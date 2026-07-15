package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"gorm.io/gorm"
)

var creativeProtocols = map[string]struct {
	operation, assetKind, executionMode string
}{
	"openai_image":           {"generate|edit", "raster", "sync"},
	"openai_responses_image": {"generate", "raster", "sync"},
	"advanced_custom_image":  {"generate|edit", "raster", "sync"},
	"midjourney_image":       {"generate|edit", "raster", "async"},
	"claude_svg":             {"generate", "vector", "sync_artifact"},
}

const defaultOpenAIImageSchema = `{"properties":{"prompt":{"type":"textarea","label":"Prompt","required":true},"n":{"type":"number","label":"Quantity","presentation":"segmented","options":[1,2,4],"min":1,"max":4},"size":{"type":"select","label":"Size plan","presentation":"size_cards","description":"Choose the image composition and orientation.","options":[{"label":"1:1 Square","value":"1024x1024","aspect_ratio":"1:1"},{"label":"Landscape","value":"1536x1024","aspect_ratio":"3:2"},{"label":"Portrait","value":"1024x1536","aspect_ratio":"2:3"},{"label":"Custom","value":"auto","aspect_ratio":"auto"}]},"download_resolution":{"type":"select","label":"Download resolution","presentation":"segmented","local_only":true,"options":["1K","2K","4K"],"description":"Higher download resolutions are upscaled locally and do not affect generation billing."},"quality":{"type":"select","label":"Quality","options":[{"label":"Auto","value":"auto"},{"label":"Low","value":"low"},{"label":"Medium","value":"medium"},{"label":"High","value":"high"}]},"output_format":{"type":"select","label":"Format","options":[{"label":"PNG","value":"png"},{"label":"JPEG","value":"jpeg"},{"label":"WebP","value":"webp"}]},"background":{"type":"select","label":"Background","presentation":"segmented","options":[{"label":"Auto","value":"auto"},{"label":"Opaque","value":"opaque"},{"label":"Transparent","value":"transparent"}]}}}`
const defaultOpenAIImageParams = `{"n":1,"size":"1024x1024","download_resolution":"1K","quality":"auto","output_format":"png","background":"auto"}`
const defaultClaudeSVGSchema = `{"properties":{"prompt":{"type":"textarea","label":"Prompt","required":true},"n":{"type":"number","label":"Quantity","presentation":"segmented","options":[1,2,4],"min":1,"max":4},"aspect_ratio":{"type":"select","label":"Size plan","presentation":"size_cards","allow_custom":true,"description":"Choose the composition ratio for the SVG illustration.","options":[{"label":"1:1 Square","value":"1:1","aspect_ratio":"1:1"},{"label":"16:9 Landscape","value":"16:9","aspect_ratio":"16:9"},{"label":"9:16 Portrait","value":"9:16","aspect_ratio":"9:16"},{"label":"3:2 Landscape","value":"3:2","aspect_ratio":"3:2"},{"label":"2:3 Portrait","value":"2:3","aspect_ratio":"2:3"}]},"download_resolution":{"type":"select","label":"Download resolution","presentation":"segmented","local_only":true,"options":["1K","2K","4K"],"description":"SVG stays vector-based; 2K and 4K downloads are rendered locally without extra generation charges."}}}`
const defaultClaudeSVGParams = `{"n":1,"aspect_ratio":"1:1","download_resolution":"1K"}`
const defaultMidjourneyImageSchema = `{"properties":{"prompt":{"type":"textarea","label":"Prompt","required":true},"aspect_ratio":{"type":"select","label":"Size plan","presentation":"size_cards","options":[{"label":"1:1 Square","value":"1:1","aspect_ratio":"1:1"},{"label":"16:9 Landscape","value":"16:9","aspect_ratio":"16:9"},{"label":"9:16 Portrait","value":"9:16","aspect_ratio":"9:16"},{"label":"3:2 Landscape","value":"3:2","aspect_ratio":"3:2"},{"label":"2:3 Portrait","value":"2:3","aspect_ratio":"2:3"}]}}`
const defaultMidjourneyImageParams = `{"aspect_ratio":"1:1"}`

