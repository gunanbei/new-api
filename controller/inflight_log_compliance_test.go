package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestUpdateOptionBlocksInflightTraceEnableWithoutCompliance(t *testing.T) {
	originalSelfUse := operation_setting.SelfUseModeEnabled
	originalDemo := operation_setting.DemoSiteEnabled
	setting := operation_setting.GetInflightLogSetting()
	originalConfirmed := setting.ComplianceConfirmed
	originalTermsVersion := setting.ComplianceTermsVersion
	t.Cleanup(func() {
		operation_setting.SelfUseModeEnabled = originalSelfUse
		operation_setting.DemoSiteEnabled = originalDemo
		setting.ComplianceConfirmed = originalConfirmed
		setting.ComplianceTermsVersion = originalTermsVersion
	})

	operation_setting.SelfUseModeEnabled = false
	operation_setting.DemoSiteEnabled = false
	setting.ComplianceConfirmed = false
	setting.ComplianceTermsVersion = ""

	recorder := httptest.NewRecorder()
	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(
		http.MethodPut,
		"/api/option/",
		strings.NewReader(`{"key":"InflightTaskTraceEnabled","value":true}`),
	)

	UpdateOption(ctx)

	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.DecodeJson(recorder.Body, &response))
	require.False(t, response.Success)
	require.NotEmpty(t, response.Message)
}
