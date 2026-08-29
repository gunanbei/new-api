package service

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"
)

func formatNotifyType(channelId int, status int) string {
	return fmt.Sprintf("%s_%d_%d", dto.NotifyTypeChannelUpdate, channelId, status)
}

// disable & notify
func DisableChannel(channelError types.ChannelError, reason string) {
	common.SysLog(fmt.Sprintf("通道「%s」（#%d）发生错误，准备禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, common.LocalLogPreview(reason)))

	// 检查是否启用自动禁用功能
	if !channelError.AutoBan {
		common.SysLog(fmt.Sprintf("通道「%s」（#%d）未启用自动禁用功能，跳过禁用操作", channelError.ChannelName, channelError.ChannelId))
		return
	}

	success := model.UpdateChannelStatus(channelError.ChannelId, channelError.UsingKey, common.ChannelStatusAutoDisabled, reason)
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelError.ChannelName, channelError.ChannelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelError.ChannelName, channelError.ChannelId, reason)
		NotifyRootUser(formatNotifyType(channelError.ChannelId, common.ChannelStatusAutoDisabled), subject, content)
	}
}

// RecordChannelFailure records every failure after a channel has been tried.
// It is intentionally independent of retry/status-code rules.
func RecordChannelFailure(channelError types.ChannelError, reason string) {
	threshold := common.AutomaticDisableFailureThreshold
	if threshold <= 0 || !common.AutomaticDisableChannelEnabled || !channelError.AutoBan {
		return
	}
	count, err := model.RecordChannelFailure(channelError.ChannelId)
	if err != nil {
		common.SysLog(fmt.Sprintf("failed to record channel failure: channel_id=%d, error=%v", channelError.ChannelId, err))
		return
	}
	if count >= threshold {
		DisableChannel(channelError, fmt.Sprintf("连续失败 %d 次：%s", count, reason))
	}
}

func ResetChannelFailures(channelId int) { model.ResetChannelFailures(channelId) }

func EnableChannel(channelId int, usingKey string, channelName string) {
	success := model.UpdateChannelStatus(channelId, usingKey, common.ChannelStatusEnabled, "")
	if success {
		subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
		NotifyRootUser(formatNotifyType(channelId, common.ChannelStatusEnabled), subject, content)
	}
}

func ShouldDisableChannel(err *types.NewAPIError, groups ...string) bool {
	if !common.AutomaticDisableChannelEnabled {
		return false
	}
	if err == nil {
		return false
	}
	if types.IsSkipRetryError(err) {
		return false
	}

	lowerMessage := strings.ToLower(err.Error())
	globalKeywordMatched, _ := AcSearch(lowerMessage, operation_setting.AutomaticDisableKeywords, true)
	if len(groups) == 0 {
		groups = []string{""}
	}
	for _, group := range groups {
		if operation_setting.ShouldDisableInGroupWithMessage(group, err.StatusCode, types.IsChannelError(err), globalKeywordMatched, lowerMessage) {
			return true
		}
	}
	return false
}

func ShouldEnableChannel(newAPIError *types.NewAPIError, status int) bool {
	if !common.AutomaticEnableChannelEnabled {
		return false
	}
	if newAPIError != nil {
		return false
	}
	if status != common.ChannelStatusAutoDisabled {
		return false
	}
	return true
}
