package operation_setting

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCrossGroupRetryRulesNormalizesAndRejectsDuplicates(t *testing.T) {
	rules, err := ParseCrossGroupRetryRules(`[
		{"group":" default ","http_status_codes":" 500,501-503 "},
		{"group":"vip","http_status_codes":"429"}
	]`)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	assert.Equal(t, "default", rules[0].Group)
	assert.Equal(t, "500-503", rules[0].HTTPStatusCodes)
	assert.Equal(t, "429", rules[1].HTTPStatusCodes)

	_, err = ParseCrossGroupRetryRules(`[
		{"group":"default","http_status_codes":"429"},
		{"group":"default","http_status_codes":"500"}
	]`)
	assert.Error(t, err)

	rules, err = ParseCrossGroupRetryRules(`null`)
	require.NoError(t, err)
	assert.Empty(t, rules)
}

func TestShouldRetryAcrossGroupMatchesOnlyConfiguredGroup(t *testing.T) {
	original := routingReliabilitySetting.CrossGroupRetryRules
	routingReliabilitySetting.CrossGroupRetryRules = []CrossGroupRetryRule{
		{Group: "default", HTTPStatusCodes: "429,500-599"},
	}
	t.Cleanup(func() { routingReliabilitySetting.CrossGroupRetryRules = original })

	assert.True(t, ShouldRetryAcrossGroup("default", http.StatusTooManyRequests))
	assert.True(t, ShouldRetryAcrossGroup("default", http.StatusBadGateway))
	assert.False(t, ShouldRetryAcrossGroup("vip", http.StatusBadGateway))
	assert.False(t, ShouldRetryAcrossGroup("default", http.StatusBadRequest))
	assert.False(t, ShouldRetryAcrossGroup("default", http.StatusGatewayTimeout))
}

func TestGroupAutoDisableRulesOverrideStatusCodesAndSwitches(t *testing.T) {
	rules, err := ParseGroupAutoDisableRules(`[
		{"group":" default ","enabled":true,"disable_threshold_seconds":3,"http_status_codes":"401,500-501"},
		{"group":"vip","enabled":false,"disable_threshold_seconds":7,"http_status_codes":"","failure_keywords":" Account blocked "}
	]`)
	require.NoError(t, err)
	require.Len(t, rules, 2)
	assert.Equal(t, "default", rules[0].Group)
	assert.Equal(t, "401,500-501", rules[0].HTTPStatusCodes)
	assert.Empty(t, rules[1].HTTPStatusCodes)
	assert.Equal(t, "account blocked", rules[1].FailureKeywords)

	original := routingReliabilitySetting.GroupAutoDisableRules
	routingReliabilitySetting.GroupAutoDisableRules = rules
	t.Cleanup(func() { routingReliabilitySetting.GroupAutoDisableRules = original })

	assert.True(t, ShouldDisableInGroup("default", http.StatusUnauthorized, false, false))
	assert.False(t, ShouldDisableInGroup("default", http.StatusTooManyRequests, false, false))
	assert.False(t, ShouldDisableInGroup("vip", http.StatusTooManyRequests, false, false))
	assert.False(t, ShouldDisableInGroup("vip", http.StatusInternalServerError, true, true))
	assert.True(t, ShouldDisableInGroup("missing", http.StatusUnauthorized, false, false))
}

func TestEffectiveAutoDisableThresholdUsesEnabledGroupRules(t *testing.T) {
	original := routingReliabilitySetting.GroupAutoDisableRules
	routingReliabilitySetting.GroupAutoDisableRules = []GroupAutoDisableRule{
		{Group: "default", Enabled: true, DisableThresholdSeconds: 3, HTTPStatusCodes: "401"},
		{Group: "vip", Enabled: false, DisableThresholdSeconds: 1, HTTPStatusCodes: "429"},
	}
	t.Cleanup(func() { routingReliabilitySetting.GroupAutoDisableRules = original })

	assert.Equal(t, 3.0, EffectiveAutoDisableThreshold([]string{"default", "vip"}, 5))
	assert.Equal(t, 5.0, EffectiveAutoDisableThreshold([]string{"missing"}, 5))
	assert.Equal(t, 0.0, EffectiveAutoDisableThreshold([]string{"vip"}, 5))
}
