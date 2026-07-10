package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type listModelsResponse struct {
	Success bool               `json:"success"`
	Data    []dto.OpenAIModels `json:"data"`
	Object  string             `json:"object"`
}

type userModelsResponse struct {
	Success bool     `json:"success"`
	Data    []string `json:"data"`
}

type imagePlaygroundBootstrapResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Tokens []struct {
			ID               int     `json:"id"`
			Group            string  `json:"group"`
			GroupDisplayName string  `json:"group_display_name"`
			GroupRatio       float64 `json:"group_ratio"`
			Disabled         bool    `json:"disabled"`
			Models           []struct {
				Name          string `json:"name"`
				AdapterType   string `json:"adapter_type"`
				DisplayVendor string `json:"display_vendor"`
			} `json:"models"`
		} `json:"tokens"`
	} `json:"data"`
}

func setupModelListControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	initModelListColumnNames(t)

	gin.SetMode(gin.TestMode)
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	common.RedisEnabled = false

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db

	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}, &model.Channel{}, &model.Ability{}, &model.Model{}, &model.Vendor{}))

	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func initModelListColumnNames(t *testing.T) {
	t.Helper()

	originalIsMasterNode := common.IsMasterNode
	originalSQLitePath := common.SQLitePath
	originalMainDatabaseType := common.MainDatabaseType()
	originalLogDatabaseType := common.LogDatabaseType()
	originalSQLDSN, hadSQLDSN := os.LookupEnv("SQL_DSN")
	defer func() {
		common.IsMasterNode = originalIsMasterNode
		common.SQLitePath = originalSQLitePath
		common.SetDatabaseTypes(originalMainDatabaseType, originalLogDatabaseType)
		if hadSQLDSN {
			require.NoError(t, os.Setenv("SQL_DSN", originalSQLDSN))
		} else {
			require.NoError(t, os.Unsetenv("SQL_DSN"))
		}
	}()

	common.IsMasterNode = false
	common.SQLitePath = fmt.Sprintf("file:%s_init?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)
	require.NoError(t, os.Setenv("SQL_DSN", "local"))

	require.NoError(t, model.InitDB())
	if model.DB != nil {
		sqlDB, err := model.DB.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	}
}

func withTieredBillingConfig(t *testing.T, modes map[string]string, exprs map[string]string) {
	t.Helper()

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		if strings.HasPrefix(key, "billing_setting.") {
			saved[key] = value
		}
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		model.InvalidatePricingCache()
	})

	modeBytes, err := common.Marshal(modes)
	require.NoError(t, err)
	exprBytes, err := common.Marshal(exprs)
	require.NoError(t, err)

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": string(modeBytes),
		"billing_setting.billing_expr": string(exprBytes),
	}))
	model.InvalidatePricingCache()
}

func withSelfUseModeDisabled(t *testing.T) {
	t.Helper()

	original := operation_setting.SelfUseModeEnabled
	operation_setting.SelfUseModeEnabled = false
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = original
	})
}

func decodeListModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]struct{} {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload listModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	require.Equal(t, "list", payload.Object)

	ids := make(map[string]struct{}, len(payload.Data))
	for _, item := range payload.Data {
		ids[item.Id] = struct{}{}
	}
	return ids
}

func pricingByModelName(pricings []model.Pricing) map[string]model.Pricing {
	byName := make(map[string]model.Pricing, len(pricings))
	for _, pricing := range pricings {
		byName[pricing.ModelName] = pricing
	}
	return byName
}

func decodeUserModelsResponse(t *testing.T, recorder *httptest.ResponseRecorder) []string {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload userModelsResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	return payload.Data
}

func decodeImagePlaygroundBootstrapResponse(t *testing.T, recorder *httptest.ResponseRecorder) imagePlaygroundBootstrapResponse {
	t.Helper()

	require.Equal(t, http.StatusOK, recorder.Code)
	var payload imagePlaygroundBootstrapResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &payload))
	require.True(t, payload.Success)
	return payload
}

