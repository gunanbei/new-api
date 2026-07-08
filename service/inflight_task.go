package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/go-redis/redis/v8"
)

const (
	inflightTaskItemKeyPrefix = "inflight:item:"
	inflightTaskUserKeyPrefix = "inflight:user:"

	InflightTaskStatusAccepted        = "accepted"
	InflightTaskStatusRouting         = "routing"
	InflightTaskStatusUpstreamPending = "upstream_pending"
	InflightTaskStatusStreaming       = "streaming"
	InflightTaskStatusCompleted       = "completed"
	InflightTaskStatusFailed          = "failed"

	inflightTaskCleanupRuleOptionKey            = "InflightTaskCleanupRule"
	inflightTaskCleanupIntervalMinutesOptionKey = "InflightTaskCleanupIntervalMinutes"
	InflightTaskCleanupRuleDisabled             = 1
	InflightTaskCleanupRuleTerminal             = 2
	inflightTaskCleanupRuleDefault              = InflightTaskCleanupRuleTerminal
	inflightTaskCleanupIntervalMinutesDefault   = 10
	inflightTaskCleanupIntervalMinutesMax       = 1440
	inflightTaskWriteMaxRetries                 = 3
	inflightTaskCleanupBatchSize                = 100
)

var errInflightTaskUnavailable = errors.New("功能暂不可用，请联系管理员")

const inflightTaskFinalizeTimeout = 5 * time.Second

func IsInflightTaskUnavailable(err error) bool {
	return errors.Is(err, errInflightTaskUnavailable)
}

func NewInflightTaskFinalizeContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(context.WithoutCancel(parent), inflightTaskFinalizeTimeout)
}

type InflightTask struct {
	RequestID string              `json:"request_id"`
	UserID    int                 `json:"user_id"`
	Status    string              `json:"status"`
	Kind      string              `json:"kind"`
	ModelName string              `json:"model_name"`
	IsStream  bool                `json:"is_stream"`
	CreatedAt int64               `json:"created_at"`
	UpdatedAt int64               `json:"updated_at"`
	Detail    *InflightTaskDetail `json:"detail,omitempty"`
}

type InflightTaskDetail struct {
	ChannelID    int                          `json:"channel_id,omitempty"`
	ChannelName  string                       `json:"channel_name,omitempty"`
	RetryIndex   int                          `json:"retry_index,omitempty"`
	LatestError  string                       `json:"latest_error,omitempty"`
	CurrentStage string                       `json:"current_stage,omitempty"`
	ChannelChain []InflightTaskChannelAttempt `json:"channel_chain,omitempty"`
	Timeline     []InflightTaskStatusStep     `json:"timeline,omitempty"`
}

type InflightTaskChannelAttempt struct {
	RetryIndex  int    `json:"retry_index"`
	ChannelID   int    `json:"channel_id,omitempty"`
	ChannelName string `json:"channel_name,omitempty"`
	Status      string `json:"status,omitempty"`
	Error       string `json:"error,omitempty"`
	StartedAt   int64  `json:"started_at,omitempty"`
	UpdatedAt   int64  `json:"updated_at,omitempty"`
}

type InflightTaskStatusStep struct {
	Status          string `json:"status"`
	StartedAt       int64  `json:"started_at"`
	UpdatedAt       int64  `json:"updated_at"`
	DurationSeconds int64  `json:"duration_seconds"`
}

type InflightTaskQuery struct {
	Status         string
	Kind           string
	Channel        string
	ModelName      string
	RequestID      string
	StartTimestamp int64
	EndTimestamp   int64
	IsStream       *bool
	StartIdx       int
	Num            int
}

func inflightTaskStatusRank(status string) int {
	switch status {
	case InflightTaskStatusAccepted:
		return 1
	case InflightTaskStatusRouting:
		return 2
	case InflightTaskStatusUpstreamPending:
		return 3
	case InflightTaskStatusStreaming:
		return 4
	case InflightTaskStatusCompleted, InflightTaskStatusFailed:
		return 100
	default:
		return 0
	}
}

func isInflightTaskTerminalStatus(status string) bool {
	return status == InflightTaskStatusCompleted || status == InflightTaskStatusFailed
}