func effectiveCreativeCapability(capability model.CreativeModelCapability) model.CreativeModelCapability {
	var schema map[string]any
	if common.UnmarshalJsonStr(capability.InputSchema, &schema) != nil || len(schema) > 0 {
		return capability
	}
	switch capability.Protocol {
	case "openai_image", "advanced_custom_image":
		capability.InputSchema, capability.DefaultParams = defaultOpenAIImageSchema, defaultOpenAIImageParams
	case "claude_svg":
		capability.InputSchema, capability.DefaultParams = defaultClaudeSVGSchema, defaultClaudeSVGParams
	case "midjourney_image":
		capability.InputSchema, capability.DefaultParams = defaultMidjourneyImageSchema, defaultMidjourneyImageParams
	}
	return capability
}

type CreativeTaskListFilter struct {
	Status string
	Model  string
	From   *time.Time
	To     *time.Time
}

type CreativeTaskPage struct {
	Items []model.CreativeTask `json:"items"`
	Total int64                `json:"total"`
}

func PrepareCreativeTask(userID int64, input dto.CreativeTaskCreateRequest) (*model.CreativeTask, error) {
	return prepareCreativeTask(userID, input, 0)
}

func PrepareCreativeTaskWithReferences(userID int64, input dto.CreativeTaskCreateRequest, referenceCount int) (*model.CreativeTask, error) {
	return prepareCreativeTask(userID, input, referenceCount)
}

