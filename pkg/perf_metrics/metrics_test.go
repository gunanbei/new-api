package perfmetrics

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildGroupSummaries(t *testing.T) {
	summaries := buildGroupSummaries(map[string]map[int64]counters{
		"standard": {
			2: {requestCount: 2, successCount: 1, totalLatencyMs: 400, ttftSumMs: 80, ttftCount: 2, outputTokens: 40, generationMs: 200, cacheHitTokens: 60, inputTokens: 300},
			1: {requestCount: 2, successCount: 2, totalLatencyMs: 200, ttftSumMs: 40, ttftCount: 2, outputTokens: 20, generationMs: 100, cacheHitTokens: 40, inputTokens: 200},
		},
	})

	require.Len(t, summaries, 1)
	require.Equal(t, "standard", summaries[0].Group)
	require.EqualValues(t, 4, summaries[0].RequestCount)
	require.Equal(t, 75.0, summaries[0].SuccessRate)
	require.Equal(t, 75.0, summaries[0].Availability)
	require.EqualValues(t, 150, summaries[0].AvgLatencyMs)
	require.EqualValues(t, 30, summaries[0].AvgTtftMs)
	require.Equal(t, 200.0, summaries[0].AvgTps)
	require.Equal(t, "error", summaries[0].Status)
	require.EqualValues(t, 2, summaries[0].LastUpdated)
	require.EqualValues(t, 40, summaries[0].LatestTtftMs)
	require.Equal(t, 200.0, summaries[0].LatestTps)
	require.Equal(t, 20.0, summaries[0].CacheHitRate)
	require.EqualValues(t, 100, summaries[0].CacheHitTokens)
	require.EqualValues(t, 500, summaries[0].InputTokens)
	require.Equal(t, 20.0, summaries[0].Series[1].CacheHitRate)
	require.Equal(t, []int64{1, 2}, []int64{summaries[0].Series[0].Ts, summaries[0].Series[1].Ts})
}

func TestApplyGroupStatus(t *testing.T) {
	groupMonitorStates = sync.Map{}
	now := time.Now()
	for range 20 {
		recordGroupMonitorSample("available", true, 0, 0, now)
	}
	available := GroupSummary{Group: "available"}
	ApplyGroupStatus(&available, 1, 0)
	require.Equal(t, "available", available.Status)

	for range 13 {
		recordGroupMonitorSample("error", true, 0, 0, now)
	}
	for range 7 {
		recordGroupMonitorSample("error", false, 0, 0, now)
	}
	errorSummary := GroupSummary{Group: "error"}
	ApplyGroupStatus(&errorSummary, 1, 0)
	require.Equal(t, "error", errorSummary.Status)
	require.Contains(t, errorSummary.StatusReasons, StatusReason{Code: "recent_success_rate_error", SuccessRate: 65})
}

func TestApplyGroupStatusWithInsufficientSamples(t *testing.T) {
	groupMonitorStates = sync.Map{}

	historicalSuccess := GroupSummary{Group: "historical-success", RequestCount: 1, Availability: 100}
	ApplyGroupStatus(&historicalSuccess, 1, 0)
	require.Equal(t, "available", historicalSuccess.Status)

	unknown := GroupSummary{Group: "unknown", RequestCount: 1}
	ApplyGroupStatus(&unknown, 1, 0)
	require.Equal(t, "unknown", unknown.Status)
	require.Contains(t, unknown.StatusReasons, StatusReason{Code: "insufficient_sampling"})
}

func TestApplyGroupStatusUsesPersistedMetricsAfterRestart(t *testing.T) {
	groupMonitorStates = sync.Map{}

	summary := buildGroupSummaries(map[string]map[int64]counters{
		"historical-success": {
			1: {requestCount: 522, successCount: 505},
			2: {requestCount: 20, successCount: 20},
		},
	})[0]
	ApplyGroupStatus(&summary, 1, 0)

	require.Equal(t, "available", summary.Status)
	require.EqualValues(t, 542, summary.RequestCount)
	require.InDelta(t, 96.86, summary.Availability, 0.01)
	require.Empty(t, summary.StatusReasons)
}
