package service

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withTraceOptions(enabled, menuVisible string, maxRequest, maxResponse string, fn func()) {
	common.OptionMapRWMutex.Lock()
	originalOptionMap := common.OptionMap
	if common.OptionMap == nil {
		common.OptionMap = make(map[string]string)
	}
	originalEnabled := common.OptionMap[inflightTaskTraceEnabledOptionKey]
	originalMenu := common.OptionMap[inflightTaskTraceMenuVisibleOptionKey]
	originalMaxRequest := common.OptionMap[inflightTaskTraceMaxRequestBytesOptionKey]
	originalMaxResponse := common.OptionMap[inflightTaskTraceMaxResponseBytesOptionKey]
	common.OptionMap[inflightTaskTraceEnabledOptionKey] = enabled
	common.OptionMap[inflightTaskTraceMenuVisibleOptionKey] = menuVisible
	common.OptionMap[inflightTaskTraceMaxRequestBytesOptionKey] = maxRequest
	common.OptionMap[inflightTaskTraceMaxResponseBytesOptionKey] = maxResponse
	common.OptionMapRWMutex.Unlock()

	defer func() {
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		if common.OptionMap != nil {
			common.OptionMap[inflightTaskTraceEnabledOptionKey] = originalEnabled
			common.OptionMap[inflightTaskTraceMenuVisibleOptionKey] = originalMenu
			common.OptionMap[inflightTaskTraceMaxRequestBytesOptionKey] = originalMaxRequest
			common.OptionMap[inflightTaskTraceMaxResponseBytesOptionKey] = originalMaxResponse
		}
		common.OptionMapRWMutex.Unlock()
	}()

	fn()
}

func TestInflightTraceRedactHeaders(t *testing.T) {
	headers := http.Header{
		"Authorization": []string{"Bearer secret"},
		"Content-Type":  []string{"application/json"},
		"X-Api-Key":     []string{"sk-test"},
	}
	redacted := redactTraceHeaders(headers)
	assert.Equal(t, "***", redacted["Authorization"])
	assert.Equal(t, "***", redacted["X-Api-Key"])
	assert.Equal(t, "application/json", redacted["Content-Type"])
}

func TestInflightTraceTruncateBody(t *testing.T) {
	raw := []byte("abcdefghijklmnopqrstuvwxyz")
	truncated, wasTruncated, originalBytes := truncateTraceBody(raw, 10)
	require.True(t, wasTruncated)
	assert.Equal(t, int64(26), originalBytes)
	assert.Equal(t, "abcdefghij", string(truncated))
}

func TestInflightTraceBodyEncoding(t *testing.T) {
	textBody, textEncoding, textBytes := encodeTraceBody([]byte(`{"ok":true}`))
	assert.Equal(t, "text", textEncoding)
	assert.Equal(t, `{"ok":true}`, textBody)
	assert.Equal(t, int64(11), textBytes)

	binaryBody, binaryEncoding, binaryBytes := encodeTraceBody([]byte{0xff, 0xfe, 0x00})
	assert.Equal(t, "base64", binaryEncoding)
	assert.NotEmpty(t, binaryBody)
	assert.Equal(t, int64(3), binaryBytes)
}

func TestInflightTraceMenuVisible(t *testing.T) {
	withTraceOptions("false", "true", "0", "0", func() {
		assert.False(t, InflightTaskTraceMenuVisible())
	})
	withTraceOptions("true", "true", "0", "0", func() {
		assert.True(t, InflightTaskTraceMenuVisible())
	})
	withTraceOptions("true", "false", "0", "0", func() {
		assert.False(t, InflightTaskTraceMenuVisible())
	})
}

func TestInflightTraceMaxBytesZeroMeansUnlimited(t *testing.T) {
	raw := []byte("0123456789")
	truncated, wasTruncated, originalBytes := truncateTraceBody(raw, 0)
	assert.False(t, wasTruncated)
	assert.Equal(t, raw, truncated)
	assert.Equal(t, int64(10), originalBytes)
}

func TestBuildInflightTaskTraceInProgressFlags(t *testing.T) {
	capture := &InflightTraceCapture{
		requestMethod:  http.MethodPost,
		requestPath:    "/v1/chat/completions",
		requestHeaders: http.Header{"Content-Type": []string{"application/json"}},
		requestBody:    []byte(`{"model":"gpt-4"}`),
		writer: &traceResponseWriter{
			buf:        bytes.NewBufferString("data: hello\n\n"),
			statusCode: http.StatusOK,
			headerSnap: http.Header{"Content-Type": []string{"text/event-stream"}},
		},
		createdAt: 100,
	}
	info := &relaycommon.RelayInfo{
		RequestId:       "req-live",
		UserId:          1,
		OriginModelName: "gpt-4",
		IsStream:        true,
	}

	trace := buildInflightTaskTraceFromCapture(
		capture,
		info,
		InflightTaskStatusStreaming,
		capture.createdAt,
		true,
	)
	require.NotNil(t, trace)
	assert.True(t, trace.Flags.InProgress)
	assert.True(t, trace.Flags.ResponseIncomplete)
	assert.Equal(t, InflightTaskStatusStreaming, trace.Status)
	assert.Equal(t, int64(100), trace.CreatedAt)
	assert.NotNil(t, trace.ClientResponse)

	finalTrace := buildInflightTaskTraceFromCapture(
		capture,
		info,
		InflightTaskStatusCompleted,
		capture.createdAt,
		false,
	)
	assert.False(t, finalTrace.Flags.InProgress)
}