func shouldOverwriteInflightTaskStatus(current string, next string) bool {
	if current == "" {
		return true
	}
	if current == next {
		return isInflightTaskTerminalStatus(next)
	}
	if isInflightTaskTerminalStatus(current) {
		return false
	}
	if isInflightTaskTerminalStatus(next) {
		return true
	}
	return inflightTaskStatusRank(next) > inflightTaskStatusRank(current)
}

func shouldReopenInflightTask(stored *InflightTask, next *InflightTask) bool {
	if stored == nil || next == nil {
		return false
	}
	if stored.Status != InflightTaskStatusFailed || isInflightTaskTerminalStatus(next.Status) {
		return false
	}
	if stored.Detail == nil || next.Detail == nil {
		return false
	}
	return next.Detail.RetryIndex > stored.Detail.RetryIndex
}

func shouldApplyReconciledTerminalStatus(task *InflightTask, terminalStatus string) bool {
	if task == nil || !isInflightTaskTerminalStatus(terminalStatus) {
		return false
	}
	if !shouldOverwriteInflightTaskStatus(task.Status, terminalStatus) {
		return false
	}
	if terminalStatus != InflightTaskStatusFailed {
		return true
	}
	if task.Detail == nil {
		return true
	}
	return task.Detail.RetryIndex >= common.RetryTimes
}

func inflightTaskRetentionTTL(status string) time.Duration {
	switch status {
	case InflightTaskStatusCompleted, InflightTaskStatusFailed:
		return 0
	default:
		return common.RateLimitKeyExpirationDuration
	}
}

func inflightTaskKindFromRelayMode(relayMode int) string {
	switch relayMode {
	case relayconstant.RelayModeImagesGenerations, relayconstant.RelayModeImagesEdits:
		return "image"
	case relayconstant.RelayModeAudioSpeech, relayconstant.RelayModeAudioTranslation, relayconstant.RelayModeAudioTranscription:
		return "audio"
	default:
		return "chat"
	}
}

func InflightTaskCleanupRule() int {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskCleanupRuleOptionKey]
	common.OptionMapRWMutex.RUnlock()

	rule, err := strconv.Atoi(raw)
	if err != nil || rule == 0 {
		return inflightTaskCleanupRuleDefault
	}
	if rule != InflightTaskCleanupRuleDisabled {
		return InflightTaskCleanupRuleTerminal
	}
	return rule
}

func InflightTaskCleanupInterval() time.Duration {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskCleanupIntervalMinutesOptionKey]
	common.OptionMapRWMutex.RUnlock()

	minutes, err := strconv.Atoi(raw)
	if err != nil || minutes <= 0 {
		minutes = inflightTaskCleanupIntervalMinutesDefault
	}
	if minutes > inflightTaskCleanupIntervalMinutesMax {
		minutes = inflightTaskCleanupIntervalMinutesMax
	}
	return time.Duration(minutes) * time.Minute
}

func inflightTaskRedis() (*redis.Client, error) {
	if !common.RedisEnabled || common.RDB == nil {
		return nil, errInflightTaskUnavailable
	}
	return common.RDB, nil
}

func inflightTaskItemKey(requestID string) string {
	return inflightTaskItemKeyPrefix + requestID
}

func inflightTaskUserKey(userID int) string {
	return inflightTaskUserKeyPrefix + strconv.Itoa(userID)
}

type inflightTaskRedisOps interface {
	TxPipeline() redis.Pipeliner
}