func TestGetUserModelsFiltersByRequestedGroup(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1002,
		Username: "playground-model-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-default-only-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-disabled-model", ChannelId: 1, Enabled: false},
	}).Error)

	defaultRecorder := httptest.NewRecorder()
	defaultContext, _ := gin.CreateTestContext(defaultRecorder)
	defaultContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=default", nil)
	defaultContext.Set("id", 1002)

	GetUserModels(defaultContext)

	defaultModels := decodeUserModelsResponse(t, defaultRecorder)
	require.ElementsMatch(t, []string{"zz-default-only-model"}, defaultModels)

	vipRecorder := httptest.NewRecorder()
	vipContext, _ := gin.CreateTestContext(vipRecorder)
	vipContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/models?group=vip", nil)
	vipContext.Set("id", 1002)

	GetUserModels(vipContext)

	require.Empty(t, decodeUserModelsResponse(t, vipRecorder))
}

func TestGetImagePlaygroundBootstrapUsesTokenOwnGroupAndImageModels(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	common.OptionMap = map[string]string{
		"image_playground.model_registry": `{"version":1,"items":[{"model_name":"gpt-image-1","enabled":true,"adapter_type":"openai_images","display_vendor":"OpenAI Images"},{"model_name":"GPT-IMAGE-1","enabled":true,"adapter_type":"custom","display_vendor":"Duplicate"}]}`,
	}
	require.NoError(t, db.Create(&model.User{
		Id:        1003,
		Username:  "bootstrap-user",
		Password:  "password",
		Group:     "default",
		Status:    common.UserStatusEnabled,
		Quota:     12345,
		UsedQuota: 678,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "gpt-image-1", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "gpt-image-1", ChannelId: 2, Enabled: true},
		{Group: "default", Model: "gpt-4o", ChannelId: 1, Enabled: true},
		{Group: "vip", Model: "gpt-image-1", ChannelId: 1, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&[]model.Token{
		{Id: 2001, UserId: 1003, Name: "image-key", Key: "sk-image-key", Status: common.TokenStatusEnabled, Group: "default", ModelLimitsEnabled: true, ModelLimits: "gpt-image-1"},
		{Id: 2002, UserId: 1003, Name: "disabled-key", Key: "sk-disabled", Status: common.TokenStatusEnabled, Group: "vip", ModelLimitsEnabled: true, ModelLimits: "gpt-4o"},
		{Id: 2003, UserId: 1003, Name: "off-key", Key: "sk-off", Status: common.TokenStatusDisabled, Group: "default"},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/image-playground/bootstrap", nil)
	ctx.Set("id", 1003)

	GetImagePlaygroundBootstrap(ctx)

	payload := decodeImagePlaygroundBootstrapResponse(t, recorder)
	require.Len(t, payload.Data.Tokens, 2)

	tokensByID := map[int]struct {
		ID               int     `json:"id"`
		Group            string  `json:"group"`
		GroupDisplayName string  `json:"group_display_name"`
		GroupRatio       float64 `json:"group_ratio"`
		Disabled         bool    `json:"disabled"`
		Models           []struct {
			Name          string `json:"name"`
			AdapterType   string `json:"adapter_type"`
			DisplayVendor string `json:"display_vendor"`
		} `json:"models"`
	}{}
	for _, token := range payload.Data.Tokens {
		tokensByID[token.ID] = token
	}

	imageToken := tokensByID[2001]
	assert.Equal(t, "default", imageToken.Group)
	assert.Equal(t, "默认分组", imageToken.GroupDisplayName)
	assert.Equal(t, 1.0, imageToken.GroupRatio)
	require.Len(t, imageToken.Models, 1)
	assert.Equal(t, "gpt-image-1", imageToken.Models[0].Name)
	assert.Equal(t, "openai_images", imageToken.Models[0].AdapterType)
	assert.Equal(t, "OpenAI Images", imageToken.Models[0].DisplayVendor)

	disabledToken := tokensByID[2002]
	assert.True(t, disabledToken.Disabled)
	assert.Empty(t, disabledToken.Models)
}

func setupImagePlaygroundBootstrapBaseFixture(t *testing.T, db *gorm.DB, token model.Token) {
	t.Helper()

	require.NoError(t, db.Create(&model.User{
		Id:       1004,
		Username: "registry-bootstrap-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "gpt-image-1", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "gpt-4o", ChannelId: 1, Enabled: true},
	}).Error)
	require.NoError(t, db.Create(&token).Error)
}

func requestImagePlaygroundBootstrap(t *testing.T, userID int) imagePlaygroundBootstrapResponse {
	t.Helper()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/image-playground/bootstrap", nil)
	ctx.Set("id", userID)

	GetImagePlaygroundBootstrap(ctx)

	return decodeImagePlaygroundBootstrapResponse(t, recorder)
}

func TestGetImagePlaygroundBootstrapRegistryEdgeCases(t *testing.T) {
	baseToken := model.Token{
		Id: 2101, UserId: 1004, Name: "registry-key", Key: "sk-registry-key",
		Status: common.TokenStatusEnabled, Group: "default",
		ModelLimitsEnabled: false,
	}

	tests := []struct {
		name          string
		registry      string
		token         model.Token
		wantDisabled  bool
		wantModelLen  int
		wantModelName string
	}{
		{
			name:         "empty registry returns no models",
			registry:     "",
			token:        baseToken,
			wantDisabled: true,
			wantModelLen: 0,
		},
		{
			name:         "invalid registry json returns no models",
			registry:     `{not-json`,
			token:        baseToken,
			wantDisabled: true,
			wantModelLen: 0,
		},
		{
			name:     "disabled registry item is filtered",
			registry: `{"version":1,"items":[{"model_name":"gpt-image-1","enabled":false,"adapter_type":"openai_images","display_vendor":"OpenAI Images"}]}`,
			token:    baseToken,
			wantDisabled: true,
			wantModelLen: 0,
		},
		{
			name:     "channel model not in registry is filtered",
			registry: `{"version":1,"items":[{"model_name":"gpt-image-1","enabled":true,"adapter_type":"openai_images","display_vendor":"OpenAI Images"}]}`,
			token:    baseToken,
			wantDisabled:  false,
			wantModelLen:  1,
			wantModelName: "gpt-image-1",
		},
		{
			name:     "model limits match case insensitively",
			registry: `{"version":1,"items":[{"model_name":"gpt-image-1","enabled":true,"adapter_type":"openai_images","display_vendor":"OpenAI Images"}]}`,
			token: model.Token{
				Id: 2101, UserId: 1004, Name: "registry-key", Key: "sk-registry-key",
				Status: common.TokenStatusEnabled, Group: "default",
				ModelLimitsEnabled: true, ModelLimits: "GPT-IMAGE-1",
			},
			wantDisabled:  false,
			wantModelLen:  1,
			wantModelName: "gpt-image-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupModelListControllerTestDB(t)
			setupImagePlaygroundBootstrapBaseFixture(t, db, tt.token)
			common.OptionMap = map[string]string{}
			if tt.registry != "" {
				common.OptionMap["image_playground.model_registry"] = tt.registry
			}

			payload := requestImagePlaygroundBootstrap(t, 1004)
			require.Len(t, payload.Data.Tokens, 1)

			token := payload.Data.Tokens[0]
			assert.Equal(t, tt.wantDisabled, token.Disabled)
			assert.Len(t, token.Models, tt.wantModelLen)
			if tt.wantModelLen > 0 {
				assert.Equal(t, tt.wantModelName, token.Models[0].Name)
			}
		})
	}
}

func TestListModelsIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-tiered-visible-model":      "tiered_expr",
		"zz-tiered-empty-expr-model":   "tiered_expr",
		"zz-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-tiered-empty-expr-model": "   ",
	})

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1001,
		Username: "model-list-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)
	require.NoError(t, db.Create(&[]model.Ability{
		{Group: "default", Model: "zz-tiered-visible-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-empty-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-tiered-missing-expr-model", ChannelId: 1, Enabled: true},
		{Group: "default", Model: "zz-unpriced-model", ChannelId: 1, Enabled: true},
	}).Error)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	ctx.Set("id", 1001)

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-tiered-visible-model")
	require.NotContains(t, ids, "zz-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-unpriced-model")

	pricingByName := pricingByModelName(model.GetPricing())
	visiblePricing, ok := pricingByName["zz-tiered-visible-model"]
	require.True(t, ok)
	require.Equal(t, "tiered_expr", visiblePricing.BillingMode)
	require.NotEmpty(t, visiblePricing.BillingExpr)

	emptyExprPricing, ok := pricingByName["zz-tiered-empty-expr-model"]
	require.True(t, ok)
	require.Empty(t, emptyExprPricing.BillingMode)
	require.Empty(t, emptyExprPricing.BillingExpr)

	missingExprPricing, ok := pricingByName["zz-tiered-missing-expr-model"]
	require.True(t, ok)
	require.Empty(t, missingExprPricing.BillingMode)
	require.Empty(t, missingExprPricing.BillingExpr)
}

