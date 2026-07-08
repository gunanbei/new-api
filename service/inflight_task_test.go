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

	updateInflightTaskDetail(detail, InflightTaskStatusRouting, 100)
	updateInflightTaskDetail(detail, InflightTaskStatusUpstreamPending, 105)
	detail.LatestError = "timeout"
	updateInflightTaskDetail(detail, InflightTaskStatusFailed, 110)

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
	require.Len(t, next.Detail.Attempts[1].Timeline, 1)
	assert.Equal(t, InflightTaskStatusRouting, next.Detail.Attempts[1].Timeline[0].Status)
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
