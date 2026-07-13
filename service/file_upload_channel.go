package service

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const (
	FileUploadChannelTypeLocal              = "0"
	FileUploadChannelTypeWebDAV             = "1"
	FileUploadChannelTypeCloudflareImageBed = "2"
	FileUploadChannelTypeS3                 = "3"

	FileUploadChannelStatusDisabled = "0"
	FileUploadChannelStatusEnabled  = "1"

	FileUploadChannelNotDefault = "0"
	FileUploadChannelIsDefault  = "1"

	DefaultFileUploadChunkThreshold = int64(16777216)
	DefaultFileUploadChunkSize      = int64(8388608)
	DefaultFileUploadMaxSize        = int64(0)
	DefaultWebDAVTimeoutMS          = int64(600000)
)

//go:embed assets/file_upload_channel_probe_test.txt
var embeddedFileUploadChannelProbeContent []byte

type FileUploadChannelProbeResult struct {
	Ok        bool   `json:"ok"`
	Type      string `json:"type"`
	LatencyMs int64  `json:"latency_ms"`
	Detail    string `json:"detail"`
	FileName  string `json:"file_name"`
	FileSize  int64  `json:"file_size"`
	URL       string `json:"url"`
}

type FileUploadChannelProbeFileInfo struct {
	FileName string `json:"file_name"`
	FileSize int64  `json:"file_size"`
}

func IsSupportedFileUploadChannelType(channelType string) bool {
	switch channelType {
	case FileUploadChannelTypeWebDAV, FileUploadChannelTypeCloudflareImageBed, FileUploadChannelTypeS3:
		return true
	default:
		return false
	}
}

func IsValidFileUploadChannelStatus(status string) bool {
	return status == FileUploadChannelStatusDisabled || status == FileUploadChannelStatusEnabled
}

func IsValidFileUploadChannelDefaultFlag(flag string) bool {
	return flag == FileUploadChannelNotDefault || flag == FileUploadChannelIsDefault
}

func ParseFileUploadChannelConfig(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}, nil
	}
	config := make(map[string]any)
	if err := common.UnmarshalJsonStr(raw, &config); err != nil {
		return nil, err
	}
	return config, nil
}

func MarshalFileUploadChannelConfig(config map[string]any) (string, error) {
	if config == nil {
		config = map[string]any{}
	}
	data, err := common.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func CloneConfigMap(config map[string]any) map[string]any {
	if config == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(config))
	for key, value := range config {
		cloned[key] = value
	}
	return cloned
}

func MaskFileUploadChannelConfig(channelType string, config map[string]any) map[string]any {
	masked := CloneConfigMap(config)
	for _, key := range getSensitiveConfigKeys(channelType) {
		value := strings.TrimSpace(getStringValue(masked[key]))
		if value == "" {
			masked[key] = ""
			continue
		}
		masked[key] = maskSensitiveConfigValue(value)
	}
	return masked
}

func maskSensitiveConfigValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	if len(runes) <= 6 {
		return strings.Repeat("*", len(runes))
	}

	prefixLen := 3
	suffixLen := 3
	if len(runes) >= 16 {
		prefixLen = 4
		suffixLen = 4
	}

	if prefixLen+suffixLen >= len(runes) {
		return strings.Repeat("*", len(runes))
	}

	return string(runes[:prefixLen]) +
		strings.Repeat("*", len(runes)-prefixLen-suffixLen) +
		string(runes[len(runes)-suffixLen:])
}

func MergeAndValidateFileUploadChannelConfig(
	channelType string,
	input map[string]any,
	existing map[string]any,
	isCreate bool,
) (map[string]any, error) {
	if channelType == FileUploadChannelTypeLocal {
		return nil, fmt.Errorf("local file upload channel is not supported in this iteration")
	}
	if !IsSupportedFileUploadChannelType(channelType) {
		return nil, fmt.Errorf("unsupported file upload channel type: %s", channelType)
	}

	merged := CloneConfigMap(existing)
	for key, value := range CloneConfigMap(input) {
		merged[key] = value
	}

	switch channelType {
	case FileUploadChannelTypeWebDAV:
		return normalizeWebDAVConfig(merged, existing, isCreate)
	case FileUploadChannelTypeCloudflareImageBed:
		return normalizeCloudflareImageBedConfig(merged, existing, isCreate)
	case FileUploadChannelTypeS3:
		return normalizeS3Config(merged, existing, isCreate)
	default:
		return nil, fmt.Errorf("unsupported file upload channel type: %s", channelType)
	}
}

