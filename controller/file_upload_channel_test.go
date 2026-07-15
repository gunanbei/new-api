package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestCopyFileUploadChannelCreatesIndependentNonDefaultClone(t *testing.T) {
	gin.SetMode(gin.TestMode)
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
	})

	database, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))), &gorm.Config{})
	require.NoError(t, err)
	model.DB = database
	model.LOG_DB = database
	require.NoError(t, database.AutoMigrate(&model.FileUploadChannel{}))

	original := model.FileUploadChannel{
		Name:           "primary",
		Type:           service.FileUploadChannelTypeCloudflareImageBed,
		Status:         service.FileUploadChannelStatusEnabled,
		IsDefault:      service.FileUploadChannelIsDefault,
		ChunkThreshold: 16 * 1024 * 1024,
		ChunkSize:      8 * 1024 * 1024,
		MaxSize:        64 * 1024 * 1024,
		ConfigProflle:  `{"api_token":"secret","base_url":"https://example.com"}`,
		CreateUserId:   1,
		UpdateUserId:   1,
	}
	require.NoError(t, database.Create(&original).Error)

	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(http.MethodPost, "/api/file-upload-channel/1/copy", strings.NewReader(`{"suffix":"_clone"}`))
	context.Params = gin.Params{{Key: "id", Value: fmt.Sprint(original.Id)}}
	context.Set("id", 42)

	CopyFileUploadChannel(context)

	require.Equal(t, http.StatusOK, recorder.Code)
	var response struct {
		Success bool `json:"success"`
	}
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	require.True(t, response.Success)

	var clone model.FileUploadChannel
	require.NoError(t, database.Where("name = ?", "primary_clone").First(&clone).Error)
	assert.NotEqual(t, original.Id, clone.Id)
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, original.Status, clone.Status)
	assert.Equal(t, service.FileUploadChannelNotDefault, clone.IsDefault)
	assert.Equal(t, original.ConfigProflle, clone.ConfigProflle)
	assert.Equal(t, int64(42), clone.CreateUserId)
	assert.Equal(t, int64(42), clone.UpdateUserId)
}
