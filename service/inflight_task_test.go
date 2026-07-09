package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInflightTaskKindFromRelayMode(t *testing.T) {
	assert.Equal(t, "chat", inflightTaskKindFromRelayMode(relayconstant.RelayModeChatCompletions))
	assert.Equal(t, "image", inflightTaskKindFromRelayMode(relayconstant.RelayModeImagesGenerations))
	assert.Equal(t, "audio", inflightTaskKindFromRelayMode(relayconstant.RelayModeAudioSpeech))
}

func TestInflightTaskRetentionTTL(t *testing.T) {
	assert.Equal(t, time.Duration(0), inflightTaskRetentionTTL(InflightTaskStatusCompleted))
	assert.Equal(t, time.Duration(0), inflightTaskRetentionTTL(InflightTaskStatusFailed))
	assert.NotZero(t, inflightTaskRetentionTTL(InflightTaskStatusAccepted))
}
func TestInflightTaskCleanupSettings(t *testing.T) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalRule := common.OptionMap[inflightTaskCleanupRuleOptionKey]
	originalInterval := common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey]
	common.OptionMap[inflightTaskCleanupRuleOptionKey] = "1"
	common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey] = "15"
	common.OptionMapRWMutex.Unlock()
	defer func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap[inflightTaskCleanupRuleOptionKey] = originalRule
			common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey] = originalInterval
		}
		common.OptionMapRWMutex.Unlock()
	}()

	assert.Equal(t, InflightTaskCleanupRuleDisabled, InflightTaskCleanupRule())
	assert.Equal(t, 15*time.Minute, InflightTaskCleanupInterval())
}

func TestShouldOverwriteInflightTaskStatus(t *testing.T) {
	assert.True(t, shouldOverwriteInflightTaskStatus("", InflightTaskStatusAccepted))
	assert.True(t, shouldOverwriteInflightTaskStatus(InflightTaskStatusAccepted, InflightTaskStatusStreaming))
	assert.True(t, shouldOverwriteInflightTaskStatus(InflightTaskStatusStreaming, InflightTaskStatusCompleted))
	assert.True(t, shouldOverwriteInflightTaskStatus(InflightTaskStatusFailed, InflightTaskStatusCompleted))
	assert.False(t, shouldOverwriteInflightTaskStatus(InflightTaskStatusCompleted, InflightTaskStatusStreaming))
	assert.False(t, shouldOverwriteInflightTaskStatus(InflightTaskStatusFailed, InflightTaskStatusRouting))
}

func TestShouldReopenInflightTaskForLaterRetry(t *testing.T) {
	stored := &InflightTask{
		Status: InflightTaskStatusFailed,
		Detail: &InflightTaskDetail{RetryIndex: 0},
	}
	next := &InflightTask{
		Status: InflightTaskStatusRouting,
		Detail: &InflightTaskDetail{RetryIndex: 1},
	}

	assert.True(t, shouldReopenInflightTask(stored, next))
	assert.False(t, shouldReopenInflightTask(stored, &InflightTask{
		Status: InflightTaskStatusRouting,
		Detail: &InflightTaskDetail{RetryIndex: 0},
	}))
	assert.False(t, shouldReopenInflightTask(stored, &InflightTask{
		Status: InflightTaskStatusCompleted,
		Detail: &InflightTaskDetail{RetryIndex: 1},
	}))
}

func TestShouldApplyReconciledTerminalStatus(t *testing.T) {
	originalRetryTimes := common.RetryTimes
	common.RetryTimes = 2
	defer func() {
		common.RetryTimes = originalRetryTimes
	}()

	inflight := &InflightTask{
		Status: InflightTaskStatusUpstreamPending,
		Detail: &InflightTaskDetail{RetryIndex: 1},
	}
	assert.False(t, shouldApplyReconciledTerminalStatus(inflight, InflightTaskStatusFailed))

	inflight.Detail.RetryIndex = 2
	assert.True(t, shouldApplyReconciledTerminalStatus(inflight, InflightTaskStatusFailed))
	assert.True(t, shouldApplyReconciledTerminalStatus(inflight, InflightTaskStatusCompleted))
	assert.False(t, shouldApplyReconciledTerminalStatus(inflight, InflightTaskStatusRouting))
}

