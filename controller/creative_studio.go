package controller

import (
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

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

func creativeID(c *gin.Context, name string) (uint64, error) {
	id, err := strconv.ParseUint(c.Param(name), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid id")
	}
	return id, nil
}
