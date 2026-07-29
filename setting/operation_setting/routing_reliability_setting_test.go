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
