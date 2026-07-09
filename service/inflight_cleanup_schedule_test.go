package service

import (
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInflightTaskCleanupIntervalMinutes(t *testing.T) {
	require.NoError(t, ValidateInflightTaskCleanupIntervalMinutes("1"))
	require.NoError(t, ValidateInflightTaskCleanupIntervalMinutes("10080"))
	require.Error(t, ValidateInflightTaskCleanupIntervalMinutes("0"))
	require.Error(t, ValidateInflightTaskCleanupIntervalMinutes("10081"))
}

func TestValidateInflightTaskCleanupCron(t *testing.T) {
	require.NoError(t, ValidateInflightTaskCleanupCron("0 3 * * *"))
	require.Error(t, ValidateInflightTaskCleanupCron("not a cron"))
}

func TestIsInflightTaskCleanupCronDue(t *testing.T) {
	now := time.Date(2026, 7, 10, 4, 0, 0, 0, time.Local).Unix()
	lastRun := time.Date(2026, 7, 9, 3, 0, 0, 0, time.Local).Unix()
	assert.True(t, IsInflightTaskCleanupCronDue("0 3 * * *", lastRun, now))

	sameDayLastRun := time.Date(2026, 7, 10, 3, 0, 0, 0, time.Local).Unix()
	assert.False(t, IsInflightTaskCleanupCronDue("0 3 * * *", sameDayLastRun, now))
}

func TestPreviewInflightTaskCleanupCron(t *testing.T) {
	preview, err := PreviewInflightTaskCleanupCron("0 3 * * *", 3)
	require.NoError(t, err)
	require.NotNil(t, preview)
	assert.NotEmpty(t, preview.Timezone)
	require.Len(t, preview.NextRuns, 3)
	for i := 1; i < len(preview.NextRuns); i++ {
		assert.Greater(t, preview.NextRuns[i], preview.NextRuns[i-1])
	}
}

func TestNextInflightTaskCleanupRunAtInterval(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalMode := common.OptionMap[inflightTaskCleanupScheduleModeOptionKey]
	originalInterval := common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey]
	common.OptionMap[inflightTaskCleanupScheduleModeOptionKey] = InflightTaskCleanupScheduleModeInterval
	common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey] = "60"
	common.OptionMapRWMutex.Unlock()
	defer func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap[inflightTaskCleanupScheduleModeOptionKey] = originalMode
			common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey] = originalInterval
		}
		common.OptionMapRWMutex.Unlock()
	}()

	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local).Unix()
	lastRun := time.Date(2026, 7, 10, 11, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, time.Date(2026, 7, 10, 12, 0, 0, 0, time.Local).Unix(), NextInflightTaskCleanupRunAt(lastRun, now))

	overdueLastRun := time.Date(2026, 7, 10, 10, 0, 0, 0, time.Local).Unix()
	assert.Equal(t, now, NextInflightTaskCleanupRunAt(overdueLastRun, now))
}