func ProbeFileUploadChannel(ctx context.Context, channelType string, config map[string]any) (*FileUploadChannelProbeResult, error) {
	if channelType == FileUploadChannelTypeLocal {
		return nil, fmt.Errorf("local file upload channel probe is not supported")
	}

	probeFile, err := loadFileUploadChannelProbeFile()
	if err != nil {
		return nil, err
	}

	startedAt := time.Now()
	var uploadedURL string
	switch channelType {
	case FileUploadChannelTypeWebDAV:
		uploadedURL, err = probeWebDAV(ctx, config, probeFile)
		if err != nil {
			return nil, err
		}
	case FileUploadChannelTypeCloudflareImageBed:
		uploadedURL, err = probeCloudflareImageBed(ctx, config, probeFile)
		if err != nil {
			return nil, err
		}
	case FileUploadChannelTypeS3:
		uploadedURL, err = probeS3(ctx, config, probeFile)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported file upload channel type: %s", channelType)
	}

	return &FileUploadChannelProbeResult{
		Ok:        true,
		Type:      channelType,
		LatencyMs: time.Since(startedAt).Milliseconds(),
		Detail:    "ok",
		FileName:  probeFile.Name,
		FileSize:  int64(len(probeFile.Content)),
		URL:       uploadedURL,
	}, nil
}

func GetFileUploadChannelProbeFileInfo() (*FileUploadChannelProbeFileInfo, error) {
	probeFile, err := loadFileUploadChannelProbeFile()
	if err != nil {
		return nil, err
	}
	return &FileUploadChannelProbeFileInfo{
		FileName: probeFile.Name,
		FileSize: int64(len(probeFile.Content)),
	}, nil
}

type fileUploadChannelProbeFile struct {
	Name        string
	ContentType string
	Content     []byte
}

func loadFileUploadChannelProbeFile() (*fileUploadChannelProbeFile, error) {
	if len(embeddedFileUploadChannelProbeContent) == 0 {
		return nil, fmt.Errorf("embedded probe file is empty")
	}
	return &fileUploadChannelProbeFile{
		Name:        "test.txt",
		ContentType: "text/plain; charset=utf-8",
		Content:     append([]byte(nil), embeddedFileUploadChannelProbeContent...),
	}, nil
}

func normalizeWebDAVConfig(input map[string]any, existing map[string]any, isCreate bool) (map[string]any, error) {
	config := map[string]any{
		"base_url":        strings.TrimSpace(getStringValue(input["base_url"])),
		"username":        strings.TrimSpace(getStringValue(input["username"])),
		"password":        resolveSensitiveString(input, existing, "password"),
		"auth_type":       strings.ToLower(strings.TrimSpace(getStringValue(input["auth_type"]))),
		"path_prefix":     strings.TrimSpace(getStringValue(input["path_prefix"])),
		"public_base_url": strings.TrimSpace(getStringValue(input["public_base_url"])),
		"timeout_ms":      getInt64Value(input["timeout_ms"], DefaultWebDAVTimeoutMS),
	}
	if config["auth_type"] == "" {
		config["auth_type"] = "basic"
	}
	if config["timeout_ms"].(int64) <= 0 {
		config["timeout_ms"] = DefaultWebDAVTimeoutMS
	}

	baseURL := config["base_url"].(string)
	if baseURL == "" {
		return nil, fmt.Errorf("config_proflle.base_url is required")
	}
	authType := config["auth_type"].(string)
	if authType != "basic" && authType != "bearer" {
		return nil, fmt.Errorf("config_proflle.auth_type must be basic or bearer")
	}
	if authType == "basic" && config["username"].(string) == "" {
		return nil, fmt.Errorf("config_proflle.username is required")
	}
	if isCreate && strings.TrimSpace(config["password"].(string)) == "" {
		return nil, fmt.Errorf("config_proflle.password is required")
	}
	return config, nil
}

