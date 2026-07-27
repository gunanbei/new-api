package common

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttemptResponseWriterCommitsOnlyMeaningfulOutput(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	gate := NewAttemptResponseWriter(ctx.Writer)

	_, err := gate.WriteString("data: {\"choices\":[{\"delta\":{\"role\":\"assistant\"}}]}\n\n")
	require.NoError(t, err)
	assert.False(t, gate.Committed())
	assert.Empty(t, recorder.Body.String())

	_, err = gate.WriteString("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n")
	require.NoError(t, err)
	assert.True(t, gate.Committed())
	assert.Contains(t, recorder.Body.String(), "hello")
}

func TestFinalizeFailoverAttemptRejectsEmptyResponse(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	info := &RelayInfo{AttemptResponse: NewAttemptResponseWriter(ctx.Writer)}
	_, err := info.AttemptResponse.WriteString("data: [DONE]\n\n")
	require.NoError(t, err)

	apiErr := info.FinalizeFailoverAttempt()
	require.NotNil(t, apiErr)
	assert.Equal(t, types.ErrorCodeEmptyResponse, apiErr.GetErrorCode())
	assert.Empty(t, recorder.Body.String())
}

func TestAttemptResponseWriterTreatsToolCallAsContent(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	gate := NewAttemptResponseWriter(ctx.Writer)

	_, err := gate.WriteString(`data: {"choices":[{"delta":{"tool_calls":[{"function":{"name":"lookup"}}]}}]}` + "\n\n")
	require.NoError(t, err)
	assert.True(t, gate.HasContent())
	assert.True(t, gate.Committed())
}

func TestAttemptResponseWriterTreatsResponsesTextDeltaAsContent(t *testing.T) {
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	gate := NewAttemptResponseWriter(ctx.Writer)

	_, err := gate.WriteString("event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"hello\"}\n\n")
	require.NoError(t, err)
	assert.True(t, gate.HasContent())
	assert.True(t, gate.Committed())
}
