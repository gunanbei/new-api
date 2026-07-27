package common

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/QuantumNous/new-api/types"
)

type FailoverState string

const (
	FailoverStateIdle              FailoverState = "idle"
	FailoverStateSelecting         FailoverState = "selecting"
	FailoverStateAttempting        FailoverState = "attempting"
	FailoverStateResponseCommitted FailoverState = "response_committed"
	FailoverStateRollingBack       FailoverState = "rolling_back"
	FailoverStateRolledBack        FailoverState = "rolled_back"
	FailoverStateReadyToRetry      FailoverState = "ready_to_retry"
	FailoverStateSucceeded         FailoverState = "succeeded"
	FailoverStateFailed            FailoverState = "failed"
	FailoverStateCanceled          FailoverState = "canceled"
)

type FailoverAuditEvent struct {
	Sequence          int             `json:"sequence"`
	TimestampMS       int64           `json:"timestamp_ms"`
	RetryIndex        int             `json:"retry_index"`
	From              FailoverState   `json:"from"`
	To                FailoverState   `json:"to"`
	Event             string          `json:"event"`
	Trigger           string          `json:"trigger,omitempty"`
	Reason            string          `json:"reason,omitempty"`
	ChannelID         int             `json:"channel_id,omitempty"`
	ChannelName       string          `json:"channel_name,omitempty"`
	Group             string          `json:"group,omitempty"`
	PreviousChannelID int             `json:"previous_channel_id,omitempty"`
	PreviousGroup     string          `json:"previous_group,omitempty"`
	ErrorCode         types.ErrorCode `json:"error_code,omitempty"`
	StatusCode        int             `json:"status_code,omitempty"`
	Error             string          `json:"error,omitempty"`
	Rollback          string          `json:"rollback,omitempty"`
}

type FailoverAuditTrail struct {
	State        FailoverState            `json:"state"`
	RulesEnabled bool                     `json:"rules_enabled"`
	Rules        types.TokenFailoverRules `json:"rules"`
	Events       []FailoverAuditEvent     `json:"events"`
}

type failoverAttemptSnapshot struct {
	attemptResponse           *AttemptResponseWriter
	channelMeta               *ChannelMeta
	isStream                  bool
	sendResponseCount         int
	receivedResponseCount     int
	streamStatus              *StreamStatus
	firstResponseTime         time.Time
	isFirstResponse           bool
	thinkingContentInfo       ThinkingContentInfo
	claudeConvertInfo         *ClaudeConvertInfo
	responsesToolCounts       map[string]int
	runtimeHeadersOverride    map[string]interface{}
	useRuntimeHeadersOverride bool
	paramOverrideAudit        []string
	upstreamRequestBodySize   int64
	requestConversionChain    []types.RelayFormat
	finalRequestRelayFormat   types.RelayFormat
}

type FailoverStateMachine struct {
	state             FailoverState
	rules             types.TokenFailoverRules
	events            []FailoverAuditEvent
	snapshot          *failoverAttemptSnapshot
	previousChannelID int
	previousGroup     string
}

func NewFailoverStateMachine(rules types.TokenFailoverRules) *FailoverStateMachine {
	return &FailoverStateMachine{state: FailoverStateIdle, rules: rules}
}

func (m *FailoverStateMachine) State() FailoverState {
	if m == nil {
		return FailoverStateIdle
	}
	return m.state
}

func (m *FailoverStateMachine) Trail() FailoverAuditTrail {
	if m == nil {
		return FailoverAuditTrail{State: FailoverStateIdle}
	}
	events := append([]FailoverAuditEvent(nil), m.events...)
	return FailoverAuditTrail{State: m.state, RulesEnabled: m.rules.Enabled, Rules: m.rules, Events: events}
}

func MergeFailoverAuditEvents(current, incoming []FailoverAuditEvent) []FailoverAuditEvent {
	bySequence := make(map[int]FailoverAuditEvent, len(current)+len(incoming))
	for _, event := range current {
		bySequence[event.Sequence] = event
	}
	for _, event := range incoming {
		bySequence[event.Sequence] = event
	}
	merged := make([]FailoverAuditEvent, 0, len(bySequence))
	for _, event := range bySequence {
		merged = append(merged, event)
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].Sequence < merged[j].Sequence })
	return merged
}

func (m *FailoverStateMachine) BeginSelection(info *RelayInfo) error {
	return m.transition(info, FailoverStateSelecting, "selection_started", "", "", nil, "")
}

func (m *FailoverStateMachine) StartAttempt(info *RelayInfo, writer *AttemptResponseWriter) error {
	if info == nil {
		return errors.New("failover state machine requires relay info")
	}
	if m.state != FailoverStateSelecting {
		return m.invalid(FailoverStateAttempting)
	}
	m.snapshot = snapshotFailoverAttempt(info)
	info.ResetFailoverAttempt(writer)
	return m.transition(info, FailoverStateAttempting, "attempt_started", "", "", nil, "")
}