func persistInflightTask(ctx context.Context, ops inflightTaskRedisOps, task *InflightTask) error {
	if task == nil {
		return nil
	}
	data, err := common.Marshal(task)
	if err != nil {
		return err
	}
	userKey := inflightTaskUserKey(task.UserID)
	ttl := inflightTaskRetentionTTL(task.Status)

	pipe := ops.TxPipeline()
	pipe.Set(ctx, inflightTaskItemKey(task.RequestID), string(data), ttl)
	pipe.ZAdd(ctx, userKey, &redis.Z{
		Score:  float64(task.UpdatedAt),
		Member: task.RequestID,
	})
	if ttl > 0 {
		pipe.Expire(ctx, userKey, ttl)
	} else {
		pipe.Persist(ctx, userKey)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func loadTerminalStatusesByRequestID(userID int, requestIDs []string) (map[string]string, error) {
	statuses := make(map[string]string, len(requestIDs))
	if userID <= 0 || len(requestIDs) == 0 {
		return statuses, nil
	}

	var logs []model.Log
	err := model.LOG_DB.
		Select("request_id, type").
		Where("user_id = ? AND request_id IN ? AND type IN ?", userID, requestIDs, []int{model.LogTypeConsume, model.LogTypeError}).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	for _, log := range logs {
		if log.RequestId == "" {
			continue
		}
		if log.Type == model.LogTypeConsume {
			statuses[log.RequestId] = InflightTaskStatusCompleted
		}
	}
	return statuses, nil
}

func inflightTaskFromRelayInfo(info *relaycommon.RelayInfo, status string) *InflightTask {
	now := time.Now().Unix()
	task := &InflightTask{
		RequestID: info.RequestId,
		UserID:    info.UserId,
		Status:    status,
		Kind:      inflightTaskKindFromRelayMode(info.RelayMode),
		ModelName: info.OriginModelName,
		IsStream:  info.IsStream,
		CreatedAt: now,
		UpdatedAt: now,
	}
	task.Detail = inflightTaskDetailFromRelayInfo(info, status, now)
	return task
}

func inflightTaskDetailFromRelayInfo(info *relaycommon.RelayInfo, status string, now int64) *InflightTaskDetail {
	if info == nil {
		return nil
	}
	detail := &InflightTaskDetail{
		RetryIndex:   info.RetryIndex,
		CurrentStage: status,
	}
	if info.LastError != nil {
		detail.LatestError = info.LastError.Error()
	}
	if info.ChannelMeta != nil {
		detail.ChannelID = info.ChannelId
		detail.ChannelName = strings.TrimSpace(info.ChannelMeta.ChannelName)
		attempt := InflightTaskChannelAttempt{
			RetryIndex:  info.RetryIndex,
			ChannelID:   info.ChannelId,
			ChannelName: detail.ChannelName,
			Status:      status,
			Error:       detail.LatestError,
			StartedAt:   now,
			UpdatedAt:   now,
		}
		detail.ChannelChain = []InflightTaskChannelAttempt{attempt}
	}
	detail.Timeline = []InflightTaskStatusStep{{
		Status:          status,
		StartedAt:       now,
		UpdatedAt:       now,
		DurationSeconds: 0,
	}}
	return detail
}

func mergeInflightTaskDetail(stored *InflightTask, next *InflightTask) {
	if next == nil {
		return
	}
	now := next.UpdatedAt
	if now == 0 {
		now = time.Now().Unix()
		next.UpdatedAt = now
	}
	if next.Detail == nil {
		if stored != nil {
			next.Detail = stored.Detail
		}
		return
	}
	if stored == nil || stored.Detail == nil {
		updateInflightTaskDetail(next.Detail, next.Status, now)
		return
	}

	detail := stored.Detail
	if next.Detail.ChannelID != 0 {
		detail.ChannelID = next.Detail.ChannelID
	}
	if next.Detail.ChannelName != "" {
		detail.ChannelName = next.Detail.ChannelName
	}
	detail.RetryIndex = next.Detail.RetryIndex
	if next.Detail.LatestError != "" {
		detail.LatestError = next.Detail.LatestError
	}
	detail.CurrentStage = next.Status
	if detail.ChannelChain == nil {
		detail.ChannelChain = make([]InflightTaskChannelAttempt, 0)
	}
	if len(next.Detail.ChannelChain) > 0 {
		attempt := next.Detail.ChannelChain[0]
		lastIdx := len(detail.ChannelChain) - 1
		if lastIdx < 0 || detail.ChannelChain[lastIdx].RetryIndex != attempt.RetryIndex || detail.ChannelChain[lastIdx].ChannelID != attempt.ChannelID {
			detail.ChannelChain = append(detail.ChannelChain, attempt)
		} else {
			current := &detail.ChannelChain[lastIdx]
			if current.StartedAt == 0 {
				current.StartedAt = attempt.StartedAt
			}
			current.UpdatedAt = attempt.UpdatedAt
			if attempt.ChannelName != "" {
				current.ChannelName = attempt.ChannelName
			}
			if attempt.Status != "" {
				current.Status = attempt.Status
			}
			if attempt.Error != "" {
				current.Error = attempt.Error
			}
		}
	}
	next.Detail = detail
	updateInflightTaskDetail(next.Detail, next.Status, now)
}

func updateInflightTaskDetail(detail *InflightTaskDetail, status string, now int64) {
	if detail == nil {
		return
	}
	detail.CurrentStage = status
	if len(detail.ChannelChain) > 0 {
		lastAttempt := &detail.ChannelChain[len(detail.ChannelChain)-1]
		if lastAttempt.StartedAt == 0 {
			lastAttempt.StartedAt = now
		}
		lastAttempt.UpdatedAt = now
		if status != "" {
			lastAttempt.Status = status
		}
		if detail.LatestError != "" {
			lastAttempt.Error = detail.LatestError
		}
	}
	if len(detail.Timeline) == 0 {
		detail.Timeline = append(detail.Timeline, InflightTaskStatusStep{
			Status:          status,
			StartedAt:       now,
			UpdatedAt:       now,
			DurationSeconds: 0,
		})
		return
	}
	lastStep := &detail.Timeline[len(detail.Timeline)-1]
	if lastStep.Status == status {
		lastStep.UpdatedAt = now
		lastStep.DurationSeconds = maxInt64(0, now-lastStep.StartedAt)
		return
	}
	lastStep.UpdatedAt = now
	lastStep.DurationSeconds = maxInt64(0, now-lastStep.StartedAt)
	detail.Timeline = append(detail.Timeline, InflightTaskStatusStep{
		Status:          status,
		StartedAt:       now,
		UpdatedAt:       now,
		DurationSeconds: 0,
	})
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func inflightTaskMatchesChannel(task InflightTask, channel string) bool {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return true
	}
	if task.Detail == nil {
		return false
	}
	if strings.Contains(strconv.Itoa(task.Detail.ChannelID), channel) ||
		strings.Contains(strings.ToLower(task.Detail.ChannelName), strings.ToLower(channel)) {
		return true
	}
	for _, attempt := range task.Detail.ChannelChain {
		if strings.Contains(strconv.Itoa(attempt.ChannelID), channel) ||
			strings.Contains(strings.ToLower(attempt.ChannelName), strings.ToLower(channel)) {
			return true
		}
	}
	return false
}

func updateInflightTask(ctx context.Context, client *redis.Client, task *InflightTask) error {
	itemKey := inflightTaskItemKey(task.RequestID)
	for i := 0; i < inflightTaskWriteMaxRetries; i++ {
		err := client.Watch(ctx, func(tx *redis.Tx) error {
			existing, getErr := tx.Get(ctx, itemKey).Result()
			if getErr == nil {
				var stored InflightTask
				if err := common.UnmarshalJsonStr(existing, &stored); err == nil {
					task.CreatedAt = stored.CreatedAt
					mergeInflightTaskDetail(&stored, task)
					if !shouldOverwriteInflightTaskStatus(stored.Status, task.Status) && !shouldReopenInflightTask(&stored, task) {
						return nil
					}
				}
			} else if !errors.Is(getErr, redis.Nil) {
				return getErr
			} else {
				mergeInflightTaskDetail(nil, task)
			}
			return persistInflightTask(ctx, tx, task)
		}, itemKey)
		if !errors.Is(err, redis.TxFailedErr) {
			return err
		}
	}
	return redis.TxFailedErr
}

func UpdateInflightTaskStatus(c context.Context, info *relaycommon.RelayInfo, status string) error {
	if info == nil || info.UserId <= 0 || info.RequestId == "" {
		return nil
	}
	client, err := inflightTaskRedis()
	if err != nil {
		return err
	}
	return updateInflightTask(c, client, inflightTaskFromRelayInfo(info, status))
}

func UpdateInflightTaskStatusAsync(parent context.Context, info *relaycommon.RelayInfo, status string) {
	if info == nil || info.UserId <= 0 || info.RequestId == "" {
		return
	}
	task := inflightTaskFromRelayInfo(info, status)
	gopool.Go(func() {
		ctx, cancel := NewInflightTaskFinalizeContext(parent)
		defer cancel()

		client, err := inflightTaskRedis()
		if err != nil {
			if !errors.Is(err, errInflightTaskUnavailable) {
				common.SysError(fmt.Sprintf("update inflight log failed: %v", err))
			}
			return
		}
		if err = updateInflightTask(ctx, client, task); err != nil && !errors.Is(err, errInflightTaskUnavailable) {
			common.SysError(fmt.Sprintf("update inflight log failed: %v", err))
		}
	})
}

func MarkInflightTaskStreamStarted(c context.Context, info *relaycommon.RelayInfo) {
	if info == nil || !info.IsStream {
		return
	}
	UpdateInflightTaskStatusAsync(c, info, InflightTaskStatusStreaming)
}

func FinalInflightTaskStatus(info *relaycommon.RelayInfo) string {
	if info != nil && info.IsStream && info.StreamStatus != nil && (!info.StreamStatus.IsNormalEnd() || info.StreamStatus.HasErrors()) {
		return InflightTaskStatusFailed
	}
	return InflightTaskStatusCompleted
}

func ListUserInflightTasks(ctx context.Context, userID int, query InflightTaskQuery) ([]InflightTask, int, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return nil, 0, err
	}

	requestIDs, err := client.ZRevRange(ctx, inflightTaskUserKey(userID), 0, -1).Result()
	if err != nil {
		return nil, 0, err
	}
	if len(requestIDs) == 0 {
		return []InflightTask{}, 0, nil
	}

	keys := make([]string, 0, len(requestIDs))
	for _, requestID := range requestIDs {
		keys = append(keys, inflightTaskItemKey(requestID))
	}

	values, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, 0, err
	}

	tasks := make([]InflightTask, 0, len(values))
	missing := make([]interface{}, 0)
	activeRequestIDs := make([]string, 0, len(values))
	for i, value := range values {
		if value == nil {
			missing = append(missing, requestIDs[i])
			continue
		}
		raw, ok := value.(string)
		if !ok {
			continue
		}
		var task InflightTask
		if err = common.UnmarshalJsonStr(raw, &task); err != nil {
			continue
		}
		if task.UserID != userID {
			continue
		}
		if query.Status != "" && task.Status != query.Status {
			continue
		}
		if query.Kind != "" && task.Kind != query.Kind {
			continue
		}
		if !inflightTaskMatchesChannel(task, query.Channel) {
			continue
		}
		if query.ModelName != "" && !strings.Contains(task.ModelName, query.ModelName) {
			continue
		}
		if query.RequestID != "" && !strings.Contains(task.RequestID, query.RequestID) {
			continue
		}
		if query.StartTimestamp != 0 && task.CreatedAt < query.StartTimestamp {
			continue
		}
		if query.EndTimestamp != 0 && task.CreatedAt > query.EndTimestamp {
			continue
		}
		if query.IsStream != nil && task.IsStream != *query.IsStream {
			continue
		}
		if !isInflightTaskTerminalStatus(task.Status) {
			activeRequestIDs = append(activeRequestIDs, task.RequestID)
		}
		tasks = append(tasks, task)
	}

	terminalStatuses, err := loadTerminalStatusesByRequestID(userID, activeRequestIDs)
	if err != nil {
		return nil, 0, err
	}
	reconciled := make([]*InflightTask, 0)
	for i := range tasks {
		if terminalStatus, ok := terminalStatuses[tasks[i].RequestID]; ok && shouldApplyReconciledTerminalStatus(&tasks[i], terminalStatus) {
			tasks[i].Status = terminalStatus
			tasks[i].UpdatedAt = time.Now().Unix()
			reconciled = append(reconciled, &tasks[i])
		}
	}

	for _, task := range reconciled {
		_ = persistInflightTask(ctx, client, task)
	}

	if len(missing) > 0 {
		_, _ = client.ZRem(ctx, inflightTaskUserKey(userID), missing...).Result()
	}

	sort.Slice(tasks, func(i, j int) bool {
		if tasks[i].UpdatedAt == tasks[j].UpdatedAt {
			return tasks[i].CreatedAt > tasks[j].CreatedAt
		}
		return tasks[i].UpdatedAt > tasks[j].UpdatedAt
	})
	total := len(tasks)
	if query.StartIdx >= total {
		return []InflightTask{}, total, nil
	}
	end := query.StartIdx + query.Num
	if query.Num <= 0 || end > total {
		end = total
	}
	return tasks[query.StartIdx:end], total, nil
}