func TestInflightTaskDetailFromRelayInfoIncludesChannelWhenMetaSet(t *testing.T) {
	info := &relaycommon.RelayInfo{
		RetryIndex: 0,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   16,
			ChannelName: "Hiyo",
		},
	}

	detail := inflightTaskDetailFromRelayInfo(info, InflightTaskStatusUpstreamPending, 100)
	require.NotNil(t, detail)
	assert.Equal(t, 16, detail.ChannelID)
	assert.Equal(t, "Hiyo", detail.ChannelName)
	require.Len(t, detail.ChannelChain, 1)
	assert.Equal(t, 16, detail.ChannelChain[0].ChannelID)
	require.Len(t, detail.Attempts, 1)
	assert.Equal(t, InflightTaskStatusUpstreamPending, detail.Attempts[0].Status)
}

func TestInflightTaskDetailFromRelayInfoOmitsChannelWithoutMeta(t *testing.T) {
	info := &relaycommon.RelayInfo{RetryIndex: 0}

	detail := inflightTaskDetailFromRelayInfo(info, InflightTaskStatusUpstreamPending, 100)
	require.NotNil(t, detail)
	assert.Zero(t, detail.ChannelID)
	assert.Empty(t, detail.ChannelName)
	assert.Empty(t, detail.ChannelChain)
	assert.Empty(t, detail.Attempts)
}

func TestUpdateInflightTaskDetailBuildsAttemptTimeline(t *testing.T) {
	detail := &InflightTaskDetail{
		RetryIndex: 0,
		ChannelID:  10,
		ChannelName: "first",
		ChannelChain: []InflightTaskChannelAttempt{{
			RetryIndex:  0,
			ChannelID:   10,
			ChannelName: "first",
			StartedAt:   100,
			UpdatedAt:   100,
		}},
		Attempts: []InflightTaskAttempt{{
			RetryIndex:  0,
			ChannelID:   10,
			ChannelName: "first",
			StartedAt:   100,
			UpdatedAt:   100,
		}},
	}

	updateInflightTaskDetail(detail, InflightTaskStatusRouting, 100, 0)
	updateInflightTaskDetail(detail, InflightTaskStatusUpstreamPending, 105, 0)
	detail.LatestError = "timeout"
	updateInflightTaskDetail(detail, InflightTaskStatusFailed, 110, 0)

	require.Len(t, detail.Attempts, 1)
	require.Len(t, detail.Attempts[0].Timeline, 3)
	assert.Equal(t, InflightTaskStatusRouting, detail.Attempts[0].Timeline[0].Status)
	assert.Equal(t, InflightTaskStatusUpstreamPending, detail.Attempts[0].Timeline[1].Status)
	assert.Equal(t, InflightTaskStatusFailed, detail.Attempts[0].Timeline[2].Status)
	assert.Equal(t, "timeout", detail.Attempts[0].Error)
}

