package perfmetrics

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/perf_metrics_setting"
)

var hotBuckets sync.Map
var groupMonitorStates sync.Map

// seriesSchema is a stable client cache/schema marker. Do not change it when
// hiding fields or making response-only privacy hardening changes.
const seriesSchema = "a1f1a225cd623533"

func Init() {
	go flushLoop()
}

func RecordRelaySample(info *relaycommon.RelayInfo, success bool, outputTokens int64) {
	RecordRelaySampleWithCache(info, success, outputTokens, 0, 0)
}

func RecordRelaySampleWithCache(info *relaycommon.RelayInfo, success bool, outputTokens int64, cacheHitTokens int64, inputTokens int64) {
	if info == nil {
		return
	}
	now := time.Now()
	hasTtft := info.IsStream && info.HasSendResponse()
	ttftMs := int64(0)
	if hasTtft {
		ttftMs = info.FirstResponseTime.Sub(info.StartTime).Milliseconds()
	}
	latencyMs := now.Sub(info.StartTime).Milliseconds()
	generationMs := latencyMs
	if hasTtft {
		generationMs = now.Sub(info.FirstResponseTime).Milliseconds()
	}
	if generationMs <= 0 {
		generationMs = latencyMs
	}
	Record(Sample{
		Model:          info.OriginModelName,
		Group:          info.UsingGroup,
		LatencyMs:      latencyMs,
		TtftMs:         ttftMs,
		HasTtft:        hasTtft,
		Success:        success,
		OutputTokens:   outputTokens,
		GenerationMs:   generationMs,
		CacheHitTokens: cacheHitTokens,
		InputTokens:    inputTokens,
	})
}

func Record(sample Sample) {
	setting := perf_metrics_setting.GetSetting()
	if !setting.Enabled || sample.Model == "" {
		return
	}
	if sample.Group == "" {
		sample.Group = "default"
	}
	if sample.LatencyMs < 0 {
		sample.LatencyMs = 0
	}
	if sample.CacheHitTokens < 0 {
		sample.CacheHitTokens = 0
	}
	if sample.InputTokens < sample.CacheHitTokens {
		sample.InputTokens = sample.CacheHitTokens
	}
	now := time.Now()
	recordGroupMonitorSample(sample.Group, sample.Success, sample.CacheHitTokens, sample.InputTokens, now)

	key := bucketKey{
		model:    sample.Model,
		group:    sample.Group,
		bucketTs: bucketStart(now.Unix()),
	}
	actual, _ := hotBuckets.LoadOrStore(key, &atomicBucket{})
	actual.(*atomicBucket).add(sample)
	recordRedis(key, sample)
}

func Query(params QueryParams) (QueryResult, error) {
	if params.Hours <= 0 {
		params.Hours = 24
	}
	if params.Hours > 24*30 {
		params.Hours = 24 * 30
	}
	endTs := time.Now().Unix()
	startTs := endTs - int64(params.Hours)*3600

	merged := map[bucketKey]counters{}
	rows, err := model.GetPerfMetrics(params.Model, params.Group, startTs, endTs)
	if err != nil {
		return QueryResult{}, err
	}
	for _, row := range rows {
		mergeCounters(merged, bucketKey{
			model:    row.ModelName,
			group:    row.Group,
			bucketTs: row.BucketTs,
		}, counters{
			requestCount:   row.RequestCount,
			successCount:   row.SuccessCount,
			totalLatencyMs: row.TotalLatencyMs,
			ttftSumMs:      row.TtftSumMs,
			ttftCount:      row.TtftCount,
			outputTokens:   row.OutputTokens,
			generationMs:   row.GenerationMs,
			cacheHitTokens: row.CacheHitTokens,
			inputTokens:    row.InputTokens,
		})
	}

	hotBuckets.Range(func(key, value any) bool {
		k := key.(bucketKey)
		if k.model != params.Model || k.bucketTs < startTs || k.bucketTs > endTs {
			return true
		}
		if params.Group != "" && k.group != params.Group {
			return true
		}
		mergeCounters(merged, k, value.(*atomicBucket).snapshot())
		return true
	})

	return buildQueryResult(params.Model, merged), nil
}

