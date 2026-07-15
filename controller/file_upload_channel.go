package controller

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type FileUploadChannelUpsertRequest struct {
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Status         *string        `json:"status"`
	IsDefault      *string        `json:"is_default"`
	ChunkThreshold *int64         `json:"chunk_threshold"`
	ChunkSize      *int64         `json:"chunk_size"`
	MaxSize        *int64         `json:"max_size"`
	ConfigProflle  map[string]any `json:"config_proflle"`
}

type FileUploadChannelProbeRequest struct {
	Id            uint64         `json:"id"`
	Type          string         `json:"type"`
	ConfigProflle map[string]any `json:"config_proflle"`
}

type FileUploadChannelStatusRequest struct {
	Status string `json:"status"`
}

type FileUploadChannelCopyRequest struct {
	Suffix *string `json:"suffix"`
}

func GetFileUploadChannelProbeFileInfo(c *gin.Context) {
	info, err := service.GetFileUploadChannelProbeFileInfo()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, info)
}

type fileUploadChannelResponse struct {
	Id             uint64         `json:"id"`
	Name           string         `json:"name"`
	Type           string         `json:"type"`
	Status         string         `json:"status"`
	IsDefault      string         `json:"is_default"`
	ChunkThreshold int64          `json:"chunk_threshold"`
	ChunkSize      int64          `json:"chunk_size"`
	MaxSize        int64          `json:"max_size"`
	ConfigProflle  map[string]any `json:"config_proflle,omitempty"`
	CreateUserId   int64          `json:"create_user_id"`
	UpdateUserId   int64          `json:"update_user_id"`
	CreateTime     int64          `json:"create_time"`
	UpdateTime     int64          `json:"update_time"`
}

