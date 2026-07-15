package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func OpenCreativeUserFile(ctx context.Context, userID int64, userFileID uint64) (io.ReadCloser, *FileView, error) {
	fileView, err := Get(userFileID, &userID)
	if err != nil || fileView.Status != "1" || !validGeneratedMime(fileView.MimeType) {
		return nil, nil, errors.New("creative input file is unavailable")
	}
	var file model.File
	if err := model.DB.First(&file, fileView.FileID).Error; err != nil {
		return nil, nil, errors.New("creative input file is unavailable")
	}
	var channel model.FileUploadChannel
	if err := model.DB.Where("id = ? AND status = ?", file.FileChannelId, service.FileUploadChannelStatusEnabled).First(&channel).Error; err != nil {
		return nil, nil, errors.New("creative input storage is unavailable")
	}
	switch channel.Type {
	case service.FileUploadChannelTypeS3:
		client, bucket, err := s3Client(&channel)
		if err != nil {
			return nil, nil, err
		}
		response, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(bucket), Key: aws.String(file.ObjectKey)})
		if err != nil {
			return nil, nil, err
		}
		return response.Body, fileView, nil
	case service.FileUploadChannelTypeWebDAV:
		config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
		if err != nil {
			return nil, nil, err
		}
		target, err := joinURL(stringValue(config["base_url"]), file.ObjectKey)
		if err != nil {
			return nil, nil, err
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
		if err != nil {
			return nil, nil, err
		}
		applyWebDAVAuth(request, config)
		response, err := (&http.Client{Timeout: webDAVTimeout(config)}).Do(request)
		if err != nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			if response != nil {
				response.Body.Close()
			}
			return nil, nil, errors.New("creative input file is unavailable")
		}
		return response.Body, fileView, nil
	case service.FileUploadChannelTypeCloudflareImageBed:
		if strings.TrimSpace(file.FileUrl) == "" {
			return nil, nil, errors.New("creative input file is unavailable")
		}
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, file.FileUrl, nil)
		if err != nil {
			return nil, nil, err
		}
		response, err := (&http.Client{Timeout: 30 * time.Second}).Do(request)
		if err != nil || response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			if response != nil {
				response.Body.Close()
			}
			return nil, nil, errors.New("creative input file is unavailable")
		}
		return response.Body, fileView, nil
	default:
		return nil, nil, errors.New("creative input storage is unsupported")
	}
}