func prepareCreativeTask(userID int64, input dto.CreativeTaskCreateRequest, referenceCount int) (*model.CreativeTask, error) {
	if err := ValidateCreativeTaskRequest(input); err != nil {
		return nil, err
	}
	var user model.User
	if err := model.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	var capability model.CreativeModelCapability
	if err := model.DB.Where("id = ? AND enabled = ? AND category = ?", input.CapabilityID, true, "image").First(&capability).Error; err != nil || !creativeProtocolMatches(capability) {
		return nil, errors.New("creative capability is unavailable")
	}
	var creativeModel model.CreativeModel
	if err := model.DB.Where("id = ? AND status = ?", capability.ModelID, "enabled").First(&creativeModel).Error; err != nil {
		return nil, errors.New("creative model is unavailable")
	}
	var publication model.CreativeModelPublication
	if err := model.DB.Where("capability_id = ? AND group_name = ? AND enabled = ?", capability.ID, strings.TrimSpace(input.GroupName), true).First(&publication).Error; err != nil {
		return nil, errors.New("creative publication is unavailable")
	}
	groups := GetUserUsableGroups(user.Group)
	if _, ok := groups[publication.GroupName]; !ok {
		return nil, errors.New("group is unavailable for this user")
	}
	settings, err := GetCreativeStudioSettings()
	if err != nil || !creativeStorageAvailable(settings) {
		return nil, errors.New("creative storage is unavailable")
	}
	var bindings []model.CreativeChannelBinding
	if err := model.DB.Where("publication_id = ? AND enabled = ? AND validation_status = ?", publication.ID, true, "valid").Order("priority desc, id asc").Find(&bindings).Error; err != nil {
		return nil, err
	}
	if len(bindings) == 0 {
		return nil, errors.New("no verified creative channel binding")
	}
	var binding *model.CreativeChannelBinding
	for index := range bindings {
		candidate := &bindings[index]
		valid, _ := validateCreativeBinding(candidate)
		if valid {
			binding = candidate
			break
		}
	}
	if binding == nil {
		return nil, errors.New("no verified creative channel binding")
	}
	requested, resolved, inputAssets, err := resolveCreativeTaskParams(userID, capability, publication, input.Params, referenceCount)
	if err != nil {
		return nil, err
	}
	startedAt := time.Now()
	task := &model.CreativeTask{TaskKey: "ct_" + common.NewRequestId(), UserID: userID, Category: "image", Operation: capability.Operation, CapabilityID: capability.ID, BindingID: binding.ID, ModelName: creativeModel.ModelName, DisplayName: creativeModel.DisplayName, GroupName: publication.GroupName, Protocol: capability.Protocol, ExecutionMode: capability.ExecutionMode, ChannelID: binding.ChannelID, RequestID: common.NewRequestId(), Status: model.CreativeTaskStatusDispatching, RequestedParams: requested, ResolvedParams: resolved, ResultManifest: "", StartedAt: &startedAt}
	if err := model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(task).Error; err != nil {
			return err
		}
		for position, asset := range inputAssets {
			asset.TaskID, asset.Position = task.ID, position
			if err := tx.Create(&asset).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return task, nil
}

func CreativeUserBootstrap(userID int64) (map[string]any, error) {
	var user model.User
	if err := model.DB.First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	settings, err := GetCreativeStudioSettings()
	if err != nil {
		return nil, err
	}
	if !creativeStorageAvailable(settings) {
		return map[string]any{"models": []model.CreativeModel{}, "capabilities": map[uint64][]model.CreativeModelCapability{}, "groups": GetUserUsableGroups(user.Group), "storage_available": false}, nil
	}
	groups := GetUserUsableGroups(user.Group)
	models, err := ListCreativeModels()
	if err != nil {
		return nil, err
	}
	visibleModels := make([]model.CreativeModel, 0, len(models))
	capabilities := make(map[uint64][]model.CreativeModelCapability)
	publications := make(map[uint64][]model.CreativeModelPublication)
	for _, creativeModel := range models {
		if creativeModel.Status != "enabled" {
			continue
		}
		items, err := ListCreativeCapabilities(creativeModel.ID)
		if err != nil {
			return nil, err
		}
		for _, capability := range items {
			if !capability.Enabled || capability.Category != "image" || !creativeProtocolMatches(capability) {
				continue
			}
			visiblePublications, err := creativeVisiblePublications(capability, creativeModel, groups)
			if err != nil {
				return nil, err
			}
			if len(visiblePublications) == 0 {
				continue
			}
			capabilities[creativeModel.ID] = append(capabilities[creativeModel.ID], effectiveCreativeCapability(capability))
			publications[capability.ID] = visiblePublications
		}
		if len(capabilities[creativeModel.ID]) > 0 {
			visibleModels = append(visibleModels, creativeModel)
		}
	}
	return map[string]any{"models": visibleModels, "capabilities": capabilities, "publications": publications, "groups": groups, "storage_available": true}, nil
}

func ListCreativeTasks(userID int64, filter CreativeTaskListFilter, offset, limit int) (*CreativeTaskPage, error) {
	query := model.DB.Where("user_id = ?", userID)
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Model != "" {
		query = query.Where("model_name = ?", filter.Model)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", *filter.To)
	}
	var page CreativeTaskPage
	if err := query.Model(&model.CreativeTask{}).Count(&page.Total).Error; err != nil {
		return nil, err
	}
	if err := query.Order("created_at desc").Offset(offset).Limit(limit).Find(&page.Items).Error; err != nil {
		return nil, err
	}
	return &page, nil
}

func GetCreativeTask(userID int64, taskKey string) (*model.CreativeTask, []model.CreativeTaskAsset, error) {
	var task model.CreativeTask
	if err := model.DB.Where("user_id = ? AND task_key = ?", userID, taskKey).First(&task).Error; err != nil {
		return nil, nil, err
	}
	var assets []model.CreativeTaskAsset
	if err := model.DB.Where("task_id = ?", task.ID).Order("role asc, position asc").Find(&assets).Error; err != nil {
		return nil, nil, err
	}
	return &task, assets, nil
}

func DeleteCreativeTask(userID int64, taskKey string) error {
	task, _, err := GetCreativeTask(userID, taskKey)
	if err != nil {
		return err
	}
	if task.Status != model.CreativeTaskStatusSucceeded && task.Status != model.CreativeTaskStatusFailed {
		return errors.New("only completed creative tasks can be deleted")
	}
	return model.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("task_id = ?", task.ID).Delete(&model.CreativeTaskAsset{}).Error; err != nil {
			return err
		}
		return tx.Delete(task).Error
	})
}