func TestMergeInflightTaskDetailSplitsAttemptsByRetryIndex(t *testing.T) {
	stored := &InflightTask{
		Detail: &InflightTaskDetail{
			RetryIndex: 0,
			ChannelChain: []InflightTaskChannelAttempt{{
				RetryIndex:  0,
				ChannelID:   10,
				ChannelName: "first",
				Status:      InflightTaskStatusFailed,
				StartedAt:   100,
				UpdatedAt:   110,
			}},
			Attempts: []InflightTaskAttempt{{
				RetryIndex:  0,
				ChannelID:   10,
				ChannelName: "first",
				Status:      InflightTaskStatusFailed,
				StartedAt:   100,
				UpdatedAt:   110,
				Timeline: []InflightTaskStatusStep{
					{Status: InflightTaskStatusRouting, StartedAt: 100, UpdatedAt: 100},
					{Status: InflightTaskStatusUpstreamPending, StartedAt: 100, UpdatedAt: 105},
					{Status: InflightTaskStatusFailed, StartedAt: 105, UpdatedAt: 110},
				},
			}},
		},
	}
	next := &InflightTask{
		Status:    InflightTaskStatusRouting,
		UpdatedAt: 120,
		Detail: &InflightTaskDetail{
			RetryIndex: 1,
			ChannelID:  20,
			ChannelName: "second",
			ChannelChain: []InflightTaskChannelAttempt{{
				RetryIndex:  1,
				ChannelID:   20,
				ChannelName: "second",
				Status:      InflightTaskStatusRouting,
				StartedAt:   120,
				UpdatedAt:   120,
			}},
			Attempts: []InflightTaskAttempt{{
				RetryIndex:  1,
				ChannelID:   20,
				ChannelName: "second",
				Status:      InflightTaskStatusRouting,
				StartedAt:   120,
				UpdatedAt:   120,
				Timeline: []InflightTaskStatusStep{
					{Status: InflightTaskStatusRouting, StartedAt: 120, UpdatedAt: 120},
				},
			}},
		},
	}

	mergeInflightTaskDetail(stored, next)

	require.Len(t, next.Detail.Attempts, 2)
	assert.Equal(t, 0, next.Detail.Attempts[0].RetryIndex)
	assert.Equal(t, 1, next.Detail.Attempts[1].RetryIndex)
	assert.Equal(t, "second", next.Detail.Attempts[1].ChannelName)
	assert.Equal(t, 20, next.Detail.ChannelID)
	assert.Equal(t, "second", next.Detail.ChannelName)
	require.Len(t, next.Detail.ChannelChain, 2)
	assert.Equal(t, 10, next.Detail.ChannelChain[0].ChannelID)
	assert.Equal(t, 20, next.Detail.ChannelChain[1].ChannelID)
	require.Len(t, next.Detail.Attempts[1].Timeline, 1)
	assert.Equal(t, InflightTaskStatusRouting, next.Detail.Attempts[1].Timeline[0].Status)
}

func TestMergeInflightTaskDetailRetryUpstreamPendingShowsCurrentChannel(t *testing.T) {
	stored := &InflightTask{
		Status: InflightTaskStatusRouting,
		Detail: &InflightTaskDetail{
			RetryIndex:  1,
			ChannelID:   20,
			ChannelName: "second",
			ChannelChain: []InflightTaskChannelAttempt{
				{RetryIndex: 0, ChannelID: 10, ChannelName: "first", Status: InflightTaskStatusFailed, StartedAt: 100, UpdatedAt: 110},
				{RetryIndex: 1, ChannelID: 20, ChannelName: "second", Status: InflightTaskStatusRouting, StartedAt: 120, UpdatedAt: 120},
			},
			Attempts: []InflightTaskAttempt{
				{
					RetryIndex: 0, ChannelID: 10, ChannelName: "first", Status: InflightTaskStatusFailed,
					StartedAt: 100, UpdatedAt: 110,
					Timeline: []InflightTaskStatusStep{
						{Status: InflightTaskStatusRouting, StartedAt: 100, UpdatedAt: 100},
						{Status: InflightTaskStatusUpstreamPending, StartedAt: 100, UpdatedAt: 105},
						{Status: InflightTaskStatusFailed, StartedAt: 105, UpdatedAt: 110},
					},
				},
				{
					RetryIndex: 1, ChannelID: 20, ChannelName: "second", Status: InflightTaskStatusRouting,
					StartedAt: 120, UpdatedAt: 120,
					Timeline: []InflightTaskStatusStep{
						{Status: InflightTaskStatusRouting, StartedAt: 120, UpdatedAt: 120},
					},
				},
			},
		},
	}
	next := inflightTaskFromRelayInfo(&relaycommon.RelayInfo{
		RetryIndex: 1,
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelId:   20,
			ChannelName: "second",
		},
	}, InflightTaskStatusUpstreamPending)
	next.UpdatedAt = 125

	mergeInflightTaskDetail(stored, next)

	assert.Equal(t, InflightTaskStatusUpstreamPending, next.Status)
	assert.Equal(t, 1, next.Detail.RetryIndex)
	assert.Equal(t, 20, next.Detail.ChannelID)
	assert.Equal(t, "second", next.Detail.ChannelName)
	require.Len(t, next.Detail.Attempts, 2)
	assert.Equal(t, InflightTaskStatusUpstreamPending, next.Detail.Attempts[1].Status)
	require.GreaterOrEqual(t, len(next.Detail.Attempts[1].Timeline), 2)
	assert.Equal(t, InflightTaskStatusUpstreamPending, next.Detail.Attempts[1].Timeline[len(next.Detail.Attempts[1].Timeline)-1].Status)
}

