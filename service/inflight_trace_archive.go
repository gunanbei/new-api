package service

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

const (
	inflightTraceStorageModeOptionKey            = "InflightTaskTraceStorageMode"
	inflightTraceArchiveChannelIDOptionKey       = "InflightTaskTraceArchiveChannelID"
	inflightTraceArchiveThresholdBytesOptionKey  = "InflightTaskTraceArchiveThresholdBytes"
	inflightTraceArchiveRetentionYearsOptionKey  = "InflightTaskTraceArchiveRetentionYears"
	inflightTraceArchiveRetentionMonthsOptionKey = "InflightTaskTraceArchiveRetentionMonths"
	inflightTraceArchiveRetentionDaysOptionKey   = "InflightTaskTraceArchiveRetentionDays"
	inflightTraceArchiveRetentionHoursOptionKey  = "InflightTaskTraceArchiveRetentionHours"
	inflightTraceArchiveStatusActive             = "active"
	inflightTraceArchiveStatusUploading          = "uploading"
	inflightTraceArchiveStatusUploaded           = "uploaded"
	inflightTraceArchiveStatusFailed             = "failed"
)

var inflightTraceArchiveLocks sync.Map

func inflightTaskTraceStorageMode() string {
	if inflightTraceOption(inflightTraceStorageModeOptionKey) == "disk" {
		return "disk"
	}
	return "memory"
}

func inflightTraceOption(key string) string {
	common.OptionMapRWMutex.RLock()
	value := common.OptionMap[key]
	common.OptionMapRWMutex.RUnlock()
	return strings.TrimSpace(value)
}

