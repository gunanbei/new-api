package service

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"

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

func withInflightLogComplianceState(selfUse, demo, confirmed bool, fn func()) {
	originalSelfUse := operation_setting.SelfUseModeEnabled
	originalDemo := operation_setting.DemoSiteEnabled
	setting := operation_setting.GetInflightLogSetting()
	originalConfirmed := setting.ComplianceConfirmed
	originalTermsVersion := setting.ComplianceTermsVersion

	operation_setting.SelfUseModeEnabled = selfUse
	operation_setting.DemoSiteEnabled = demo
	setting.ComplianceConfirmed = confirmed
	if confirmed {
		setting.ComplianceTermsVersion = operation_setting.CurrentInflightLogComplianceTermsVersion
	} else {
		setting.ComplianceTermsVersion = ""
	}

	defer func() {
		operation_setting.SelfUseModeEnabled = originalSelfUse
		operation_setting.DemoSiteEnabled = originalDemo
		setting.ComplianceConfirmed = originalConfirmed
		setting.ComplianceTermsVersion = originalTermsVersion
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

func TestInflightTaskTraceEnabledRequiresComplianceOutsideSelfUseMode(t *testing.T) {
	withTraceOptions("true", "true", "0", "0", func() {
		withInflightLogComplianceState(true, false, false, func() {
			assert.True(t, InflightTaskTraceEnabled())
		})
		withInflightLogComplianceState(false, false, false, func() {
			assert.False(t, InflightTaskTraceEnabled())
		})
		withInflightLogComplianceState(true, true, false, func() {
			assert.False(t, InflightTaskTraceEnabled())
		})
		withInflightLogComplianceState(false, false, true, func() {
			assert.True(t, InflightTaskTraceEnabled())
		})
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
	tempPath, tempFile, err := createTestInflightTraceResponseTempFile(t)
	require.NoError(t, err)
	_, err = tempFile.WriteString("data: hello\n\n")
	require.NoError(t, err)
	capture := &InflightTraceCapture{
		requestMethod:  http.MethodPost,
		requestPath:    "/v1/chat/completions",
		requestHeaders: http.Header{"Content-Type": []string{"application/json"}},
		requestBody:    []byte(`{"model":"gpt-4"}`),
		writer: &traceResponseWriter{
			tempPath:      tempPath,
			tempFile:      tempFile,
			capturedBytes: int64(len("data: hello\n\n")),
			statusCode:    http.StatusOK,
			headerSnap:    http.Header{"Content-Type": []string{"text/event-stream"}},
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

func TestInflightTraceNoteResponseCaptureFlushesByBytes(t *testing.T) {
	capture := &InflightTraceCapture{
		info: &relaycommon.RelayInfo{
			RequestId: "req-chunk",
			UserId:    1,
		},
	}
	capture.noteResponseCapture(inflightTraceFlushMinBytes)
	assert.Equal(t, int64(0), capture.bytesSinceFlush)
	assert.False(t, capture.lastFlushedAt.IsZero())
}

func TestInflightTraceFlushSingleFlightState(t *testing.T) {
	capture := &InflightTraceCapture{}
	now := time.Unix(100, 0)

	require.True(t, capture.startResponseCaptureFlush(now))
	require.False(t, capture.startResponseCaptureFlush(now.Add(time.Second)))
	assert.True(t, capture.flushPending)

	require.True(t, capture.finishResponseCaptureFlush())
	assert.True(t, capture.flushInFlight)
	assert.False(t, capture.flushPending)

	require.False(t, capture.finishResponseCaptureFlush())
	assert.False(t, capture.flushInFlight)
}

func TestInflightTraceSnapshotIsImmutable(t *testing.T) {
	tempPath, tempFile, err := createTestInflightTraceResponseTempFile(t)
	require.NoError(t, err)
	_, err = tempFile.WriteString("first")
	require.NoError(t, err)
	writer := &traceResponseWriter{
		tempPath:      tempPath,
		tempFile:      tempFile,
		capturedBytes: int64(len("first")),
		statusCode:    http.StatusOK,
		headerSnap:    http.Header{"X-Test": []string{"before"}},
	}

	snapshot := writer.snapshot()
	writer.mu.Lock()
	_, err = writer.tempFile.WriteString(" second")
	require.NoError(t, err)
	writer.capturedBytes += int64(len(" second"))
	writer.headerSnap.Set("X-Test", "after")
	writer.mu.Unlock()

	assert.Equal(t, "first", string(snapshot.buf))
	assert.Equal(t, "before", snapshot.headerSnap.Get("X-Test"))
}

func TestInflightTraceAppendBufferedSSEChunks(t *testing.T) {
	tempPath, tempFile, err := createTestInflightTraceResponseTempFile(t)
	require.NoError(t, err)
	capture := &InflightTraceCapture{
		writer: &traceResponseWriter{
			tempPath: tempPath,
			tempFile: tempFile,
		},
		info: &relaycommon.RelayInfo{
			RequestId: "req-buffered-sse",
			UserId:    1,
		},
	}
	capture.AppendInflightTraceResponseChunk([]byte("data: {\"delta\":\"hi\"}\n\n"))
	capture.AppendInflightTraceResponseChunk([]byte("data: [DONE]\n\n"))
	require.Equal(t, http.StatusOK, capture.writer.statusCode)
	require.Equal(t, "text/event-stream", capture.writer.headerSnap.Get("Content-Type"))
	snapshot := capture.writer.snapshot()
	require.Contains(t, string(snapshot.buf), `"delta":"hi"`)
	require.Contains(t, string(snapshot.buf), "data: [DONE]")

	capture.ResetInflightTraceResponseBody()
	require.Empty(t, capture.writer.snapshot().buf)
	require.False(t, capture.writer.wroteHeader)
	capture.cleanupResponseTempFile()
}

func createTestInflightTraceResponseTempFile(t *testing.T) (string, *os.File, error) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "response-*.tmp")
	if err != nil {
		return "", nil, err
	}
	return file.Name(), file, nil
}
