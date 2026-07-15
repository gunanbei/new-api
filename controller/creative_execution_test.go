package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestSetupCreativeRelayContextUsesPlaygroundBilling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/images/generations", nil)
	user := &model.User{Id: 7, Group: "default", Quota: 1000}
	task := &model.CreativeTask{UserID: 7, GroupName: "creative", ModelName: "gpt-image"}

	require.NoError(t, setupCreativeRelayContext(ctx, user, task))
	info, err := relaycommon.GenRelayInfo(ctx, types.RelayFormatOpenAIImage, nil, nil)
	require.NoError(t, err)
	assert.True(t, info.IsPlayground)
	assert.Equal(t, "creative", info.TokenGroup)
	assert.Equal(t, "/v1/images/generations", info.RequestURLPath)
}

func TestCreativeRasterMIME(t *testing.T) {
	assert.True(t, isCreativeRasterMIME("image/png"))
	assert.True(t, isCreativeRasterMIME("image/webp"))
	assert.False(t, isCreativeRasterMIME("application/octet-stream"))
}

func TestMidjourneyAspectRatio(t *testing.T) {
	assert.Equal(t, "a cat --ar 16:9", withMidjourneyAspectRatio("a cat", "16:9"))
	assert.Equal(t, "a cat --ar 3:2", withMidjourneyAspectRatio("a cat --ar 3:2", "16:9"))
	assert.Equal(t, "a cat --ar=2:3", withMidjourneyAspectRatio("a cat --ar=2:3", "16:9"))
}

func TestCreativeRelayError(t *testing.T) {
	assert.EqualError(t, creativeRelayError(http.StatusBadRequest, []byte(`{"error":{"message":"model is unavailable"}}`)), "model is unavailable")
	assert.EqualError(t, creativeRelayError(http.StatusBadGateway, []byte("invalid response")), "creative relay failed (HTTP 502)")
}

func TestDispatchCreativeTaskKeepsFailureReason(t *testing.T) {
	database, err := gorm.Open(sqlite.Open("file:creative-dispatch-error?mode=memory&cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	previousDB := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previousDB })
	require.NoError(t, database.AutoMigrate(&model.CreativeTask{}))
	task := model.CreativeTask{TaskKey: "ct_dispatch_error", UserID: 7, Category: "image", Status: model.CreativeTaskStatusDispatching, RequestedParams: "{}", ResolvedParams: "{}"}
	require.NoError(t, database.Create(&task).Error)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	err = dispatchCreativeTask(ctx, &task)
	require.EqualError(t, err, "creative protocol execution is not available")
	var stored model.CreativeTask
	require.NoError(t, database.First(&stored, task.ID).Error)
	assert.Equal(t, err.Error(), stored.ErrorMessage)
}
