package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type userGroupsResponse struct {
	Success bool `json:"success"`
	Data    map[string]struct {
		CrossGroupRetryStatusCodes string `json:"cross_group_retry_status_codes"`
	} `json:"data"`
}

func TestGetUserGroupsOnlyExposesCrossGroupRetryRulesToAuthenticatedUsers(t *testing.T) {
	originalGroupRatios := ratio_setting.GroupRatio2JSONString()
	originalUsableGroups := setting.UserUsableGroups2JSONString()
	originalRules := operation_setting.GetRoutingReliabilitySetting().CrossGroupRetryRules
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(originalGroupRatios))
		require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(originalUsableGroups))
		operation_setting.GetRoutingReliabilitySetting().CrossGroupRetryRules = originalRules
	})

	require.NoError(t, ratio_setting.UpdateGroupRatioByJSONString(`{"default":1,"vip":2}`))
	require.NoError(t, setting.UpdateUserUsableGroupsByJSONString(`{"default":"Default","vip":"VIP"}`))
	operation_setting.GetRoutingReliabilitySetting().CrossGroupRetryRules = []operation_setting.CrossGroupRetryRule{
		{Group: "default", HTTPStatusCodes: "429,500-599"},
	}

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.Create(&model.User{
		Id:       1005,
		Username: "group-rule-user",
		Password: "password",
		Group:    "default",
		Status:   common.UserStatusEnabled,
	}).Error)

	authenticatedRecorder := httptest.NewRecorder()
	authenticatedContext, _ := gin.CreateTestContext(authenticatedRecorder)
	authenticatedContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/self/groups", nil)
	authenticatedContext.Set("id", 1005)
	GetUserGroups(authenticatedContext)

	var authenticated userGroupsResponse
	require.Equal(t, http.StatusOK, authenticatedRecorder.Code)
	require.NoError(t, common.Unmarshal(authenticatedRecorder.Body.Bytes(), &authenticated))
	require.True(t, authenticated.Success)
	assert.Equal(t, "429,500-599", authenticated.Data["default"].CrossGroupRetryStatusCodes)
	assert.Empty(t, authenticated.Data["vip"].CrossGroupRetryStatusCodes)

	anonymousRecorder := httptest.NewRecorder()
	anonymousContext, _ := gin.CreateTestContext(anonymousRecorder)
	anonymousContext.Request = httptest.NewRequest(http.MethodGet, "/api/user/groups", nil)
	GetUserGroups(anonymousContext)

	var anonymous userGroupsResponse
	require.Equal(t, http.StatusOK, anonymousRecorder.Code)
	require.NoError(t, common.Unmarshal(anonymousRecorder.Body.Bytes(), &anonymous))
	require.True(t, anonymous.Success)
	assert.Empty(t, anonymous.Data["default"].CrossGroupRetryStatusCodes)
}
