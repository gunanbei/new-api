package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	inflightTaskTraceKeyPrefix                 = "inflight:trace:"
	inflightTaskTraceEnabledOptionKey          = "InflightTaskTraceEnabled"
	inflightTaskTraceMenuVisibleOptionKey      = "InflightTaskTraceMenuVisible"
	inflightTaskTraceMaxRequestBytesOptionKey  = "InflightTaskTraceMaxRequestBytes"
	inflightTaskTraceMaxResponseBytesOptionKey = "InflightTaskTraceMaxResponseBytes"
	inflightTraceFlushInterval                 = 2 * time.Second
	inflightTraceFlushMinBytes                 = 64 * 1024
	inflightTraceLiveSnapshotMaxBytes          = 64 * 1024
)

var sensitiveTraceHeaderNames = map[string]struct{}{
	"authorization":       {},
	"x-api-key":           {},
	"api-key":             {},
	"cookie":              {},
	"set-cookie":          {},
	"x-goog-api-key":      {},
	"proxy-authorization": {},
}

var (
	errInflightTaskTraceNotFound  = errors.New("debug log not found")
	errInflightTaskTraceForbidden = errors.New("forbidden")
)

type InflightTaskTrace struct {
	RequestID string `json:"request_id"`
	UserID    int    `json:"user_id"`
	Status    string `json:"status"`
	// Error contains a lifecycle summary for failed streams. The raw
	// upstream response remains in ClientResponse when available; this field
	// keeps the terminal reason queryable even when the provider closes early.
	Error       string                         `json:"error,omitempty"`
	Kind        string                         `json:"kind"`
	ModelName   string                         `json:"model_name"`
	IsStream    bool                           `json:"is_stream"`
	CreatedAt   int64                          `json:"created_at"`
	UpdatedAt   int64                          `json:"updated_at"`
	RecordedAt  int64                          `json:"recorded_at"`
	ArchiveID   uint64                         `json:"archive_id,omitempty"`
	StorageMode string                         `json:"storage_mode,omitempty"`
	Archive     *InflightTraceArchiveReference `json:"archive,omitempty"`

	ClientRequest  *InflightTraceHTTPPart `json:"client_request,omitempty"`
	ClientResponse *InflightTraceHTTPPart `json:"client_response,omitempty"`

	Flags InflightTaskTraceFlags `json:"flags"`
}

type InflightTraceArchiveReference struct {
	ID        uint64 `json:"id"`
	FileName  string `json:"file_name"`
	Status    string `json:"status"`
	RemoteURL string `json:"remote_url,omitempty"`
}

type InflightTaskTraceFlags struct {
	RequestTruncated    bool `json:"request_truncated"`
	ResponseTruncated   bool `json:"response_truncated"`
	ResponseIncomplete  bool `json:"response_incomplete"`
	InProgress          bool `json:"in_progress,omitempty"`
	UnsupportedRealtime bool `json:"unsupported_realtime,omitempty"`
}

type InflightTraceHTTPPart struct {
	Method       string            `json:"method,omitempty"`
	Path         string            `json:"path,omitempty"`
	Query        string            `json:"query,omitempty"`
	Protocol     string            `json:"protocol,omitempty"`
	StatusCode   int               `json:"status_code,omitempty"`
	Headers      map[string]string `json:"headers,omitempty"`
	Body         string            `json:"body,omitempty"`
	BodyEncoding string            `json:"body_encoding,omitempty"`
	BodyBytes    int64             `json:"body_bytes,omitempty"`
	ContentType  string            `json:"content_type,omitempty"`
}

type InflightTaskListMeta struct {
	TraceMenuVisible bool `json:"trace_menu_visible"`
}

type traceResponseWriter struct {
	gin.ResponseWriter
	mu            sync.Mutex
	tempPath      string
	tempFile      *os.File
	capturedBytes int64
	maxBytes      int64
	totalWritten  int64
	truncated     bool
	statusCode    int
	headerSnap    http.Header
	wroteHeader   bool
	owner         *InflightTraceCapture
}

