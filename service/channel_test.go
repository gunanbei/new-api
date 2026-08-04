package service

import (
	"errors"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldDisableChannelHonorsGroupAutoDisableSwitch(t *testing.T) {
	originalEnabled := common.AutomaticDisableChannelEnabled
	originalKeywords := append([]string(nil), operation_setting.AutomaticDisableKeywords...)
	settings := operation_setting.GetRoutingReliabilitySetting()
	originalRules := append([]operation_setting.GroupAutoDisableRule(nil), settings.GroupAutoDisableRules...)
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = originalEnabled
		operation_setting.AutomaticDisableKeywords = originalKeywords
		settings.GroupAutoDisableRules = originalRules
	})

	common.AutomaticDisableChannelEnabled = true
	operation_setting.AutomaticDisableKeywords = []string{"account disabled"}
	settings.GroupAutoDisableRules = []operation_setting.GroupAutoDisableRule{
		{Group: "paused", Enabled: false, DisableThresholdSeconds: 5, HTTPStatusCodes: "401"},
		{Group: "active", Enabled: true, DisableThresholdSeconds: 5, HTTPStatusCodes: "401"},
	}

	statusError := types.NewOpenAIError(errors.New("unauthorized"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized)
	keywordError := types.NewOpenAIError(errors.New("account disabled"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
	channelError := types.NewError(errors.New("invalid channel key"), types.ErrorCodeChannelInvalidKey)

	assert.False(t, ShouldDisableChannel(statusError, "paused"))
	assert.False(t, ShouldDisableChannel(keywordError, "paused"))
	assert.False(t, ShouldDisableChannel(channelError, "paused"))
	assert.True(t, ShouldDisableChannel(statusError, "active"))

	common.AutomaticDisableChannelEnabled = false
	assert.False(t, ShouldDisableChannel(statusError, "active"))
}

func TestShouldDisableChannelUsesGroupKeywordsWithoutStatusCodes(t *testing.T) {
	originalEnabled := common.AutomaticDisableChannelEnabled
	settings := operation_setting.GetRoutingReliabilitySetting()
	originalRules := append([]operation_setting.GroupAutoDisableRule(nil), settings.GroupAutoDisableRules...)
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = originalEnabled
		settings.GroupAutoDisableRules = originalRules
	})

	common.AutomaticDisableChannelEnabled = true
	settings.GroupAutoDisableRules = []operation_setting.GroupAutoDisableRule{
		{Group: "vip", Enabled: true, FailureKeywords: "provider account blocked"},
	}

	groupKeywordError := types.NewOpenAIError(errors.New("provider account blocked"), types.ErrorCodeBadResponseStatusCode, http.StatusBadRequest)
	statusOnlyError := types.NewOpenAIError(errors.New("unauthorized"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized)

	assert.True(t, ShouldDisableChannel(groupKeywordError, "vip"))
	assert.False(t, ShouldDisableChannel(statusOnlyError, "vip"))
}

func TestShouldDisableChannelUsesGroupStatusCodes(t *testing.T) {
	originalEnabled := common.AutomaticDisableChannelEnabled
	settings := operation_setting.GetRoutingReliabilitySetting()
	originalRules := append([]operation_setting.GroupAutoDisableRule(nil), settings.GroupAutoDisableRules...)
	t.Cleanup(func() {
		common.AutomaticDisableChannelEnabled = originalEnabled
		settings.GroupAutoDisableRules = originalRules
	})

	common.AutomaticDisableChannelEnabled = true
	settings.GroupAutoDisableRules = []operation_setting.GroupAutoDisableRule{
		{Group: "default", Enabled: true, DisableThresholdSeconds: 5, HTTPStatusCodes: "429"},
	}

	tooManyRequests := types.NewOpenAIError(errors.New("rate limited"), types.ErrorCodeBadResponseStatusCode, http.StatusTooManyRequests)
	unauthorized := types.NewOpenAIError(errors.New("unauthorized"), types.ErrorCodeBadResponseStatusCode, http.StatusUnauthorized)

	require.True(t, ShouldDisableChannel(tooManyRequests, "default"))
	assert.False(t, ShouldDisableChannel(unauthorized, "default"))
}