func QuerySummaryAll(hours int, groups []string) (SummaryAllResult, error) {
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*30 {
		hours = 24 * 30
	}
	endTs := time.Now().Unix()
	startTs := endTs - int64(hours)*3600
	allowedGroups := allowedGroupSet(groups)

	rows, err := model.GetPerfMetricsSummaryBucketsAll(startTs, endTs, groups)
	if err != nil {
		return SummaryAllResult{}, err
	}

	totals := map[string]counters{}
	modelBuckets := map[string]map[int64]counters{}
	for _, row := range rows {
		value := counters{
			requestCount:   row.RequestCount,
			successCount:   row.SuccessCount,
			totalLatencyMs: row.TotalLatencyMs,
			outputTokens:   row.OutputTokens,
			generationMs:   row.GenerationMs,
		}
		mergeModelTotals(totals, row.ModelName, value)
		mergeModelBucket(modelBuckets, row.ModelName, row.BucketTs, value)
	}

	hotBuckets.Range(func(key, value any) bool {
		k := key.(bucketKey)
		if k.bucketTs < startTs || k.bucketTs > endTs {
			return true
		}
		if allowedGroups != nil {
			if _, ok := allowedGroups[k.group]; !ok {
				return true
			}
		}
		snap := value.(*atomicBucket).snapshot()
		if snap.requestCount == 0 {
			return true
		}
		mergeModelTotals(totals, k.model, snap)
		mergeModelBucket(modelBuckets, k.model, k.bucketTs, snap)
		return true
	})

	models := make([]ModelSummary, 0, len(totals))
	for name, total := range totals {
		if total.requestCount == 0 {
			continue
		}
		avgLatency := total.totalLatencyMs / total.requestCount
		successRate := float64(total.successCount) / float64(total.requestCount) * 100
		avgTps := 0.0
		if total.generationMs > 0 {
			avgTps = float64(total.outputTokens) / (float64(total.generationMs) / 1000.0)
		}
		models = append(models, ModelSummary{
			ModelName:          name,
			AvgLatencyMs:       avgLatency,
			SuccessRate:        math.Round(successRate*100) / 100,
			AvgTps:             math.Round(avgTps*100) / 100,
			RecentSuccessRates: recentSuccessRates(modelBuckets[name], 3),
			RequestCount:       total.requestCount,
		})
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i].RequestCount > models[j].RequestCount
	})

	return SummaryAllResult{Models: models}, nil
}

func QueryGroupSummaryAll(hours int, groups []string) ([]GroupSummary, error) {
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*30 {
		hours = 24 * 30
	}
	endTs := time.Now().Unix()
	startTs := endTs - int64(hours)*3600
	allowedGroups := allowedGroupSet(groups)

	rows, err := model.GetPerfMetricsGroupSummaryBucketsAll(startTs, endTs, groups)
	if err != nil {
		return nil, err
	}

	groupBuckets := map[string]map[int64]counters{}
	for _, row := range rows {
		mergeGroupBucket(groupBuckets, row.Group, row.BucketTs, counters{
			requestCount:   row.RequestCount,
			successCount:   row.SuccessCount,
			totalLatencyMs: row.TotalLatencyMs,
			ttftSumMs:      row.TtftSumMs,
			ttftCount:      row.TtftCount,
			outputTokens:   row.OutputTokens,
			generationMs:   row.GenerationMs,
			cacheHitTokens: row.CacheHitTokens,
			inputTokens:    row.InputTokens,
		})
	}

	// Redis retains the active bucket across process restarts. Prefer it over the
	// local bucket snapshot so current metrics are not counted twice.
	redisCurrentGroups := mergeRedisActiveGroupBuckets(groupBuckets, allowedGroups, startTs, endTs)
	currentBucketTs := bucketStart(endTs)
	hotBuckets.Range(func(key, value any) bool {
		k := key.(bucketKey)
		if k.bucketTs < startTs || k.bucketTs > endTs {
			return true
		}
		if allowedGroups != nil {
			if _, ok := allowedGroups[k.group]; !ok {
				return true
			}
		}
		if k.bucketTs == currentBucketTs && redisCurrentGroups[k.group] {
			return true
		}
		mergeGroupBucket(groupBuckets, k.group, k.bucketTs, value.(*atomicBucket).snapshot())
		return true
	})

	return buildGroupSummaries(groupBuckets), nil
}