func (w *traceResponseWriter) setHeaderLocked(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.headerSnap = w.Header().Clone()
		w.wroteHeader = true
	}
}

func (w *traceResponseWriter) WriteHeader(code int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.setHeaderLocked(code)
	w.ResponseWriter.WriteHeader(code)
}

func (w *traceResponseWriter) Write(b []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.wroteHeader {
		w.setHeaderLocked(http.StatusOK)
		w.ResponseWriter.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	if n > 0 {
		w.captureLocked(b[:n])
	}
	return n, err
}

func (w *traceResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *traceResponseWriter) capture(chunk []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.captureLocked(chunk)
}

func (w *traceResponseWriter) captureLocked(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	w.totalWritten += int64(len(chunk))
	if w.tempFile == nil {
		w.truncated = true
		return
	}
	if w.maxBytes > 0 && w.capturedBytes >= w.maxBytes {
		w.truncated = true
		return
	}
	if w.maxBytes > 0 {
		remain := w.maxBytes - w.capturedBytes
		if int64(len(chunk)) > remain {
			chunk = chunk[:remain]
			w.truncated = true
		}
	}
	if len(chunk) == 0 {
		return
	}
	n, err := w.tempFile.Write(chunk)
	if n > 0 {
		w.capturedBytes += int64(n)
		if w.owner != nil {
			w.owner.noteResponseCapture(n)
		}
	}
	if err == nil && n != len(chunk) {
		err = io.ErrShortWrite
	}
	if err != nil {
		w.truncated = true
		common.SysError(fmt.Sprintf("write inflight trace response temp file failed: %v", err))
	}
}

type traceResponseSnapshot struct {
	statusCode    int
	headerSnap    http.Header
	buf           []byte
	capturedBytes int64
	truncated     bool
	totalWritten  int64
}

func (w *traceResponseWriter) snapshot() traceResponseSnapshot {
	return w.snapshotWithLimit(0)
}

func (w *traceResponseWriter) snapshotWithLimit(maxBytes int64) traceResponseSnapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	snap := traceResponseSnapshot{
		statusCode:    w.statusCode,
		capturedBytes: w.capturedBytes,
		truncated:     w.truncated,
		totalWritten:  w.totalWritten,
	}
	if w.headerSnap != nil {
		snap.headerSnap = w.headerSnap.Clone()
	} else {
		snap.headerSnap = w.Header().Clone()
	}
	readBytes := w.capturedBytes
	if maxBytes > 0 && readBytes > maxBytes {
		readBytes = maxBytes
		snap.truncated = true
	}
	if w.tempFile != nil && readBytes > 0 {
		reader := io.NewSectionReader(w.tempFile, 0, readBytes)
		if body, err := io.ReadAll(reader); err == nil {
			snap.buf = body
		} else {
			common.SysError(fmt.Sprintf("read inflight trace response temp file failed: %v", err))
		}
	}
	return snap
}

func createInflightTraceResponseTempFile() (string, *os.File, error) {
	dir := filepath.Join(common.GetDiskCacheDir(), "inflight-trace-responses")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", nil, err
	}
	file, err := os.CreateTemp(dir, "inflight-trace-response-*.tmp")
	if err != nil {
		return "", nil, err
	}
	return file.Name(), file, nil
}

func CleanupInflightTraceResponseTempFiles(maxAge time.Duration) error {
	dir := filepath.Join(common.GetDiskCacheDir(), "inflight-trace-responses")
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "inflight-trace-response-") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr != nil || info.ModTime().After(cutoff) {
			continue
		}
		if removeErr := os.Remove(filepath.Join(dir, entry.Name())); removeErr != nil && !os.IsNotExist(removeErr) {
			return removeErr
		}
	}
	return nil
}

type InflightTraceCapture struct {
	writer          *traceResponseWriter
	requestBody     []byte
	requestBodyErr  error
	requestMethod   string
	requestPath     string
	requestQuery    string
	requestProto    string
	requestHeaders  http.Header
	info            *relaycommon.RelayInfo
	parentCtx       context.Context
	createdAt       int64
	persistMu       sync.Mutex
	flushMu         sync.Mutex
	lastFlushedAt   time.Time
	bytesSinceFlush int64
	flushInFlight   bool
	flushPending    bool
}