func RetryCreativeTask(userID int64, taskKey string) (*model.CreativeTask, error) {
	previous, _, err := GetCreativeTask(userID, taskKey)
	if err != nil {
		return nil, err
	}
	var params map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(previous.RequestedParams, &params); err != nil {
		return nil, err
	}
	next, err := PrepareCreativeTask(userID, dto.CreativeTaskCreateRequest{CapabilityID: previous.CapabilityID, GroupName: previous.GroupName, Params: params})
	if err != nil {
		return nil, err
	}
	if err := model.DB.Model(next).Update("retry_of_id", previous.ID).Error; err != nil {
		return nil, err
	}
	next.RetryOfID = previous.ID
	return next, nil
}

func RetryCreativeImport(userID int64, taskKey string) (*model.CreativeTask, error) {
	task, _, err := GetCreativeTask(userID, taskKey)
	if err != nil {
		return nil, err
	}
	if task.Status != model.CreativeTaskStatusFailed || task.ErrorCode != "asset_import_failed" || task.ImportExpiresAt == nil || !task.ImportExpiresAt.After(time.Now()) || task.ResultManifest == "" {
		return nil, errors.New("generated asset is no longer available for import retry")
	}
	transitioned, err := TransitionCreativeTask(task.ID, []string{model.CreativeTaskStatusFailed}, model.CreativeTaskStatusImporting, map[string]any{"error_code": "", "error_message": "", "finished_at": nil})
	if err != nil {
		return nil, err
	}
	if !transitioned {
		return nil, errors.New("creative task state changed unexpectedly")
	}
	task.Status, task.ErrorCode, task.ErrorMessage = model.CreativeTaskStatusImporting, "", ""
	return task, nil
}

func TransitionCreativeTask(taskID uint64, from []string, to string, updates map[string]any) (bool, error) {
	if !validCreativeTaskStatus(to) || len(from) == 0 {
		return false, errors.New("invalid creative task transition")
	}
	if updates == nil {
		updates = make(map[string]any)
	}
	updates["status"] = to
	result := model.DB.Model(&model.CreativeTask{}).Where("id = ? AND status IN ?", taskID, from).Updates(updates)
	if result.Error != nil {
		common.SysError(fmt.Sprintf("creative task %d transition to %s failed: %v", taskID, to, result.Error))
		return false, result.Error
	}
	if result.RowsAffected == 1 {
		return true, nil
	}
	var task model.CreativeTask
	if err := model.DB.Select("status").Where("id = ?", taskID).First(&task).Error; err != nil {
		return false, err
	}
	if task.Status == to {
		return true, nil
	}
	for _, status := range from {
		if task.Status != status {
			continue
		}
		retry := model.DB.Model(&model.CreativeTask{}).Where("id = ? AND status = ?", taskID, status).Updates(updates)
		if retry.Error != nil {
			common.SysError(fmt.Sprintf("creative task %d transition retry to %s failed: %v", taskID, to, retry.Error))
			return false, retry.Error
		}
		if retry.RowsAffected == 1 {
			return true, nil
		}
		if err := model.DB.Select("status").Where("id = ?", taskID).First(&task).Error; err != nil {
			return false, err
		}
		return task.Status == to, nil
	}
	common.SysError(fmt.Sprintf("creative task %d transition to %s rejected from %s", taskID, to, task.Status))
	return false, nil
}

