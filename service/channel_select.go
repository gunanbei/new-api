package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
)

type RetryParam struct {
	Ctx          *gin.Context
	TokenGroup   string
	ModelName    string
	RequestPath  string
	Retry        *int
	resetNextTry bool
}

func (p *RetryParam) GetRetry() int {
	if p.Retry == nil {
		return 0
	}
	return *p.Retry
}

func (p *RetryParam) SetRetry(retry int) {
	p.Retry = &retry
}

func (p *RetryParam) IncreaseRetry() {
	if p.resetNextTry {
		p.resetNextTry = false
		return
	}
	if p.Retry == nil {
		p.Retry = new(int)
	}
	*p.Retry++
}

func (p *RetryParam) ResetRetryNextTry() {
	p.resetNextTry = true
}

// CacheGetRandomSatisfiedChannel tries to get a random channel that satisfies the requirements.
// 尝试获取一个满足要求的随机渠道。
//
// For "auto" tokenGroup with cross-group Retry enabled:
// 对于启用了跨分组重试的 "auto" tokenGroup：
//
//   - Each group will exhaust all its priorities before moving to the next group.
//     每个分组会用完所有优先级后才会切换到下一个分组。
//
//   - Uses ContextKeyAutoGroupIndex to track current group index.
//     使用 ContextKeyAutoGroupIndex 跟踪当前分组索引。
//
//   - Uses ContextKeyAutoGroupRetryIndex to track the global Retry count when current group started.
//     使用 ContextKeyAutoGroupRetryIndex 跟踪当前分组开始时的全局重试次数。
//
//   - priorityRetry = Retry - startRetryIndex, represents the priority level within current group.
//     priorityRetry = Retry - startRetryIndex，表示当前分组内的优先级级别。
//
//   - When GetRandomSatisfiedChannel returns nil (priorities exhausted), moves to next group.
//     当 GetRandomSatisfiedChannel 返回 nil（优先级用完）时，切换到下一个分组。
//
// Example flow (2 groups, each with 2 priorities, RetryTimes=3):
// 示例流程（2个分组，每个有2个优先级，RetryTimes=3）：
//
//	Retry=0: GroupA, priority0 (startRetryIndex=0, priorityRetry=0)
//	         分组A, 优先级0
//
//	Retry=1: GroupA, priority1 (startRetryIndex=0, priorityRetry=1)
//	         分组A, 优先级1
//
//	Retry=2: GroupA exhausted → GroupB, priority0 (startRetryIndex=2, priorityRetry=0)
//	         分组A用完 → 分组B, 优先级0
//
//	Retry=3: GroupB, priority1 (startRetryIndex=2, priorityRetry=1)
//	         分组B, 优先级1
func CacheGetRandomSatisfiedChannel(param *RetryParam) (*model.Channel, string, error) {
	var channel *model.Channel
	var err error
	selectGroup := param.TokenGroup
	userGroup := common.GetContextKeyString(param.Ctx, constant.ContextKeyUserGroup)

	if failoverGroups := GetFailoverGroups(param.Ctx); len(failoverGroups) > 0 {
		return selectFailoverChannel(param, failoverGroups)
	}

	if param.TokenGroup == "auto" {
		if len(setting.GetAutoGroups()) == 0 {
			return nil, selectGroup, errors.New("auto groups is not enabled")
		}
		autoGroups := GetUserAutoGroup(userGroup)

		// startGroupIndex: the group index to start searching from
		// startGroupIndex: 开始搜索的分组索引
		startGroupIndex := 0
		crossGroupRetry := common.GetContextKeyBool(param.Ctx, constant.ContextKeyTokenCrossGroupRetry)

		if lastGroupIndex, exists := common.GetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex); exists {
			if idx, ok := lastGroupIndex.(int); ok {
				startGroupIndex = idx
			}
		}

		for i := startGroupIndex; i < len(autoGroups); i++ {
			autoGroup := autoGroups[i]
			// Calculate priorityRetry for current group
			// 计算当前分组的 priorityRetry
			priorityRetry := param.GetRetry()
			// If moved to a new group, reset priorityRetry and update startRetryIndex
			// 如果切换到新分组，重置 priorityRetry 并更新 startRetryIndex
			if i > startGroupIndex {
				priorityRetry = 0
			}
			logger.LogDebug(param.Ctx, "Auto selecting group: %s, priorityRetry: %d", autoGroup, priorityRetry)

			channel, _ = model.GetRandomSatisfiedChannel(autoGroup, param.ModelName, priorityRetry, param.RequestPath)
			if channel == nil {
				// Current group has no available channel for this model, try next group
				// 当前分组没有该模型的可用渠道，尝试下一个分组
				logger.LogDebug(param.Ctx, "No available channel in group %s for model %s at priorityRetry %d, trying next group", autoGroup, param.ModelName, priorityRetry)
				// 重置状态以尝试下一个分组
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupRetryIndex, 0)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				continue
			}
			common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, autoGroup)
			selectGroup = autoGroup
			logger.LogDebug(param.Ctx, "Auto selected group: %s", autoGroup)

			// Prepare state for next retry
			// 为下一次重试准备状态
			if crossGroupRetry && priorityRetry >= common.RetryTimes {
				// Current group has exhausted all retries, prepare to switch to next group
				// This request still uses current group, but next retry will use next group
				// 当前分组已用完所有重试次数，准备切换到下一个分组
				// 本次请求仍使用当前分组，但下次重试将使用下一个分组
				logger.LogDebug(param.Ctx, "Current group %s retries exhausted (priorityRetry=%d >= RetryTimes=%d), preparing switch to next group for next retry", autoGroup, priorityRetry, common.RetryTimes)
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i+1)
				// Reset retry counter so outer loop can continue for next group
				// 重置重试计数器，以便外层循环可以为下一个分组继续
				param.SetRetry(0)
				param.ResetRetryNextTry()
			} else {
				// Stay in current group, save current state
				// 保持在当前分组，保存当前状态
				common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroupIndex, i)
			}
			break
		}
	} else {
		channel, err = model.GetRandomSatisfiedChannel(param.TokenGroup, param.ModelName, param.GetRetry(), param.RequestPath)
		if err != nil {
			return nil, param.TokenGroup, err
		}
	}
	return channel, selectGroup, nil
}

