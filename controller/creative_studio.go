package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/service/storage"
	"github.com/gin-gonic/gin"
)

func GetCreativeUserBootstrap(c *gin.Context) {
	data, err := service.CreativeUserBootstrap(int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetCreativeTasks(c *gin.Context) {
	filter := service.CreativeTaskListFilter{Status: c.Query("status"), Model: c.Query("model")}
	if value := c.Query("from"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			common.ApiErrorMsg(c, "invalid from timestamp")
			return
		}
		filter.From = &parsed
	}
	if value := c.Query("to"); value != "" {
		parsed, err := time.Parse(time.RFC3339, value)
		if err != nil {
			common.ApiErrorMsg(c, "invalid to timestamp")
			return
		}
		filter.To = &parsed
	}
	pageInfo := common.GetPageQuery(c)
	page, err := service.ListCreativeTasks(int64(c.GetInt("id")), filter, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(page.Total))
	pageInfo.SetItems(page.Items)
	common.ApiSuccess(c, pageInfo)
}

func CreateCreativeTask(c *gin.Context) {
	var request dto.CreativeTaskCreateRequest
	var references []*multipart.FileHeader
	if strings.Contains(c.GetHeader("Content-Type"), "multipart/form-data") {
		settings, err := service.GetCreativeStudioSettings()
		if err != nil {
			common.ApiError(c, err)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, settings.MaxRasterBytes*5+1<<20)
		if err := c.Request.ParseMultipartForm(32 << 20); err != nil {
			common.ApiErrorMsg(c, "invalid multipart task request")
			return
		}
		rawRequest := c.Request.MultipartForm.Value["request"]
		if len(rawRequest) != 1 || common.UnmarshalJsonStr(rawRequest[0], &request) != nil {
			common.ApiErrorMsg(c, "invalid task request")
			return
		}
		references = c.Request.MultipartForm.File["images"]
		if err := validateCreativeReferenceFiles(references, settings); err != nil {
			common.ApiError(c, err)
			return
		}
	} else if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	task, err := service.PrepareCreativeTaskWithReferences(int64(c.GetInt("id")), request, len(references))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := dispatchCreativeTask(c, task, references); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func validateCreativeReferenceFiles(files []*multipart.FileHeader, settings service.CreativeStudioSettings) error {
	if len(files) == 0 || len(files) > 5 {
		return errors.New("reference images must contain between 1 and 5 files")
	}
	for _, header := range files {
		if header.Size <= 0 || header.Size > settings.MaxRasterBytes {
			return errors.New("reference image size is not allowed")
		}
		file, err := header.Open()
		if err != nil {
			return err
		}
		magic := make([]byte, 32)
		count, readErr := io.ReadFull(file, magic)
		_ = file.Close()
		if readErr != nil && !errors.Is(readErr, io.ErrUnexpectedEOF) {
			return errors.New("reference image is invalid")
		}
		mimeType := http.DetectContentType(magic[:count])
		if !service.CreativeImageMIMEAllowed(settings, mimeType) {
			return errors.New("reference image type is not allowed")
		}
	}
	return nil
}

func GetCreativeTask(c *gin.Context) {
	task, assets, err := service.GetCreativeTask(int64(c.GetInt("id")), c.Param("task_key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	assetViews := make([]gin.H, 0, len(assets))
	for _, asset := range assets {
		view := gin.H{"id": asset.ID, "user_file_id": asset.UserFileID, "role": asset.Role, "position": asset.Position, "mime_type": asset.MimeType, "file_size": asset.FileSize, "file_name": asset.FileName}
		if file, err := storage.Get(asset.UserFileID, &task.UserID); err == nil {
			view["file_url"] = file.FileURL
		}
		assetViews = append(assetViews, view)
	}
	if previews, ok := creativePreviews.Load(task.ID); ok {
		for position, preview := range previews.([]creativePreview) {
			assetViews = append(assetViews, gin.H{"id": 0, "role": "output", "position": position, "mime_type": preview.MimeType, "file_name": "creative-image", "preview_base64": preview.Base64, "temporary": true})
		}
	}
	if task.Status == model.CreativeTaskStatusImporting && task.ResultManifest != "" {
		var manifest struct {
			Items []struct {
				URL      string `json:"url"`
				FileName string `json:"file_name"`
			} `json:"items"`
		}
		if common.UnmarshalJsonStr(task.ResultManifest, &manifest) == nil {
			for position, item := range manifest.Items {
				if item.URL != "" {
					assetViews = append(assetViews, gin.H{"id": 0, "role": "output", "position": position, "file_name": item.FileName, "file_url": item.URL, "temporary": true})
				}
			}
		}
	}
	taskView := gin.H{"task_key": task.TaskKey, "display_name": task.DisplayName, "model_name": task.ModelName, "group_name": task.GroupName, "operation": task.Operation, "status": task.Status, "requested_params": task.RequestedParams, "resolved_params": task.ResolvedParams, "started_at": task.StartedAt, "finished_at": task.FinishedAt, "created_at": task.CreatedAt, "error_code": task.ErrorCode, "error_message": task.ErrorMessage}
	var creativeModel model.CreativeModel
	if err := model.DB.Where("model_name = ?", task.ModelName).Order("id asc").First(&creativeModel).Error; err == nil {
		taskView["source_vendor"] = creativeModel.Vendor
	}
	var binding model.CreativeChannelBinding
	if err := model.DB.Where("id = ?", task.BindingID).First(&binding).Error; err == nil {
		taskView["request_model"] = binding.RequestModel
		var channel model.Channel
		if err := model.DB.Select("name").Where("id = ?", binding.ChannelID).First(&channel).Error; err == nil {
			taskView["source_channel_name"] = channel.Name
		}
	}
	common.ApiSuccess(c, gin.H{"task": taskView, "assets": assetViews})
}

func DeleteCreativeTask(c *gin.Context) {
	userID := int64(c.GetInt("id"))
	if err := service.DeleteCreativeTask(userID, c.Param("task_key")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetCreativeTaskAssetBase64(c *gin.Context) {
	task, _, err := service.GetCreativeTask(int64(c.GetInt("id")), c.Param("task_key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	assetID, err := creativeID(c, "asset_id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var asset model.CreativeTaskAsset
	if err := model.DB.Where("id = ? AND task_id = ? AND role = ?", assetID, task.ID, "output").First(&asset).Error; err != nil {
		common.ApiErrorMsg(c, "creative asset is unavailable")
		return
	}
	reader, file, err := storage.OpenCreativeUserFile(c.Request.Context(), task.UserID, asset.UserFileID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer reader.Close()
	data, err := io.ReadAll(io.LimitReader(reader, file.FileSize+1))
	if err != nil || int64(len(data)) > file.FileSize {
		common.ApiErrorMsg(c, "creative asset is unavailable")
		return
	}
	common.ApiSuccess(c, gin.H{"base64": base64.StdEncoding.EncodeToString(data)})
}

func RetryCreativeTask(c *gin.Context) {
	task, err := service.RetryCreativeTask(int64(c.GetInt("id")), c.Param("task_key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := dispatchCreativeTask(c, task); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func dispatchCreativeTask(c *gin.Context, task *model.CreativeTask, references ...[]*multipart.FileHeader) error {
	var err error
	switch task.Protocol {
	case "openai_image", "advanced_custom_image":
		var files []*multipart.FileHeader
		if len(references) > 0 {
			files = references[0]
		}
		err = executeCreativeImageTask(c, task, files)
	case "openai_responses_image":
		err = executeCreativeResponsesImageTask(c, task)
	case "claude_svg":
		err = executeCreativeSVGTask(c, task)
	case "midjourney_image":
		err = executeCreativeMidjourneyTask(c, task)
	default:
		err = fmt.Errorf("creative protocol execution is not available")
	}
	if err == nil {
		return nil
	}
	service.MarkCreativeTaskFailed(task.ID, "generation_failed", err.Error())
	return err
}

func RetryCreativeImport(c *gin.Context) {
	task, err := service.RetryCreativeImport(int64(c.GetInt("id")), c.Param("task_key"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := storage.ImportCreativeTaskManifest(c.Request.Context(), task); err != nil {
		service.MarkCreativeTaskImportFailed(task.ID, err)
		common.ApiError(c, err)
		return
	}
	if err := model.DB.First(task, task.ID).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, task)
}

func GetCreativeStudioBootstrap(c *gin.Context) {
	data, err := service.CreativeStudioBootstrap()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func GetCreativeStudioSettings(c *gin.Context) {
	settings, err := service.GetCreativeStudioSettings()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func UpdateCreativeStudioSettings(c *gin.Context) {
	var request dto.CreativeStudioSettingsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	settings, err := service.UpdateCreativeStudioSettings(request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, settings)
}

func GetCreativeModels(c *gin.Context) {
	models, err := service.ListCreativeModels()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, models)
}

func CreateCreativeModel(c *gin.Context) {
	var request dto.CreativeModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	creativeModel, err := service.CreateCreativeModel(request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, creativeModel)
}

func UpdateCreativeModel(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativeModelRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	creativeModel, err := service.UpdateCreativeModel(id, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, creativeModel)
}

func DeleteCreativeModel(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeleteCreativeModel(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

func GetCreativeCapabilities(c *gin.Context) {
	modelID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	capabilities, err := service.ListCreativeCapabilities(modelID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capabilities)
}

func CreateCreativeCapability(c *gin.Context) {
	modelID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativeCapabilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	capability, err := service.CreateCreativeCapability(modelID, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capability)
}

func UpdateCreativeCapability(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativeCapabilityRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	capability, err := service.UpdateCreativeCapability(id, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, capability)
}

func DeleteCreativeCapability(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeleteCreativeCapability(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

func GetCreativePublications(c *gin.Context) {
	capabilityID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	publications, err := service.ListCreativePublications(capabilityID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, publications)
}

func CreateCreativePublication(c *gin.Context) {
	capabilityID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativePublicationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	publication, err := service.CreateCreativePublication(capabilityID, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, publication)
}

func UpdateCreativePublication(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativePublicationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	publication, err := service.UpdateCreativePublication(id, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, publication)
}

func DeleteCreativePublication(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeleteCreativePublication(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

func GetCreativeBindings(c *gin.Context) {
	publicationID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	bindings, err := service.ListCreativeBindings(publicationID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, bindings)
}

func CreateCreativeBinding(c *gin.Context) {
	publicationID, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativeBindingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	binding, err := service.CreateCreativeBinding(publicationID, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func UpdateCreativeBinding(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var request dto.CreativeBindingRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body")
		return
	}
	binding, err := service.UpdateCreativeBinding(id, request, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func DeleteCreativeBinding(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := service.DeleteCreativeBinding(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"id": id})
}

func RevalidateCreativeBinding(c *gin.Context) {
	id, err := creativeID(c, "id")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	binding, err := service.RevalidateCreativeBinding(id, int64(c.GetInt("id")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, binding)
}

func creativeID(c *gin.Context, name string) (uint64, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}