func (m *FailoverStateMachine) MarkResponseCommitted(info *RelayInfo) error {
	if m.state == FailoverStateResponseCommitted {
		return nil
	}
	return m.transition(info, FailoverStateResponseCommitted, "response_committed", "response_committed", "downstream response is no longer reversible", nil, "")
}

func (m *FailoverStateMachine) CompleteAttempt(info *RelayInfo) error {
	if err := m.transition(info, FailoverStateSucceeded, "attempt_succeeded", "", "", nil, ""); err != nil {
		return err
	}
	m.snapshot = nil
	return nil
}

func (m *FailoverStateMachine) RollbackAttempt(info *RelayInfo, writer *AttemptResponseWriter, apiErr *types.NewAPIError, trigger string) error {
	if writer != nil && writer.Committed() {
		return errors.New("cannot roll back a committed downstream response")
	}
	if err := m.transition(info, FailoverStateRollingBack, "rollback_started", trigger, "", apiErr, "started"); err != nil {
		return err
	}
	if writer != nil {
		writer.Discard()
		if writer.Committed() || writer.Size() != 0 {
			_ = m.transition(info, FailoverStateFailed, "rollback_failed", trigger, "response buffer could not be discarded", apiErr, "failed")
			return errors.New("failover response rollback verification failed")
		}
	}
	if m.snapshot == nil {
		_ = m.transition(info, FailoverStateFailed, "rollback_failed", trigger, "attempt snapshot is missing", apiErr, "failed")
		return errors.New("failover attempt snapshot is missing")
	}
	m.snapshot.restore(info)
	m.snapshot = nil
	return m.transition(info, FailoverStateRolledBack, "rollback_completed", trigger, "attempt state restored", apiErr, "completed")
}

func (m *FailoverStateMachine) SelectRetry(info *RelayInfo, trigger, reason string) error {
	if err := m.transition(info, FailoverStateReadyToRetry, "retry_selected", trigger, reason, nil, "completed"); err != nil {
		return err
	}
	if info != nil && info.ChannelMeta != nil {
		m.previousChannelID = info.ChannelId
		m.previousGroup = info.UsingGroup
	}
	return nil
}

func (m *FailoverStateMachine) RejectRetry(info *RelayInfo, apiErr *types.NewAPIError, trigger, reason string, canceled bool) error {
	target := FailoverStateFailed
	event := "retry_rejected"
	if canceled {
		target = FailoverStateCanceled
		event = "client_canceled"
	}
	if err := m.transition(info, target, event, trigger, reason, apiErr, "completed"); err != nil {
		return err
	}
	m.snapshot = nil
	return nil
}

func (m *FailoverStateMachine) FailSelection(info *RelayInfo, apiErr *types.NewAPIError, reason string) error {
	return m.transition(info, FailoverStateFailed, "selection_failed", "channel_selection", reason, apiErr, "")
}

func (m *FailoverStateMachine) transition(info *RelayInfo, target FailoverState, event, trigger, reason string, apiErr *types.NewAPIError, rollback string) error {
	if !validFailoverTransition(m.state, target) {
		return m.invalid(target)
	}
	audit := FailoverAuditEvent{
		Sequence:          len(m.events) + 1,
		TimestampMS:       time.Now().UnixMilli(),
		From:              m.state,
		To:                target,
		Event:             event,
		Trigger:           trigger,
		Reason:            reason,
		PreviousChannelID: m.previousChannelID,
		PreviousGroup:     m.previousGroup,
		Rollback:          rollback,
	}
	if info != nil {
		audit.RetryIndex = info.RetryIndex
		audit.Group = info.UsingGroup
		if info.ChannelMeta != nil {
			audit.ChannelID = info.ChannelId
			audit.ChannelName = info.ChannelName
		}
	}
	if apiErr != nil {
		audit.ErrorCode = apiErr.GetErrorCode()
		audit.StatusCode = apiErr.StatusCode
		audit.Error = apiErr.MaskSensitiveError()
	}
	m.state = target
	m.events = append(m.events, audit)
	return nil
}

func (m *FailoverStateMachine) invalid(target FailoverState) error {
	return fmt.Errorf("invalid failover state transition: %s -> %s", m.state, target)
}

func validFailoverTransition(from, to FailoverState) bool {
	switch from {
	case FailoverStateIdle, FailoverStateReadyToRetry:
		return to == FailoverStateSelecting
	case FailoverStateSelecting:
		return to == FailoverStateAttempting || to == FailoverStateFailed || to == FailoverStateCanceled
	case FailoverStateAttempting:
		return to == FailoverStateResponseCommitted || to == FailoverStateRollingBack || to == FailoverStateSucceeded || to == FailoverStateFailed || to == FailoverStateCanceled
	case FailoverStateResponseCommitted:
		return to == FailoverStateSucceeded || to == FailoverStateFailed || to == FailoverStateCanceled
	case FailoverStateRollingBack:
		return to == FailoverStateRolledBack || to == FailoverStateFailed
	case FailoverStateRolledBack:
		return to == FailoverStateReadyToRetry || to == FailoverStateFailed || to == FailoverStateCanceled
	default:
		return false
	}
}