// failoverCursor walks a token's failover group sequence. Its position lives in the
// request context so every retry resumes where the previous attempt stopped.
type failoverCursor struct {
	Groups     []string
	GroupIndex int
	StartRetry int
}

// nextTarget returns the group and the in-group priority level to use for the given
// global retry index. Groups that have nothing left to offer are skipped without
// consuming a retry. ok is false once the whole sequence is exhausted.
func (cursor *failoverCursor) nextTarget(retry int, levelsOf func(group string) int) (string, int, bool) {
	for cursor.GroupIndex < len(cursor.Groups) {
		group := cursor.Groups[cursor.GroupIndex]
		priorityRetry := retry - cursor.StartRetry
		if levels := levelsOf(group); levels > 0 && priorityRetry < levels {
			return group, priorityRetry, true
		}
		cursor.skip(retry)
	}
	return "", 0, false
}

// skip abandons the current group and restarts the priority counter on the next one.
func (cursor *failoverCursor) skip(retry int) {
	cursor.GroupIndex++
	cursor.StartRetry = retry
}

// selectFailoverChannel serves a token in failover mode: it exhausts the channel
// priority levels of the current group, then moves on to the next group in the
// sequence. Unlike the auto cross-group path it never resets the caller's retry
// counter, so the total number of attempts stays bounded by EffectiveRetryTimes.
func selectFailoverChannel(param *RetryParam, groups []string) (*model.Channel, string, error) {
	cursor := &failoverCursor{
		Groups:     groups,
		GroupIndex: common.GetContextKeyInt(param.Ctx, constant.ContextKeyFailoverGroupIndex),
		StartRetry: common.GetContextKeyInt(param.Ctx, constant.ContextKeyFailoverGroupStartRetry),
	}
	levelsOf := func(group string) int {
		return model.CountPriorityLevels(group, param.ModelName, param.RequestPath)
	}

	defer func() {
		common.SetContextKey(param.Ctx, constant.ContextKeyFailoverGroupIndex, cursor.GroupIndex)
		common.SetContextKey(param.Ctx, constant.ContextKeyFailoverGroupStartRetry, cursor.StartRetry)
	}()

	for {
		group, priorityRetry, ok := cursor.nextTarget(param.GetRetry(), levelsOf)
		if !ok {
			return nil, strings.Join(groups, ","), nil
		}
		channel, err := model.GetRandomSatisfiedChannel(group, param.ModelName, priorityRetry, param.RequestPath)
		if err != nil || channel == nil {
			// The database selection path cannot pre-filter Advanced Custom channels by
			// request path, so a level counted as available can still yield nothing.
			// Give up on this group rather than the whole request.
			if err != nil {
				logger.LogWarn(param.Ctx, fmt.Sprintf("failover: group %s failed to provide a channel for model %s: %s", group, param.ModelName, err.Error()))
			}
			cursor.skip(param.GetRetry())
			continue
		}
		logger.LogDebug(param.Ctx, "Failover selected group: %s, priorityRetry: %d", group, priorityRetry)
		common.SetContextKey(param.Ctx, constant.ContextKeyAutoGroup, group)
		return channel, group, nil
	}
}