func MarkCreativeTaskFailed(taskID uint64, errorCode, errorMessage string) {
	_, _ = TransitionCreativeTask(taskID, []string{model.CreativeTaskStatusValidating, model.CreativeTaskStatusDispatching}, model.CreativeTaskStatusFailed, map[string]any{"error_code": errorCode, "error_message": errorMessage, "finished_at": time.Now()})
}

func MarkCreativeTaskImportFailed(taskID uint64, err error) {
	if err == nil {
		return
	}
	_, _ = TransitionCreativeTask(taskID, []string{model.CreativeTaskStatusImporting}, model.CreativeTaskStatusFailed, map[string]any{"error_code": "asset_import_failed", "error_message": err.Error(), "finished_at": time.Now()})
}

func RecoverCreativeTasks() error {
	cutoff := time.Now().Add(-10 * time.Minute)
	return model.DB.Model(&model.CreativeTask{}).
		Where("status IN ? AND updated_at < ?", []string{model.CreativeTaskStatusValidating, model.CreativeTaskStatusDispatching}, cutoff).
		Updates(map[string]any{"status": model.CreativeTaskStatusFailed, "error_code": "interrupted_before_result", "error_message": "generation was interrupted before a result was available", "finished_at": time.Now()}).Error
}

func CreativeTaskChannelCandidates(task *model.CreativeTask) ([]int, error) {
	if task == nil {
		return nil, errors.New("creative task is required")
	}
	var publication model.CreativeModelPublication
	if err := model.DB.Where("capability_id = ? AND group_name = ? AND enabled = ?", task.CapabilityID, task.GroupName, true).First(&publication).Error; err != nil {
		return nil, errors.New("creative publication is unavailable")
	}
	var bindings []model.CreativeChannelBinding
	if err := model.DB.Where("publication_id = ? AND enabled = ? AND validation_status = ?", publication.ID, true, "valid").Order("priority desc, id asc").Find(&bindings).Error; err != nil {
		return nil, err
	}
	channelIDs := make([]int, 0, len(bindings))
	for index := range bindings {
		valid, _ := validateCreativeBinding(&bindings[index])
		if valid {
			channelIDs = append(channelIDs, bindings[index].ChannelID)
		}
	}
	if len(channelIDs) == 0 {
		return nil, errors.New("no verified creative channel binding")
	}
	return channelIDs, nil
}

func creativeStorageAvailable(settings CreativeStudioSettings) bool {
	if settings.DefaultFileChannelID == 0 {
		return false
	}
	var channel model.FileUploadChannel
	return model.DB.Where("id = ?", settings.DefaultFileChannelID).First(&channel).Error == nil && channel.Status == FileUploadChannelStatusEnabled && channel.Type != FileUploadChannelTypeLocal
}

func creativeProtocolMatches(capability model.CreativeModelCapability) bool {
	contract, ok := creativeProtocols[capability.Protocol]
	return ok && capability.ExecutionMode == contract.executionMode && capability.AssetKind == contract.assetKind && containsCreativeOperation(contract.operation, capability.Operation)
}

func containsCreativeOperation(allowed, value string) bool {
	for _, operation := range strings.Split(allowed, "|") {
		if operation == value {
			return true
		}
	}
	return false
}