func InflightTaskTraceEnabled() bool {
	if operation_setting.InflightLogComplianceRequired() &&
		!operation_setting.IsInflightLogComplianceConfirmed() {
		return false
	}
	return InflightTaskTraceConfigured()
}

func InflightTaskTraceConfigured() bool {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskTraceEnabledOptionKey]
	common.OptionMapRWMutex.RUnlock()
	configured, err := strconv.ParseBool(strings.TrimSpace(raw))
	return err == nil && configured
}

func InflightTaskTraceMenuVisible() bool {
	if !InflightTaskTraceEnabled() {
		return false
	}
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskTraceMenuVisibleOptionKey]
	common.OptionMapRWMutex.RUnlock()
	return raw == "true"
}

func inflightTaskTraceMaxRequestBytes() int64 {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskTraceMaxRequestBytesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func inflightTaskTraceMaxResponseBytes() int64 {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskTraceMaxResponseBytesOptionKey]
	common.OptionMapRWMutex.RUnlock()
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func inflightTaskTraceKey(requestID string) string {
	return inflightTaskTraceKeyPrefix + requestID
}

func redactTraceHeaders(headers http.Header) map[string]string {
	if headers == nil {
		return nil
	}
	result := make(map[string]string, len(headers))
	for key, values := range headers {
		if _, sensitive := sensitiveTraceHeaderNames[strings.ToLower(key)]; sensitive {
			result[key] = "***"
			continue
		}
		result[key] = strings.Join(values, ", ")
	}
	return result
}

func truncateTraceBody(raw []byte, maxBytes int64) (truncated []byte, wasTruncated bool, originalBytes int64) {
	originalBytes = int64(len(raw))
	if maxBytes <= 0 || originalBytes <= maxBytes {
		return raw, false, originalBytes
	}
	return raw[:maxBytes], true, originalBytes
}

func encodeTraceBody(raw []byte) (body string, encoding string, bodyBytes int64) {
	bodyBytes = int64(len(raw))
	if len(raw) == 0 {
		return "", "empty", 0
	}
	if utf8.Valid(raw) {
		return string(raw), "text", bodyBytes
	}
	return base64.StdEncoding.EncodeToString(raw), "base64", bodyBytes
}

func buildInflightTraceHTTPPart(method, path, query, protocol string, statusCode int, headers http.Header, raw []byte, maxBytes int64) (*InflightTraceHTTPPart, bool) {
	part := &InflightTraceHTTPPart{
		Method:     method,
		Path:       path,
		Query:      query,
		Protocol:   protocol,
		StatusCode: statusCode,
		Headers:    redactTraceHeaders(headers),
	}
	if contentType := headers.Get("Content-Type"); contentType != "" {
		part.ContentType = contentType
	}
	body, truncated, originalBytes := truncateTraceBody(raw, maxBytes)
	encodedBody, encoding, _ := encodeTraceBody(body)
	part.Body = encodedBody
	part.BodyEncoding = encoding
	part.BodyBytes = originalBytes
	return part, truncated
}

func StartInflightTraceCapture(c *gin.Context, info *relaycommon.RelayInfo, relayFormat types.RelayFormat) *InflightTraceCapture {
	if c == nil || info == nil || !InflightTaskTraceEnabled() {
		return nil
	}
	if relayFormat == types.RelayFormatOpenAIRealtime {
		return nil
	}

	capture := &InflightTraceCapture{}
	if c.Request != nil {
		capture.requestMethod = c.Request.Method
		capture.requestPath = c.Request.URL.Path
		capture.requestQuery = c.Request.URL.RawQuery
		capture.requestProto = c.Request.Proto
		capture.requestHeaders = c.Request.Header.Clone()
	}
	if bodyStorage, err := common.GetBodyStorage(c); err == nil {
		capture.requestBody, capture.requestBodyErr = bodyStorage.Bytes()
	} else {
		capture.requestBodyErr = err
	}

	writer := &traceResponseWriter{
		ResponseWriter: c.Writer,
		maxBytes:       inflightTaskTraceMaxResponseBytes(),
	}
	if tempPath, tempFile, err := createInflightTraceResponseTempFile(); err == nil {
		writer.tempPath = tempPath
		writer.tempFile = tempFile
	} else {
		writer.truncated = true
		common.SysError(fmt.Sprintf("create inflight trace response temp file failed: %v", err))
	}
	c.Writer = writer
	capture.writer = writer
	writer.owner = capture
	capture.info = info
	capture.parentCtx = c.Request.Context()
	capture.createdAt = time.Now().Unix()
	common.SetContextKey(c, constant.ContextKeyInflightTraceCapture, capture)
	if capture.startResponseCaptureFlush(time.Time{}) {
		flushInflightTraceSnapshotAsync(capture)
	}
	return capture
}

func GetInflightTraceCapture(c *gin.Context) *InflightTraceCapture {
	if c == nil {
		return nil
	}
	raw, ok := common.GetContextKey(c, constant.ContextKeyInflightTraceCapture)
	if !ok {
		return nil
	}
	capture, _ := raw.(*InflightTraceCapture)
	return capture
}

func (capture *InflightTraceCapture) StageInflightTraceResponse(contentType string, statusCode int) {
	if capture == nil || capture.writer == nil {
		return
	}
	if statusCode <= 0 {
		statusCode = http.StatusOK
	}
	capture.writer.mu.Lock()
	defer capture.writer.mu.Unlock()
	if capture.writer.wroteHeader {
		return
	}
	capture.writer.statusCode = statusCode
	capture.writer.headerSnap = http.Header{}
	if contentType != "" {
		capture.writer.headerSnap.Set("Content-Type", contentType)
	}
	capture.writer.wroteHeader = true
}

func (capture *InflightTraceCapture) AppendInflightTraceResponseChunk(chunk []byte) {
	if capture == nil || capture.writer == nil || len(chunk) == 0 {
		return
	}
	capture.writer.mu.Lock()
	wroteHeader := capture.writer.wroteHeader
	capture.writer.mu.Unlock()
	if !wroteHeader {
		capture.StageInflightTraceResponse("text/event-stream", http.StatusOK)
	}
	capture.writer.capture(chunk)
}

func (capture *InflightTraceCapture) ResetInflightTraceResponseBody() {
	if capture == nil || capture.writer == nil {
		return
	}
	capture.writer.mu.Lock()
	defer capture.writer.mu.Unlock()
	if capture.writer.tempFile != nil {
		if err := capture.writer.tempFile.Truncate(0); err != nil {
			common.SysError(fmt.Sprintf("reset inflight trace response temp file failed: %v", err))
		}
		if _, err := capture.writer.tempFile.Seek(0, 0); err != nil {
			common.SysError(fmt.Sprintf("seek inflight trace response temp file failed: %v", err))
		}
	}
	capture.writer.capturedBytes = 0
	capture.writer.truncated = false
	capture.writer.totalWritten = 0
	capture.writer.wroteHeader = false
	capture.writer.statusCode = 0
	capture.writer.headerSnap = nil
}

func (capture *InflightTraceCapture) cleanupResponseTempFile() {
	if capture == nil || capture.writer == nil {
		return
	}
	capture.writer.mu.Lock()
	defer capture.writer.mu.Unlock()
	if capture.writer.tempFile != nil {
		_ = capture.writer.tempFile.Close()
		capture.writer.tempFile = nil
	}
	if capture.writer.tempPath != "" {
		_ = os.Remove(capture.writer.tempPath)
		capture.writer.tempPath = ""
	}
}

func (capture *InflightTraceCapture) noteResponseCapture(n int) {
	if capture == nil {
		return
	}
	capture.flushMu.Lock()
	if n > 0 {
		capture.bytesSinceFlush += int64(n)
	}
	now := time.Now()
	shouldFlush := capture.bytesSinceFlush >= inflightTraceFlushMinBytes ||
		capture.lastFlushedAt.IsZero() ||
		now.Sub(capture.lastFlushedAt) >= inflightTraceFlushInterval
	if !shouldFlush {
		capture.flushMu.Unlock()
		return
	}
	if capture.flushInFlight {
		capture.flushPending = true
		capture.flushMu.Unlock()
		return
	}
	capture.flushInFlight = true
	capture.bytesSinceFlush = 0
	capture.lastFlushedAt = now
	capture.flushMu.Unlock()
	flushInflightTraceSnapshotAsync(capture)
}

func (capture *InflightTraceCapture) startResponseCaptureFlush(now time.Time) bool {
	capture.flushMu.Lock()
	defer capture.flushMu.Unlock()
	if capture.flushInFlight {
		capture.flushPending = true
		return false
	}
	capture.flushInFlight = true
	if !now.IsZero() {
		capture.bytesSinceFlush = 0
		capture.lastFlushedAt = now
	}
	return true
}

func (capture *InflightTraceCapture) finishResponseCaptureFlush() bool {
	capture.flushMu.Lock()
	defer capture.flushMu.Unlock()
	if capture.flushPending {
		capture.flushPending = false
		capture.bytesSinceFlush = 0
		capture.lastFlushedAt = time.Now()
		return true
	}
	capture.flushInFlight = false
	return false
}

func flushInflightTraceSnapshotAsync(capture *InflightTraceCapture) {
	if capture == nil {
		return
	}
	if capture.info == nil || capture.info.UserId <= 0 || capture.info.RequestId == "" {
		capture.flushMu.Lock()
		capture.flushInFlight = false
		capture.flushPending = false
		capture.flushMu.Unlock()
		return
	}
	gopool.Go(func() {
		for {
			ctx, cancel := NewInflightTaskFinalizeContext(capture.parentCtx)
			err := persistInflightTraceSnapshot(ctx, capture)
			cancel()
			if err != nil && !errors.Is(err, errInflightTaskUnavailable) {
				common.SysError(fmt.Sprintf("flush inflight trace snapshot failed: %v", err))
			}
			if !capture.finishResponseCaptureFlush() {
				return
			}
		}
	})
}

func ResolveInflightTraceFinalStatus(info *relaycommon.RelayInfo, newAPIError *types.NewAPIError) string {
	if newAPIError != nil {
		return InflightTaskStatusFailed
	}
	return FinalInflightTaskStatus(info)
}

func PersistInflightTaskTraceAsync(parent context.Context, capture *InflightTraceCapture, info *relaycommon.RelayInfo, status string) {
	if capture == nil || info == nil || info.UserId <= 0 || info.RequestId == "" {
		return
	}
	if !isInflightTaskTerminalStatus(status) {
		return
	}
	gopool.Go(func() {
		ctx, cancel := NewInflightTaskFinalizeContext(parent)
		defer cancel()
		if err := persistInflightTaskTrace(ctx, capture, info, status); err != nil && !errors.Is(err, errInflightTaskUnavailable) {
			common.SysError(fmt.Sprintf("persist inflight trace failed: %v", err))
		}
	})
}

func persistInflightTaskTrace(ctx context.Context, capture *InflightTraceCapture, info *relaycommon.RelayInfo, status string) error {
	capture.persistMu.Lock()
	defer capture.persistMu.Unlock()
	createdAt := capture.createdAt
	if existing, err := loadInflightTaskTraceCreatedAt(ctx, info.RequestId); err == nil && existing > 0 {
		createdAt = existing
	}
	trace := buildInflightTaskTraceFromCapture(capture, info, status, createdAt, false)
	if err := writeInflightTaskTrace(ctx, info.RequestId, status, trace); err != nil {
		return err
	}
	capture.cleanupResponseTempFile()
	return nil
}

func persistInflightTraceSnapshot(ctx context.Context, capture *InflightTraceCapture) error {
	capture.persistMu.Lock()
	defer capture.persistMu.Unlock()
	info := capture.info
	if info == nil || info.UserId <= 0 || info.RequestId == "" {
		return nil
	}

	item, err := loadInflightTaskItem(ctx, info.RequestId)
	if errors.Is(err, redis.Nil) {
		// The trace capture starts immediately after relay info is created, while
		// the accepted task record is written asynchronously. Keep an initial
		// request snapshot instead of dropping it during that short race.
		trace := buildInflightTaskTraceFromCapture(
			capture,
			info,
			InflightTaskStatusAccepted,
			capture.createdAt,
			true,
		)
		return writeInflightTaskTrace(ctx, info.RequestId, InflightTaskStatusAccepted, trace)
	}
	if err != nil {
		return err
	}
	if item.UserID != info.UserId {
		return nil
	}
	if isInflightTaskTerminalStatus(item.Status) {
		return nil
	}

	trace := buildInflightTaskTraceFromCapture(capture, info, item.Status, capture.createdAt, true)
	return writeInflightTaskTrace(ctx, info.RequestId, item.Status, trace)
}

func loadInflightTaskItem(ctx context.Context, requestID string) (*InflightTask, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return nil, err
	}
	itemRaw, err := client.Get(ctx, inflightTaskItemKey(requestID)).Result()
	if err != nil {
		return nil, err
	}
	var item InflightTask
	if err = common.UnmarshalJsonStr(itemRaw, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func loadInflightTaskTraceCreatedAt(ctx context.Context, requestID string) (int64, error) {
	client, err := inflightTaskRedis()
	if err != nil {
		return 0, err
	}
	traceRaw, err := client.Get(ctx, inflightTaskTraceKey(requestID)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var trace InflightTaskTrace
	if err = common.UnmarshalJsonStr(traceRaw, &trace); err != nil {
		return 0, err
	}
	return trace.CreatedAt, nil
}

func buildInflightTaskTraceFromCapture(
	capture *InflightTraceCapture,
	info *relaycommon.RelayInfo,
	status string,
	createdAt int64,
	inProgress bool,
) *InflightTaskTrace {
	now := time.Now().Unix()
	if createdAt <= 0 {
		createdAt = now
	}

	trace := &InflightTaskTrace{
		RequestID:  info.RequestId,
		UserID:     info.UserId,
		Status:     status,
		Kind:       inflightTaskKindFromRelayMode(info.RelayMode),
		ModelName:  info.OriginModelName,
		IsStream:   info.IsStream,
		CreatedAt:  createdAt,
		UpdatedAt:  now,
		RecordedAt: now,
	}
	if info.StreamStatus != nil && status == InflightTaskStatusFailed {
		trace.Error = info.StreamStatus.Summary()
	}

	requestPart, requestTruncated := buildInflightTraceHTTPPart(
		capture.requestMethod,
		capture.requestPath,
		capture.requestQuery,
		capture.requestProto,
		0,
		capture.requestHeaders,
		capture.requestBody,
		inflightTaskTraceMaxRequestBytes(),
	)
	trace.ClientRequest = requestPart
	trace.Flags.RequestTruncated = requestTruncated

	if capture.writer != nil {
		var snapshot traceResponseSnapshot
		if inProgress {
			snapshot = capture.writer.snapshotWithLimit(inflightTraceLiveSnapshotMaxBytes)
		} else {
			snapshot = capture.writer.snapshot()
		}
		responseHeaders := snapshot.headerSnap
		responsePart, responseTruncated := buildInflightTraceHTTPPart(
			"",
			"",
			"",
			"",
			snapshot.statusCode,
			responseHeaders,
			snapshot.buf,
			inflightTaskTraceMaxResponseBytes(),
		)
		responsePart.BodyBytes = snapshot.capturedBytes
		trace.ClientResponse = responsePart
		trace.Flags.ResponseTruncated = responseTruncated || snapshot.truncated
		if inProgress {
			trace.Flags.InProgress = true
			trace.Flags.ResponseIncomplete = true
		} else if snapshot.totalWritten > 0 && len(snapshot.buf) == 0 && snapshot.statusCode == 0 {
			trace.Flags.ResponseIncomplete = true
		} else if len(snapshot.buf) == 0 && snapshot.totalWritten == 0 && snapshot.statusCode == 0 {
			trace.Flags.ResponseIncomplete = true
		}
	} else if inProgress {
		trace.Flags.InProgress = true
		trace.Flags.ResponseIncomplete = true
	}

	return trace
}

func writeInflightTaskTrace(ctx context.Context, requestID, status string, trace *InflightTaskTrace) error {
	client, err := inflightTaskRedis()
	if err != nil {
		return err
	}
	if inflightTaskTraceStorageMode() == "disk" {
		trace.StorageMode = "disk"
		if isInflightTaskTerminalStatus(status) {
			if err := persistInflightTraceToDisk(ctx, trace); err != nil {
				return err
			}
			if trace.ClientRequest != nil {
				trace.ClientRequest.Body = ""
				trace.ClientRequest.BodyEncoding = "disk"
			}
			if trace.ClientResponse != nil {
				trace.ClientResponse.Body = ""
				trace.ClientResponse.BodyEncoding = "disk"
			}
		}
	}
	data, err := common.Marshal(trace)
	if err != nil {
		return err
	}
	ttl := inflightTaskRetentionTTL(status)
	return client.Set(ctx, inflightTaskTraceKey(requestID), string(data), ttl).Err()
}

func attachInflightTaskHasTrace(ctx context.Context, tasks []InflightTask) error {
	if len(tasks) == 0 {
		return nil
	}
	client, err := inflightTaskRedis()
	if err != nil {
		return err
	}

	pipe := client.Pipeline()
	cmds := make([]*redis.IntCmd, 0)
	indices := make([]int, 0, len(tasks))
	for i := range tasks {
		cmds = append(cmds, pipe.Exists(ctx, inflightTaskTraceKey(tasks[i].RequestID)))
		indices = append(indices, i)
	}
	if len(cmds) == 0 {
		return nil
	}
	if _, err = pipe.Exec(ctx); err != nil {
		return err
	}
	for i, idx := range indices {
		tasks[idx].HasTrace = cmds[i].Val() > 0
	}
	return nil
}

func GetInflightTaskTrace(ctx context.Context, userID int, requestID string) (*InflightTaskTrace, error) {
	if userID <= 0 || requestID == "" {
		return nil, errInflightTaskTraceForbidden
	}
	client, err := inflightTaskRedis()
	if err != nil {
		return nil, err
	}

	itemRaw, err := client.Get(ctx, inflightTaskItemKey(requestID)).Result()
	itemExists := err == nil
	if err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	var item InflightTask
	if itemExists {
		if err = common.UnmarshalJsonStr(itemRaw, &item); err != nil {
			return nil, err
		}
		if item.UserID != userID {
			return nil, errInflightTaskTraceForbidden
		}
	}

	traceRaw, err := client.Get(ctx, inflightTaskTraceKey(requestID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, errInflightTaskTraceNotFound
	}
	if err != nil {
		return nil, err
	}
	var trace InflightTaskTrace
	if err = common.UnmarshalJsonStr(traceRaw, &trace); err != nil {
		return nil, err
	}
	if trace.UserID != userID {
		return nil, errInflightTaskTraceForbidden
	}
	if !itemExists && !trace.Flags.InProgress {
		return nil, errInflightTaskTraceNotFound
	}
	if trace.StorageMode == "disk" && trace.ArchiveID > 0 {
		var archive model.InflightTraceArchive
		if err = model.DB.Where("id = ? AND user_id = ?", trace.ArchiveID, userID).First(&archive).Error; err != nil {
			return nil, errInflightTaskTraceNotFound
		}
		trace.Archive = &InflightTraceArchiveReference{ID: archive.ID, FileName: archive.FileName, Status: archive.Status, RemoteURL: archive.RemoteURL}
		if archive.LocalPath != "" {
			if archivedTrace, archiveErr := loadInflightTraceFromCSV(archive.LocalPath, requestID); archiveErr == nil && archivedTrace != nil {
				archivedTrace.Archive = trace.Archive
				return archivedTrace, nil
			}
		}
	}
	return &trace, nil
}

func IsInflightTaskTraceNotFound(err error) bool {
	return errors.Is(err, errInflightTaskTraceNotFound)
}

func IsInflightTaskTraceForbidden(err error) bool {
	return errors.Is(err, errInflightTaskTraceForbidden)
}