func inflightTraceOptionInt64(key string) int64 {
	value, err := strconv.ParseInt(inflightTraceOption(key), 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func ValidateInflightTraceStorageOption(key, raw string) error {
	value := strings.TrimSpace(raw)
	switch key {
	case inflightTraceStorageModeOptionKey:
		if value != "memory" && value != "disk" {
			return errors.New("inflight trace storage mode must be memory or disk")
		}
	case inflightTraceArchiveChannelIDOptionKey:
		channelID, err := strconv.ParseUint(value, 10, 64)
		if err != nil || (value != "0" && channelID == 0) {
			return errors.New("invalid inflight trace archive storage channel")
		}
	case inflightTraceArchiveThresholdBytesOptionKey:
		threshold, err := strconv.ParseInt(value, 10, 64)
		if err != nil || threshold < 1 {
			return errors.New("inflight trace archive threshold must be at least 1 byte")
		}
	case inflightTraceArchiveRetentionYearsOptionKey, inflightTraceArchiveRetentionMonthsOptionKey, inflightTraceArchiveRetentionDaysOptionKey, inflightTraceArchiveRetentionHoursOptionKey:
		amount, err := strconv.ParseInt(value, 10, 64)
		if err != nil || amount < 0 || amount > 100000 {
			return errors.New("invalid inflight trace archive retention value")
		}
	}
	return nil
}

func inflightTraceArchiveLock(userID int) *sync.Mutex {
	value, _ := inflightTraceArchiveLocks.LoadOrStore(userID, &sync.Mutex{})
	return value.(*sync.Mutex)
}

func inflightTraceArchiveDir() string {
	if path := strings.TrimSpace(common.GetInflightTracePath()); path != "" {
		return filepath.Join(path, "inflight-traces")
	}
	return filepath.Join(common.GetDiskCacheDir(), "inflight-traces")
}

type InflightTraceArchiveStats struct {
	Directory          string
	FileCount          int64
	TotalSize          int64
	PendingUploadCount int64
}

func GetInflightTraceArchiveStats() (InflightTraceArchiveStats, error) {
	stats := InflightTraceArchiveStats{Directory: inflightTraceArchiveDir()}
	err := filepath.WalkDir(stats.Directory, func(_ string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		stats.FileCount++
		stats.TotalSize += info.Size()
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		err = nil
	}
	if err != nil {
		return stats, err
	}
	if err = model.DB.Model(&model.InflightTraceArchive{}).Where("status IN ?", []string{inflightTraceArchiveStatusUploading, inflightTraceArchiveStatusFailed}).Count(&stats.PendingUploadCount).Error; err != nil {
		return stats, err
	}
	return stats, nil
}

func TriggerInflightTraceArchiveUploads() (int64, error) {
	var archives []model.InflightTraceArchive
	if err := model.DB.Where("status IN ?", []string{inflightTraceArchiveStatusActive, inflightTraceArchiveStatusFailed}).Find(&archives).Error; err != nil {
		return 0, err
	}
	var started int64
	for index := range archives {
		archive := archives[index]
		lock := inflightTraceArchiveLock(archive.UserID)
		lock.Lock()
		if err := model.DB.Model(&model.InflightTraceArchive{}).Where("id = ? AND status IN ?", archive.ID, []string{inflightTraceArchiveStatusActive, inflightTraceArchiveStatusFailed}).Update("status", inflightTraceArchiveStatusUploading).Error; err == nil {
			started++
			common.SysLog(fmt.Sprintf("manual inflight trace archive upload started: id=%d file=%s", archive.ID, archive.FileName))
			gopool.Go(func() { uploadInflightTraceArchive(archive.ID) })
		}
		lock.Unlock()
	}
	return started, nil
}

func persistInflightTraceToDisk(ctx context.Context, trace *InflightTaskTrace) error {
	if trace == nil || trace.UserID <= 0 || trace.RequestID == "" {
		return nil
	}
	lock := inflightTraceArchiveLock(trace.UserID)
	lock.Lock()
	defer lock.Unlock()
	if err := os.MkdirAll(inflightTraceArchiveDir(), 0700); err != nil {
		return err
	}
	archive, err := activeInflightTraceArchive(trace.UserID)
	if err != nil {
		return err
	}
	if archive == nil {
		archive, err = createInflightTraceArchive(trace.UserID)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(archive.LocalPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	data, err := common.Marshal(trace)
	if err == nil {
		writer := csv.NewWriter(file)
		if err = writer.Write([]string{trace.RequestID, strconv.FormatInt(trace.RecordedAt, 10), string(data)}); err == nil {
			writer.Flush()
			err = writer.Error()
		}
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	trace.ArchiveID = archive.ID
	trace.StorageMode = "disk"
	archive.LatestRecordedAt = trace.RecordedAt
	if err = model.DB.Model(archive).Update("latest_recorded_at", archive.LatestRecordedAt).Error; err != nil {
		return err
	}
	if threshold := inflightTraceOptionInt64(inflightTraceArchiveThresholdBytesOptionKey); threshold > 0 {
		if info, statErr := os.Stat(archive.LocalPath); statErr == nil && info.Size() >= threshold {
			archive.Status = inflightTraceArchiveStatusUploading
			if err = model.DB.Save(archive).Error; err != nil {
				return err
			}
			gopool.Go(func() { uploadInflightTraceArchive(archive.ID) })
		}
	}
	return nil
}

func activeInflightTraceArchive(userID int) (*model.InflightTraceArchive, error) {
	var archive model.InflightTraceArchive
	err := model.DB.Where("user_id = ? AND status = ?", userID, inflightTraceArchiveStatusActive).Order("id desc").First(&archive).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err != nil {
		return nil, nil
	}
	return &archive, nil
}

func createInflightTraceArchive(userID int) (*model.InflightTraceArchive, error) {
	now := time.Now()
	baseName := fmt.Sprintf("Inflight_%d_%s", userID, now.Format("20060102150405"))
	var fileName string
	var path string
	var file *os.File
	var err error
	for sequence := 0; ; sequence++ {
		fileName = baseName + ".csv"
		if sequence > 0 {
			fileName = fmt.Sprintf("%s_%d.csv", baseName, sequence)
		}
		var existing int64
		if err = model.DB.Model(&model.InflightTraceArchive{}).Where("user_id = ? AND file_name = ?", userID, fileName).Count(&existing).Error; err != nil {
			return nil, err
		}
		if existing > 0 {
			continue
		}
		path = filepath.Join(inflightTraceArchiveDir(), fileName)
		file, err = os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if errors.Is(err, os.ErrExist) {
			continue
		}
		if err != nil {
			return nil, err
		}
		break
	}
	writer := csv.NewWriter(file)
	err = writer.Write([]string{"request_id", "recorded_at", "trace_json"})
	writer.Flush()
	if err == nil {
		err = writer.Error()
	}
	if closeErr := file.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	archive := &model.InflightTraceArchive{UserID: userID, FileName: fileName, LocalPath: path, StorageChannelID: uint64(inflightTraceOptionInt64(inflightTraceArchiveChannelIDOptionKey)), Status: inflightTraceArchiveStatusActive, CreatedAt: now.Unix()}
	if err = model.DB.Create(archive).Error; err != nil {
		_ = os.Remove(path)
		return nil, err
	}
	return archive, nil
}

func loadInflightTraceFromCSV(filePath, requestID string) (*InflightTaskTrace, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	if _, err = reader.Read(); err != nil {
		return nil, err
	}
	var result *InflightTaskTrace
	for {
		record, readErr := reader.Read()
		if errors.Is(readErr, io.EOF) {
			return result, nil
		}
		if readErr != nil {
			return nil, readErr
		}
		if len(record) != 3 || record[0] != requestID {
			continue
		}
		trace := &InflightTaskTrace{}
		if err = common.UnmarshalJsonStr(record[2], trace); err == nil {
			result = trace
		}
	}
}

func uploadInflightTraceArchive(archiveID uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	var archive model.InflightTraceArchive
	if err := model.DB.First(&archive, archiveID).Error; err != nil {
		return
	}
	channelID := archive.StorageChannelID
	if channelID == 0 {
		archive.Status, archive.LastError = inflightTraceArchiveStatusFailed, "archive storage channel is required"
		_ = model.DB.Save(&archive).Error
		return
	}
	for attempt := 1; attempt <= 3; attempt++ {
		objectKey, url, err := uploadInflightTraceArchiveFile(ctx, channelID, archive.FileName, archive.LocalPath)
		if err == nil {
			archive.StorageChannelID, archive.ObjectKey, archive.RemoteURL = channelID, objectKey, url
			archive.Status, archive.UploadedAt, archive.RetryCount, archive.LastError = inflightTraceArchiveStatusUploaded, time.Now().Unix(), attempt, ""
			_ = model.DB.Save(&archive).Error
			if url != "" {
				_ = os.Remove(archive.LocalPath)
			}
			common.SysLog(fmt.Sprintf("inflight trace archive uploaded: id=%d file=%s", archive.ID, archive.FileName))
			return
		}
		archive.RetryCount, archive.LastError = attempt, err.Error()
		_ = model.DB.Save(&archive).Error
		common.SysError(fmt.Sprintf("inflight trace archive upload failed: id=%d attempt=%d err=%v", archive.ID, attempt, err))
	}
	archive.Status = inflightTraceArchiveStatusFailed
	_ = model.DB.Save(&archive).Error
}

func OpenInflightTraceArchive(ctx context.Context, userID int, archiveID uint64) (io.ReadCloser, string, error) {
	var archive model.InflightTraceArchive
	if err := model.DB.Where("id = ? AND user_id = ?", archiveID, userID).First(&archive).Error; err != nil {
		return nil, "", errors.New("inflight trace archive not found")
	}
	if archive.LocalPath != "" {
		file, err := os.Open(archive.LocalPath)
		if err == nil {
			return file, archive.FileName, nil
		}
	}
	if archive.RemoteURL == "" {
		return nil, "", errors.New("inflight trace archive is not available for download")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, archive.RemoteURL, nil)
	if err != nil {
		return nil, "", err
	}
	response, err := (&http.Client{Timeout: 10 * time.Minute}).Do(request)
	if err != nil {
		return nil, "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		response.Body.Close()
		return nil, "", fmt.Errorf("archive download returned status %d", response.StatusCode)
	}
	return response.Body, archive.FileName, nil
}

func CleanupExpiredInflightTraceArchives(ctx context.Context) error {
	years := int(inflightTraceOptionInt64(inflightTraceArchiveRetentionYearsOptionKey))
	months := int(inflightTraceOptionInt64(inflightTraceArchiveRetentionMonthsOptionKey))
	days := int(inflightTraceOptionInt64(inflightTraceArchiveRetentionDaysOptionKey))
	hours := int(inflightTraceOptionInt64(inflightTraceArchiveRetentionHoursOptionKey))
	if years == 0 && months == 0 && days == 0 && hours == 0 {
		return nil
	}
	cutoff := time.Now().AddDate(-years, -months, -days).Add(-time.Duration(hours) * time.Hour).Unix()
	var archives []model.InflightTraceArchive
	if err := model.DB.Where("created_at <= ?", cutoff).Find(&archives).Error; err != nil {
		return err
	}
	for _, archive := range archives {
		if err := deleteInflightTraceArchive(ctx, &archive); err != nil {
			return err
		}
	}
	return nil
}

func DeleteInflightTraceArchivesBefore(ctx context.Context, targetTimestamp int64) (int64, error) {
	var archives []model.InflightTraceArchive
	if err := model.DB.Where("(latest_recorded_at > 0 AND latest_recorded_at <= ?) OR (latest_recorded_at = 0 AND created_at <= ?)", targetTimestamp, targetTimestamp).Find(&archives).Error; err != nil {
		return 0, err
	}
	var deleted int64
	for index := range archives {
		lock := inflightTraceArchiveLock(archives[index].UserID)
		lock.Lock()
		var archive model.InflightTraceArchive
		err := model.DB.First(&archive, archives[index].ID).Error
		if err == nil && (archive.LatestRecordedAt == 0 || archive.LatestRecordedAt <= targetTimestamp) {
			err = deleteInflightTraceArchive(ctx, &archive)
			if err == nil {
				deleted++
			}
		}
		lock.Unlock()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return deleted, err
		}
	}
	return deleted, nil
}

func deleteInflightTraceArchive(ctx context.Context, archive *model.InflightTraceArchive) error {
	if archive.ObjectKey != "" && archive.StorageChannelID != 0 {
		if err := deleteInflightTraceArchiveFile(ctx, archive.StorageChannelID, archive.ObjectKey); err != nil {
			return err
		}
	}
	if archive.LocalPath != "" {
		_ = os.Remove(archive.LocalPath)
	}
	return model.DB.Delete(archive).Error
}

func deleteInflightTraceArchiveFile(ctx context.Context, channelID uint64, objectKey string) error {
	var channel model.FileUploadChannel
	if err := model.DB.First(&channel, channelID).Error; err != nil {
		return err
	}
	config, err := ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return err
	}
	switch channel.Type {
	case FileUploadChannelTypeS3:
		region, bucket := inflightTraceConfigString(config["region"]), inflightTraceConfigString(config["bucket"])
		accessKey, secret := inflightTraceConfigString(config["access_key_id"]), inflightTraceConfigString(config["secret_access_key"])
		if region == "" || bucket == "" || accessKey == "" || secret == "" {
			return errors.New("S3 channel configuration is incomplete")
		}
		awsConfig := aws.Config{Region: region, Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secret, ""))}
		forcePathStyle, _ := config["force_path_style"].(bool)
		client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
			options.UsePathStyle = forcePathStyle
			if endpoint := strings.TrimSpace(inflightTraceConfigString(config["endpoint"])); endpoint != "" {
				options.BaseEndpoint = aws.String(endpoint)
			}
		})
		_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(objectKey)})
		return err
	case FileUploadChannelTypeWebDAV:
		target := strings.TrimRight(inflightTraceConfigString(config["base_url"]), "/") + "/" + objectKey
		req, err := http.NewRequestWithContext(ctx, http.MethodDelete, target, nil)
		if err != nil {
			return err
		}
		if username, password := inflightTraceConfigString(config["username"]), inflightTraceConfigString(config["password"]); username != "" || password != "" {
			req.SetBasicAuth(username, password)
		}
		response, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return fmt.Errorf("WebDAV archive delete returned status %d", response.StatusCode)
		}
		return nil
	case FileUploadChannelTypeCloudflareImageBed:
		baseURL := strings.TrimRight(strings.TrimSpace(inflightTraceConfigString(config["base_url"])), "/")
		token := strings.TrimSpace(inflightTraceConfigString(config["api_token"]))
		if baseURL == "" || token == "" {
			return errors.New("image bed storage configuration is incomplete")
		}
		archivePath := strings.Trim(strings.TrimSpace(inflightTraceConfigString(config["upload_folder"]))+"/"+filepath.Base(objectKey), "/")
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/api/manage/delete/"+strings.ReplaceAll(url.PathEscape(archivePath), "%2F", "/"), nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		response, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
		if err != nil {
			return err
		}
		defer response.Body.Close()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return fmt.Errorf("image bed archive delete returned status %d", response.StatusCode)
		}
		return nil
	default:
		return errors.New("inflight trace archive deletion is unsupported for this storage channel")
	}
}

