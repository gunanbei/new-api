package common

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"

	basecommon "github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

// ponytail: cap pre-commit buffering at 1 MiB; raise it only if real providers
// emit larger metadata preambles before their first valid content.
const maxFailoverResponseBuffer = 1 << 20

var ErrFirstContentTimeout = errors.New("upstream first content timeout")

// AttemptResponseWriter delays the downstream response until an attempt emits
// meaningful model output. Failed attempts can therefore be discarded safely.
type AttemptResponseWriter struct {
	gin.ResponseWriter
	mu        sync.Mutex
	header    http.Header
	status    int
	body      bytes.Buffer
	committed bool
	content   bool
	ready     chan struct{}
	readyOnce sync.Once
}

func NewAttemptResponseWriter(writer gin.ResponseWriter) *AttemptResponseWriter {
	return &AttemptResponseWriter{ResponseWriter: writer, header: writer.Header().Clone(), status: http.StatusOK, ready: make(chan struct{})}
}

func (w *AttemptResponseWriter) Header() http.Header {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.committed {
		return w.ResponseWriter.Header()
	}
	return w.header
}
func (w *AttemptResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
func (w *AttemptResponseWriter) Status() int                 { w.mu.Lock(); defer w.mu.Unlock(); return w.status }
func (w *AttemptResponseWriter) Size() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.committed {
		return w.ResponseWriter.Size()
	}
	return w.body.Len()
}
func (w *AttemptResponseWriter) Written() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.committed || w.body.Len() > 0
}
func (w *AttemptResponseWriter) ContentReady() <-chan struct{} { return w.ready }
func (w *AttemptResponseWriter) HasContent() bool              { w.mu.Lock(); defer w.mu.Unlock(); return w.content }
func (w *AttemptResponseWriter) Committed() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.committed
}

func (w *AttemptResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.committed && code > 0 {
		w.status = code
	}
}

func (w *AttemptResponseWriter) WriteHeaderNow() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.content {
		_ = w.commitLocked()
	}
}

func (w *AttemptResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.committed {
		return w.ResponseWriter.Write(data)
	}
	n, _ := w.body.Write(data)
	if hasMeaningfulModelOutput(w.body.Bytes()) {
		w.markContentLocked()
		return n, w.commitLocked()
	}
	if w.body.Len() > maxFailoverResponseBuffer {
		w.markContentLocked()
		return n, w.commitLocked()
	}
	return n, nil
}

func (w *AttemptResponseWriter) WriteString(data string) (int, error) { return w.Write([]byte(data)) }
func (w *AttemptResponseWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.committed {
		w.ResponseWriter.Flush()
	}
}
func (w *AttemptResponseWriter) Commit() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.commitLocked()
}
func (w *AttemptResponseWriter) Discard() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.committed {
		w.body.Reset()
		w.header = make(http.Header)
	}
}

func (w *AttemptResponseWriter) commitLocked() error {
	if w.committed {
		return nil
	}
	for key := range w.ResponseWriter.Header() {
		w.ResponseWriter.Header().Del(key)
	}
	for key, values := range w.header {
		for _, value := range values {
			w.ResponseWriter.Header().Add(key, value)
		}
	}
	w.ResponseWriter.WriteHeader(w.status)
	w.committed = true
	if w.body.Len() == 0 {
		return nil
	}
	_, err := w.ResponseWriter.Write(w.body.Bytes())
	w.body.Reset()
	return err
}

func (w *AttemptResponseWriter) markContentLocked() {
	w.content = true
	w.readyOnce.Do(func() { close(w.ready) })
}

func (w *AttemptResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return w.ResponseWriter.Hijack()
}
func (w *AttemptResponseWriter) CloseNotify() <-chan bool { return w.ResponseWriter.CloseNotify() }
func (w *AttemptResponseWriter) Pusher() http.Pusher      { return w.ResponseWriter.Pusher() }

func hasMeaningfulModelOutput(data []byte) bool {
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("data:")) {
			line = bytes.TrimSpace(bytes.TrimPrefix(line, []byte("data:")))
		}
		if len(line) == 0 || bytes.Equal(line, []byte("[DONE]")) || line[0] != '{' {
			continue
		}
		var value any
		if basecommon.Unmarshal(line, &value) == nil && containsModelContent(value, "") {
			return true
		}
	}
	var value any
	return basecommon.Unmarshal(data, &value) == nil && containsModelContent(value, "")
}