func creativeVisiblePublications(capability model.CreativeModelCapability, creativeModel model.CreativeModel, groups map[string]string) ([]model.CreativeModelPublication, error) {
	publications, err := ListCreativePublications(capability.ID)
	if err != nil {
		return nil, err
	}
	visible := make([]model.CreativeModelPublication, 0, len(publications))
	for _, publication := range publications {
		if !publication.Enabled {
			continue
		}
		if _, ok := groups[publication.GroupName]; !ok {
			continue
		}
		var bindings []model.CreativeChannelBinding
		if err := model.DB.Where("publication_id = ? AND enabled = ? AND validation_status = ?", publication.ID, true, "valid").Order("priority desc, id asc").Find(&bindings).Error; err != nil {
			return nil, err
		}
		for _, binding := range bindings {
			var channel model.Channel
			result := model.DB.Where("id = ? AND status = ?", binding.ChannelID, common.ChannelStatusEnabled).Limit(1).Find(&channel)
			if result.Error != nil {
				return nil, result.Error
			}
			if channel.Id == 0 {
				continue
			}
			var count int64
			if err := model.DB.Model(&model.Ability{}).Where(&model.Ability{ChannelId: binding.ChannelID, Enabled: true, Model: creativeModel.ModelName, Group: publication.GroupName}).Count(&count).Error; err != nil {
				return nil, err
			}
			if count > 0 {
				visible = append(visible, publication)
				break
			}
		}
	}
	return visible, nil
}

func validCreativeTaskStatus(status string) bool {
	switch status {
	case model.CreativeTaskStatusValidating, model.CreativeTaskStatusDispatching, model.CreativeTaskStatusProcessing, model.CreativeTaskStatusImporting, model.CreativeTaskStatusSucceeded, model.CreativeTaskStatusFailed:
		return true
	}
	return false
}

func ValidateCreativeTaskRequest(input dto.CreativeTaskCreateRequest) error {
	if input.CapabilityID == 0 || strings.TrimSpace(input.GroupName) == "" {
		return errors.New("capability_id and group_name are required")
	}
	if _, exists := input.Params["stream"]; exists {
		return errors.New("stream is not supported for creative tasks")
	}
	if raw, exists := input.Params["n"]; exists {
		var count uint
		if err := common.Unmarshal(raw, &count); err != nil || count == 0 || count > dto.MaxImageN {
			return fmt.Errorf("n must be an integer between 1 and %d", dto.MaxImageN)
		}
	}
	return nil
}