func normalizeCloudflareImageBedConfig(input map[string]any, existing map[string]any, isCreate bool) (map[string]any, error) {
	config := map[string]any{
		"base_url":       strings.TrimSpace(getStringValue(input["base_url"])),
		"api_token":      resolveSensitiveString(input, existing, "api_token"),
		"upload_channel": strings.TrimSpace(getStringValue(input["upload_channel"])),
		"upload_folder":  strings.TrimSpace(getStringValue(input["upload_folder"])),
		"return_format":  strings.TrimSpace(getStringValue(input["return_format"])),
	}
	if config["upload_channel"] == "" {
		config["upload_channel"] = "cfr2"
	}
	if config["return_format"] == "" {
		config["return_format"] = "default"
	}

	if config["base_url"].(string) == "" {
		return nil, fmt.Errorf("config_proflle.base_url is required")
	}
	if isCreate && strings.TrimSpace(config["api_token"].(string)) == "" {
		return nil, fmt.Errorf("config_proflle.api_token is required")
	}
	returnFormat := config["return_format"].(string)
	if returnFormat != "default" && returnFormat != "full" {
		return nil, fmt.Errorf("config_proflle.return_format must be default or full")
	}
	return config, nil
}

func normalizeS3Config(input map[string]any, existing map[string]any, isCreate bool) (map[string]any, error) {
	config := map[string]any{
		"endpoint":          strings.TrimSpace(getStringValue(input["endpoint"])),
		"region":            strings.TrimSpace(getStringValue(input["region"])),
		"bucket":            strings.TrimSpace(getStringValue(input["bucket"])),
		"access_key_id":     strings.TrimSpace(getStringValue(input["access_key_id"])),
		"secret_access_key": resolveSensitiveString(input, existing, "secret_access_key"),
		"force_path_style":  getBoolValue(input["force_path_style"], false),
		"public_base_url":   strings.TrimSpace(getStringValue(input["public_base_url"])),
		"key_prefix":        strings.TrimSpace(getStringValue(input["key_prefix"])),
	}

	if config["region"].(string) == "" {
		return nil, fmt.Errorf("config_proflle.region is required")
	}
	if config["bucket"].(string) == "" {
		return nil, fmt.Errorf("config_proflle.bucket is required")
	}
	if config["access_key_id"].(string) == "" {
		return nil, fmt.Errorf("config_proflle.access_key_id is required")
	}
	if isCreate && strings.TrimSpace(config["secret_access_key"].(string)) == "" {
		return nil, fmt.Errorf("config_proflle.secret_access_key is required")
	}
	return config, nil
}

func probeWebDAV(ctx context.Context, config map[string]any, probeFile *fileUploadChannelProbeFile) (string, error) {
	baseURL := strings.TrimSpace(getStringValue(config["base_url"]))
	if baseURL == "" {
		return "", fmt.Errorf("config_proflle.base_url is required")
	}

	timeout := time.Duration(getInt64Value(config["timeout_ms"], DefaultWebDAVTimeoutMS)) * time.Millisecond
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	client := &http.Client{Timeout: timeout}

	probePath := strings.Trim(strings.TrimSpace(getStringValue(config["path_prefix"])), "/")
	probePath = appendNonEmptyPath(probePath, "probe")
	probePath = appendNonEmptyPath(probePath, common.GetTimeString()+"_"+probeFile.Name)

	uploadURL, err := joinURLPath(baseURL, probePath)
	if err != nil {
		return "", fmt.Errorf("build WebDAV probe url failed: %w", err)
	}

	if err := ensureWebDAVParentCollections(ctx, client, baseURL, probePath, config); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, bytes.NewReader(probeFile.Content))
	if err != nil {
		return "", fmt.Errorf("build WebDAV probe upload request failed: %w", err)
	}
	applyWebDAVAuth(req, config)
	req.Header.Set("Content-Type", probeFile.ContentType)
	req.Header.Set("Content-Length", fmt.Sprintf("%d", len(probeFile.Content)))

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("WebDAV probe upload failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("WebDAV probe upload failed: %s", buildProbeHTTPError(resp.StatusCode, bodyBytes))
	}

	publicBaseURL := strings.TrimSpace(getStringValue(config["public_base_url"]))
	if publicBaseURL != "" {
		publicURL, joinErr := joinURLPath(publicBaseURL, probePath)
		if joinErr == nil {
			return publicURL, nil
		}
	}
	return uploadURL, nil
}

