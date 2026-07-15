package storage

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"gorm.io/gorm"
)

type GeneratedAssetInput struct {
	FileName string
	MimeType string
	MaxBytes int64
}

// ImportGeneratedAsset stores server-produced bytes through the configured
// channel. It deliberately accepts an io.Reader rather than an upstream URL.
func ImportGeneratedAsset(ctx context.Context, userID int64, channelID uint64, reader io.Reader, input GeneratedAssetInput) (*FileView, error) {
	if userID <= 0 || channelID == 0 || reader == nil || input.MaxBytes <= 0 {
		return nil, errors.New("invalid generated asset")
	}
	var channel model.FileUploadChannel
	if err := model.DB.Where("id = ? AND status = ?", channelID, service.FileUploadChannelStatusEnabled).First(&channel).Error; err != nil {
		return nil, errors.New("creative storage channel is unavailable")
	}
	if !validGeneratedMime(input.MimeType) {
		return nil, errors.New("generated asset MIME type is not allowed")
	}
	buffered := bufio.NewReader(reader)
	magic, err := buffered.Peek(32)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, bufio.ErrBufferFull) {
		return nil, err
	}
	if !generatedMagicMatches(input.MimeType, magic) {
		return nil, errors.New("generated asset content does not match its MIME type")
	}
	fileName := path.Base(strings.TrimSpace(input.FileName))
	if fileName == "." || fileName == "" {
		fileName = "creative-asset"
	}
	suffix := strings.TrimPrefix(strings.ToLower(path.Ext(fileName)), ".")
	if suffix == "" {
		suffix = generatedAssetSuffix(input.MimeType)
		fileName += "." + suffix
	}
	objectKey := buildObjectKey(&channel, fileName, suffix)
	hash := sha256.New()
	limited := &generatedLimitReader{reader: io.TeeReader(buffered, hash), remaining: input.MaxBytes}
	fileURL, err := uploadGeneratedAsset(ctx, &channel, objectKey, input.MimeType, limited)
	if err != nil {
		return nil, err
	}
	if limited.exceeded {
		_ = deletePhysical(ctx, &model.File{FileChannelId: channel.Id, ChannelType: channel.Type, ObjectKey: objectKey})
		return nil, errors.New("generated asset exceeds size limit")
	}
	if limited.size == 0 {
		_ = deletePhysical(ctx, &model.File{FileChannelId: channel.Id, ChannelType: channel.Type, ObjectKey: objectKey})
		return nil, errors.New("generated asset is empty")
	}
	identifier := hex.EncodeToString(hash.Sum(nil))
	if fileURL == "" {
		fileURL = publicURL(&channel, objectKey)
	}
	var userFile model.UserFile
	err = model.DB.Transaction(func(tx *gorm.DB) error {
		physical := model.File{FileChannelId: channel.Id, ChannelType: channel.Type, FileSize: limited.size, ObjectKey: objectKey, FileUrl: fileURL, Identifier: identifier, MimeType: input.MimeType, Status: "1", CreaterUserId: userID}
		if err := tx.Create(&physical).Error; err != nil {
			return err
		}
		userFile = model.UserFile{FileId: physical.Id, FileChannelId: physical.FileChannelId, FileName: fileName, FileSuffix: suffix, UserId: userID, Source: "creative_studio", Status: "1"}
		if err := tx.Create(&userFile).Error; err != nil {
			return err
		}
		return tx.Model(&model.File{}).Where("id = ?", physical.Id).UpdateColumn("ref_count", gorm.Expr("ref_count + ?", 1)).Error
	})
	if err != nil {
		_ = deletePhysical(ctx, &model.File{FileChannelId: channel.Id, ChannelType: channel.Type, ObjectKey: objectKey})
		return nil, err
	}
	return Get(userFile.Id, &userID)
}

type generatedLimitReader struct {
	reader    io.Reader
	remaining int64
	size      int64
	exceeded  bool
}

func (reader *generatedLimitReader) Read(buffer []byte) (int, error) {
	if reader.remaining <= 0 {
		var probe [1]byte
		count, err := reader.reader.Read(probe[:])
		if count > 0 {
			reader.exceeded = true
			return 0, errors.New("generated asset exceeds size limit")
		}
		return 0, err
	}
	if int64(len(buffer)) > reader.remaining {
		buffer = buffer[:reader.remaining]
	}
	count, err := reader.reader.Read(buffer)
	reader.remaining -= int64(count)
	reader.size += int64(count)
	return count, err
}

func validGeneratedMime(mimeType string) bool {
	switch mimeType {
	case "image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml":
		return true
	}
	return false
}

func generatedAssetSuffix(mimeType string) string {
	switch mimeType {
	case "image/png":
		return "png"
	case "image/jpeg":
		return "jpg"
	case "image/webp":
		return "webp"
	case "image/gif":
		return "gif"
	case "image/svg+xml":
		return "svg"
	}
	return ""
}