func TestListModelsTokenLimitIncludesTieredBillingModel(t *testing.T) {
	withSelfUseModeDisabled(t)
	withTieredBillingConfig(t, map[string]string{
		"zz-token-tiered-visible-model":      "tiered_expr",
		"zz-token-tiered-empty-expr-model":   "tiered_expr",
		"zz-token-tiered-missing-expr-model": "tiered_expr",
	}, map[string]string{
		"zz-token-tiered-visible-model":    `tier("base", p * 1 + c * 2)`,
		"zz-token-tiered-empty-expr-model": "",
	})
	setupModelListControllerTestDB(t)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	common.SetContextKey(ctx, constant.ContextKeyUserGroup, "default")
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimitEnabled, true)
	common.SetContextKey(ctx, constant.ContextKeyTokenModelLimit, map[string]bool{
		"zz-token-tiered-visible-model":      true,
		"zz-token-tiered-empty-expr-model":   true,
		"zz-token-tiered-missing-expr-model": true,
		"zz-token-unpriced-model":            true,
	})

	ListModels(ctx, constant.ChannelTypeOpenAI)

	ids := decodeListModelsResponse(t, recorder)
	require.Contains(t, ids, "zz-token-tiered-visible-model")
	require.NotContains(t, ids, "zz-token-tiered-empty-expr-model")
	require.NotContains(t, ids, "zz-token-tiered-missing-expr-model")
	require.NotContains(t, ids, "zz-token-unpriced-model")
}