func probeCloudflareImageBed(ctx context.Context, config map[string]any, probeFile *fileUploadChannelProbeFile) (string, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(getStringValue(config["base_url"])), "/")
	if baseURL == "" {
		return "", fmt.Errorf("config_proflle.base_url is required")
	}
	token := strings.TrimSpace(getStringValue(config["api_token"]))
	if token == "" {
		return "", fmt.Errorf("config_proflle.api_token is required")
	}

	query := url.Values{}
	query.Set("uploadChannel", strings.TrimSpace(getStringValue(config["upload_channel"])))
	if query.Get("uploadChannel") == "" {
		query.Set("uploadChannel", "cfr2")
	}
	query.Set("returnFormat", strings.TrimSpace(getStringValue(config["return_format"])))
	if query.Get("returnFormat") == "" {
		query.Set("returnFormat", "default")
	}
	uploadFolder := strings.TrimSpace(getStringValue(config["upload_folder"]))
	if uploadFolder != "" {
		query.Set("uploadFolder", uploadFolder)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", probeFile.Name)
	if err != nil {
		return "", fmt.Errorf("build Cloudflare ImageBed probe file field failed: %w", err)
	}
	if _, err = part.Write(probeFile.Content); err != nil {
		return "", fmt.Errorf("write Cloudflare ImageBed probe file failed: %w", err)
	}
	if err = writer.Close(); err != nil {
		return "", fmt.Errorf("close Cloudflare ImageBed probe form failed: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/upload?"+query.Encode(), &body)
	if err != nil {
		return "", fmt.Errorf("build Cloudflare ImageBed probe request failed: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Cloudflare ImageBed probe failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return "", fmt.Errorf("Cloudflare ImageBed probe failed: %s", buildProbeHTTPError(resp.StatusCode, bodyBytes))
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
	if err != nil {
		return "", fmt.Errorf("read Cloudflare ImageBed probe response failed: %w", err)
	}
	return parseCloudflareProbeURL(bodyBytes, baseURL)
}

func probeS3(ctx context.Context, config map[string]any, probeFile *fileUploadChannelProbeFile) (string, error) {
	region := strings.TrimSpace(getStringValue(config["region"]))
	bucket := strings.TrimSpace(getStringValue(config["bucket"]))
	accessKeyID := strings.TrimSpace(getStringValue(config["access_key_id"]))
	secretAccessKey := strings.TrimSpace(getStringValue(config["secret_access_key"]))
	if region == "" || bucket == "" || accessKeyID == "" || secretAccessKey == "" {
		return "", fmt.Errorf("config_proflle.region, bucket, access_key_id, secret_access_key are required")
	}

	awsConfig := aws.Config{
		Region:      region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
		HTTPClient:  &http.Client{Timeout: 15 * time.Second},
	}
	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = getBoolValue(config["force_path_style"], false)
		endpoint := strings.TrimSpace(getStringValue(config["endpoint"]))
		if endpoint != "" {
			options.BaseEndpoint = aws.String(endpoint)
		}
	})

	objectKey := strings.Trim(strings.TrimSpace(getStringValue(config["key_prefix"])), "/")
	objectKey = appendNonEmptyPath(objectKey, "probe")
	objectKey = appendNonEmptyPath(objectKey, common.GetTimeString()+"_"+probeFile.Name)

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(probeFile.Content),
		ContentType: aws.String(probeFile.ContentType),
		ACL:         types.ObjectCannedACLPrivate,
	})
	if err != nil {
		return "", fmt.Errorf("S3 probe upload failed: %w", err)
	}

	return buildS3ProbeURL(config, bucket, objectKey), nil
}

func ensureWebDAVParentCollections(ctx context.Context, client *http.Client, baseURL string, filePath string, config map[string]any) error {
	trimmed := strings.Trim(strings.TrimSpace(filePath), "/")
	if trimmed == "" {
		return nil
	}
	segments := strings.Split(trimmed, "/")
	if len(segments) <= 1 {
		return nil
	}
	current := ""
	for _, segment := range segments[:len(segments)-1] {
		current = appendNonEmptyPath(current, segment)
		targetURL, err := joinURLPath(baseURL, current)
		if err != nil {
			return fmt.Errorf("build WebDAV collection url failed: %w", err)
		}
		req, err := http.NewRequestWithContext(ctx, "MKCOL", targetURL, nil)
		if err != nil {
			return fmt.Errorf("build WebDAV MKCOL request failed: %w", err)
		}
		applyWebDAVAuth(req, config)
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("WebDAV MKCOL failed: %w", err)
		}
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		resp.Body.Close()
		if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusConflict {
			continue
		}
		if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
			continue
		}
		return fmt.Errorf("WebDAV MKCOL failed: %s", buildProbeHTTPError(resp.StatusCode, bodyBytes))
	}
	return nil
}

func applyWebDAVAuth(req *http.Request, config map[string]any) {
	authType := strings.ToLower(strings.TrimSpace(getStringValue(config["auth_type"])))
	if authType == "" {
		authType = "basic"
	}
	switch authType {
	case "bearer":
		token := getStringValue(config["password"])
		req.Header.Set("Authorization", "Bearer "+token)
	default:
		username := getStringValue(config["username"])
		password := getStringValue(config["password"])
		req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)))
	}
}

