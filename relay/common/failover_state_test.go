package common

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFailoverStateMachineRollsBackAttemptBeforeRetry(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &RelayInfo{
		RetryIndex:             0,
		UsingGroup:             "primary",
		IsStream:               true,
		RequestConversionChain: []types.RelayFormat{types.RelayFormatOpenAI},
		ChannelMeta:            &ChannelMeta{ChannelId: 10, ChannelName: "primary-channel", ParamOverride: map[string]interface{}{"keep": "yes"}},
		ResponsesUsageInfo: &ResponsesUsageInfo{BuiltInTools: map[string]*BuildInToolInfo{
			"web_search": {ToolName: "web_search"},
		}},
	}
	machine := NewFailoverStateMachine(types.DefaultTokenFailoverRules())
	gate := NewAttemptResponseWriter(ctx.Writer)

	require.NoError(t, machine.BeginSelection(info))
	require.NoError(t, machine.StartAttempt(info, gate))
	info.IsStream = false
	info.SendResponseCount = 9
	info.RequestConversionChain = append(info.RequestConversionChain, types.RelayFormatClaude)
	info.ChannelMeta.IsModelMapped = true
	info.ChannelMeta.ParamOverride["keep"] = "changed"
	info.ResponsesUsageInfo.BuiltInTools["web_search"].CallCount = 3
	_, err := gate.WriteString("data: [DONE]\n\n")
	require.NoError(t, err)

	apiErr := types.NewError(errors.New("empty response"), types.ErrorCodeEmptyResponse, types.ErrOptionWithStatusCode(http.StatusBadGateway))
	require.NoError(t, machine.RollbackAttempt(info, gate, apiErr, "empty_response"))
	require.NoError(t, machine.SelectRetry(info, "empty_response", "empty response rule"))

	assert.Equal(t, FailoverStateReadyToRetry, machine.State())
	assert.True(t, info.IsStream)
	assert.Zero(t, info.SendResponseCount)
	assert.Equal(t, []types.RelayFormat{types.RelayFormatOpenAI}, info.RequestConversionChain)
	assert.False(t, info.ChannelMeta.IsModelMapped)
	assert.Equal(t, "yes", info.ChannelMeta.ParamOverride["keep"])
	assert.Zero(t, info.ResponsesUsageInfo.BuiltInTools["web_search"].CallCount)
	assert.Zero(t, gate.Size())
	assert.Nil(t, info.AttemptResponse)

	trail := machine.Trail()
	require.Len(t, trail.Events, 5)
	assert.Equal(t, "rollback_completed", trail.Events[3].Event)
	assert.Equal(t, "completed", trail.Events[3].Rollback)
	assert.Equal(t, "retry_selected", trail.Events[4].Event)
}

func TestFailoverStateMachineRejectsCommittedRollbackAndIllegalTransition(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &RelayInfo{ChannelMeta: &ChannelMeta{ChannelId: 10}}
	machine := NewFailoverStateMachine(types.DefaultTokenFailoverRules())
	gate := NewAttemptResponseWriter(ctx.Writer)

	require.NoError(t, machine.BeginSelection(info))
	assert.Error(t, machine.BeginSelection(info))
	require.NoError(t, machine.StartAttempt(info, gate))
	_, err := gate.WriteString(`{"choices":[{"message":{"content":"hello"}}]}`)
	require.NoError(t, err)
	require.True(t, gate.Committed())

	apiErr := types.NewError(errors.New("late stream error"), types.ErrorCodeUpstreamStreamError)
	assert.Error(t, machine.RollbackAttempt(info, gate, apiErr, "stream_error"))
	require.NoError(t, machine.MarkResponseCommitted(info))
	require.NoError(t, machine.RejectRetry(info, apiErr, "response_committed", "downstream response was already committed", false))
	assert.Equal(t, FailoverStateFailed, machine.State())
}

func TestMergeFailoverAuditEventsKeepsAsyncPrefixes(t *testing.T) {
	current := []FailoverAuditEvent{{Sequence: 1, Event: "selection_started"}, {Sequence: 3, Event: "rollback_completed"}}
	incoming := []FailoverAuditEvent{{Sequence: 1, Event: "selection_started"}, {Sequence: 2, Event: "attempt_started"}}

	merged := MergeFailoverAuditEvents(current, incoming)
	require.Len(t, merged, 3)
	assert.Equal(t, []int{1, 2, 3}, []int{merged[0].Sequence, merged[1].Sequence, merged[2].Sequence})
}