func TestShouldPersistInflightTaskUpdateAllowsRetryAdvance(t *testing.T) {
	next := &InflightTask{
		Status: InflightTaskStatusRouting,
		Detail: &InflightTaskDetail{RetryIndex: 2},
	}

	assert.True(t, shouldPersistInflightTaskUpdate(InflightTaskStatusUpstreamPending, 1, false, next, nil))
	assert.False(t, shouldPersistInflightTaskUpdate(InflightTaskStatusUpstreamPending, 2, false, next, nil))
	assert.True(t, shouldPersistInflightTaskUpdate(InflightTaskStatusUpstreamPending, 1, false, &InflightTask{
		Status: InflightTaskStatusUpstreamPending,
		Detail: &InflightTaskDetail{RetryIndex: 2},
	}, nil))
	assert.False(t, shouldPersistInflightTaskUpdate(InflightTaskStatusUpstreamPending, 2, false, &InflightTask{
		Status: InflightTaskStatusRouting,
		Detail: &InflightTaskDetail{RetryIndex: 2},
	}, nil))
}

func TestShouldPersistInflightTaskUpdateBackfillsMissingAttemptSlot(t *testing.T) {
	storedDetail := &InflightTaskDetail{
		RetryIndex: 2,
		Attempts: []InflightTaskAttempt{
			{RetryIndex: 1, ChannelID: 16, Status: InflightTaskStatusFailed},
			{RetryIndex: 2, ChannelID: 10, Status: InflightTaskStatusFailed},
		},
		ChannelChain: []InflightTaskChannelAttempt{
			{RetryIndex: 1, ChannelID: 16, Status: InflightTaskStatusFailed},
			{RetryIndex: 2, ChannelID: 10, Status: InflightTaskStatusFailed},
		},
	}
	next := &InflightTask{
		Status: InflightTaskStatusUpstreamPending,
		Detail: &InflightTaskDetail{
			RetryIndex: 0,
			ChannelID:  16,
			ChannelName: "first",
			ChannelChain: []InflightTaskChannelAttempt{{
				RetryIndex: 0, ChannelID: 16, ChannelName: "first", Status: InflightTaskStatusUpstreamPending,
			}},
			Attempts: []InflightTaskAttempt{{
				RetryIndex: 0, ChannelID: 16, ChannelName: "first", Status: InflightTaskStatusUpstreamPending,
			}},
		},
	}

	assert.True(t, shouldPersistInflightTaskUpdate(InflightTaskStatusRouting, 2, false, next, storedDetail))
}