func mergeGroupBucket(groupBuckets map[string]map[int64]counters, group string, bucketTs int64, value counters) {
	if value.requestCount == 0 {
		return
	}
	if _, ok := groupBuckets[group]; !ok {
		groupBuckets[group] = map[int64]counters{}
	}
	current := groupBuckets[group][bucketTs]
	current.requestCount += value.requestCount
	current.successCount += value.successCount
	current.totalLatencyMs += value.totalLatencyMs
	current.ttftSumMs += value.ttftSumMs
	current.ttftCount += value.ttftCount
	current.outputTokens += value.outputTokens
	current.generationMs += value.generationMs
	current.cacheHitTokens += value.cacheHitTokens
	current.inputTokens += value.inputTokens
	groupBuckets[group][bucketTs] = current
}

func buildGroupSummaries(groupBuckets map[string]map[int64]counters) []GroupSummary {
	groups := make([]string, 0, len(groupBuckets))
	for group := range groupBuckets {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	summaries := make([]GroupSummary, 0, len(groups))
	for _, group := range groups {
		buckets := groupBuckets[group]
		timestamps := make([]int64, 0, len(buckets))
		for ts := range buckets {
			timestamps = append(timestamps, ts)
		}
		sort.Slice(timestamps, func(i, j int) bool { return timestamps[i] < timestamps[j] })

		total := counters{}
		series := make([]BucketPoint, 0, len(timestamps))
		for _, ts := range timestamps {
			value := buckets[ts]
			total.requestCount += value.requestCount
			total.successCount += value.successCount
			total.totalLatencyMs += value.totalLatencyMs
			total.ttftSumMs += value.ttftSumMs
			total.ttftCount += value.ttftCount
			total.outputTokens += value.outputTokens
			total.generationMs += value.generationMs
			total.cacheHitTokens += value.cacheHitTokens
			total.inputTokens += value.inputTokens
			series = append(series, bucketPoint(ts, value))
		}
		lastUpdated := int64(0)
		latestTtftMs := int64(0)
		latestTps := 0.0
		status := "unknown"
		if len(series) > 0 {
			last := series[len(series)-1]
			lastUpdated = last.Ts
			latestTtftMs = last.AvgTtftMs
			latestTps = last.AvgTps
			if last.SuccessRate >= 90 {
				status = "available"
			} else {
				status = "warning"
			}
		}
		summaries = append(summaries, GroupSummary{
			Group:          group,
			Status:         status,
			AvgTtftMs:      avg(total.ttftSumMs, total.ttftCount),
			AvgLatencyMs:   avg(total.totalLatencyMs, total.requestCount),
			SuccessRate:    successRate(total),
			Availability:   successRate(total),
			AvgTps:         avgTps(total),
			RequestCount:   total.requestCount,
			LastUpdated:    lastUpdated,
			LatestTtftMs:   latestTtftMs,
			LatestTps:      latestTps,
			CacheHitRate:   cacheHitRate(total),
			CacheHitTokens: total.cacheHitTokens,
			InputTokens:    total.inputTokens,
			Series:         series,
		})
	}
	return summaries
}

func recordGroupMonitorSample(group string, success bool, cacheHitTokens int64, inputTokens int64, now time.Time) {
	stateValue, _ := groupMonitorStates.LoadOrStore(group, &groupMonitorState{})
	state := stateValue.(*groupMonitorState)
	state.mu.Lock()
	defer state.mu.Unlock()

	state.recentCalls = append(state.recentCalls, success)
	if len(state.recentCalls) > 20 {
		state.recentCalls = state.recentCalls[len(state.recentCalls)-20:]
	}
	state.hasLastCall = true
	state.lastCallSuccess = success
	if inputTokens <= 0 {
		return
	}

	cacheHitRate := float64(cacheHitTokens) / float64(inputTokens) * 100
	if state.hasLastCacheHitRate && math.Abs(state.lastCacheHitRate-cacheHitRate) > 20 {
		state.cacheFluctuationEvents = append(state.cacheFluctuationEvents, now)
	}
	state.hasLastCacheHitRate = true
	state.lastCacheHitRate = cacheHitRate
	state.cacheFluctuationEvents = filterCacheFluctuationEvents(state.cacheFluctuationEvents, now.Add(-6*time.Hour))
}

type groupMonitorState struct {
	mu                     sync.Mutex
	recentCalls            []bool
	hasLastCall            bool
	lastCallSuccess        bool
	hasLastCacheHitRate    bool
	lastCacheHitRate       float64
	cacheFluctuationEvents []time.Time
}

type groupMonitorSnapshot struct {
	recentCalls       []bool
	hasLastCall       bool
	lastCallSuccess   bool
	cacheFluctuations int
}

func groupMonitorStatus(group string) groupMonitorSnapshot {
	stateValue, ok := groupMonitorStates.Load(group)
	if !ok {
		return groupMonitorSnapshot{}
	}
	state := stateValue.(*groupMonitorState)
	state.mu.Lock()
	defer state.mu.Unlock()
	state.cacheFluctuationEvents = filterCacheFluctuationEvents(state.cacheFluctuationEvents, time.Now().Add(-6*time.Hour))
	return groupMonitorSnapshot{
		recentCalls:       append([]bool(nil), state.recentCalls...),
		hasLastCall:       state.hasLastCall,
		lastCallSuccess:   state.lastCallSuccess,
		cacheFluctuations: len(state.cacheFluctuationEvents),
	}
}

func filterCacheFluctuationEvents(events []time.Time, cutoff time.Time) []time.Time {
	first := 0
	for first < len(events) && events[first].Before(cutoff) {
		first++
	}
	return events[first:]
}

func ApplyGroupStatus(summary *GroupSummary, channelCount int64, disabledChannelCount int64) {
	if summary == nil {
		return
	}

	status := groupMonitorStatus(summary.Group)
	if channelCount > 0 && disabledChannelCount >= channelCount {
		summary.Status = "error"
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{Code: "all_channels_disabled"})
		return
	}

	if summary.RequestCount == 0 {
		summary.Status = "unknown"
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{Code: "insufficient_sampling"})
		return
	}
	if len(status.recentCalls) > 0 {
		recentSuccessRate := recentSuccessRate(status.recentCalls)
		if recentSuccessRate < 70 {
			summary.StatusReasons = append(summary.StatusReasons, StatusReason{
				Code:        "recent_success_rate_error",
				SuccessRate: recentSuccessRate,
			})
		}
	}
	if status.hasLastCall && !status.lastCallSuccess {
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{Code: "last_call_failed"})
	}
	if len(summary.StatusReasons) > 0 {
		summary.Status = "error"
		return
	}

	if summary.Availability < 90 {
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{
			Code:        "selected_period_success_rate",
			SuccessRate: summary.Availability,
		})
	}
	if status.cacheFluctuations > 20 {
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{
			Code:             "cache_hit_rate_volatility",
			FluctuationCount: status.cacheFluctuations,
		})
	}
	if len(status.recentCalls) > 0 {
		recentSuccessRate := recentSuccessRate(status.recentCalls)
		if recentSuccessRate >= 70 && recentSuccessRate < 98 {
			summary.StatusReasons = append(summary.StatusReasons, StatusReason{
				Code:        "recent_success_rate_warning",
				SuccessRate: recentSuccessRate,
			})
		}
	}
	if channelCount > 0 && disabledChannelCount > 0 {
		summary.StatusReasons = append(summary.StatusReasons, StatusReason{Code: "channels_disabled"})
	}
	if len(summary.StatusReasons) > 0 {
		summary.Status = "warning"
		return
	}

	summary.Status = "available"
}