func resolveCreativeTaskParams(userID int64, capability model.CreativeModelCapability, publication model.CreativeModelPublication, submitted map[string]json.RawMessage, directReferenceCounts ...int) (string, string, []model.CreativeTaskAsset, error) {
	directReferenceCount := 0
	if len(directReferenceCounts) > 0 {
		directReferenceCount = directReferenceCounts[0]
	}
	capability = effectiveCreativeCapability(capability)
	var schema map[string]any
	if err := common.UnmarshalJsonStr(capability.InputSchema, &schema); err != nil {
		return "", "", nil, errors.New("invalid capability input schema")
	}
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		properties = schema
	}
	if _, exists := properties["prompt"]; !exists {
		properties["prompt"] = map[string]any{"type": "textarea", "required": true}
	}
	if capability.Operation == "edit" {
		if _, exists := properties["image"]; !exists {
			properties["image"] = map[string]any{"type": "file", "required": true, "multiple": true, "max_items": float64(5)}
		}
	}
	directReferencesAccepted := false
	if directReferenceCount > 0 {
		for _, definition := range properties {
			field, ok := definition.(map[string]any)
			if !ok || field["type"] != "file" {
				continue
			}
			directReferencesAccepted = true
			maxItems := 1
			if multiple, _ := field["multiple"].(bool); multiple {
				maxItems = 5
				if configured, ok := field["max_items"].(float64); ok && configured > 0 && configured < float64(maxItems) {
					maxItems = int(configured)
				}
			}
			if directReferenceCount > maxItems {
				return "", "", nil, fmt.Errorf("reference images: accepts between 1 and %d input images", maxItems)
			}
		}
		if !directReferencesAccepted {
			return "", "", nil, errors.New("reference images are not supported")
		}
	}
	var defaults map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(capability.DefaultParams, &defaults); err != nil {
		return "", "", nil, err
	}
	var groupDefaults map[string]json.RawMessage
	if err := common.UnmarshalJsonStr(publication.GroupDefaultParams, &groupDefaults); err != nil {
		return "", "", nil, err
	}
	resolved := make(map[string]json.RawMessage, len(defaults)+len(groupDefaults)+len(submitted))
	for key, value := range defaults {
		resolved[key] = value
	}
	for key, value := range groupDefaults {
		resolved[key] = value
	}
	settings, err := GetCreativeStudioSettings()
	if err != nil {
		return "", "", nil, err
	}
	assets := make([]model.CreativeTaskAsset, 0)
	for key, value := range submitted {
		if forbiddenCreativeParameter(key) {
			return "", "", nil, fmt.Errorf("unsupported creative parameter: %s", key)
		}
		definition, exists := properties[key]
		if !exists || key == "stream" {
			return "", "", nil, fmt.Errorf("unsupported creative parameter: %s", key)
		}
		field, ok := definition.(map[string]any)
		if !ok {
			return "", "", nil, fmt.Errorf("invalid input schema for %s", key)
		}
		fieldType, _ := field["type"].(string)
		if !validCreativeInputType(fieldType) {
			return "", "", nil, fmt.Errorf("unsupported input type for %s", key)
		}
		if err := validateCreativeInputValue(fieldType, value, field); err != nil {
			return "", "", nil, fmt.Errorf("%s: %w", key, err)
		}
		if fieldType == "file" {
			if directReferenceCount > 0 {
				return "", "", nil, fmt.Errorf("%s: direct reference images cannot be combined with saved files", key)
			}
			inputAssets, err := resolveCreativeInputFiles(userID, key, value, field, settings)
			if err != nil {
				return "", "", nil, err
			}
			assets = append(assets, inputAssets...)
		}
		resolved[key] = value
	}
	for key, definition := range properties {
		field, ok := definition.(map[string]any)
		if !ok || field["required"] != true {
			continue
		}
		if field["type"] == "file" && directReferenceCount > 0 {
			continue
		}
		if _, exists := resolved[key]; !exists {
			return "", "", nil, fmt.Errorf("%s: is required", key)
		}
	}
	requested, err := common.Marshal(submitted)
	if err != nil {
		return "", "", nil, err
	}
	resolvedRaw, err := common.Marshal(resolved)
	if err != nil {
		return "", "", nil, err
	}
	return string(requested), string(resolvedRaw), assets, nil
}

func resolveCreativeInputFiles(userID int64, key string, value json.RawMessage, definition map[string]any, settings CreativeStudioSettings) ([]model.CreativeTaskAsset, error) {
	multiple, _ := definition["multiple"].(bool)
	maxItems := 1
	if multiple {
		maxItems = 5
		if configured, ok := definition["max_items"].(float64); ok && configured > 0 && configured < float64(maxItems) {
			maxItems = int(configured)
		}
	}
	var ids []uint64
	if multiple {
		if err := common.Unmarshal(value, &ids); err != nil {
			return nil, fmt.Errorf("%s: user_file_ids must be an array", key)
		}
	} else {
		var id uint64
		if err := common.Unmarshal(value, &id); err != nil {
			return nil, fmt.Errorf("%s: user_file_id is required", key)
		}
		ids = []uint64{id}
	}
	if len(ids) == 0 || len(ids) > maxItems {
		return nil, fmt.Errorf("%s: accepts between 1 and %d input images", key, maxItems)
	}
	seen := make(map[uint64]struct{}, len(ids))
	assets := make([]model.CreativeTaskAsset, 0, len(ids))
	for _, userFileID := range ids {
		if userFileID == 0 {
			return nil, fmt.Errorf("%s: user_file_id is required", key)
		}
		if _, exists := seen[userFileID]; exists {
			return nil, fmt.Errorf("%s: duplicate input file", key)
		}
		seen[userFileID] = struct{}{}
		var userFile model.UserFile
		if err := model.DB.Where("id = ? AND user_id = ? AND status = ?", userFileID, userID, "1").First(&userFile).Error; err != nil {
			return nil, fmt.Errorf("%s: input file is unavailable", key)
		}
		var file model.File
		if err := model.DB.Where("id = ? AND status = ?", userFile.FileId, "1").First(&file).Error; err != nil {
			return nil, fmt.Errorf("%s: input file is unavailable", key)
		}
		if !CreativeImageMIMEAllowed(settings, file.MimeType) || file.FileSize > settings.MaxRasterBytes {
			return nil, fmt.Errorf("%s: input file type or size is not allowed", key)
		}
		assets = append(assets, model.CreativeTaskAsset{UserFileID: userFile.Id, FileID: file.Id, Role: "input", MimeType: file.MimeType, FileSize: file.FileSize, FileName: userFile.FileName})
	}
	return assets, nil
}