func TestMergeInflightTaskDetailFinalizesPreviousAttemptOnRetryAdvance(t *testing.T) {
	stored := &InflightTask{
		Detail: &InflightTaskDetail{
			RetryIndex:  0,
			LatestError: "daily usage limit exceeded",
			Attempts: []InflightTaskAttempt{{
				RetryIndex:  0,
				ChannelID:   10,
				ChannelName: "first",
				Status:      InflightTaskStatusUpstreamPending,
				StartedAt:   100,
				UpdatedAt:   110,
			}},
		},
	}
	next := &InflightTask{
		Status:    InflightTaskStatusRouting,
		UpdatedAt: 120,
		Detail: &InflightTaskDetail{
			RetryIndex:  1,
			LatestError: "daily usage limit exceeded",
			ChannelID:   20,
			ChannelName: "second",
			Attempts: []InflightTaskAttempt{{
				RetryIndex:  1,
				ChannelID:   20,
				ChannelName: "second",
				Status:      InflightTaskStatusRouting,
				StartedAt:   120,
				UpdatedAt:   120,
			}},
		},
	}

	mergeInflightTaskDetail(stored, next)

	require.Len(t, next.Detail.Attempts, 2)
	assert.Equal(t, InflightTaskStatusFailed, next.Detail.Attempts[0].Status)
	assert.Equal(t, "daily usage limit exceeded", next.Detail.Attempts[0].Error)
	assert.Equal(t, 1, next.Detail.Attempts[1].RetryIndex)
}

func TestMergeInflightTaskDetailDedupesAndOrdersOutOfOrderRetries(t *testing.T) {
	// Simulate async status writes arriving out of order: a delayed retry_index=1
	// write shows up after retry_index=2 has already been recorded. It must update
	// the existing slot in place (no duplicate) and stay ordered by retry_index.
	stored := &InflightTask{
		Detail: &InflightTaskDetail{
			RetryIndex: 2,
			ChannelChain: []InflightTaskChannelAttempt{
				{RetryIndex: 0, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 100, UpdatedAt: 110},
				{RetryIndex: 2, ChannelID: 13, Status: InflightTaskStatusRouting, StartedAt: 130, UpdatedAt: 130},
			},
			Attempts: []InflightTaskAttempt{
				{RetryIndex: 0, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 100, UpdatedAt: 110},
				{RetryIndex: 2, ChannelID: 13, Status: InflightTaskStatusRouting, StartedAt: 130, UpdatedAt: 130},
			},
		},
	}
	next := &InflightTask{
		Status:    InflightTaskStatusUpstreamPending,
		UpdatedAt: 135,
		Detail: &InflightTaskDetail{
			RetryIndex:  1,
			ChannelID:   12,
			ChannelName: "second",
			ChannelChain: []InflightTaskChannelAttempt{{
				RetryIndex: 1, ChannelID: 12, ChannelName: "second", Status: InflightTaskStatusUpstreamPending, StartedAt: 120, UpdatedAt: 135,
			}},
			Attempts: []InflightTaskAttempt{{
				RetryIndex: 1, ChannelID: 12, ChannelName: "second", Status: InflightTaskStatusUpstreamPending, StartedAt: 120, UpdatedAt: 135,
			}},
		},
	}

	mergeInflightTaskDetail(stored, next)

	require.Len(t, next.Detail.Attempts, 3)
	assert.Equal(t, []int{0, 1, 2}, []int{
		next.Detail.Attempts[0].RetryIndex,
		next.Detail.Attempts[1].RetryIndex,
		next.Detail.Attempts[2].RetryIndex,
	})
	require.Len(t, next.Detail.ChannelChain, 3)
	assert.Equal(t, []int{0, 1, 2}, []int{
		next.Detail.ChannelChain[0].RetryIndex,
		next.Detail.ChannelChain[1].RetryIndex,
		next.Detail.ChannelChain[2].RetryIndex,
	})
	// retry_index 1 is superseded by 2, so it must be finalized as failed.
	assert.Equal(t, InflightTaskStatusFailed, next.Detail.Attempts[1].Status)
	// The displayed current retry must not regress below the furthest attempt.
	assert.Equal(t, 2, next.Detail.RetryIndex)
}