func recentSuccessRate(calls []bool) float64 {
	successCount := 0
	for _, success := range calls {
		if success {
			successCount++
		}
	}
	return float64(successCount) / float64(len(calls)) * 100
}

func mergeModelTotals(totals map[string]counters, modelName string, value counters) {
	if value.requestCount == 0 {
		return
	}
	current := totals[modelName]
	current.requestCount += value.requestCount
	current.successCount += value.successCount
	current.totalLatencyMs += value.totalLatencyMs
	current.ttftSumMs += value.ttftSumMs
	current.ttftCount += value.ttftCount
	current.outputTokens += value.outputTokens
	current.generationMs += value.generationMs
	current.cacheHitTokens += value.cacheHitTokens
	current.inputTokens += value.inputTokens
	totals[modelName] = current
}

func mergeModelBucket(modelBuckets map[string]map[int64]counters, modelName string, bucketTs int64, value counters) {
	if value.requestCount == 0 {
		return
	}
	if _, ok := modelBuckets[modelName]; !ok {
		modelBuckets[modelName] = map[int64]counters{}
	}
	current := modelBuckets[modelName][bucketTs]
	current.requestCount += value.requestCount
	current.successCount += value.successCount
	current.totalLatencyMs += value.totalLatencyMs
	current.ttftSumMs += value.ttftSumMs
	current.ttftCount += value.ttftCount
	current.outputTokens += value.outputTokens
	current.generationMs += value.generationMs
	current.cacheHitTokens += value.cacheHitTokens
	current.inputTokens += value.inputTokens
	modelBuckets[modelName][bucketTs] = current
}

