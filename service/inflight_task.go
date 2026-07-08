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
	RequestID string `json:"request_id"`
	UserID    int    `json:"user_id"`
	Status    string `json:"status"`
	Kind      string `json:"kind"`
	ModelName string `json:"model_name"`
	IsStream  bool   `json:"is_stream"`
	CreatedAt int64  `json:"created_at"`
	UpdatedAt int64  `json:"updated_at"`
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
			continue
		}
		if _, ok := statuses[log.RequestId]; !ok && log.Type == model.LogTypeError {
			statuses[log.RequestId] = InflightTaskStatusFailed
		}
	}
	return statuses, nil
}

func inflightTaskFromRelayInfo(info *relaycommon.RelayInfo, status string) *InflightTask {
	now := time.Now().Unix()
	return &InflightTask{
		RequestID: info.RequestId,
		UserID:    info.UserId,
		Status:    status,
		Kind:      inflightTaskKindFromRelayMode(info.RelayMode),
		ModelName: info.OriginModelName,
		IsStream:  info.IsStream,
		CreatedAt: now,
		UpdatedAt: now,
	}
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
					if !shouldOverwriteInflightTaskStatus(stored.Status, task.Status) {
						return nil
					}
				}
			} else if !errors.Is(getErr, redis.Nil) {
				return getErr
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

func ListUserInflightTasks(ctx context.Context, userID int) ([]InflightTask, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return nil, err
	}

	requestIDs, err := client.ZRevRange(ctx, inflightTaskUserKey(userID), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	if len(requestIDs) == 0 {
		return []InflightTask{}, nil
	}

	keys := make([]string, 0, len(requestIDs))
	for _, requestID := range requestIDs {
		keys = append(keys, inflightTaskItemKey(requestID))
	}

	values, err := client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, err
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
		if !isInflightTaskTerminalStatus(task.Status) {
			activeRequestIDs = append(activeRequestIDs, task.RequestID)
		}
		tasks = append(tasks, task)
	}

	terminalStatuses, err := loadTerminalStatusesByRequestID(userID, activeRequestIDs)
	if err != nil {
		return nil, err
	}
	reconciled := make([]*InflightTask, 0)
	for i := range tasks {
		if terminalStatus, ok := terminalStatuses[tasks[i].RequestID]; ok && shouldOverwriteInflightTaskStatus(tasks[i].Status, terminalStatus) {
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
	return tasks, nil
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