func uploadInflightTraceArchiveFile(ctx context.Context, channelID uint64, fileName, localPath string) (string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()
	var channel model.FileUploadChannel
	if err = model.DB.Where("id = ? AND status = ?", channelID, FileUploadChannelStatusEnabled).First(&channel).Error; err != nil {
		return "", "", errors.New("archive storage channel is unavailable")
	}
	config, err := ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return "", "", err
	}
	prefix := strings.Trim(strings.TrimSpace(inflightTraceConfigString(config["key_prefix"])), "/")
	objectKey := strings.Trim(prefix+"/"+fileName, "/")
	switch channel.Type {
	case FileUploadChannelTypeS3:
		region, bucket := strings.TrimSpace(inflightTraceConfigString(config["region"])), strings.TrimSpace(inflightTraceConfigString(config["bucket"]))
		accessKey, secret := strings.TrimSpace(inflightTraceConfigString(config["access_key_id"])), strings.TrimSpace(inflightTraceConfigString(config["secret_access_key"]))
		if region == "" || bucket == "" || accessKey == "" || secret == "" {
			return "", "", errors.New("S3 channel configuration is incomplete")
		}
		awsConfig := aws.Config{Region: region, Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secret, ""))}
		client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
			if endpoint := strings.TrimSpace(inflightTraceConfigString(config["endpoint"])); endpoint != "" {
				options.BaseEndpoint = aws.String(endpoint)
			}
		})
		if _, err = client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(objectKey), Body: file, ContentType: aws.String("text/csv")}); err != nil {
			return "", "", err
		}
	case FileUploadChannelTypeWebDAV:
		baseURL := strings.TrimSpace(inflightTraceConfigString(config["base_url"]))
		if baseURL == "" {
			return "", "", errors.New("WebDAV channel configuration is incomplete")
		}
		target := strings.TrimRight(baseURL, "/") + "/" + objectKey
		req, reqErr := http.NewRequestWithContext(ctx, http.MethodPut, target, file)
		if reqErr != nil {
			return "", "", reqErr
		}
		req.Header.Set("Content-Type", "text/csv")
		if username, password := strings.TrimSpace(inflightTraceConfigString(config["username"])), strings.TrimSpace(inflightTraceConfigString(config["password"])); username != "" || password != "" {
			req.SetBasicAuth(username, password)
		}
		response, requestErr := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
		if requestErr != nil {
			return "", "", requestErr
		}
		defer response.Body.Close()
		if response.StatusCode < 200 || response.StatusCode >= 300 {
			return "", "", fmt.Errorf("WebDAV archive upload returned status %d", response.StatusCode)
		}
	case FileUploadChannelTypeCloudflareImageBed:
		archiveURL, uploadErr := uploadInflightTraceImageBed(ctx, config, fileName, file)
		if uploadErr != nil {
			return "", "", uploadErr
		}
		return objectKey, archiveURL, nil
	default:
		return "", "", errors.New("inflight trace archive storage channel is unsupported")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(inflightTraceConfigString(config["public_base_url"])), "/")
	if baseURL == "" {
		return objectKey, "", nil
	}
	return objectKey, baseURL + "/" + objectKey, nil
}