func recentSuccessRates(buckets map[int64]counters, limit int) []float64 {
	if len(buckets) == 0 || limit <= 0 {
		return nil
	}
	timestamps := make([]int64, 0, len(buckets))
	for ts := range buckets {
		timestamps = append(timestamps, ts)
	}
	sort.Slice(timestamps, func(i, j int) bool {
		return timestamps[i] < timestamps[j]
	})
	if len(timestamps) > limit {
		timestamps = timestamps[len(timestamps)-limit:]
	}
	rates := make([]float64, 0, len(timestamps))
	for _, ts := range timestamps {
		rates = append(rates, math.Round(successRate(buckets[ts])*100)/100)
	}
	return rates
}

func allowedGroupSet(groups []string) map[string]struct{} {
	if groups == nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		allowed[group] = struct{}{}
	}
	return allowed
}

func bucketStart(ts int64) int64 {
	bucketSeconds := perf_metrics_setting.GetBucketSeconds()
	if bucketSeconds <= 0 {
		bucketSeconds = 3600
	}
	return ts - (ts % bucketSeconds)
}

func mergeCounters(merged map[bucketKey]counters, key bucketKey, value counters) {
	if value.requestCount == 0 {
		return
	}
	current := merged[key]
	current.requestCount += value.requestCount
	current.successCount += value.successCount
	current.totalLatencyMs += value.totalLatencyMs
	current.ttftSumMs += value.ttftSumMs
	current.ttftCount += value.ttftCount
	current.outputTokens += value.outputTokens
	current.generationMs += value.generationMs
	current.cacheHitTokens += value.cacheHitTokens
	current.inputTokens += value.inputTokens
	merged[key] = current
}