func snapshotFailoverAttempt(info *RelayInfo) *failoverAttemptSnapshot {
	snapshot := &failoverAttemptSnapshot{
		attemptResponse:           info.AttemptResponse,
		isStream:                  info.IsStream,
		sendResponseCount:         info.SendResponseCount,
		receivedResponseCount:     info.ReceivedResponseCount,
		streamStatus:              info.StreamStatus,
		firstResponseTime:         info.FirstResponseTime,
		isFirstResponse:           info.isFirstResponse,
		thinkingContentInfo:       info.ThinkingContentInfo,
		useRuntimeHeadersOverride: info.UseRuntimeHeadersOverride,
		paramOverrideAudit:        append([]string(nil), info.ParamOverrideAudit...),
		upstreamRequestBodySize:   info.UpstreamRequestBodySize,
		requestConversionChain:    append([]types.RelayFormat(nil), info.RequestConversionChain...),
		finalRequestRelayFormat:   info.FinalRequestRelayFormat,
	}
	if info.ChannelMeta != nil {
		channelMeta := *info.ChannelMeta
		channelMeta.ParamOverride = cloneInterfaceMap(info.ChannelMeta.ParamOverride)
		channelMeta.HeadersOverride = cloneInterfaceMap(info.ChannelMeta.HeadersOverride)
		snapshot.channelMeta = &channelMeta
	}
	if info.ClaudeConvertInfo != nil {
		copy := *info.ClaudeConvertInfo
		snapshot.claudeConvertInfo = &copy
	}
	if info.ResponsesUsageInfo != nil {
		snapshot.responsesToolCounts = make(map[string]int, len(info.ResponsesUsageInfo.BuiltInTools))
		for name, tool := range info.ResponsesUsageInfo.BuiltInTools {
			if tool != nil {
				snapshot.responsesToolCounts[name] = tool.CallCount
			}
		}
	}
	if info.RuntimeHeadersOverride != nil {
		snapshot.runtimeHeadersOverride = make(map[string]interface{}, len(info.RuntimeHeadersOverride))
		for key, value := range info.RuntimeHeadersOverride {
			snapshot.runtimeHeadersOverride[key] = value
		}
	}
	return snapshot
}

func (snapshot *failoverAttemptSnapshot) restore(info *RelayInfo) {
	info.AttemptResponse = snapshot.attemptResponse
	if snapshot.channelMeta == nil {
		info.ChannelMeta = nil
	} else {
		channelMeta := *snapshot.channelMeta
		channelMeta.ParamOverride = cloneInterfaceMap(snapshot.channelMeta.ParamOverride)
		channelMeta.HeadersOverride = cloneInterfaceMap(snapshot.channelMeta.HeadersOverride)
		info.ChannelMeta = &channelMeta
	}
	info.IsStream = snapshot.isStream
	info.SendResponseCount = snapshot.sendResponseCount
	info.ReceivedResponseCount = snapshot.receivedResponseCount
	info.StreamStatus = snapshot.streamStatus
	info.FirstResponseTime = snapshot.firstResponseTime
	info.isFirstResponse = snapshot.isFirstResponse
	info.ThinkingContentInfo = snapshot.thinkingContentInfo
	info.UseRuntimeHeadersOverride = snapshot.useRuntimeHeadersOverride
	info.ParamOverrideAudit = append([]string(nil), snapshot.paramOverrideAudit...)
	info.UpstreamRequestBodySize = snapshot.upstreamRequestBodySize
	info.RequestConversionChain = append([]types.RelayFormat(nil), snapshot.requestConversionChain...)
	info.FinalRequestRelayFormat = snapshot.finalRequestRelayFormat
	if snapshot.claudeConvertInfo == nil {
		info.ClaudeConvertInfo = nil
	} else {
		copy := *snapshot.claudeConvertInfo
		info.ClaudeConvertInfo = &copy
	}
	if info.ResponsesUsageInfo != nil {
		for name, tool := range info.ResponsesUsageInfo.BuiltInTools {
			if tool != nil {
				tool.CallCount = snapshot.responsesToolCounts[name]
			}
		}
	}
	if snapshot.runtimeHeadersOverride == nil {
		info.RuntimeHeadersOverride = nil
	} else {
		info.RuntimeHeadersOverride = make(map[string]interface{}, len(snapshot.runtimeHeadersOverride))
		for key, value := range snapshot.runtimeHeadersOverride {
			info.RuntimeHeadersOverride[key] = value
		}
	}
}

func cloneInterfaceMap(source map[string]interface{}) map[string]interface{} {
	if source == nil {
		return nil
	}
	clone := make(map[string]interface{}, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}
