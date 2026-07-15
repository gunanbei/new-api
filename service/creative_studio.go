package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"gorm.io/gorm"
)

type CreativeStudioSettings struct {
	Version                      int      `json:"version"`
	DefaultFileChannelID         uint64   `json:"default_file_channel_id"`
	AllowedImageMIMETypes        []string `json:"allowed_image_mime_types"`
	MaxRasterBytes               int64    `json:"max_raster_bytes"`
	MaxSVGBytes                  int64    `json:"max_svg_bytes"`
	ReferenceUploadQueueSize     int      `json:"reference_upload_queue_size"`
	ReferenceUploadMaxConcurrent int      `json:"reference_upload_max_concurrent"`
}

var defaultCreativeImageMIMETypes = []string{"image/png", "image/jpeg", "image/webp", "image/gif", "image/svg+xml"}

func defaultCreativeStudioSettings() CreativeStudioSettings {
	return CreativeStudioSettings{Version: 3, AllowedImageMIMETypes: append([]string(nil), defaultCreativeImageMIMETypes...), MaxRasterBytes: 32 << 20, MaxSVGBytes: 1 << 20, ReferenceUploadQueueSize: 16, ReferenceUploadMaxConcurrent: 1}
}

func GetCreativeStudioSettings() (CreativeStudioSettings, error) {
	settings := defaultCreativeStudioSettings()
	raw, err := model.GetOptionValue(model.CreativeStudioSettingsOption)
	if errors.Is(err, gorm.ErrRecordNotFound) || strings.TrimSpace(raw) == "" {
		return settings, nil
	}
	if err != nil {
		return settings, err
	}
	if err := common.UnmarshalJsonStr(raw, &settings); err != nil {
		return settings, err
	}
	settings.Version = 3
	if len(settings.AllowedImageMIMETypes) == 0 {
		settings.AllowedImageMIMETypes = append([]string(nil), defaultCreativeImageMIMETypes...)
	}
	if settings.MaxRasterBytes <= 0 {
		settings.MaxRasterBytes = 32 << 20
	}
	if settings.MaxSVGBytes <= 0 {
		settings.MaxSVGBytes = 1 << 20
	}
	if settings.ReferenceUploadQueueSize < 0 {
		settings.ReferenceUploadQueueSize = 16
	}
	if settings.ReferenceUploadMaxConcurrent < 1 {
		settings.ReferenceUploadMaxConcurrent = 1
	}
	return settings, nil
}

func UpdateCreativeStudioSettings(input dto.CreativeStudioSettingsRequest) (CreativeStudioSettings, error) {
	settings := defaultCreativeStudioSettings()
	settings.DefaultFileChannelID = input.DefaultFileChannelID
	if len(input.AllowedImageMIMETypes) > 0 {
		settings.AllowedImageMIMETypes = input.AllowedImageMIMETypes
	}
	if input.MaxRasterBytes > 0 {
		settings.MaxRasterBytes = input.MaxRasterBytes
	}
	if input.MaxSVGBytes > 0 {
		settings.MaxSVGBytes = input.MaxSVGBytes
	}
	if input.ReferenceUploadQueueSize > 0 {
		settings.ReferenceUploadQueueSize = input.ReferenceUploadQueueSize
	}
	if input.ReferenceUploadMaxConcurrent > 0 {
		settings.ReferenceUploadMaxConcurrent = input.ReferenceUploadMaxConcurrent
	}
	if settings.MaxRasterBytes > 32<<20 || settings.MaxSVGBytes > 1<<20 {
		return settings, fmt.Errorf("creative asset size limit exceeds the safe maximum")
	}
	if input.DefaultFileChannelID > 0 {
		var channel model.FileUploadChannel
		if err := model.DB.Where("id = ?", input.DefaultFileChannelID).First(&channel).Error; err != nil {
			return settings, fmt.Errorf("file upload channel not found")
		}
		if channel.Status != FileUploadChannelStatusEnabled || channel.Type == FileUploadChannelTypeLocal {
			return settings, fmt.Errorf("default file channel must be an enabled non-local channel")
		}
	}
	raw, err := common.Marshal(settings)
	if err != nil {
		return settings, err
	}
	return settings, model.UpdateOption(model.CreativeStudioSettingsOption, string(raw))
}