func buildQueryResult(modelName string, merged map[bucketKey]counters) QueryResult {
	groupBuckets := map[string]map[int64]counters{}
	for key, value := range merged {
		if value.requestCount == 0 {
			continue
		}
		if _, ok := groupBuckets[key.group]; !ok {
			groupBuckets[key.group] = map[int64]counters{}
		}
		groupBuckets[key.group][key.bucketTs] = value
	}

	groups := make([]string, 0, len(groupBuckets))
	for group := range groupBuckets {
		groups = append(groups, group)
	}
	sort.Strings(groups)

	results := make([]GroupResult, 0, len(groups))
	for _, group := range groups {
		buckets := groupBuckets[group]
		timestamps := make([]int64, 0, len(buckets))
		for ts := range buckets {
			timestamps = append(timestamps, ts)
		}
		sort.Slice(timestamps, func(i, j int) bool {
			return timestamps[i] < timestamps[j]
		})

		total := counters{}
		series := make([]BucketPoint, 0, len(timestamps))
		for _, ts := range timestamps {
			value := buckets[ts]
			total.requestCount += value.requestCount
			total.successCount += value.successCount
			total.totalLatencyMs += value.totalLatencyMs
			total.ttftSumMs += value.ttftSumMs
			total.ttftCount += value.ttftCount
			total.outputTokens += value.outputTokens
			total.generationMs += value.generationMs
			total.cacheHitTokens += value.cacheHitTokens
			total.inputTokens += value.inputTokens
			series = append(series, bucketPoint(ts, value))
		}

		results = append(results, GroupResult{
			Group:        group,
			AvgTtftMs:    avg(total.ttftSumMs, total.ttftCount),
			AvgLatencyMs: avg(total.totalLatencyMs, total.requestCount),
			SuccessRate:  successRate(total),
			AvgTps:       avgTps(total),
			Series:       series,
		})
	}

	return QueryResult{
		ModelName:    modelName,
		SeriesSchema: seriesSchema,
		Groups:       results,
	}
}

func bucketPoint(ts int64, value counters) BucketPoint {
	return BucketPoint{
		Ts:           ts,
		AvgTtftMs:    avg(value.ttftSumMs, value.ttftCount),
		AvgLatencyMs: avg(value.totalLatencyMs, value.requestCount),
		SuccessRate:  successRate(value),
		AvgTps:       avgTps(value),
		CacheHitRate: cacheHitRate(value),
	}
}

func avg(sum int64, count int64) int64 {
	if count <= 0 {
		return 0
	}
	return sum / count
}

func successRate(value counters) float64 {
	if value.requestCount <= 0 {
		return 0
	}
	return float64(value.successCount) / float64(value.requestCount) * 100
}

func avgTps(value counters) float64 {
	if value.outputTokens <= 0 || value.generationMs <= 0 {
		return 0
	}
	return float64(value.outputTokens) / (float64(value.generationMs) / 1000)
}

func cacheHitRate(value counters) float64 {
	if value.inputTokens <= 0 {
		return 0
	}
	return float64(value.cacheHitTokens) / float64(value.inputTokens) * 100
}

