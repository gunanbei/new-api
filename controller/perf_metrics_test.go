package controller

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestActivePerfMetricGroupsIncludesAutoOnce(t *testing.T) {
	groups := activePerfMetricGroups(map[string]float64{
		"auto":     1,
		"standard": 1,
		"vip":      1,
	})

	require.Equal(t, []string{"auto", "standard", "vip"}, groups)
}
