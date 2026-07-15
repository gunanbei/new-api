// Package storage contains the file-storage control plane. It deliberately
// never accepts file bytes: S3 receives browser uploads through presigned URLs.
package storage

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	ticketTTL        = 15 * time.Minute
	minS3PartSize    = int64(5 * 1024 * 1024)
	maxS3Parts       = int64(10000)
	storageTicketKey = "storage:ticket:"
)

type PrepareInput struct {
	ChannelType   string `json:"channel_type"`
	FileChannelID uint64 `json:"file_channel_id"`
	FileName      string `json:"file_name"`
	FileSuffix    string `json:"file_suffix"`
	MimeType      string `json:"mime_type"`
	FileSize      int64  `json:"file_size"`
	Identifier    string `json:"identifier"`
	Source        string `json:"source"`
}

type CompleteInput struct {
	TicketID    string `json:"ticket_id"`
	UploadToken string `json:"upload_token"`
	Proof       Proof  `json:"proof"`
}

type AbortInput struct {
	TicketID    string `json:"ticket_id"`
	UploadToken string `json:"upload_token"`
}

type PresignPartInput struct {
	TicketID    string `json:"ticket_id"`
	UploadToken string `json:"upload_token"`
	PartNumber  int32  `json:"part_number"`
}

type Proof struct {
	ETag  string          `json:"etag"`
	Parts []CompletedPart `json:"parts"`
	Src   string          `json:"src"`
}

type CompletedPart struct {
	PartNumber int32  `json:"part_number"`
	ETag       string `json:"etag"`
}

type PartPlan struct {
	PartNumber int32             `json:"part_number"`
	UploadURL  string            `json:"upload_url"`
	Headers    map[string]string `json:"headers"`
	ExpiresAt  int64             `json:"expires_at"`
}