type fileUploadChannelSummaryResponse struct {
	Id             uint64 `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Status         string `json:"status"`
	IsDefault      string `json:"is_default"`
	ChunkThreshold int64  `json:"chunk_threshold"`
	ChunkSize      int64  `json:"chunk_size"`
	MaxSize        int64  `json:"max_size"`
}

func GetFileUploadChannels(c *gin.Context) {
	channelType := strings.TrimSpace(c.Query("type"))
	status := strings.TrimSpace(c.Query("status"))

	query := model.DB.Model(&model.FileUploadChannel{}).Order("id desc")
	if channelType != "" {
		query = query.Where("type = ?", channelType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var channels []*model.FileUploadChannel
	if err := query.Find(&channels).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	response := make([]*fileUploadChannelResponse, 0, len(channels))
	for _, channel := range channels {
		item, err := buildFileUploadChannelResponse(channel, true)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		response = append(response, item)
	}
	common.ApiSuccess(c, response)
}

func GetFileUploadChannel(c *gin.Context) {
	channel, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(channel, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func CreateFileUploadChannel(c *gin.Context) {
	var req FileUploadChannelUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}

	channel, err := buildFileUploadChannelForCreate(c, &req)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if channel.IsDefault == service.FileUploadChannelIsDefault {
			if err := clearDefaultFileUploadChannel(tx); err != nil {
				return err
			}
		}
		return tx.Create(channel).Error
	}); err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(channel, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func CopyFileUploadChannel(c *gin.Context) {
	original, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req FileUploadChannelCopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}

	suffix := "_copy"
	if req.Suffix != nil {
		suffix = *req.Suffix
	}

	copy := *original
	copy.Id = 0
	copy.Name = original.Name + suffix
	copy.IsDefault = service.FileUploadChannelNotDefault
	copy.CreateTime = time.Time{}
	copy.UpdateTime = time.Time{}
	copy.CreateUserId = int64(c.GetInt("id"))
	copy.UpdateUserId = copy.CreateUserId

	if err := model.DB.Create(&copy).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(&copy, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func UpdateFileUploadChannel(c *gin.Context) {
	existing, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req FileUploadChannelUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}

	if strings.TrimSpace(req.Type) != "" && req.Type != existing.Type {
		common.ApiErrorMsg(c, "channel type cannot be changed")
		return
	}
	if req.IsDefault != nil && strings.TrimSpace(*req.IsDefault) == service.FileUploadChannelIsDefault &&
		req.Status != nil && strings.TrimSpace(*req.Status) != service.FileUploadChannelStatusEnabled {
		common.ApiErrorMsg(c, "only enabled channels can be set as default")
		return
	}

	oldIsDefault := existing.IsDefault
	if err := applyFileUploadChannelUpdate(c, existing, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if existing.Status == service.FileUploadChannelStatusDisabled {
		referenced, err := model.IsCreativeStudioFileChannel(existing.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if referenced {
			common.ApiErrorMsg(c, "channel is the creative studio default; choose another channel first")
			return
		}
	}
	if existing.IsDefault == service.FileUploadChannelIsDefault && existing.Status != service.FileUploadChannelStatusEnabled {
		common.ApiErrorMsg(c, "only enabled channels can be set as default")
		return
	}
	if oldIsDefault == service.FileUploadChannelIsDefault &&
		(existing.Status != service.FileUploadChannelStatusEnabled || existing.IsDefault != service.FileUploadChannelIsDefault) {
		if err := ensureAnotherDefaultFileUploadChannel(model.DB, existing.Id); err != nil {
			common.ApiError(c, err)
			return
		}
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if existing.IsDefault == service.FileUploadChannelIsDefault {
			if err := clearDefaultFileUploadChannel(tx); err != nil {
				return err
			}
		}
		return tx.Save(existing).Error
	}); err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(existing, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func SetDefaultFileUploadChannel(c *gin.Context) {
	channel, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if channel.Status != service.FileUploadChannelStatusEnabled {
		common.ApiErrorMsg(c, "only enabled channels can be set as default")
		return
	}

	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := clearDefaultFileUploadChannel(tx); err != nil {
			return err
		}
		channel.IsDefault = service.FileUploadChannelIsDefault
		channel.UpdateUserId = int64(c.GetInt("id"))
		return tx.Model(&model.FileUploadChannel{}).
			Where("id = ?", channel.Id).
			Updates(map[string]any{
				"is_default":     service.FileUploadChannelIsDefault,
				"update_user_id": channel.UpdateUserId,
			}).Error
	}); err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(channel, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func UpdateFileUploadChannelStatus(c *gin.Context) {
	channel, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	var req FileUploadChannelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	req.Status = strings.TrimSpace(req.Status)
	if !service.IsValidFileUploadChannelStatus(req.Status) {
		common.ApiErrorMsg(c, "invalid status")
		return
	}

	if req.Status == service.FileUploadChannelStatusDisabled && channel.IsDefault == service.FileUploadChannelIsDefault {
		if err := ensureAnotherDefaultFileUploadChannel(model.DB, channel.Id); err != nil {
			common.ApiError(c, err)
			return
		}
	}
	if req.Status == service.FileUploadChannelStatusDisabled {
		referenced, err := model.IsCreativeStudioFileChannel(channel.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if referenced {
			common.ApiErrorMsg(c, "channel is the creative studio default; choose another channel first")
			return
		}
	}

	channel.Status = req.Status
	channel.UpdateUserId = int64(c.GetInt("id"))
	if err := model.DB.Model(&model.FileUploadChannel{}).
		Where("id = ?", channel.Id).
		Updates(map[string]any{
			"status":         channel.Status,
			"update_user_id": channel.UpdateUserId,
		}).Error; err != nil {
		common.ApiError(c, err)
		return
	}

	response, err := buildFileUploadChannelResponse(channel, true)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, response)
}

func DeleteFileUploadChannel(c *gin.Context) {
	channel, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := ensureFileUploadChannelNotReferenced(channel.Id); err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DB.Delete(&model.FileUploadChannel{}, channel.Id).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": channel.Id})
}

func ProbeSavedFileUploadChannel(c *gin.Context) {
	channel, err := getFileUploadChannelByParamID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.ProbeFileUploadChannel(c.Request.Context(), channel.Type, config)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func ProbeDraftFileUploadChannel(c *gin.Context) {
	var req FileUploadChannelProbeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}

	existingConfig := map[string]any{}
	if req.Id > 0 {
		channel, err := getFileUploadChannelByID(req.Id)
		if err != nil {
			common.ApiError(c, err)
			return
		}
		if strings.TrimSpace(req.Type) != "" && strings.TrimSpace(req.Type) != channel.Type {
			common.ApiErrorMsg(c, "channel type does not match the saved channel")
			return
		}
		if strings.TrimSpace(req.Type) == "" {
			req.Type = channel.Type
		}
		existingConfig, err = service.ParseFileUploadChannelConfig(channel.ConfigProflle)
		if err != nil {
			common.ApiError(c, err)
			return
		}
	}

	config, err := service.MergeAndValidateFileUploadChannelConfig(
		strings.TrimSpace(req.Type),
		req.ConfigProflle,
		existingConfig,
		req.Id == 0,
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	result, err := service.ProbeFileUploadChannel(c.Request.Context(), strings.TrimSpace(req.Type), config)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetDefaultFileUploadChannel(c *gin.Context) {
	channel, err := resolveFileUploadChannel("", 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if channel == nil {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, buildFileUploadChannelSummary(channel))
}

func ResolveFileUploadChannel(c *gin.Context) {
	channelType := strings.TrimSpace(c.Query("channel_type"))
	channel, err := resolveFileUploadChannel(channelType, 0)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if channel == nil {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, buildFileUploadChannelSummary(channel))
}

func buildFileUploadChannelForCreate(c *gin.Context, req *FileUploadChannelUpsertRequest) (*model.FileUploadChannel, error) {
	channelType := strings.TrimSpace(req.Type)
	if channelType == service.FileUploadChannelTypeLocal {
		return nil, fmt.Errorf("type=0 local channel cannot be created by API")
	}
	if !service.IsSupportedFileUploadChannelType(channelType) {
		return nil, fmt.Errorf("invalid file upload channel type: %s", channelType)
	}

	status := service.FileUploadChannelStatusEnabled
	if req.Status != nil {
		status = strings.TrimSpace(*req.Status)
	}
	if !service.IsValidFileUploadChannelStatus(status) {
		return nil, fmt.Errorf("invalid status")
	}

	isDefault := service.FileUploadChannelNotDefault
	if req.IsDefault != nil {
		isDefault = strings.TrimSpace(*req.IsDefault)
	}
	if !service.IsValidFileUploadChannelDefaultFlag(isDefault) {
		return nil, fmt.Errorf("invalid is_default value")
	}
	if isDefault == service.FileUploadChannelIsDefault && status != service.FileUploadChannelStatusEnabled {
		return nil, fmt.Errorf("only enabled channels can be set as default")
	}

	config, err := service.MergeAndValidateFileUploadChannelConfig(channelType, req.ConfigProflle, nil, true)
	if err != nil {
		return nil, err
	}
	configRaw, err := service.MarshalFileUploadChannelConfig(config)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	chunkThreshold := service.DefaultFileUploadChunkThreshold
	if req.ChunkThreshold != nil {
		chunkThreshold = *req.ChunkThreshold
	}
	chunkSize := service.DefaultFileUploadChunkSize
	if req.ChunkSize != nil {
		chunkSize = *req.ChunkSize
	}
	maxSize := service.DefaultFileUploadMaxSize
	if req.MaxSize != nil {
		maxSize = *req.MaxSize
	}
	if chunkThreshold < 0 || chunkSize < 0 || maxSize < 0 {
		return nil, fmt.Errorf("chunk_threshold, chunk_size, and max_size must be >= 0")
	}

	userID := int64(c.GetInt("id"))
	return &model.FileUploadChannel{
		Name:           name,
		Type:           channelType,
		Status:         status,
		IsDefault:      isDefault,
		ChunkThreshold: chunkThreshold,
		ChunkSize:      chunkSize,
		MaxSize:        maxSize,
		ConfigProflle:  configRaw,
		CreateUserId:   userID,
		UpdateUserId:   userID,
	}, nil
}

func applyFileUploadChannelUpdate(c *gin.Context, channel *model.FileUploadChannel, req *FileUploadChannelUpsertRequest) error {
	if req.Name != "" {
		channel.Name = strings.TrimSpace(req.Name)
	}
	if channel.Name == "" {
		return fmt.Errorf("name is required")
	}

	if req.Status != nil {
		status := strings.TrimSpace(*req.Status)
		if !service.IsValidFileUploadChannelStatus(status) {
			return fmt.Errorf("invalid status")
		}
		channel.Status = status
	}

	if req.IsDefault != nil {
		isDefault := strings.TrimSpace(*req.IsDefault)
		if !service.IsValidFileUploadChannelDefaultFlag(isDefault) {
			return fmt.Errorf("invalid is_default value")
		}
		channel.IsDefault = isDefault
	}

	if req.ChunkThreshold != nil {
		channel.ChunkThreshold = *req.ChunkThreshold
	}
	if req.ChunkSize != nil {
		channel.ChunkSize = *req.ChunkSize
	}
	if req.MaxSize != nil {
		channel.MaxSize = *req.MaxSize
	}
	if channel.ChunkThreshold < 0 || channel.ChunkSize < 0 || channel.MaxSize < 0 {
		return fmt.Errorf("chunk_threshold, chunk_size, and max_size must be >= 0")
	}

	existingConfig, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return err
	}
	mergedConfig, err := service.MergeAndValidateFileUploadChannelConfig(
		channel.Type,
		req.ConfigProflle,
		existingConfig,
		false,
	)
	if err != nil {
		return err
	}
	configRaw, err := service.MarshalFileUploadChannelConfig(mergedConfig)
	if err != nil {
		return err
	}
	channel.ConfigProflle = configRaw
	channel.UpdateUserId = int64(c.GetInt("id"))
	return nil
}

func buildFileUploadChannelResponse(channel *model.FileUploadChannel, includeConfig bool) (*fileUploadChannelResponse, error) {
	response := &fileUploadChannelResponse{
		Id:             channel.Id,
		Name:           channel.Name,
		Type:           channel.Type,
		Status:         channel.Status,
		IsDefault:      channel.IsDefault,
		ChunkThreshold: channel.ChunkThreshold,
		ChunkSize:      channel.ChunkSize,
		MaxSize:        channel.MaxSize,
		CreateUserId:   channel.CreateUserId,
		UpdateUserId:   channel.UpdateUserId,
		CreateTime:     channel.CreateTime.Unix(),
		UpdateTime:     channel.UpdateTime.Unix(),
	}
	if !includeConfig {
		return response, nil
	}

	config, err := service.ParseFileUploadChannelConfig(channel.ConfigProflle)
	if err != nil {
		return nil, err
	}
	response.ConfigProflle = service.MaskFileUploadChannelConfig(channel.Type, config)
	return response, nil
}

func buildFileUploadChannelSummary(channel *model.FileUploadChannel) *fileUploadChannelSummaryResponse {
	return &fileUploadChannelSummaryResponse{
		Id:             channel.Id,
		Name:           channel.Name,
		Type:           channel.Type,
		Status:         channel.Status,
		IsDefault:      channel.IsDefault,
		ChunkThreshold: channel.ChunkThreshold,
		ChunkSize:      channel.ChunkSize,
		MaxSize:        channel.MaxSize,
	}
}

func getFileUploadChannelByParamID(c *gin.Context) (*model.FileUploadChannel, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid channel id")
	}
	return getFileUploadChannelByID(id)
}

func getFileUploadChannelByID(id uint64) (*model.FileUploadChannel, error) {
	var channel model.FileUploadChannel
	if err := model.DB.Where("id = ?", id).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("file upload channel not found")
		}
		return nil, err
	}
	return &channel, nil
}

func clearDefaultFileUploadChannel(tx *gorm.DB) error {
	return tx.Model(&model.FileUploadChannel{}).
		Where("is_default = ?", service.FileUploadChannelIsDefault).
		Update("is_default", service.FileUploadChannelNotDefault).Error
}

func ensureAnotherDefaultFileUploadChannel(db *gorm.DB, excludedID uint64) error {
	var count int64
	if err := db.Model(&model.FileUploadChannel{}).
		Where("id <> ? AND is_default = ? AND status = ?", excludedID, service.FileUploadChannelIsDefault, service.FileUploadChannelStatusEnabled).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("another enabled default channel is required before disabling or unsetting the current default")
	}
	return nil
}

func ensureFileUploadChannelNotReferenced(channelID uint64) error {
	referenced, err := model.IsCreativeStudioFileChannel(channelID)
	if err != nil {
		return err
	}
	if referenced {
		return fmt.Errorf("channel is the creative studio default; choose another channel first")
	}
	for _, tableName := range []string{"file", "user_file"} {
		if !model.DB.Migrator().HasTable(tableName) {
			continue
		}
		var count int64
		if err := model.DB.Table(tableName).
			Where("file_channel_id = ?", channelID).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return fmt.Errorf("channel is in use and cannot be deleted")
		}
	}
	return nil
}

func resolveFileUploadChannel(channelType string, preferredID uint64) (*model.FileUploadChannel, error) {
	query := model.DB.Model(&model.FileUploadChannel{}).
		Where("status = ?", service.FileUploadChannelStatusEnabled)

	if preferredID > 0 {
		query = query.Where("id = ?", preferredID)
	}
	if channelType != "" {
		query = query.Where("type = ?", channelType)
	}

	var channel model.FileUploadChannel
	err := query.
		Where("is_default = ?", service.FileUploadChannelIsDefault).
		Order("id asc").
		First(&channel).Error
	if err == nil {
		return &channel, nil
	}
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if preferredID > 0 || channelType == "" {
		return nil, nil
	}

	err = model.DB.Model(&model.FileUploadChannel{}).
		Where("status = ? AND type = ?", service.FileUploadChannelStatusEnabled, channelType).
		Order("id asc").
		First(&channel).Error
	if err == nil {
		return &channel, nil
	}
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return nil, err
}