func recordRedis(key bucketKey, sample Sample) {
	if !common.RedisEnabled || common.RDB == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	redisKey := redisBucketKey(key)
	groupRedisKey := redisGroupBucketKey(key.group, key.bucketTs)
	groupsRedisKey := redisActiveGroupsKey(key.bucketTs)
	pipe := common.RDB.TxPipeline()
	pipe.HIncrBy(ctx, redisKey, "req", 1)
	if sample.Success {
		pipe.HIncrBy(ctx, redisKey, "ok", 1)
	}
	if sample.LatencyMs > 0 {
		pipe.HIncrBy(ctx, redisKey, "lat", sample.LatencyMs)
	}
	if sample.HasTtft && sample.TtftMs >= 0 {
		pipe.HIncrBy(ctx, redisKey, "ttft", sample.TtftMs)
		pipe.HIncrBy(ctx, redisKey, "ttft_n", 1)
	}
	if sample.OutputTokens > 0 && sample.GenerationMs > 0 {
		pipe.HIncrBy(ctx, redisKey, "out", sample.OutputTokens)
		pipe.HIncrBy(ctx, redisKey, "gen_ms", sample.GenerationMs)
	}
	if sample.CacheHitTokens > 0 {
		pipe.HIncrBy(ctx, redisKey, "cache", sample.CacheHitTokens)
	}
	if sample.InputTokens > 0 {
		pipe.HIncrBy(ctx, redisKey, "in", sample.InputTokens)
	}
	pipe.Expire(ctx, redisKey, time.Hour)

	pipe.HIncrBy(ctx, groupRedisKey, "req", 1)
	if sample.Success {
		pipe.HIncrBy(ctx, groupRedisKey, "ok", 1)
	}
	if sample.LatencyMs > 0 {
		pipe.HIncrBy(ctx, groupRedisKey, "lat", sample.LatencyMs)
	}
	if sample.HasTtft && sample.TtftMs >= 0 {
		pipe.HIncrBy(ctx, groupRedisKey, "ttft", sample.TtftMs)
		pipe.HIncrBy(ctx, groupRedisKey, "ttft_n", 1)
	}
	if sample.OutputTokens > 0 && sample.GenerationMs > 0 {
		pipe.HIncrBy(ctx, groupRedisKey, "out", sample.OutputTokens)
		pipe.HIncrBy(ctx, groupRedisKey, "gen_ms", sample.GenerationMs)
	}
	if sample.CacheHitTokens > 0 {
		pipe.HIncrBy(ctx, groupRedisKey, "cache", sample.CacheHitTokens)
	}
	if sample.InputTokens > 0 {
		pipe.HIncrBy(ctx, groupRedisKey, "in", sample.InputTokens)
	}
	pipe.Expire(ctx, groupRedisKey, time.Hour)
	pipe.SAdd(ctx, groupsRedisKey, key.group)
	pipe.Expire(ctx, groupsRedisKey, time.Hour)
	_, _ = pipe.Exec(ctx)
}