type UploadPlan struct {
	Strategy    string            `json:"strategy"`
	Method      string            `json:"method,omitempty"`
	UploadURL   string            `json:"upload_url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	UploadID    string            `json:"upload_id,omitempty"`
	PartSize    int64             `json:"part_size,omitempty"`
	PartCount   int64             `json:"part_count,omitempty"`
	Parts       []PartPlan        `json:"parts,omitempty"`
	PresignMode string            `json:"presign_mode,omitempty"`
}

type PrepareResult struct {
	Hit            bool        `json:"hit"`
	File           *FileView   `json:"file,omitempty"`
	TicketID       string      `json:"ticket_id,omitempty"`
	UploadToken    string      `json:"upload_token,omitempty"`
	ExpireAt       int64       `json:"expire_at,omitempty"`
	FileChannelID  uint64      `json:"file_channel_id,omitempty"`
	ChannelType    string      `json:"channel_type,omitempty"`
	ChunkThreshold int64       `json:"chunk_threshold,omitempty"`
	ChunkSize      int64       `json:"chunk_size,omitempty"`
	ObjectKey      string      `json:"object_key,omitempty"`
	Plan           *UploadPlan `json:"plan,omitempty"`
}

type FileView struct {
	ID            uint64    `json:"id"`
	FileID        uint64    `json:"file_id"`
	FileChannelID uint64    `json:"file_channel_id"`
	ChannelType   string    `json:"channel_type"`
	ChannelName   string    `json:"channel_name,omitempty"`
	FileName      string    `json:"file_name"`
	FileSuffix    string    `json:"file_suffix"`
	UserID        int64     `json:"user_id"`
	Username      string    `json:"username,omitempty"`
	Source        string    `json:"source"`
	Status        string    `json:"status"`
	FileSize      int64     `json:"file_size"`
	MimeType      string    `json:"mime_type"`
	Identifier    string    `json:"identifier"`
	FileURL       string    `json:"file_url"`
	ObjectKey     string    `json:"object_key"`
	RefCount      int       `json:"ref_count,omitempty"`
	CreateTime    time.Time `json:"create_time"`
	UpdateTime    time.Time `json:"update_time"`
}

type ListFilter struct {
	UserID        *int64
	FileSuffixes  []string
	Status        string
	Source        string
	Keyword       string
	ChannelType   string
	Identifier    string
	FileChannelID uint64
	Username      string
	Order         string
}

type SuffixCount struct {
	FileSuffix string `json:"file_suffix"`
	Count      int64  `json:"count"`
}

type ChannelStat struct {
	FileChannelID uint64 `json:"file_channel_id"`
	ChannelType   string `json:"channel_type"`
	ChannelName   string `json:"channel_name"`
	UserFileCount int64  `json:"user_file_count"`
	FileCount     int64  `json:"file_count"`
	TotalSize     int64  `json:"total_size"`
}

type ChannelStatsResult struct {
	Items []ChannelStat   `json:"items"`
	Sum   ChannelStatsSum `json:"sum"`
}

type ChannelStatsSum struct {
	UserFileCount int64 `json:"user_file_count"`
	FileCount     int64 `json:"file_count"`
	TotalSize     int64 `json:"total_size"`
}

type Summary struct {
	UserFileCount          int64 `json:"user_file_count"`
	FileCount              int64 `json:"file_count"`
	TotalSize              int64 `json:"total_size"`
	UploadingUserFileCount int64 `json:"uploading_user_file_count"`
}

type DeleteResult struct {
	Deleted []uint64  `json:"deleted"`
	Failed  []Failure `json:"failed"`
}

type Failure struct {
	ID      uint64 `json:"id"`
	Message string `json:"message"`
}

type ticket struct {
	ID            string `json:"id"`
	UserID        int64  `json:"user_id"`
	FileChannelID uint64 `json:"file_channel_id"`
	ChannelType   string `json:"channel_type"`
	FileName      string `json:"file_name"`
	FileSuffix    string `json:"file_suffix"`
	MimeType      string `json:"mime_type"`
	FileSize      int64  `json:"file_size"`
	Identifier    string `json:"identifier"`
	Source        string `json:"source"`
	ObjectKey     string `json:"object_key"`
	Strategy      string `json:"strategy"`
	UploadID      string `json:"upload_id"`
	PartSize      int64  `json:"part_size"`
	PartCount     int64  `json:"part_count"`
	Nonce         string `json:"nonce"`
	ExpireAt      int64  `json:"expire_at"`
}

var memoryTickets = struct {
	sync.Mutex
	items map[string]ticket
}{items: make(map[string]ticket)}

var ticketOperations = struct {
	sync.Mutex
	locks map[string]*ticketOperation
}{locks: make(map[string]*ticketOperation)}

type ticketOperation struct {
	mu      sync.Mutex
	waiters int
}

func Prepare(ctx context.Context, userID int64, input PrepareInput) (*PrepareResult, error) {
	if err := normalizePrepareInput(&input); err != nil {
		return nil, err
	}
	if input.ChannelType == service.FileUploadChannelTypeLocal {
		return nil, errors.New("Local storage channel is not supported")
	}
	channel, err := resolveChannel(input.ChannelType, input.FileChannelID)
	if err != nil {
		return nil, err
	}
	if channel.Type == service.FileUploadChannelTypeLocal {
		return nil, errors.New("Local storage channel is not supported")
	}
	if channel.MaxSize > 0 && input.FileSize > channel.MaxSize {
		return nil, errors.New("File exceeds channel max size")
	}

	var existing model.File
	err = model.DB.Where("identifier = ? AND channel_type = ? AND status = ?", input.Identifier, channel.Type, "1").First(&existing).Error
	if err == nil {
		view, err := attachExistingFile(userID, channel, &existing, input)
		if err != nil {
			return nil, err
		}
		return &PrepareResult{Hit: true, File: view}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	if channel.Type != service.FileUploadChannelTypeS3 {
		// WebDAV has no standard multipart or presigned PUT. ImgBed's documented
		// chunk API also needs its long-lived bearer token. Neither credential is
		// safe to return to a browser, so this API rejects both rather than proxying
		// bytes through a handler; S3 is the supported browser-direct path.
		return nil, errors.New("Browser direct upload is not supported for this channel")
	}

	objectKey := buildObjectKey(channel, userID, input.FileName, input.FileSuffix)
	t := ticket{
		ID: uuid.NewString(), UserID: userID, FileChannelID: channel.Id, ChannelType: channel.Type,
		FileName: input.FileName, FileSuffix: input.FileSuffix, MimeType: input.MimeType,
		FileSize: input.FileSize, Identifier: input.Identifier, Source: input.Source, ObjectKey: objectKey,
		Nonce: uuid.NewString(), ExpireAt: time.Now().Add(ticketTTL).Unix(),
	}
	plan, err := makeS3Plan(ctx, channel, &t)
	if err != nil {
		return nil, err
	}
	if err := saveTicket(t); err != nil {
		if t.UploadID != "" {
			_ = abortS3(ctx, channel, &t)
		}
		return nil, err
	}
	return &PrepareResult{
		Hit: false, TicketID: t.ID, UploadToken: ticketToken(t), ExpireAt: t.ExpireAt,
		FileChannelID: t.FileChannelID, ChannelType: t.ChannelType,
		ChunkThreshold: channel.ChunkThreshold, ChunkSize: channel.ChunkSize,
		ObjectKey: t.ObjectKey, Plan: plan,
	}, nil
}

func PresignPart(ctx context.Context, userID int64, input PresignPartInput) (*PartPlan, error) {
	t, channel, err := loadTicketForUser(input.TicketID, input.UploadToken, userID)
	if err != nil {
		return nil, err
	}
	if t.Strategy != "multipart" || input.PartNumber < 1 || int64(input.PartNumber) > t.PartCount {
		return nil, errors.New("invalid multipart part number")
	}
	return presignS3Part(ctx, channel, t, input.PartNumber)
}

func Complete(ctx context.Context, userID int64, input CompleteInput) (*FileView, error) {
	t, channel, err := loadTicketForUser(input.TicketID, input.UploadToken, userID)
	if err != nil {
		return nil, err
	}
	release, err := claimTicket(t.ID)
	if err != nil {
		return nil, err
	}
	defer release()
	if current, loadErr := loadTicket(t.ID); loadErr != nil || current == nil || !validTicketToken(*current, input.UploadToken) {
		return nil, errors.New("Upload ticket invalid or expired")
	}
	if err = completeS3(ctx, channel, t, input.Proof); err != nil {
		if t.Strategy == "multipart" {
			_ = abortS3(ctx, channel, t)
		}
		return nil, err
	}
	view, err := persistCompletedFile(channel, t, input.Proof.ETag)
	if err != nil {
		return nil, err
	}
	if err := deleteTicket(t.ID); err != nil {
		common.SysError("delete storage ticket: " + err.Error())
	}
	return view, nil
}

func Abort(ctx context.Context, userID int64, input AbortInput) error {
	t, channel, err := loadTicketForUser(input.TicketID, input.UploadToken, userID)
	if err != nil {
		return err
	}
	release, err := claimTicket(t.ID)
	if err != nil {
		return err
	}
	defer release()
	if current, loadErr := loadTicket(t.ID); loadErr != nil || current == nil || !validTicketToken(*current, input.UploadToken) {
		return errors.New("Upload ticket invalid or expired")
	}
	if t.Strategy == "multipart" {
		if err := abortS3(ctx, channel, t); err != nil {
			return err
		}
	}
	return deleteTicket(t.ID)
}

func List(filter ListFilter, offset, limit int) ([]FileView, int64, error) {
	query := listQuery(filter)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyListOrder(query, filter.Order).Offset(offset).Limit(limit)
	var items []FileView
	if err := query.Select(fileViewSelect()).Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	if err := fillChannelData(items); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func Get(id uint64, userID *int64) (*FileView, error) {
	filter := ListFilter{UserID: userID}
	query := listQuery(filter).Where("uf.id = ?", id)
	var item FileView
	if err := query.Select(fileViewSelect()).Scan(&item).Error; err != nil {
		return nil, err
	}
	if item.ID == 0 {
		return nil, errors.New("File not found")
	}
	items := []FileView{item}
	if err := fillChannelData(items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func Suffixes(filter ListFilter) ([]SuffixCount, error) {
	query := listQuery(filter).Select("uf.file_suffix, COUNT(*) AS count").Group("uf.file_suffix").Order("uf.file_suffix asc")
	var items []SuffixCount
	return items, query.Scan(&items).Error
}

func StatsByChannel(channelType, status string) (*ChannelStatsResult, error) {
	if channelType != "" && !validChannelType(channelType, true) {
		return nil, errors.New("Invalid channel_type")
	}
	fileQuery := model.DB.Table("file AS f").Select("f.file_channel_id, f.channel_type, COUNT(*) AS file_count, COALESCE(SUM(f.file_size), 0) AS total_size").Where("f.status IN ?", []string{"0", "1"})
	if channelType != "" {
		fileQuery = fileQuery.Where("f.channel_type = ?", channelType)
	}
	fileQuery = fileQuery.Group("f.file_channel_id, f.channel_type")
	var physical []ChannelStat
	if err := fileQuery.Scan(&physical).Error; err != nil {
		return nil, err
	}

	logicalQuery := model.DB.Table("user_file AS uf").Joins("JOIN file AS f ON f.id = uf.file_id").Select("uf.file_channel_id, COUNT(*) AS user_file_count").Group("uf.file_channel_id")
	if status != "" {
		logicalQuery = logicalQuery.Where("uf.status = ?", status)
	}
	if channelType != "" {
		logicalQuery = logicalQuery.Where("f.channel_type = ?", channelType)
	}
	var logical []ChannelStat
	if err := logicalQuery.Scan(&logical).Error; err != nil {
		return nil, err
	}
	byID := make(map[uint64]*ChannelStat, len(physical)+len(logical))
	for i := range physical {
		value := physical[i]
		byID[value.FileChannelID] = &value
	}
	for _, value := range logical {
		if stat := byID[value.FileChannelID]; stat != nil {
			stat.UserFileCount = value.UserFileCount
		} else {
			byID[value.FileChannelID] = &ChannelStat{FileChannelID: value.FileChannelID, UserFileCount: value.UserFileCount}
		}
	}
	channels, err := channelMap(byID)
	if err != nil {
		return nil, err
	}
	result := &ChannelStatsResult{Items: make([]ChannelStat, 0, len(byID))}
	for id, stat := range byID {
		if channel := channels[id]; channel != nil {
			stat.ChannelName, stat.ChannelType = channel.Name, channel.Type
		}
		result.Items = append(result.Items, *stat)
		result.Sum.UserFileCount += stat.UserFileCount
		result.Sum.FileCount += stat.FileCount
		result.Sum.TotalSize += stat.TotalSize
	}
	sort.Slice(result.Items, func(i, j int) bool { return result.Items[i].FileChannelID < result.Items[j].FileChannelID })
	return result, nil
}

func GetSummary() (*Summary, error) {
	result := &Summary{}
	if err := model.DB.Model(&model.UserFile{}).Count(&result.UserFileCount).Error; err != nil {
		return nil, err
	}
	if err := model.DB.Model(&model.UserFile{}).Where("status = ?", "0").Count(&result.UploadingUserFileCount).Error; err != nil {
		return nil, err
	}
	row := model.DB.Model(&model.File{}).Select("COUNT(*) AS file_count, COALESCE(SUM(file_size), 0) AS total_size").Where("status IN ?", []string{"0", "1"}).Scan(result)
	return result, row.Error
}

func Delete(ctx context.Context, id uint64, userID *int64) (*FileView, error) {
	var removed *FileView
	var physical *model.File
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		query := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id)
		if userID != nil {
			query = query.Where("user_id = ?", *userID)
		}
		var userFile model.UserFile
		if err := query.First(&userFile).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("File not found")
			}
			return err
		}
		var file model.File
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&file, userFile.FileId).Error; err != nil {
			return err
		}
		if tx.Migrator().HasTable(&model.CreativeTaskAsset{}) {
			var outputCount int64
			if err := tx.Model(&model.CreativeTaskAsset{}).Where("user_file_id = ? AND role = ?", userFile.Id, "output").Count(&outputCount).Error; err != nil {
				return err
			}
			if outputCount > 0 {
				return errors.New("creative output files cannot be deleted")
			}
		}
		if err := tx.Delete(&userFile).Error; err != nil {
			return err
		}
		file.RefCount--
		if file.RefCount < 0 {
			file.RefCount = 0
		}
		if file.RefCount == 0 {
			file.Status = "2"
			physical = &file
		}
		if err := tx.Save(&file).Error; err != nil {
			return err
		}
		removed = &FileView{ID: userFile.Id, FileID: file.Id, UserID: userFile.UserId}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if physical != nil {
		if err := deletePhysical(ctx, physical); err != nil {
			common.SysError(fmt.Sprintf("delete physical file %d failed: %v", physical.Id, err))
		} else if err := model.DB.Where("id = ? AND ref_count = ?", physical.Id, 0).Delete(&model.File{}).Error; err != nil {
			common.SysError(fmt.Sprintf("remove deleted physical file %d failed: %v", physical.Id, err))
		}
	}
	return removed, nil
}

func DeleteMany(ctx context.Context, ids []uint64, userID *int64) (*DeleteResult, error) {
	if len(ids) > 100 {
		return nil, errors.New("Too many ids")
	}
	result := &DeleteResult{Deleted: make([]uint64, 0, len(ids)), Failed: make([]Failure, 0)}
	seen := make(map[uint64]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			result.Failed = append(result.Failed, Failure{ID: id, Message: "File not found"})
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if _, err := Delete(ctx, id, userID); err != nil {
			result.Failed = append(result.Failed, Failure{ID: id, Message: err.Error()})
			continue
		}
		result.Deleted = append(result.Deleted, id)
	}
	return result, nil
}

func normalizePrepareInput(input *PrepareInput) error {
	input.FileName = strings.TrimSpace(input.FileName)
	input.FileSuffix = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(input.FileSuffix), "."))
	input.MimeType = strings.TrimSpace(input.MimeType)
	input.Identifier = strings.ToLower(strings.TrimSpace(input.Identifier))
	input.Source = strings.TrimSpace(input.Source)
	input.ChannelType = strings.TrimSpace(input.ChannelType)
	if input.FileName == "" || input.FileSuffix == "" || input.MimeType == "" || input.FileSize <= 0 || input.Identifier == "" {
		return errors.New("invalid upload request")
	}
	if len(input.FileName) > 1000 || len(input.FileSuffix) > 255 || len(input.MimeType) > 128 || len(input.Identifier) > 255 || len(input.Source) > 64 {
		return errors.New("invalid upload request")
	}
	if len(input.Identifier) != 32 {
		return errors.New("identifier must be an MD5 hex string")
	}
	if _, err := hex.DecodeString(input.Identifier); err != nil {
		return errors.New("identifier must be an MD5 hex string")
	}
	if input.ChannelType != "" && !validChannelType(input.ChannelType, true) {
		return errors.New("Invalid channel_type")
	}
	return nil
}

func validChannelType(value string, includeLocal bool) bool {
	if includeLocal && value == service.FileUploadChannelTypeLocal {
		return true
	}
	return value == service.FileUploadChannelTypeWebDAV || value == service.FileUploadChannelTypeCloudflareImageBed || value == service.FileUploadChannelTypeS3
}

func resolveChannel(channelType string, preferredID uint64) (*model.FileUploadChannel, error) {
	if preferredID > 0 {
		var channel model.FileUploadChannel
		if err := model.DB.First(&channel, preferredID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("No default file upload channel")
			}
			return nil, err
		}
		if channel.Status != service.FileUploadChannelStatusEnabled {
			return nil, errors.New("File upload channel is disabled")
		}
		if channelType != "" && channel.Type != channelType {
			return nil, errors.New("Invalid channel_type")
		}
		return &channel, nil
	}
	query := model.DB.Where("status = ? AND is_default = ?", service.FileUploadChannelStatusEnabled, service.FileUploadChannelIsDefault)
	if channelType != "" {
		query = query.Where("type = ?", channelType)
	}
	var channel model.FileUploadChannel
	if err := query.Order("id asc").First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("No default file upload channel")
		}
		return nil, err
	}
	return &channel, nil
}

func attachExistingFile(userID int64, channel *model.FileUploadChannel, file *model.File, input PrepareInput) (*FileView, error) {
	var userFile model.UserFile
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("user_id = ? AND file_id = ?", userID, file.Id).First(&userFile).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		userFile = model.UserFile{FileId: file.Id, FileChannelId: file.FileChannelId, UserId: userID, FileName: input.FileName, FileSuffix: input.FileSuffix, Source: input.Source, Status: "1"}
		if err := tx.Create(&userFile).Error; err != nil {
			return err
		}
		return tx.Model(&model.File{}).Where("id = ?", file.Id).UpdateColumn("ref_count", gorm.Expr("ref_count + ?", 1)).Error
	})
	if err != nil {
		return nil, err
	}
	view := &FileView{ID: userFile.Id, FileID: file.Id, FileChannelID: file.FileChannelId, ChannelType: file.ChannelType, FileName: userFile.FileName, FileSuffix: userFile.FileSuffix, UserID: userID, Source: userFile.Source, Status: userFile.Status, FileSize: file.FileSize, MimeType: file.MimeType, Identifier: file.Identifier, FileURL: file.FileUrl, ObjectKey: file.ObjectKey, RefCount: file.RefCount, CreateTime: userFile.CreateTime, UpdateTime: userFile.UpdateTime}
	items := []FileView{*view}
	if err := fillChannelData(items); err != nil {
		return nil, err
	}
	return &items[0], nil
}

func buildObjectKey(channel *model.FileUploadChannel, userID int64, fileName, suffix string) string {
	config, _ := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	prefix := strings.Trim(strings.TrimSpace(stringValue(config["key_prefix"])), "/")
	if prefix == "" {
		prefix = strings.Trim(strings.TrimSpace(stringValue(config["path_prefix"])), "/")
	}
	cleanName := strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, fileName)
	if strings.EqualFold(path.Ext(cleanName), "."+suffix) {
		return strings.Trim(prefix+"/"+time.Now().UTC().Format("2006/01/02")+"/"+strconv.FormatInt(userID, 10)+"/"+uuid.NewString()+"_"+cleanName, "/")
	}
	return strings.Trim(prefix+"/"+time.Now().UTC().Format("2006/01/02")+"/"+strconv.FormatInt(userID, 10)+"/"+uuid.NewString()+"_"+cleanName+"."+suffix, "/")
}

func makeS3Plan(ctx context.Context, channel *model.FileUploadChannel, t *ticket) (*UploadPlan, error) {
	if t.FileSize > channel.ChunkThreshold {
		partSize := channel.ChunkSize
		if partSize < minS3PartSize {
			return nil, fmt.Errorf("S3 chunk_size must be at least %d bytes", minS3PartSize)
		}
		partCount := int64(math.Ceil(float64(t.FileSize) / float64(partSize)))
		if partCount > maxS3Parts {
			return nil, errors.New("file requires too many multipart parts")
		}
		client, bucket, err := s3Client(channel)
		if err != nil {
			return nil, err
		}
		output, err := client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey), ContentType: aws.String(t.MimeType)})
		if err != nil {
			return nil, fmt.Errorf("create S3 multipart upload: %w", err)
		}
		if output.UploadId == nil || *output.UploadId == "" {
			return nil, errors.New("S3 did not return multipart upload id")
		}
		t.Strategy, t.UploadID, t.PartSize, t.PartCount = "multipart", *output.UploadId, partSize, partCount
		first, err := presignS3Part(ctx, channel, t, 1)
		if err != nil {
			_ = abortS3(ctx, channel, t)
			return nil, err
		}
		return &UploadPlan{Strategy: "multipart", UploadID: t.UploadID, PartSize: partSize, PartCount: partCount, Parts: []PartPlan{*first}, PresignMode: "lazy"}, nil
	}
	t.Strategy = "single"
	client, bucket, err := s3Client(channel)
	if err != nil {
		return nil, err
	}
	presigned, err := s3.NewPresignClient(client).PresignPutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey), ContentType: aws.String(t.MimeType)}, func(options *s3.PresignOptions) { options.Expires = ticketTTL })
	if err != nil {
		return nil, fmt.Errorf("presign S3 upload: %w", err)
	}
	return &UploadPlan{Strategy: "single", Method: http.MethodPut, UploadURL: presigned.URL, Headers: headersFromHTTP(presigned.SignedHeader), PresignMode: "eager"}, nil
}

func presignS3Part(ctx context.Context, channel *model.FileUploadChannel, t *ticket, partNumber int32) (*PartPlan, error) {
	client, bucket, err := s3Client(channel)
	if err != nil {
		return nil, err
	}
	presigned, err := s3.NewPresignClient(client).PresignUploadPart(ctx, &s3.UploadPartInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey), UploadId: aws.String(t.UploadID), PartNumber: aws.Int32(partNumber)}, func(options *s3.PresignOptions) { options.Expires = ticketTTL })
	if err != nil {
		return nil, fmt.Errorf("presign S3 multipart part: %w", err)
	}
	return &PartPlan{PartNumber: partNumber, UploadURL: presigned.URL, Headers: headersFromHTTP(presigned.SignedHeader), ExpiresAt: t.ExpireAt}, nil
}

func completeS3(ctx context.Context, channel *model.FileUploadChannel, t *ticket, proof Proof) error {
	client, bucket, err := s3Client(channel)
	if err != nil {
		return err
	}
	if t.Strategy == "multipart" {
		if int64(len(proof.Parts)) != t.PartCount {
			return errors.New("invalid multipart completion proof")
		}
		parts := make([]types.CompletedPart, 0, len(proof.Parts))
		seen := make(map[int32]struct{}, len(proof.Parts))
		for _, part := range proof.Parts {
			if part.PartNumber < 1 || int64(part.PartNumber) > t.PartCount || strings.TrimSpace(part.ETag) == "" {
				return errors.New("invalid multipart completion proof")
			}
			if _, ok := seen[part.PartNumber]; ok {
				return errors.New("invalid multipart completion proof")
			}
			seen[part.PartNumber] = struct{}{}
			parts = append(parts, types.CompletedPart{PartNumber: aws.Int32(part.PartNumber), ETag: aws.String(part.ETag)})
		}
		sort.Slice(parts, func(i, j int) bool { return *parts[i].PartNumber < *parts[j].PartNumber })
		if _, err := client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey), UploadId: aws.String(t.UploadID), MultipartUpload: &types.CompletedMultipartUpload{Parts: parts}}); err != nil {
			return fmt.Errorf("complete S3 multipart upload: %w", err)
		}
	}
	head, err := client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey)})
	if err != nil {
		return fmt.Errorf("verify S3 upload: %w", err)
	}
	if head.ContentLength == nil || *head.ContentLength != t.FileSize {
		return errors.New("uploaded file size does not match prepare request")
	}
	return nil
}

func abortS3(ctx context.Context, channel *model.FileUploadChannel, t *ticket) error {
	client, bucket, err := s3Client(channel)
	if err != nil {
		return err
	}
	_, err = client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{Bucket: aws.String(bucket), Key: aws.String(t.ObjectKey), UploadId: aws.String(t.UploadID)})
	if err != nil {
		return fmt.Errorf("abort S3 multipart upload: %w", err)
	}
	return nil
}

func s3Client(channel *model.FileUploadChannel) (*s3.Client, string, error) {
	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return nil, "", err
	}
	region, bucket, accessKeyID, secretAccessKey := strings.TrimSpace(stringValue(config["region"])), strings.TrimSpace(stringValue(config["bucket"])), strings.TrimSpace(stringValue(config["access_key_id"])), strings.TrimSpace(stringValue(config["secret_access_key"]))
	if region == "" || bucket == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, "", errors.New("S3 channel configuration is incomplete")
	}
	awsConfig := aws.Config{Region: region, Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, ""))}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = boolValue(config["force_path_style"])
		if endpoint := strings.TrimSpace(stringValue(config["endpoint"])); endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})
	return client, bucket, nil
}

func persistCompletedFile(channel *model.FileUploadChannel, t *ticket, etag string) (*FileView, error) {
	var userFile model.UserFile
	var physical model.File
	err := model.DB.Transaction(func(tx *gorm.DB) error {
		err := tx.Where("identifier = ? AND channel_type = ?", t.Identifier, t.ChannelType).First(&physical).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			physical = model.File{FileChannelId: t.FileChannelID, ChannelType: t.ChannelType, FileSize: t.FileSize, ObjectKey: t.ObjectKey, FileUrl: publicURL(channel, t.ObjectKey), Identifier: t.Identifier, MimeType: t.MimeType, ETag: etag, Status: "1", CreaterUserId: t.UserID}
			if err := tx.Create(&physical).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		err = tx.Where("user_id = ? AND file_id = ?", t.UserID, physical.Id).First(&userFile).Error
		if err == nil {
			return nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		userFile = model.UserFile{FileId: physical.Id, FileChannelId: physical.FileChannelId, FileName: t.FileName, FileSuffix: t.FileSuffix, UserId: t.UserID, Source: t.Source, Status: "1"}
		if err := tx.Create(&userFile).Error; err != nil {
			return err
		}
		return tx.Model(&model.File{}).Where("id = ?", physical.Id).UpdateColumn("ref_count", gorm.Expr("ref_count + ?", 1)).Error
	})
	if err != nil {
		return nil, err
	}
	return Get(userFile.Id, &t.UserID)
}

func deletePhysical(ctx context.Context, file *model.File) error {
	var channel model.FileUploadChannel
	if err := model.DB.First(&channel, file.FileChannelId).Error; err != nil {
		return err
	}
	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return err
	}
	switch file.ChannelType {
	case service.FileUploadChannelTypeS3:
		client, bucket, err := s3Client(&channel)
		if err != nil {
			return err
		}
		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(file.ObjectKey)})
		return err
	case service.FileUploadChannelTypeWebDAV:
		target, err := joinURL(stringValue(config["base_url"]), file.ObjectKey)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
		if err != nil {
			return err
		}
		applyWebDAVAuth(req, config)
		client := &http.Client{Timeout: webDAVTimeout(config)}
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode >= 200 && response.StatusCode < 300 || response.StatusCode == http.StatusNotFound {
			return nil
		}
		return fmt.Errorf("WebDAV delete returned status %d", response.StatusCode)
	case service.FileUploadChannelTypeCloudflareImageBed:
		baseURL := strings.TrimRight(strings.TrimSpace(stringValue(config["base_url"])), "/")
		token := strings.TrimSpace(stringValue(config["api_token"]))
		if baseURL == "" || token == "" {
			return errors.New("ImgBed channel configuration is incomplete")
		}
		requestURL := baseURL + "/api/manage/delete/" + strings.ReplaceAll(url.PathEscape(imageBedObjectKey(config, file.ObjectKey)), "%2F", "/")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return nil
		}
		return fmt.Errorf("ImgBed delete returned status %d", response.StatusCode)
	default:
		return errors.New("unsupported file channel type")
	}
}

func listQuery(filter ListFilter) *gorm.DB {
	query := model.DB.Table("user_file AS uf").Joins("JOIN file AS f ON f.id = uf.file_id").Joins("LEFT JOIN users AS u ON u.id = uf.user_id")
	if filter.UserID != nil {
		query = query.Where("uf.user_id = ?", *filter.UserID)
	}
	if len(filter.FileSuffixes) > 0 {
		query = query.Where("uf.file_suffix IN ?", filter.FileSuffixes)
	}
	if filter.Status != "" {
		query = query.Where("uf.status = ?", filter.Status)
	}
	if filter.Source != "" {
		query = query.Where("uf.source = ?", filter.Source)
	}
	if filter.Keyword != "" {
		query = query.Where("uf.file_name LIKE ?", "%"+filter.Keyword+"%")
	}
	if filter.ChannelType != "" {
		query = query.Where("f.channel_type = ?", filter.ChannelType)
	}
	if filter.Identifier != "" {
		query = query.Where("f.identifier = ?", filter.Identifier)
	}
	if filter.FileChannelID > 0 {
		query = query.Where("uf.file_channel_id = ?", filter.FileChannelID)
	}
	if filter.Username != "" {
		query = query.Where("u.username LIKE ?", filter.Username+"%")
	}
	return query
}

func applyListOrder(query *gorm.DB, order string) *gorm.DB {
	switch order {
	case "create_time_asc":
		return query.Order("uf.create_time asc")
	case "file_size_desc":
		return query.Order("f.file_size desc, uf.id desc")
	default:
		return query.Order("uf.create_time desc, uf.id desc")
	}
}

func fileViewSelect() string {
	return "uf.id, uf.file_id, uf.file_channel_id, f.channel_type, uf.file_name, uf.file_suffix, uf.user_id, COALESCE(u.username, '') AS username, uf.source, uf.status, f.file_size, f.mime_type, f.identifier, f.file_url, f.object_key, f.ref_count, uf.create_time, uf.update_time"
}

func fillChannelData(items []FileView) error {
	ids := make(map[uint64]*ChannelStat)
	for i := range items {
		ids[items[i].FileChannelID] = &ChannelStat{}
	}
	channels, err := channelMap(ids)
	if err != nil {
		return err
	}
	for i := range items {
		channel := channels[items[i].FileChannelID]
		if channel == nil {
			continue
		}
		items[i].ChannelName = channel.Name
		if items[i].FileURL == "" {
			items[i].FileURL = publicURL(channel, items[i].ObjectKey)
		}
	}
	return nil
}

func channelMap(ids map[uint64]*ChannelStat) (map[uint64]*model.FileUploadChannel, error) {
	result := make(map[uint64]*model.FileUploadChannel, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	keys := make([]uint64, 0, len(ids))
	for id := range ids {
		keys = append(keys, id)
	}
	var channels []model.FileUploadChannel
	if err := model.DB.Where("id IN ?", keys).Find(&channels).Error; err != nil {
		return nil, err
	}
	for i := range channels {
		result[channels[i].Id] = &channels[i]
	}
	return result, nil
}

func publicURL(channel *model.FileUploadChannel, objectKey string) string {
	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return ""
	}
	base := strings.TrimSpace(stringValue(config["public_base_url"]))
	if base == "" {
		return ""
	}
	value, err := joinURL(base, objectKey)
	if err != nil {
		return ""
	}
	return value
}

func imageBedObjectKey(config map[string]any, objectKey string) string {
	root := strings.Trim(strings.TrimSpace(stringValue(config["upload_folder"])), "/")
	objectKey = strings.Trim(objectKey, "/")
	if root == "" {
		return objectKey
	}
	return path.Join(root, objectKey)
}

func saveTicket(value ticket) error {
	data, err := common.Marshal(value)
	if err != nil {
		return err
	}
	if common.RedisEnabled && common.RDB != nil {
		return common.RedisSet(storageTicketKey+value.ID, string(data), time.Until(time.Unix(value.ExpireAt, 0)))
	}
	memoryTickets.Lock()
	defer memoryTickets.Unlock()
	memoryTickets.items[value.ID] = value
	return nil
}

func loadTicketForUser(id, token string, userID int64) (*ticket, *model.FileUploadChannel, error) {
	value, err := loadTicket(id)
	if err != nil || value == nil || value.UserID != userID || !validTicketToken(*value, token) || value.ExpireAt < time.Now().Unix() {
		return nil, nil, errors.New("Upload ticket invalid or expired")
	}
	var channel model.FileUploadChannel
	if err := model.DB.First(&channel, value.FileChannelID).Error; err != nil {
		return nil, nil, errors.New("Upload ticket invalid or expired")
	}
	return value, &channel, nil
}

func loadTicket(id string) (*ticket, error) {
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("missing ticket")
	}
	if common.RedisEnabled && common.RDB != nil {
		raw, err := common.RedisGet(storageTicketKey + id)
		if err != nil {
			return nil, err
		}
		var value ticket
		if err := common.UnmarshalJsonStr(raw, &value); err != nil {
			return nil, err
		}
		return &value, nil
	}
	memoryTickets.Lock()
	defer memoryTickets.Unlock()
	value, ok := memoryTickets.items[id]
	if !ok {
		return nil, errors.New("ticket not found")
	}
	if value.ExpireAt < time.Now().Unix() {
		delete(memoryTickets.items, id)
		return nil, errors.New("ticket expired")
	}
	return &value, nil
}

func deleteTicket(id string) error {
	if common.RedisEnabled && common.RDB != nil {
		return common.RedisDel(storageTicketKey + id)
	}
	memoryTickets.Lock()
	defer memoryTickets.Unlock()
	delete(memoryTickets.items, id)
	return nil
}

func claimTicket(id string) (func(), error) {
	if common.RedisEnabled && common.RDB != nil {
		key := storageTicketKey + "claim:" + id
		claimed, err := common.RDB.SetNX(context.Background(), key, "1", ticketTTL).Result()
		if err != nil || !claimed {
			return nil, errors.New("Upload ticket invalid or expired")
		}
		return func() { _ = common.RDB.Del(context.Background(), key).Err() }, nil
	}
	ticketOperations.Lock()
	operation := ticketOperations.locks[id]
	if operation == nil {
		operation = &ticketOperation{}
		ticketOperations.locks[id] = operation
	}
	operation.waiters++
	ticketOperations.Unlock()
	operation.mu.Lock()
	return func() {
		operation.mu.Unlock()
		ticketOperations.Lock()
		operation.waiters--
		if operation.waiters == 0 && ticketOperations.locks[id] == operation {
			delete(ticketOperations.locks, id)
		}
		ticketOperations.Unlock()
	}, nil
}

func ticketToken(value ticket) string {
	payload := value.ID + "|" + strconv.FormatInt(value.UserID, 10) + "|" + strconv.FormatInt(value.ExpireAt, 10) + "|" + value.Nonce
	h := hmac.New(sha256.New, []byte(common.CryptoSecret))
	_, _ = h.Write([]byte(payload))
	return hex.EncodeToString(h.Sum(nil))
}

func validTicketToken(value ticket, token string) bool {
	return hmac.Equal([]byte(ticketToken(value)), []byte(strings.TrimSpace(token)))
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if result, ok := value.(string); ok {
		return result
	}
	return fmt.Sprint(value)
}
func boolValue(value any) bool {
	parsed, _ := strconv.ParseBool(strings.TrimSpace(stringValue(value)))
	return parsed
}

func joinURL(base, objectKey string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("invalid URL")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + strings.TrimLeft(objectKey, "/")
	return parsed.String(), nil
}

func applyWebDAVAuth(req *http.Request, config map[string]any) {
	if strings.EqualFold(strings.TrimSpace(stringValue(config["auth_type"])), "bearer") {
		req.Header.Set("Authorization", "Bearer "+stringValue(config["password"]))
		return
	}
	req.SetBasicAuth(stringValue(config["username"]), stringValue(config["password"]))
}

func webDAVTimeout(config map[string]any) time.Duration {
	value, err := strconv.ParseInt(strings.TrimSpace(stringValue(config["timeout_ms"])), 10, 64)
	if err != nil || value <= 0 {
		return 10 * time.Minute
	}
	return time.Duration(value) * time.Millisecond
}

func headersFromHTTP(headers http.Header) map[string]string {
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if strings.EqualFold(key, "Host") {
			continue
		}
		result[key] = strings.Join(values, ",")
	}
	return result
}