func containsModelContent(value any, key string) bool {
	switch value := value.(type) {
	case string:
		if strings.TrimSpace(value) == "" {
			return false
		}
		switch strings.ToLower(key) {
		case "content", "text", "output_text", "reasoning", "reasoning_content", "thinking", "thought", "arguments", "partial_json", "code", "encrypted_content":
			return true
		}
	case map[string]any:
		if kind, ok := value["type"].(string); ok {
			kind = strings.ToLower(kind)
			if strings.Contains(kind, "tool_use") || strings.Contains(kind, "function_call") {
				return true
			}
			if strings.Contains(kind, "text") || strings.Contains(kind, "reasoning") || strings.Contains(kind, "thinking") {
				if delta, ok := value["delta"].(string); ok && strings.TrimSpace(delta) != "" {
					return true
				}
			}
		}
		switch strings.ToLower(key) {
		case "tool_calls", "function_call", "functioncall", "inline_data", "inlinedata", "code_execution_result", "codeexecutionresult":
			return len(value) > 0
		}
		for childKey, child := range value {
			if containsModelContent(child, childKey) {
				return true
			}
		}
	case []any:
		if strings.EqualFold(key, "tool_calls") && len(value) > 0 {
			return true
		}
		for _, child := range value {
			if containsModelContent(child, key) {
				return true
			}
		}
	}
	return false
}

func (info *RelayInfo) ResetFailoverAttempt(writer *AttemptResponseWriter) {
	info.AttemptResponse = writer
	info.SendResponseCount = 0
	info.ReceivedResponseCount = 0
	info.StreamStatus = nil
	info.FirstResponseTime = info.StartTime.Add(-1)
	info.isFirstResponse = true
	info.ThinkingContentInfo = ThinkingContentInfo{IsFirstThinkingContent: true}
	if info.ClaudeConvertInfo != nil {
		info.ClaudeConvertInfo.LastMessagesType = LastMessageTypeNone
		info.ClaudeConvertInfo.Index = 0
		info.ClaudeConvertInfo.Usage = nil
		info.ClaudeConvertInfo.FinishReason = ""
		info.ClaudeConvertInfo.Done = false
		info.ClaudeConvertInfo.ToolCallBaseIndex = 0
		info.ClaudeConvertInfo.ToolCallMaxIndexOffset = 0
	}
	if info.ResponsesUsageInfo != nil {
		for _, tool := range info.ResponsesUsageInfo.BuiltInTools {
			tool.CallCount = 0
		}
	}
}

// FinalizeFailoverAttempt must run before billing settles an attempt.
func (info *RelayInfo) FinalizeFailoverAttempt() *types.NewAPIError {
	gate := info.AttemptResponse
	if gate == nil {
		return nil
	}
	if gate.HasContent() {
		info.SetFirstResponseTime()
		if err := gate.Commit(); err != nil {
			return types.NewError(errors.New("failed to commit upstream response"), types.ErrorCodeUpstreamStreamError,
				types.ErrOptionWithStatusCode(http.StatusBadGateway), types.ErrOptionWithInternalError(err))
		}
		return nil
	}
	if info.StreamStatus != nil {
		if info.StreamStatus.HasErrors() {
			cause := info.StreamStatus.EndError
			if cause == nil {
				cause = info.StreamStatus.FirstError()
			}
			return types.NewError(errors.New("upstream stream failed before returning valid content"), types.ErrorCodeUpstreamStreamError,
				types.ErrOptionWithStatusCode(http.StatusBadGateway), types.ErrOptionWithInternalError(cause))
		}
		switch info.StreamStatus.EndReason {
		case StreamEndReasonFirstContentTimeout:
			return types.NewError(errors.New("upstream timed out before returning valid content"), types.ErrorCodeUpstreamFirstContentTimeout,
				types.ErrOptionWithStatusCode(http.StatusGatewayTimeout), types.ErrOptionWithInternalError(info.StreamStatus.EndError))
		case StreamEndReasonScannerErr, StreamEndReasonTimeout, StreamEndReasonPanic, StreamEndReasonHandlerStop, StreamEndReasonPingFail:
			return types.NewError(errors.New("upstream stream ended before returning valid content"), types.ErrorCodeUpstreamStreamError,
				types.ErrOptionWithStatusCode(http.StatusBadGateway), types.ErrOptionWithInternalError(info.StreamStatus.EndError))
		case StreamEndReasonEOF:
			return types.NewError(io.EOF, types.ErrorCodeEmptyResponse, types.ErrOptionWithStatusCode(http.StatusBadGateway),
				types.ErrOptionWithInternalError(info.StreamStatus.EndError))
		}
	}
	return types.NewError(io.EOF, types.ErrorCodeEmptyResponse, types.ErrOptionWithStatusCode(http.StatusBadGateway))
}