func CreativeImageMIMEAllowed(settings CreativeStudioSettings, mimeType string) bool {
	for _, allowed := range settings.AllowedImageMIMETypes {
		if strings.EqualFold(strings.TrimSpace(allowed), strings.TrimSpace(mimeType)) {
			return true
		}
	}
	return false
}

func forbiddenCreativeParameter(key string) bool {
	switch key {
	case "model", "channel", "channel_id", "price", "status", "system", "system_prompt", "save_channel", "output_url", "url", "stream":
		return true
	}
	return false
}

func validCreativeInputType(fieldType string) bool {
	switch fieldType {
	case "text", "textarea", "number", "boolean", "select", "slider", "ratio", "file":
		return true
	}
	return false
}

func validateCreativeInputValue(fieldType string, value json.RawMessage, definition map[string]any) error {
	if fieldType == "text" || fieldType == "textarea" || fieldType == "select" {
		var text string
		if err := common.Unmarshal(value, &text); err != nil {
			return errors.New("must be a string")
		}
		if definition["required"] == true && strings.TrimSpace(text) == "" {
			return errors.New("is required")
		}
		if fieldType == "select" {
			matched := false
			options, hasOptions := definition["options"].([]any)
			if hasOptions && len(options) > 0 {
				for _, option := range options {
					candidate := option
					if object, ok := option.(map[string]any); ok {
						candidate = object["value"]
					}
					if candidate == text {
						matched = true
						break
					}
				}
			}
			if !matched && definition["allow_custom"] == true {
				left, right, found := strings.Cut(text, ":")
				if found {
					width, leftErr := strconv.ParseFloat(strings.TrimSpace(left), 64)
					height, rightErr := strconv.ParseFloat(strings.TrimSpace(right), 64)
					matched = leftErr == nil && rightErr == nil && width > 0 && height > 0 && width <= 100 && height <= 100
				}
			}
			if (hasOptions && len(options) > 0 || definition["allow_custom"] == true) && !matched {
				return errors.New("is not an allowed option")
			}
		}
		return nil
	}
	if fieldType == "boolean" {
		var boolean bool
		if err := common.Unmarshal(value, &boolean); err != nil {
			return errors.New("must be a boolean")
		}
		return nil
	}
	if fieldType == "number" || fieldType == "slider" || fieldType == "ratio" {
		var number float64
		if err := common.Unmarshal(value, &number); err != nil {
			return errors.New("must be a number")
		}
		if minimum, ok := definition["min"].(float64); ok && number < minimum {
			return errors.New("below minimum")
		}
		if maximum, ok := definition["max"].(float64); ok && number > maximum {
			return errors.New("above maximum")
		}
		if options, ok := definition["options"].([]any); ok && len(options) > 0 {
			matched := false
			for _, option := range options {
				candidate, ok := option.(float64)
				if ok && candidate == number {
					matched = true
					break
				}
			}
			if !matched {
				return errors.New("is not an allowed option")
			}
		}
		return nil
	}
	return nil
}