func generatedMagicMatches(mimeType string, bytes []byte) bool {
	switch mimeType {
	case "image/png":
		return len(bytes) >= 8 && string(bytes[:8]) == "\x89PNG\r\n\x1a\n"
	case "image/jpeg":
		return len(bytes) >= 3 && bytes[0] == 0xff && bytes[1] == 0xd8 && bytes[2] == 0xff
	case "image/gif":
		return len(bytes) >= 6 && (string(bytes[:6]) == "GIF87a" || string(bytes[:6]) == "GIF89a")
	case "image/webp":
		return len(bytes) >= 12 && string(bytes[:4]) == "RIFF" && string(bytes[8:12]) == "WEBP"
	case "image/svg+xml":
		return strings.Contains(strings.ToLower(string(bytes)), "<svg")
	}
	return false
}

func uploadGeneratedAsset(ctx context.Context, channel *model.FileUploadChannel, objectKey, mimeType string, reader io.Reader) (string, error) {
	switch channel.Type {
	case service.FileUploadChannelTypeS3:
		client, bucket, err := s3Client(channel)
		if err != nil {
			return "", err
		}
		if _, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(bucket), Key: aws.String(objectKey), Body: reader, ContentType: aws.String(mimeType)}); err != nil {
			return "", err
		}
		return publicURL(channel, objectKey), nil
	case service.FileUploadChannelTypeWebDAV:
		config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
		if err != nil {
			return "", err
		}
		target, err := joinURL(stringValue(config["base_url"]), objectKey)
		if err != nil {
			return "", err
		}
		client := &http.Client{Timeout: webDAVTimeout(config)}
		if err := ensureGeneratedWebDAVCollections(ctx, client, stringValue(config["base_url"]), objectKey, config); err != nil {
			return "", err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, target, reader)
		if err != nil {
			return "", err
		}
		req.Header.Set("Content-Type", mimeType)
		applyWebDAVAuth(req, config)
		response, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()
		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			return "", fmt.Errorf("WebDAV upload returned status %d", response.StatusCode)
		}
		return publicURL(channel, objectKey), nil
	case service.FileUploadChannelTypeCloudflareImageBed:
		return uploadGeneratedImageBed(ctx, channel, objectKey, mimeType, reader)
	default:
		return "", errors.New("generated asset import is unsupported for this channel")
	}
}

func uploadGeneratedImageBed(ctx context.Context, channel *model.FileUploadChannel, objectKey, mimeType string, reader io.Reader) (string, error) {
	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return "", err
	}
	baseURL, _ := config["base_url"].(string)
	token, _ := config["api_token"].(string)
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return "", errors.New("image bed storage configuration is incomplete")
	}
	query := url.Values{}
	if value, _ := config["upload_channel"].(string); strings.TrimSpace(value) != "" {
		query.Set("uploadChannel", strings.TrimSpace(value))
	} else {
		query.Set("uploadChannel", "cfr2")
	}
	if value, _ := config["return_format"].(string); strings.TrimSpace(value) != "" {
		query.Set("returnFormat", strings.TrimSpace(value))
	} else {
		query.Set("returnFormat", "default")
	}
	if value, _ := config["upload_folder"].(string); strings.TrimSpace(value) != "" {
		query.Set("uploadFolder", strings.TrimSpace(value))
	}
	pipeReader, pipeWriter := io.Pipe()
	defer pipeReader.Close()
	writer := multipart.NewWriter(pipeWriter)
	copyErr := make(chan error, 1)
	go func() {
		part, err := writer.CreateFormFile("file", path.Base(objectKey))
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
		pipeReader.Close()
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-File-Mime-Type", mimeType)
	response, err := (&http.Client{Timeout: 30 * time.Second}).Do(req)
	if err != nil {
		pipeReader.Close()
		return "", err
	}
	defer response.Body.Close()
	if err := <-copyErr; err != nil {
		return "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("image bed upload returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16*1024))
	if err != nil {
		return "", err
	}
	var payload []map[string]any
	if err := common.Unmarshal(body, &payload); err != nil || len(payload) == 0 {
		return "", errors.New("image bed upload returned an invalid response")
	}
	for _, key := range []string{"publicUrl", "src", "url"} {
		if value, _ := payload[0][key].(string); strings.TrimSpace(value) != "" {
			value = strings.TrimSpace(value)
			if parsed, err := url.Parse(value); err == nil && !parsed.IsAbs() {
				base, err := url.Parse(baseURL)
				if err != nil {
					return "", err
				}
				return base.ResolveReference(parsed).String(), nil
			}
			return value, nil
		}
	}
	return "", errors.New("image bed upload returned no file URL")
}

func ensureGeneratedWebDAVCollections(ctx context.Context, client *http.Client, baseURL, objectKey string, config map[string]any) error {
	segments := strings.Split(strings.Trim(objectKey, "/"), "/")
	if len(segments) <= 1 {
		return nil
	}
	current := ""
	for _, segment := range segments[:len(segments)-1] {
		current = strings.Trim(current+"/"+segment, "/")
		target, err := joinURL(baseURL, current)
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(ctx, "MKCOL", target, nil)
		if err != nil {
			return err
		}
		applyWebDAVAuth(req, config)
		response, err := client.Do(req)
		if err != nil {
			return err
		}
		response.Body.Close()
		if response.StatusCode != http.StatusCreated && response.StatusCode != http.StatusMethodNotAllowed && (response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices) {
			return fmt.Errorf("WebDAV MKCOL returned status %d", response.StatusCode)
		}
	}
	return nil
}
