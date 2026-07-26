package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Token failover advances to the next group once CountPriorityLevels says the current
// group has nothing left. The memory cache and the database read that count from
// different tables, so a disagreement between them would silently change how many
// attempts a request spends in each group.
func TestCountPriorityLevelsAgreesAcrossMemoryAndDatabasePaths(t *testing.T) {
	truncateTables(t)

	previousMemoryCache := common.MemoryCacheEnabled
	t.Cleanup(func() {
		common.MemoryCacheEnabled = previousMemoryCache
		InitChannelCache()
	})

	seed := []struct {
		id       int
		group    string
		priority int64
		enabled  bool
	}{
		{id: 1, group: "alpha", priority: 10, enabled: true},
		{id: 2, group: "alpha", priority: 10, enabled: true},
		{id: 3, group: "alpha", priority: 5, enabled: true},
		{id: 4, group: "beta", priority: 0, enabled: true},
		{id: 5, group: "alpha", priority: 1, enabled: false},
	}
	for _, entry := range seed {
		status := common.ChannelStatusEnabled
		if !entry.enabled {
			status = common.ChannelStatusManuallyDisabled
		}
		priority := entry.priority
		require.NoError(t, DB.Create(&Channel{
			Id:       entry.id,
			Name:     "channel",
			Key:      "sk-test",
			Status:   status,
			Group:    entry.group,
			Models:   "gpt-4",
			Priority: &priority,
		}).Error)
		require.NoError(t, DB.Create(&Ability{
			Group:     entry.group,
			Model:     "gpt-4",
			ChannelId: entry.id,
			Enabled:   entry.enabled,
			Priority:  &priority,
		}).Error)
	}

	cases := []struct {
		name     string
		group    string
		model    string
		expected int
	}{
		{name: "channels sharing a priority count as one level", group: "alpha", model: "gpt-4", expected: 2},
		{name: "single priority group", group: "beta", model: "gpt-4", expected: 1},
		{name: "unknown model", group: "alpha", model: "gpt-5", expected: 0},
		{name: "unknown group", group: "gamma", model: "gpt-4", expected: 0},
	}

	common.MemoryCacheEnabled = true
	InitChannelCache()
	for _, testCase := range cases {
		t.Run("memory/"+testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, CountPriorityLevels(testCase.group, testCase.model, ""))
		})
	}

	common.MemoryCacheEnabled = false
	for _, testCase := range cases {
		t.Run("database/"+testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, CountPriorityLevels(testCase.group, testCase.model, ""))
		})
	}
}
