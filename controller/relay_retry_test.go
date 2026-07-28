package controller

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestShouldRetryKeepsSystemHTTPStatusRulesWithFailoverKey(t *testing.T) {
	originalRanges := operation_setting.AutomaticRetryStatusCodeRanges
	operation_setting.AutomaticRetryStatusCodeRanges = []operation_setting.StatusCodeRange{
		{Start: http.StatusInternalServerError, End: http.StatusInternalServerError},
	}
	t.Cleanup(func() {
		operation_setting.AutomaticRetryStatusCodeRanges = originalRanges
	})

	testCases := []struct {
		name       string
		statusCode int
		errorCode  types.ErrorCode
		rules      string
		retry      bool
		trigger    string
	}{
		{
			name:       "system retry rule applies when key does not include the status code",
			statusCode: http.StatusInternalServerError,
			errorCode:  types.ErrorCodeBadResponse,
			rules:      "400",
			retry:      true,
			trigger:    "system_http_status_rule",
		},
		{
			name:       "key rule adds a status code not selected by the system",
			statusCode: http.StatusBadRequest,
			errorCode:  types.ErrorCodeBadResponse,
			rules:      "400",
			retry:      true,
			trigger:    "http_status_rule",
		},
		{
			name:       "key rule cannot override an always skipped status code",
			statusCode: http.StatusGatewayTimeout,
			errorCode:  types.ErrorCodeBadResponse,
			rules:      "504",
			retry:      false,
			trigger:    "system_skip_rule",
		},
		{
			name:       "key rule cannot override an always skipped error code",
			statusCode: http.StatusInternalServerError,
			errorCode:  types.ErrorCodeBadResponseBody,
			rules:      "500",
			retry:      false,
			trigger:    "system_skip_rule",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			ctx.Set(constant.ContextKeyTokenFailoverRules, types.TokenFailoverRules{
				Enabled:         true,
				HTTPStatusCodes: testCase.rules,
			})

			decision := shouldRetry(ctx, types.NewErrorWithStatusCode(
				errors.New("upstream error"), testCase.errorCode, testCase.statusCode,
			), 1)

			require.Equal(t, testCase.retry, decision.Retry)
			require.Equal(t, testCase.trigger, decision.Trigger)
		})
	}
}