func TestEnsureInflightAttemptCoverageBackfillsMissingInitialAttempt(t *testing.T) {
	detail := &InflightTaskDetail{
		RetryIndex:  2,
		ChannelID:   10,
		ChannelName: "last",
		LatestError: "API key is disabled",
		ChannelChain: []InflightTaskChannelAttempt{
			{RetryIndex: 0, ChannelID: 16, ChannelName: "first", Status: InflightTaskStatusFailed, StartedAt: 100, UpdatedAt: 110},
			{RetryIndex: 1, ChannelID: 16, ChannelName: "first", Status: InflightTaskStatusFailed, StartedAt: 110, UpdatedAt: 120},
			{RetryIndex: 2, ChannelID: 10, ChannelName: "last", Status: InflightTaskStatusFailed, StartedAt: 130, UpdatedAt: 140, Error: "API key is disabled"},
		},
		Attempts: []InflightTaskAttempt{
			{RetryIndex: 1, ChannelID: 16, ChannelName: "first", Status: InflightTaskStatusFailed, StartedAt: 110, UpdatedAt: 120},
			{RetryIndex: 2, ChannelID: 10, ChannelName: "last", Status: InflightTaskStatusFailed, StartedAt: 130, UpdatedAt: 140, Error: "API key is disabled"},
		},
	}

	ensureInflightAttemptCoverage(detail, InflightTaskStatusFailed, 140)

	require.Len(t, detail.Attempts, 3)
	assert.Equal(t, 0, detail.Attempts[0].RetryIndex)
	assert.Equal(t, 1, detail.Attempts[1].RetryIndex)
	assert.Equal(t, 2, detail.Attempts[2].RetryIndex)
	assert.Equal(t, "API key is disabled", detail.Attempts[2].Error)
}

func TestUpdateInflightTaskDetailCreatesMissingRetryAttempt(t *testing.T) {
	detail := &InflightTaskDetail{
		RetryIndex:  6,
		ChannelID:   10,
		ChannelName: "last",
		LatestError: "API key is disabled",
		Attempts: []InflightTaskAttempt{
			{RetryIndex: 0, ChannelID: 16, Status: InflightTaskStatusFailed, StartedAt: 100, UpdatedAt: 110},
			{RetryIndex: 1, ChannelID: 13, Status: InflightTaskStatusFailed, StartedAt: 120, UpdatedAt: 130},
			{RetryIndex: 2, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 140, UpdatedAt: 150},
			{RetryIndex: 3, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 160, UpdatedAt: 170},
			{RetryIndex: 4, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 180, UpdatedAt: 190},
			{RetryIndex: 5, ChannelID: 10, Status: InflightTaskStatusFailed, StartedAt: 200, UpdatedAt: 210},
		},
	}

	updateInflightTaskDetail(detail, InflightTaskStatusFailed, 220, 6)

	require.Len(t, detail.Attempts, 7)
	assert.Equal(t, 6, detail.Attempts[6].RetryIndex)
	assert.Equal(t, InflightTaskStatusFailed, detail.Attempts[6].Status)
	assert.Equal(t, "API key is disabled", detail.Attempts[6].Error)
	assert.Equal(t, 10, detail.Attempts[6].ChannelID)
}

func TestNewInflightTaskFinalizeContextIgnoresParentCancel(t *testing.T) {
	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()

	ctx, cancel := NewInflightTaskFinalizeContext(parent)
	defer cancel()

	require.NoError(t, ctx.Err())

	select {
	case <-ctx.Done():
		t.Fatal("finalize context should not inherit parent cancellation immediately")
	case <-time.After(10 * time.Millisecond):
	}
}

func TestFinalInflightTaskStatusUsesStreamStatus(t *testing.T) {
	info := &relaycommon.RelayInfo{IsStream: true, StreamStatus: relaycommon.NewStreamStatus()}
	info.StreamStatus.SetEndReason(relaycommon.StreamEndReasonClientGone, errors.New("context canceled"))

	assert.Equal(t, InflightTaskStatusFailed, FinalInflightTaskStatus(info))
}