type InflightTaskStats struct {
	UserCount int64 `json:"user_count"`
	ItemCount int64 `json:"item_count"`
	TotalSize int64 `json:"total_size"`
}

func GetInflightTaskStats(ctx context.Context) (InflightTaskStats, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return InflightTaskStats{}, err
	}
	var stats InflightTaskStats
	var cursor uint64
	for {
		keys, next, err := client.Scan(ctx, cursor, "inflight:*", 100).Result()
		if err != nil {
			return stats, err
		}
		cursor = next
		for _, key := range keys {
			if strings.HasPrefix(key, inflightTaskUserKeyPrefix) {
				stats.UserCount++
			} else if strings.HasPrefix(key, inflightTaskItemKeyPrefix) {
				stats.ItemCount++
			}
			size, err := client.MemoryUsage(ctx, key).Result()
			if err == nil {
				stats.TotalSize += size
			}
		}
		if cursor == 0 {
			return stats, nil
		}
	}
}

func DeleteTerminalInflightTasksBefore(ctx context.Context, targetTimestamp int64) (int64, error) {
	if targetTimestamp <= 0 {
		return 0, errors.New("target timestamp is required")
	}
	client, err := inflightTaskRedis()
	if err != nil {
		return 0, err
	}

	var deleted int64
	var cursor uint64
	for {
		userKeys, next, err := client.Scan(ctx, cursor, inflightTaskUserKeyPrefix+"*", 100).Result()
		if err != nil {
			return deleted, err
		}
		cursor = next
		for _, userKey := range userKeys {
			var offset int64
			for {
				requestIDs, err := client.ZRangeByScore(ctx, userKey, &redis.ZRangeBy{
					Min:    "0",
					Max:    strconv.FormatInt(targetTimestamp, 10),
					Offset: offset,
					Count:  inflightTaskCleanupBatchSize,
				}).Result()
				if err != nil {
					return deleted, err
				}
				if len(requestIDs) == 0 {
					break
				}

				keys := make([]string, 0, len(requestIDs))
				for _, requestID := range requestIDs {
					keys = append(keys, inflightTaskItemKey(requestID))
				}
				values, err := client.MGet(ctx, keys...).Result()
				if err != nil {
					return deleted, err
				}

				removeIDs := make([]interface{}, 0, len(requestIDs))
				removeKeys := make([]string, 0, len(requestIDs))
				for i, value := range values {
					if value == nil {
						removeIDs = append(removeIDs, requestIDs[i])
						continue
					}
					raw, ok := value.(string)
					if !ok {
						continue
					}
					var task InflightTask
					if err := common.UnmarshalJsonStr(raw, &task); err != nil {
						continue
					}
					if isInflightTaskTerminalStatus(task.Status) && task.UpdatedAt <= targetTimestamp {
						removeIDs = append(removeIDs, requestIDs[i])
						removeKeys = append(removeKeys, keys[i])
					}
				}
				if len(removeIDs) == 0 {
					offset += int64(len(requestIDs))
					if len(requestIDs) < inflightTaskCleanupBatchSize {
						break
					}
					continue
				}

				pipe := client.TxPipeline()
				pipe.ZRem(ctx, userKey, removeIDs...)
				if len(removeKeys) > 0 {
					pipe.Del(ctx, removeKeys...)
				}
				if _, err := pipe.Exec(ctx); err != nil {
					return deleted, err
				}
				deleted += int64(len(removeKeys))
				offset += int64(len(requestIDs) - len(removeIDs))
				if len(requestIDs) < inflightTaskCleanupBatchSize {
					break
				}
			}
		}
		if cursor == 0 {
			return deleted, nil
		}
	}
}