func TestCheckUpdatePasswordRequiresCurrentPassword(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	hashedPassword, err := common.Password2Hash("CurrentPassword123")
	require.NoError(t, err)
	user := &model.User{
		Username: "password-user",
		Password: hashedPassword,
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	updatePassword, err := checkUpdatePassword("", "", user.Id)
	require.NoError(t, err)
	assert.False(t, updatePassword)

	updatePassword, err = checkUpdatePassword("", "NewPassword123", user.Id)
	require.Error(t, err)
	assert.False(t, updatePassword)
	assert.ErrorIs(t, err, errOriginalPasswordFail)

	updatePassword, err = checkUpdatePassword("CurrentPassword123", "NewPassword123", user.Id)
	require.NoError(t, err)
	assert.True(t, updatePassword)
}

func TestCheckUpdatePasswordRejectsHistoricalEmptyPassword(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	user := &model.User{
		Username: "legacy-passwordless-user",
		Password: "",
		Status:   common.UserStatusEnabled,
	}
	require.NoError(t, db.Create(user).Error)

	updatePassword, err := checkUpdatePassword("", "NewPassword123", user.Id)
	require.Error(t, err)
	assert.False(t, updatePassword)
	assert.ErrorIs(t, err, errUserPasswordUnset)
}

func TestSetupLoginDoesNotTouchPasswordWhenPasswordFieldOmitted(t *testing.T) {
	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.Log{}))

	hashedPassword, err := common.Password2Hash("CurrentPassword123")
	require.NoError(t, err)
	user := &model.User{
		Username: "twofa-user",
		Password: hashedPassword,
		Role:     common.RoleCommonUser,
		Status:   common.UserStatusEnabled,
		Group:    "default",
	}
	require.NoError(t, db.Create(user).Error)

	router := gin.New()
	store := cookie.NewStore([]byte("test-session-secret"))
	router.Use(sessions.Sessions("session", store))
	router.GET("/", func(c *gin.Context) {
		setupLogin(&model.User{
			Id:       user.Id,
			Username: user.Username,
			Role:     user.Role,
			Status:   user.Status,
			Group:    user.Group,
		}, c)
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	var stored model.User
	require.NoError(t, db.First(&stored, user.Id).Error)
	assert.Equal(t, hashedPassword, stored.Password)
}