func ListCreativeModels() ([]model.CreativeModel, error) {
	var models []model.CreativeModel
	err := model.DB.Order("sort_order asc, id asc").Find(&models).Error
	return models, err
}

func CreateCreativeModel(input dto.CreativeModelRequest, userID int64) (*model.CreativeModel, error) {
	creativeModel := &model.CreativeModel{CreatedBy: userID, UpdatedBy: userID}
	if err := applyCreativeModel(creativeModel, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeModel(creativeModel); err != nil {
		return nil, err
	}
	if err := model.DB.Create(creativeModel).Error; err != nil {
		return nil, translateCreativeConflict(err, creativeModelDuplicateError(creativeModel))
	}
	return creativeModel, nil
}

func UpdateCreativeModel(id uint64, input dto.CreativeModelRequest, userID int64) (*model.CreativeModel, error) {
	var creativeModel model.CreativeModel
	if err := model.DB.Where("id = ?", id).First(&creativeModel).Error; err != nil {
		return nil, err
	}
	oldModelName := creativeModel.ModelName
	if err := applyCreativeModel(&creativeModel, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeModel(&creativeModel); err != nil {
		return nil, err
	}
	creativeModel.UpdatedBy = userID
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(&creativeModel).Error; err != nil {
			return err
		}
		if oldModelName == creativeModel.ModelName {
			return nil
		}
		var capabilityIDs []uint64
		if err := tx.Model(&model.CreativeModelCapability{}).Where("model_id = ?", creativeModel.ID).Pluck("id", &capabilityIDs).Error; err != nil {
			return err
		}
		var publicationIDs []uint64
		if err := tx.Model(&model.CreativeModelPublication{}).Where("capability_id IN ?", capabilityIDs).Pluck("id", &publicationIDs).Error; err != nil {
			return err
		}
		if len(publicationIDs) == 0 {
			return nil
		}
		return tx.Model(&model.CreativeChannelBinding{}).Where("publication_id IN ?", publicationIDs).Updates(map[string]any{"request_model": creativeModel.ModelName, "enabled": false, "validation_status": "unverified", "validation_message": "model name changed; validation required", "validation_checked_at": nil, "updated_by": userID}).Error
	}); err != nil {
		return nil, translateCreativeConflict(err, creativeModelDuplicateError(&creativeModel))
	}
	return &creativeModel, nil
}

func DeleteCreativeModel(id uint64) error {
	var count int64
	if err := model.DB.Model(&model.CreativeModelCapability{}).Where("model_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("model has capabilities and cannot be deleted")
	}
	return model.DB.Delete(&model.CreativeModel{}, id).Error
}

func ListCreativeCapabilities(modelID uint64) ([]model.CreativeModelCapability, error) {
	var capabilities []model.CreativeModelCapability
	err := model.DB.Where("model_id = ?", modelID).Order("sort_order asc, id asc").Find(&capabilities).Error
	return capabilities, err
}

