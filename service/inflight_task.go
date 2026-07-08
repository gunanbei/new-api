package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"

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

	inflightTaskUserLimitOptionKey = "InflightTaskUserLimit"
	inflightTaskUserLimitDefault   = 30
	inflightTaskUserLimitMax       = 200
)

var errInflightTaskUnavailable = errors.New("功能暂不可用，请联系管理员")

func IsInflightTaskUnavailable(err error) bool {
	return errors.Is(err, errInflightTaskUnavailable)
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

func inflightTaskRetentionTTL(status string) time.Duration {
	switch status {
	case InflightTaskStatusCompleted, InflightTaskStatusFailed:
		return 2 * time.Minute
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

func inflightTaskUserLimit() int64 {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskUserLimitOptionKey]
	common.OptionMapRWMutex.RUnlock()

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		return inflightTaskUserLimitDefault
	}
	if limit > inflightTaskUserLimitMax {
		return inflightTaskUserLimitMax
	}
	return int64(limit)
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

func UpdateInflightTaskStatus(c context.Context, info *relaycommon.RelayInfo, status string) error {
	if info == nil || info.UserId <= 0 || info.RequestId == "" {
		return nil
	}
	client, err := inflightTaskRedis()
	if err != nil {
		return err
	}
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
	itemKey := inflightTaskItemKey(info.RequestId)
	userKey := inflightTaskUserKey(info.UserId)

	existing, getErr := client.Get(c, itemKey).Result()
	if getErr == nil {
		var stored InflightTask
		if err = common.UnmarshalJsonStr(existing, &stored); err == nil {
			task.CreatedAt = stored.CreatedAt
		}
	} else if !errors.Is(getErr, redis.Nil) {
		return getErr
	}

	data, err := common.Marshal(task)
	if err != nil {
		return err
	}

	ttl := inflightTaskRetentionTTL(status)
	limit := inflightTaskUserLimit()
	overflowIDs, rangeErr := client.ZRange(c, userKey, 0, -limit-1).Result()
	if rangeErr != nil {
		return rangeErr
	}

	pipe := client.TxPipeline()
	pipe.Set(c, itemKey, string(data), ttl)
	pipe.ZAdd(c, userKey, &redis.Z{
		Score:  float64(task.UpdatedAt),
		Member: task.RequestID,
	})
	pipe.Expire(c, userKey, ttl)
	for _, overflowID := range overflowIDs {
		pipe.Del(c, inflightTaskItemKey(overflowID))
		pipe.ZRem(c, userKey, overflowID)
	}
	_, err = pipe.Exec(c)
	if err != nil {
		return err
	}
	return nil
}

func MarkInflightTaskStreamStarted(c context.Context, info *relaycommon.RelayInfo) {
	if info == nil || !info.IsStream {
		return
	}
	if err := UpdateInflightTaskStatus(c, info, InflightTaskStatusStreaming); err != nil && !errors.Is(err, errInflightTaskUnavailable) {
		common.SysError(fmt.Sprintf("update inflight task streaming status failed: %v", err))
	}
}

func ListUserInflightTasks(ctx context.Context, userID int) ([]InflightTask, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return nil, err
	}

	requestIDs, err := client.ZRevRange(ctx, inflightTaskUserKey(userID), 0, inflightTaskUserLimit()-1).Result()
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
		tasks = append(tasks, task)
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