func uploadInflightTraceImageBed(ctx context.Context, config map[string]any, fileName string, reader io.Reader) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(inflightTraceConfigString(config["base_url"])), "/")
	token := strings.TrimSpace(inflightTraceConfigString(config["api_token"]))
	if baseURL == "" || token == "" {
		return "", errors.New("image bed storage configuration is incomplete")
	}
	query := url.Values{}
	if value := strings.TrimSpace(inflightTraceConfigString(config["upload_channel"])); value != "" {
		query.Set("uploadChannel", value)
	} else {
		query.Set("uploadChannel", "cfr2")
	}
	if value := strings.TrimSpace(inflightTraceConfigString(config["return_format"])); value != "" {
		query.Set("returnFormat", value)
	}
	if value := strings.TrimSpace(inflightTraceConfigString(config["upload_folder"])); value != "" {
		query.Set("uploadFolder", value)
	}
	pipeReader, pipeWriter := io.Pipe()
	writer := multipart.NewWriter(pipeWriter)
	copyErr := make(chan error, 1)
	go func() {
		part, err := writer.CreateFormFile("file", fileName)
		if err == nil {
			_, err = io.Copy(part, reader)
		}
		if closeErr := writer.Close(); err == nil {
			err = closeErr
		}
		_ = pipeWriter.CloseWithError(err)
		copyErr <- err
	}()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/upload?"+query.Encode(), pipeReader)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-File-Mime-Type", "text/csv")
	response, err := (&http.Client{Timeout: 10 * time.Minute}).Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if err = <-copyErr; err != nil {
		return "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("image bed archive upload returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024))
	if err != nil {
		return "", err
	}
	var payload []map[string]any
	if err = common.Unmarshal(body, &payload); err != nil || len(payload) == 0 {
		return "", errors.New("image bed archive upload returned an invalid response")
	}
	for _, key := range []string{"publicUrl", "src", "url"} {
		if value := strings.TrimSpace(inflightTraceConfigString(payload[0][key])); value != "" {
			parsed, parseErr := url.Parse(value)
			if parseErr == nil && !parsed.IsAbs() {
				base, baseErr := url.Parse(baseURL)
				if baseErr != nil {
					return "", baseErr
				}
				return base.ResolveReference(parsed).String(), nil
			}
			return value, nil
		}
	}
	return "", errors.New("image bed archive upload returned no file URL")
}

func inflightTraceConfigString(value any) string {
	if result, ok := value.(string); ok {
		return result
	}
	return ""
}