func CreateCreativeCapability(modelID uint64, input dto.CreativeCapabilityRequest, userID int64) (*model.CreativeModelCapability, error) {
	if err := ensureCreativeModel(modelID); err != nil {
		return nil, err
	}
	capability := &model.CreativeModelCapability{ModelID: modelID, CreatedBy: userID, UpdatedBy: userID}
	if err := applyCreativeCapability(capability, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeCapability(capability); err != nil {
		return nil, err
	}
	if err := model.DB.Create(capability).Error; err != nil {
		return nil, translateCreativeConflict(err, creativeCapabilityDuplicateError(capability))
	}
	return capability, nil
}

func UpdateCreativeCapability(id uint64, input dto.CreativeCapabilityRequest, userID int64) (*model.CreativeModelCapability, error) {
	var capability model.CreativeModelCapability
	if err := model.DB.Where("id = ?", id).First(&capability).Error; err != nil {
		return nil, err
	}
	if err := applyCreativeCapability(&capability, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeCapability(&capability); err != nil {
		return nil, err
	}
	capability.UpdatedBy = userID
	if err := model.DB.Save(&capability).Error; err != nil {
		return nil, translateCreativeConflict(err, creativeCapabilityDuplicateError(&capability))
	}
	return &capability, nil
}

func DeleteCreativeCapability(id uint64) error {
	var count int64
	if err := model.DB.Model(&model.CreativeModelPublication{}).Where("capability_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("capability has publications and cannot be deleted")
	}
	return model.DB.Delete(&model.CreativeModelCapability{}, id).Error
}

func ListCreativePublications(capabilityID uint64) ([]model.CreativeModelPublication, error) {
	var publications []model.CreativeModelPublication
	err := model.DB.Where("capability_id = ?", capabilityID).Order("sort_order asc, id asc").Find(&publications).Error
	return publications, err
}

func CreateCreativePublication(capabilityID uint64, input dto.CreativePublicationRequest, userID int64) (*model.CreativeModelPublication, error) {
	if err := ensureCreativeCapability(capabilityID); err != nil {
		return nil, err
	}
	publication := &model.CreativeModelPublication{CapabilityID: capabilityID, CreatedBy: userID, UpdatedBy: userID}
	if err := applyCreativePublication(publication, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativePublication(publication); err != nil {
		return nil, err
	}
	if err := model.DB.Create(publication).Error; err != nil {
		return nil, translateCreativeConflict(err, creativePublicationDuplicateError(publication))
	}
	return publication, nil
}

func UpdateCreativePublication(id uint64, input dto.CreativePublicationRequest, userID int64) (*model.CreativeModelPublication, error) {
	var publication model.CreativeModelPublication
	if err := model.DB.Where("id = ?", id).First(&publication).Error; err != nil {
		return nil, err
	}
	if err := applyCreativePublication(&publication, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativePublication(&publication); err != nil {
		return nil, err
	}
	publication.UpdatedBy = userID
	if err := model.DB.Save(&publication).Error; err != nil {
		return nil, translateCreativeConflict(err, creativePublicationDuplicateError(&publication))
	}
	return &publication, nil
}

func DeleteCreativePublication(id uint64) error {
	var count int64
	if err := model.DB.Model(&model.CreativeChannelBinding{}).Where("publication_id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return fmt.Errorf("publication has channel bindings and cannot be deleted")
	}
	return model.DB.Delete(&model.CreativeModelPublication{}, id).Error
}

func ListCreativeBindings(publicationID uint64) ([]model.CreativeChannelBinding, error) {
	var bindings []model.CreativeChannelBinding
	err := model.DB.Where("publication_id = ?", publicationID).Order("priority desc, id asc").Find(&bindings).Error
	return bindings, err
}

func CreateCreativeBinding(publicationID uint64, input dto.CreativeBindingRequest, userID int64) (*model.CreativeChannelBinding, error) {
	binding := &model.CreativeChannelBinding{PublicationID: publicationID, CreatedBy: userID, UpdatedBy: userID}
	if err := applyCreativeBinding(binding, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeBinding(binding); err != nil {
		return nil, err
	}
	if err := model.DB.Create(binding).Error; err != nil {
		return nil, translateCreativeConflict(err, creativeBindingDuplicateError(binding))
	}
	return binding, nil
}

func UpdateCreativeBinding(id uint64, input dto.CreativeBindingRequest, userID int64) (*model.CreativeChannelBinding, error) {
	var binding model.CreativeChannelBinding
	if err := model.DB.Where("id = ?", id).First(&binding).Error; err != nil {
		return nil, err
	}
	if err := applyCreativeBinding(&binding, input); err != nil {
		return nil, err
	}
	if err := ensureUniqueCreativeBinding(&binding); err != nil {
		return nil, err
	}
	binding.UpdatedBy = userID
	if err := model.DB.Save(&binding).Error; err != nil {
		return nil, translateCreativeConflict(err, creativeBindingDuplicateError(&binding))
	}
	return &binding, nil
}

func DeleteCreativeBinding(id uint64) error {
	return model.DB.Delete(&model.CreativeChannelBinding{}, id).Error
}

func RevalidateCreativeBinding(id uint64, userID int64) (*model.CreativeChannelBinding, error) {
	var binding model.CreativeChannelBinding
	if err := model.DB.Where("id = ?", id).First(&binding).Error; err != nil {
		return nil, err
	}
	startedAt := time.Now()
	result := model.DB.Model(&model.CreativeChannelBinding{}).Where("id = ? AND validation_status <> ?", id, "validating").Updates(map[string]any{"enabled": false, "validation_status": "validating", "validation_message": "validation in progress", "validation_checked_at": nil, "updated_by": userID, "updated_at": startedAt})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("creative binding validation is already running")
	}
	valid, message := validateCreativeBinding(&binding)
	completedAt := time.Now()
	updates := map[string]any{"enabled": valid, "validation_status": "invalid", "validation_message": message, "validation_checked_at": completedAt, "updated_by": userID}
	if valid {
		updates["validation_status"] = "valid"
		updates["validation_message"] = "validated"
	}
	result = model.DB.Model(&model.CreativeChannelBinding{}).Where("id = ? AND validation_status = ?", id, "validating").Updates(updates)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, fmt.Errorf("creative binding validation state changed unexpectedly")
	}
	if err := model.DB.First(&binding, id).Error; err != nil {
		return nil, err
	}
	return &binding, nil
}

func validateCreativeBinding(binding *model.CreativeChannelBinding) (bool, string) {
	var publication model.CreativeModelPublication
	if err := model.DB.Where("id = ?", binding.PublicationID).First(&publication).Error; err != nil || !publication.Enabled {
		return false, "publication is unavailable"
	}
	if _, exists := setting.GetUserUsableGroupsCopy()[publication.GroupName]; !exists {
		return false, "publication group is unavailable"
	}
	var capability model.CreativeModelCapability
	if err := model.DB.Where("id = ?", publication.CapabilityID).First(&capability).Error; err != nil || !capability.Enabled {
		return false, "capability is unavailable"
	}
	if !creativeProtocolMatches(capability) {
		return false, "protocol does not match the image capability contract"
	}
	var creativeModel model.CreativeModel
	if err := model.DB.Where("id = ?", capability.ModelID).First(&creativeModel).Error; err != nil || creativeModel.Status != "enabled" {
		return false, "model is unavailable"
	}
	if binding.RequestModel != creativeModel.ModelName {
		return false, "binding model snapshot is stale"
	}
	settings, err := GetCreativeStudioSettings()
	if err != nil || !creativeStorageAvailable(settings) {
		return false, "creative storage is unavailable"
	}
	var channel model.Channel
	if err := model.DB.Where("id = ? AND status = ?", binding.ChannelID, common.ChannelStatusEnabled).First(&channel).Error; err != nil {
		return false, "channel is unavailable"
	}
	var count int64
	if err := model.DB.Model(&model.Ability{}).Where(&model.Ability{ChannelId: binding.ChannelID, Enabled: true, Model: creativeModel.ModelName, Group: publication.GroupName}).Count(&count).Error; err != nil || count == 0 {
		return false, "channel has no enabled matching ability"
	}
	if capability.Protocol == "midjourney_image" && channel.Type != constant.ChannelTypeMidjourney && channel.Type != constant.ChannelTypeMidjourneyPlus {
		return false, "Midjourney protocol requires a Midjourney channel"
	}
	if capability.Protocol == "advanced_custom_image" {
		if channel.Type != constant.ChannelTypeAdvancedCustom {
			return false, "advanced custom image protocol requires an Advanced Custom channel"
		}
		requestPath := "/v1/images/generations"
		if capability.Operation == "edit" {
			requestPath = "/v1/images/edits"
		}
		if _, matched := channel.GetOtherSettings().AdvancedCustom.MatchPath(requestPath); !matched {
			return false, "advanced custom channel has no matching image route"
		}
	}
	return true, ""
}

func CreativeStudioBootstrap() (map[string]any, error) {
	models, err := ListCreativeModels()
	if err != nil {
		return nil, err
	}
	capabilities := make(map[uint64][]model.CreativeModelCapability, len(models))
	publications := make(map[uint64][]model.CreativeModelPublication)
	bindings := make(map[uint64][]model.CreativeChannelBinding)
	for _, creativeModel := range models {
		items, err := ListCreativeCapabilities(creativeModel.ID)
		if err != nil {
			return nil, err
		}
		capabilities[creativeModel.ID] = items
		for _, capability := range items {
			publicationItems, err := ListCreativePublications(capability.ID)
			if err != nil {
				return nil, err
			}
			publications[capability.ID] = publicationItems
			for _, publication := range publicationItems {
				bindingItems, err := ListCreativeBindings(publication.ID)
				if err != nil {
					return nil, err
				}
				bindings[publication.ID] = bindingItems
			}
		}
	}
	settings, err := GetCreativeStudioSettings()
	if err != nil {
		return nil, err
	}
	var channels []struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status int    `json:"status"`
	}
	if err := model.DB.Model(&model.Channel{}).Select("id, name, status").Order("id asc").Find(&channels).Error; err != nil {
		return nil, err
	}
	var fileChannels []struct {
		ID     uint64 `json:"id"`
		Name   string `json:"name"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	if err := model.DB.Model(&model.FileUploadChannel{}).Select("id, name, type, status").Order("id asc").Find(&fileChannels).Error; err != nil {
		return nil, err
	}
	return map[string]any{"models": models, "capabilities": capabilities, "publications": publications, "bindings": bindings, "groups": setting.GetUserUsableGroupsCopy(), "channels": channels, "file_channels": fileChannels, "settings": settings}, nil
}

func applyCreativeModel(target *model.CreativeModel, input dto.CreativeModelRequest) error {
	target.ModelName = strings.TrimSpace(input.ModelName)
	if target.ModelName == "" || len(target.ModelName) > 255 {
		return fmt.Errorf("model_name must be between 1 and 255 characters")
	}
	target.ModelKey = model.CreativeModelKey(target.ModelName)
	target.DisplayName = strings.TrimSpace(input.DisplayName)
	if target.DisplayName == "" {
		target.DisplayName = target.ModelName
	}
	target.Vendor = strings.TrimSpace(input.Vendor)
	target.Description = strings.TrimSpace(input.Description)
	target.Status = strings.TrimSpace(input.Status)
	if target.Status == "" {
		target.Status = "enabled"
	}
	if target.Status != "enabled" && target.Status != "disabled" {
		return fmt.Errorf("invalid model status")
	}
	target.SortOrder = input.SortOrder
	return nil
}

func applyCreativeCapability(target *model.CreativeModelCapability, input dto.CreativeCapabilityRequest) error {
	target.Category, target.Operation, target.AssetKind = strings.TrimSpace(input.Category), strings.TrimSpace(input.Operation), strings.TrimSpace(input.AssetKind)
	target.Protocol, target.ExecutionMode = strings.TrimSpace(input.Protocol), strings.TrimSpace(input.ExecutionMode)
	if (target.Category != "image" && target.Category != "video") || target.Operation == "" || target.Protocol == "" {
		return fmt.Errorf("invalid capability")
	}
	if target.Category == "image" && target.Operation != "generate" && target.Operation != "edit" {
		return fmt.Errorf("invalid image operation")
	}
	if target.Category == "video" && target.Operation != "generate" && target.Operation != "remix" {
		return fmt.Errorf("invalid video operation")
	}
	if target.Category == "image" && target.AssetKind != "raster" && target.AssetKind != "vector" {
		return fmt.Errorf("image asset_kind must be raster or vector")
	}
	if target.Category == "video" && target.AssetKind != "video" {
		return fmt.Errorf("video asset_kind must be video")
	}
	if target.ExecutionMode != "sync" && target.ExecutionMode != "async" && target.ExecutionMode != "sync_artifact" {
		return fmt.Errorf("invalid execution_mode")
	}
	if target.Category == "image" && !creativeProtocolMatches(*target) {
		return fmt.Errorf("protocol does not match the image capability contract")
	}
	inputSchema, err := normalizeCreativeJSONObject(input.InputSchema)
	if err != nil {
		return fmt.Errorf("input_schema: %w", err)
	}
	defaultParams, err := normalizeCreativeJSONObject(input.DefaultParams)
	if err != nil {
		return fmt.Errorf("default_params: %w", err)
	}
	target.InputSchema, target.DefaultParams, target.Enabled, target.SortOrder = inputSchema, defaultParams, input.Enabled, input.SortOrder
	return nil
}

func applyCreativePublication(target *model.CreativeModelPublication, input dto.CreativePublicationRequest) error {
	target.GroupName = strings.TrimSpace(input.GroupName)
	if _, exists := setting.GetUserUsableGroupsCopy()[target.GroupName]; !exists {
		return fmt.Errorf("group_name does not exist")
	}
	raw, err := normalizeCreativeJSONObject(input.GroupDefaultParams)
	if err != nil {
		return fmt.Errorf("group_default_params: %w", err)
	}
	target.GroupDefaultParams, target.Enabled, target.SortOrder = raw, input.Enabled, input.SortOrder
	return nil
}

func applyCreativeBinding(target *model.CreativeChannelBinding, input dto.CreativeBindingRequest) error {
	if input.Enabled {
		return fmt.Errorf("stage one bindings cannot be enabled until verified")
	}
	target.ChannelID, target.RequestModel, target.Priority, target.Enabled = input.ChannelID, strings.TrimSpace(input.RequestModel), input.Priority, false
	target.ValidationStatus, target.ValidationMessage, target.ValidationCheckedAt = "unverified", "not validated", nil
	var publication model.CreativeModelPublication
	if err := model.DB.Where("id = ?", target.PublicationID).First(&publication).Error; err != nil {
		return err
	}
	var capability model.CreativeModelCapability
	if err := model.DB.Where("id = ?", publication.CapabilityID).First(&capability).Error; err != nil {
		return err
	}
	var creativeModel model.CreativeModel
	if err := model.DB.Where("id = ?", capability.ModelID).First(&creativeModel).Error; err != nil {
		return err
	}
	if target.RequestModel == "" {
		target.RequestModel = creativeModel.ModelName
	}
	return nil
}

func normalizeCreativeJSONObject(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || strings.TrimSpace(string(raw)) == "" {
		return "{}", nil
	}
	var value map[string]any
	if err := common.Unmarshal(raw, &value); err != nil {
		return "", errors.New("must be a JSON object")
	}
	if value == nil {
		return "", errors.New("must be a JSON object")
	}
	normalized, err := common.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(normalized), nil
}

func ensureCreativeModel(id uint64) error {
	return model.DB.Where("id = ?", id).First(&model.CreativeModel{}).Error
}
func ensureCreativeCapability(id uint64) error {
	return model.DB.Where("id = ?", id).First(&model.CreativeModelCapability{}).Error
}

func ensureUniqueCreativeModel(target *model.CreativeModel) error {
	var count int64
	if err := model.DB.Model(&model.CreativeModel{}).Where("model_key = ? AND id <> ?", target.ModelKey, target.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return creativeModelDuplicateError(target)
	}
	return nil
}

func ensureUniqueCreativeCapability(target *model.CreativeModelCapability) error {
	var count int64
	if err := model.DB.Model(&model.CreativeModelCapability{}).Where("model_id = ? AND category = ? AND operation = ? AND protocol = ? AND id <> ?", target.ModelID, target.Category, target.Operation, target.Protocol, target.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return creativeCapabilityDuplicateError(target)
	}
	return nil
}

func ensureUniqueCreativePublication(target *model.CreativeModelPublication) error {
	var count int64
	if err := model.DB.Model(&model.CreativeModelPublication{}).Where("capability_id = ? AND group_name = ? AND id <> ?", target.CapabilityID, target.GroupName, target.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return creativePublicationDuplicateError(target)
	}
	return nil
}

func ensureUniqueCreativeBinding(target *model.CreativeChannelBinding) error {
	var count int64
	if err := model.DB.Model(&model.CreativeChannelBinding{}).Where("publication_id = ? AND channel_id = ? AND id <> ?", target.PublicationID, target.ChannelID, target.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return creativeBindingDuplicateError(target)
	}
	return nil
}

func creativeModelDuplicateError(target *model.CreativeModel) error {
	return fmt.Errorf("模型目录：%s 已存在", target.ModelName)
}

func creativeCapabilityDuplicateError(target *model.CreativeModelCapability) error {
	return fmt.Errorf("能力：%s/%s｜%s 已存在", target.Category, target.Operation, target.Protocol)
}

func creativePublicationDuplicateError(target *model.CreativeModelPublication) error {
	return fmt.Errorf("分组：%s 已存在", target.GroupName)
}

func creativeBindingDuplicateError(target *model.CreativeChannelBinding) error {
	channelName := fmt.Sprintf("#%d", target.ChannelID)
	var channel model.Channel
	if err := model.DB.Select("name").Where("id = ?", target.ChannelID).First(&channel).Error; err == nil && strings.TrimSpace(channel.Name) != "" {
		channelName = channel.Name
	}
	return fmt.Errorf("渠道：%s 已存在", channelName)
}

func translateCreativeConflict(err error, conflict error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return conflict
	}
	return err
}