func mergeRedisActiveGroupBuckets(groupBuckets map[string]map[int64]counters, allowedGroups map[string]struct{}, startTs int64, endTs int64) map[string]bool {
	mergedGroups := make(map[string]bool)
	if !common.RedisEnabled || common.RDB == nil {
		return mergedGroups
	}

	currentBucketTs := bucketStart(endTs)
	if currentBucketTs < startTs || currentBucketTs > endTs {
		return mergedGroups
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	groups, err := common.RDB.SMembers(ctx, redisActiveGroupsKey(currentBucketTs)).Result()
	if err != nil {
		return mergedGroups
	}

	candidates := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if allowedGroups != nil {
			if _, ok := allowedGroups[group]; !ok {
				continue
			}
		}
		candidates[group] = struct{}{}
	}
	for group := range allowedGroups {
		candidates[group] = struct{}{}
	}

	missingGroups := make(map[string]struct{})
	for group := range candidates {
		values, err := common.RDB.HGetAll(ctx, redisGroupBucketKey(group, currentBucketTs)).Result()
		if err != nil || len(values) == 0 {
			missingGroups[group] = struct{}{}
			continue
		}
		value := redisCounters(values)
		if value.requestCount == 0 {
			continue
		}
		mergeGroupBucket(groupBuckets, group, currentBucketTs, value)
		mergedGroups[group] = true
	}
	if len(missingGroups) == 0 {
		return mergedGroups
	}

	// Rebuild missing group summaries from the existing per-model bucket keys.
	// This backfills the cache during a rolling upgrade without waiting for a
	// new request to reach each group.
	recovered := make(map[string]counters, len(missingGroups))
	groupSuffixes := make(map[string]string, len(missingGroups))
	groupsToRecover := make([]string, 0, len(missingGroups))
	for group := range missingGroups {
		groupSuffixes[group] = fmt.Sprintf(":%s:%d", group, currentBucketTs)
		groupsToRecover = append(groupsToRecover, group)
	}
	sort.Slice(groupsToRecover, func(i, j int) bool {
		return len(groupsToRecover[i]) > len(groupsToRecover[j])
	})
	var cursor uint64
	for {
		keys, nextCursor, err := common.RDB.Scan(ctx, cursor, "perf:*", 1000).Result()
		if err != nil {
			return mergedGroups
		}
		for _, key := range keys {
			for _, group := range groupsToRecover {
				suffix := groupSuffixes[group]
				if !strings.HasSuffix(key, suffix) {
					continue
				}
				values, err := common.RDB.HGetAll(ctx, key).Result()
				if err != nil || len(values) == 0 {
					continue
				}
				value := redisCounters(values)
				current := recovered[group]
				current.requestCount += value.requestCount
				current.successCount += value.successCount
				current.totalLatencyMs += value.totalLatencyMs
				current.ttftSumMs += value.ttftSumMs
				current.ttftCount += value.ttftCount
				current.outputTokens += value.outputTokens
				current.generationMs += value.generationMs
				current.cacheHitTokens += value.cacheHitTokens
				current.inputTokens += value.inputTokens
				recovered[group] = current
				break
			}
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	pipe := common.RDB.TxPipeline()
	for group := range missingGroups {
		value := recovered[group]
		groupRedisKey := redisGroupBucketKey(group, currentBucketTs)
		pipe.HSet(ctx, groupRedisKey, map[string]interface{}{
			"req":    value.requestCount,
			"ok":     value.successCount,
			"lat":    value.totalLatencyMs,
			"ttft":   value.ttftSumMs,
			"ttft_n": value.ttftCount,
			"out":    value.outputTokens,
			"gen_ms": value.generationMs,
			"cache":  value.cacheHitTokens,
			"in":     value.inputTokens,
		})
		pipe.Expire(ctx, groupRedisKey, time.Hour)
		pipe.SAdd(ctx, redisActiveGroupsKey(currentBucketTs), group)
		if value.requestCount > 0 {
			mergeGroupBucket(groupBuckets, group, currentBucketTs, value)
			mergedGroups[group] = true
		}
	}
	pipe.Expire(ctx, redisActiveGroupsKey(currentBucketTs), time.Hour)
	_, _ = pipe.Exec(ctx)
	return mergedGroups
}

func mergeRedisActiveBuckets(merged map[bucketKey]counters, params QueryParams, startTs int64, endTs int64) {
	if !common.RedisEnabled || common.RDB == nil || params.Model == "" || params.Group == "" {
		return
	}
	active := bucketStart(time.Now().Unix())
	if active < startTs || active > endTs {
		return
	}
	key := bucketKey{model: params.Model, group: params.Group, bucketTs: active}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	values, err := common.RDB.HGetAll(ctx, redisBucketKey(key)).Result()
	if err != nil || len(values) == 0 {
		return
	}
	mergeCounters(merged, key, redisCounters(values))
}

func redisBucketKey(key bucketKey) string {
	return fmt.Sprintf("perf:%s:%s:%d", key.model, key.group, key.bucketTs)
}

func redisGroupBucketKey(group string, bucketTs int64) string {
	return fmt.Sprintf("perf:group:%d:%s", bucketTs, group)
}

func redisActiveGroupsKey(bucketTs int64) string {
	return fmt.Sprintf("perf:groups:%d", bucketTs)
}
