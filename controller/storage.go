package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/service/storage"
	"github.com/gin-gonic/gin"
)

type storageIDsRequest struct {
	IDs []uint64 `json:"ids"`
}

func StoragePrepare(c *gin.Context) {
	var request storage.PrepareInput
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	result, err := storage.Prepare(c.Request.Context(), int64(c.GetInt("id")), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func StoragePresignPart(c *gin.Context) {
	var request storage.PresignPartInput
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	result, err := storage.PresignPart(c.Request.Context(), int64(c.GetInt("id")), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func StorageComplete(c *gin.Context) {
	var request storage.CompleteInput
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	result, err := storage.Complete(c.Request.Context(), int64(c.GetInt("id")), request)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"file": result})
}

func StorageAbort(c *gin.Context) {
	var request storage.AbortInput
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	if err := storage.Abort(c.Request.Context(), int64(c.GetInt("id")), request); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"aborted": true})
}

func GetStorageSelf(c *gin.Context) { listStorage(c, int64Pointer(int64(c.GetInt("id"))), false) }

func GetStorageSelfSuffixes(c *gin.Context) { suffixesStorage(c, int64Pointer(int64(c.GetInt("id")))) }

func GetStorageSelfFile(c *gin.Context) {
	id, err := storageID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userID := int64(c.GetInt("id"))
	file, err := storage.Get(id, &userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"file": file})
}

func DeleteStorageSelfFile(c *gin.Context) {
	id, err := storageID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	userID := int64(c.GetInt("id"))
	file, err := storage.Delete(c.Request.Context(), id, &userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true, "id": file.ID})
}

func DeleteStorageSelfFiles(c *gin.Context) {
	deleteStorageFiles(c, int64Pointer(int64(c.GetInt("id"))))
}

func GetStorageFiles(c *gin.Context) { listStorage(c, nil, true) }

func GetStorageUserFiles(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("user_id"), 10, 64)
	if err != nil || userID <= 0 {
		common.ApiErrorMsg(c, "invalid user id")
		return
	}
	listStorage(c, &userID, true)
}

func GetStorageSuffixes(c *gin.Context) { suffixesStorage(c, nil) }

func GetStorageFile(c *gin.Context) {
	id, err := storageID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	file, err := storage.Get(id, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"file": file})
}

func DeleteStorageFile(c *gin.Context) {
	id, err := storageID(c)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	file, err := storage.Delete(c.Request.Context(), id, nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"deleted": true, "id": file.ID, "user_id": file.UserID})
}

func DeleteStorageFiles(c *gin.Context) { deleteStorageFiles(c, nil) }

func GetStorageChannelStats(c *gin.Context) {
	channelType := strings.TrimSpace(c.Query("channel_type"))
	if channelType != "" && !isStorageChannelType(channelType, true) {
		common.ApiErrorMsg(c, "Invalid channel_type")
		return
	}
	result, err := storage.StatsByChannel(channelType, strings.TrimSpace(c.Query("status")))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func GetStorageSummary(c *gin.Context) {
	result, err := storage.GetSummary()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func listStorage(c *gin.Context, fixedUserID *int64, admin bool) {
	filter, err := storageListFilter(c, fixedUserID, admin)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	items, total, err := storage.List(filter, pageInfo.GetStartIdx(), pageInfo.GetPageSize())
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func suffixesStorage(c *gin.Context, fixedUserID *int64) {
	filter, err := storageListFilter(c, fixedUserID, fixedUserID == nil)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items, err := storage.Suffixes(filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"items": items})
}

func deleteStorageFiles(c *gin.Context, userID *int64) {
	var request storageIDsRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		common.ApiErrorMsg(c, "invalid request body: "+err.Error())
		return
	}
	result, err := storage.DeleteMany(c.Request.Context(), request.IDs, userID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, result)
}

func storageListFilter(c *gin.Context, fixedUserID *int64, admin bool) (storage.ListFilter, error) {
	filter := storage.ListFilter{UserID: fixedUserID, Status: strings.TrimSpace(c.Query("status")), Source: strings.TrimSpace(c.Query("source")), Keyword: strings.TrimSpace(c.Query("keyword")), ChannelType: strings.TrimSpace(c.Query("channel_type")), Identifier: strings.ToLower(strings.TrimSpace(c.Query("identifier"))), Username: strings.TrimSpace(c.Query("username")), Order: strings.TrimSpace(c.Query("order"))}
	if filter.ChannelType != "" && !isStorageChannelType(filter.ChannelType, true) {
		return filter, errors.New("Invalid channel_type")
	}
	for _, suffix := range strings.Split(c.Query("file_suffix"), ",") {
		if suffix = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(suffix), ".")); suffix != "" {
			filter.FileSuffixes = append(filter.FileSuffixes, suffix)
		}
	}
	if channelID := strings.TrimSpace(c.Query("file_channel_id")); channelID != "" {
		value, err := strconv.ParseUint(channelID, 10, 64)
		if err != nil {
			return filter, errors.New("invalid file_channel_id")
		}
		filter.FileChannelID = value
	}
	if admin && fixedUserID == nil && strings.TrimSpace(c.Query("user_id")) != "" {
		userID, err := strconv.ParseInt(c.Query("user_id"), 10, 64)
		if err != nil || userID <= 0 {
			return filter, errors.New("invalid user_id")
		}
		filter.UserID = &userID
	}
	return filter, nil
}

func storageID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, errors.New("File not found")
	}
	return id, nil
}
func int64Pointer(value int64) *int64 { return &value }
func isStorageChannelType(value string, includeLocal bool) bool {
	return (includeLocal && value == "0") || value == "1" || value == "2" || value == "3"
}