func appendNonEmptyPath(base string, segments ...string) string {
	result := strings.Trim(base, "/")
	for _, segment := range segments {
		trimmed := strings.TrimSpace(strings.Trim(segment, "/"))
		if trimmed == "" {
			continue
		}
		if result == "" {
			result = trimmed
			continue
		}
		result += "/" + trimmed
	}
	return result
}

func joinURLPath(base string, segments ...string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(base))
	if err != nil {
		return "", err
	}
	joined := strings.TrimSuffix(parsed.Path, "/")
	for _, segment := range segments {
		trimmed := strings.Trim(segment, "/")
		if trimmed == "" {
			continue
		}
		joined = joined + "/" + trimmed
	}
	if joined == "" {
		joined = "/"
	}
	parsed.Path = joined
	return parsed.String(), nil
}

func parseCloudflareProbeURL(body []byte, baseURL string) (string, error) {
	var payload []map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		return "", fmt.Errorf("parse Cloudflare ImageBed probe response failed: %w", err)
	}
	if len(payload) == 0 {
		return "", fmt.Errorf("Cloudflare ImageBed probe response is empty")
	}
	first := payload[0]
	for _, key := range []string{"publicUrl", "src", "url"} {
		value := strings.TrimSpace(getStringValue(first[key]))
		if value == "" {
			continue
		}
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value, nil
		}
		return joinURLPath(baseURL, value)
	}
	return "", fmt.Errorf("Cloudflare ImageBed probe response does not contain a file url")
}

func buildS3ProbeURL(config map[string]any, bucket string, objectKey string) string {
	publicBaseURL := strings.TrimSpace(getStringValue(config["public_base_url"]))
	if publicBaseURL != "" {
		joined, err := joinURLPath(publicBaseURL, objectKey)
		if err == nil {
			return joined
		}
	}
	endpoint := strings.TrimRight(strings.TrimSpace(getStringValue(config["endpoint"])), "/")
	if endpoint == "" {
		return fmt.Sprintf("s3://%s/%s", bucket, objectKey)
	}
	if getBoolValue(config["force_path_style"], false) {
		return endpoint + "/" + bucket + "/" + objectKey
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return endpoint + "/" + objectKey
	}
	return parsed.Scheme + "://" + bucket + "." + parsed.Host + "/" + objectKey
}

func buildProbeHTTPError(statusCode int, body []byte) string {
	message := strings.TrimSpace(string(body))
	if message == "" {
		return fmt.Sprintf("upstream returned status %d", statusCode)
	}
	message = strings.ReplaceAll(message, "\n", " ")
	return fmt.Sprintf("upstream returned status %d: %s", statusCode, message)
}

func getSensitiveConfigKeys(channelType string) []string {
	switch channelType {
	case FileUploadChannelTypeWebDAV:
		return []string{"password"}
	case FileUploadChannelTypeCloudflareImageBed:
		return []string{"api_token"}
	case FileUploadChannelTypeS3:
		return []string{"secret_access_key"}
	default:
		return nil
	}
}

func resolveSensitiveString(input map[string]any, existing map[string]any, key string) string {
	value, exists := input[key]
	if !exists || strings.TrimSpace(getStringValue(value)) == "" {
		return strings.TrimSpace(getStringValue(existing[key]))
	}
	return strings.TrimSpace(getStringValue(value))
}

func getStringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case fmt.Stringer:
		return typed.String()
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", typed)
	}
}

func getInt64Value(value any, defaultValue int64) int64 {
	switch typed := value.(type) {
	case nil:
		return defaultValue
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64:
		return typed
	case float32:
		return int64(typed)
	case float64:
		return int64(typed)
	case string:
		if strings.TrimSpace(typed) == "" {
			return defaultValue
		}
		var parsed int64
		_, err := fmt.Sscan(strings.TrimSpace(typed), &parsed)
		if err != nil {
			return defaultValue
		}
		return parsed
	default:
		return defaultValue
	}
}

func getBoolValue(value any, defaultValue bool) bool {
	switch typed := value.(type) {
	case nil:
		return defaultValue
	case bool:
		return typed
	case string:
		switch strings.ToLower(strings.TrimSpace(typed)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		default:
			return defaultValue
		}
	default:
		return defaultValue
	}
}
