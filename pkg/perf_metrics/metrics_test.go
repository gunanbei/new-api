package perfmetrics

import (
	"testing"

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
