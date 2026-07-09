package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"

	"github.com/bytedance/gopkg/util/gopool"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	inflightTaskTraceKeyPrefix                   = "inflight:trace:"
	inflightTaskTraceEnabledOptionKey            = "InflightTaskTraceEnabled"
	inflightTaskTraceMenuVisibleOptionKey        = "InflightTaskTraceMenuVisible"
	inflightTaskTraceMaxRequestBytesOptionKey    = "InflightTaskTraceMaxRequestBytes"
	inflightTaskTraceMaxResponseBytesOptionKey   = "InflightTaskTraceMaxResponseBytes"
	inflightTraceFlushInterval                   = 2 * time.Second
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
	RequestID  string `json:"request_id"`
	UserID     int    `json:"user_id"`
	Status     string `json:"status"`
	Kind       string `json:"kind"`
	ModelName  string `json:"model_name"`
	IsStream   bool   `json:"is_stream"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
	RecordedAt int64  `json:"recorded_at"`

	ClientRequest  *InflightTraceHTTPPart `json:"client_request,omitempty"`
	ClientResponse *InflightTraceHTTPPart `json:"client_response,omitempty"`

	Flags InflightTaskTraceFlags `json:"flags"`
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
	buf          *bytes.Buffer
	maxBytes     int64
	totalWritten int64
	truncated    bool
	statusCode   int
	headerSnap   http.Header
	wroteHeader  bool
	owner        *InflightTraceCapture
}

func (w *traceResponseWriter) WriteHeader(code int) {
	if !w.wroteHeader {
		w.statusCode = code
		w.headerSnap = w.Header().Clone()
		w.wroteHeader = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *traceResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(b)
	if n > 0 {
		w.capture(b[:n])
	}
	return n, err
}

func (w *traceResponseWriter) WriteString(s string) (int, error) {
	return w.Write([]byte(s))
}

func (w *traceResponseWriter) capture(chunk []byte) {
	if len(chunk) == 0 {
		return
	}
	w.totalWritten += int64(len(chunk))
	if w.maxBytes > 0 && int64(w.buf.Len()) >= w.maxBytes {
		w.truncated = true
		return
	}
	if w.maxBytes > 0 {
		remain := w.maxBytes - int64(w.buf.Len())
		if int64(len(chunk)) > remain {
			chunk = chunk[:remain]
			w.truncated = true
		}
	}
	_, _ = w.buf.Write(chunk)
	if w.owner != nil {
		w.owner.scheduleTraceFlush()
	}
}

type InflightTraceCapture struct {
	writer         *traceResponseWriter
	requestBody    []byte
	requestBodyErr error
	requestMethod  string
	requestPath    string
	requestQuery   string
	requestProto   string
	requestHeaders http.Header
	info           *relaycommon.RelayInfo
	parentCtx      context.Context
	createdAt      int64
	flushMu        sync.Mutex
	lastFlushedAt  time.Time
}

func InflightTaskTraceEnabled() bool {
	common.OptionMapRWMutex.RLock()
	raw := common.OptionMap[inflightTaskTraceEnabledOptionKey]
	common.OptionMapRWMutex.RUnlock()
	return raw == "true"
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
		Method:   method,
		Path:     path,
		Query:    query,
		Protocol: protocol,
		StatusCode: statusCode,
		Headers:  redactTraceHeaders(headers),
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
		buf:            bytes.NewBuffer(nil),
		maxBytes:       inflightTaskTraceMaxResponseBytes(),
	}
	c.Writer = writer
	capture.writer = writer
	writer.owner = capture
	capture.info = info
	capture.parentCtx = c.Request.Context()
	capture.createdAt = time.Now().Unix()
	common.SetContextKey(c, constant.ContextKeyInflightTraceCapture, capture)
	flushInflightTraceSnapshotAsync(capture)
	return capture
}

func (capture *InflightTraceCapture) scheduleTraceFlush() {
	if capture == nil {
		return
	}
	capture.flushMu.Lock()
	defer capture.flushMu.Unlock()
	now := time.Now()
	if !capture.lastFlushedAt.IsZero() && now.Sub(capture.lastFlushedAt) < inflightTraceFlushInterval {
		return
	}
	capture.lastFlushedAt = now
	flushInflightTraceSnapshotAsync(capture)
}

func flushInflightTraceSnapshotAsync(capture *InflightTraceCapture) {
	if capture == nil || capture.info == nil || capture.info.UserId <= 0 || capture.info.RequestId == "" {
		return
	}
	gopool.Go(func() {
		ctx, cancel := NewInflightTaskFinalizeContext(capture.parentCtx)
		defer cancel()
		if err := persistInflightTraceSnapshot(ctx, capture); err != nil && !errors.Is(err, errInflightTaskUnavailable) {
			common.SysError(fmt.Sprintf("flush inflight trace snapshot failed: %v", err))
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
	createdAt := capture.createdAt
	if existing, err := loadInflightTaskTraceCreatedAt(ctx, info.RequestId); err == nil && existing > 0 {
		createdAt = existing
	}
	trace := buildInflightTaskTraceFromCapture(capture, info, status, createdAt, false)
	return writeInflightTaskTrace(ctx, info.RequestId, status, trace)
}

func persistInflightTraceSnapshot(ctx context.Context, capture *InflightTraceCapture) error {
	info := capture.info
	if info == nil || info.UserId <= 0 || info.RequestId == "" {
		return nil
	}

	item, err := loadInflightTaskItem(ctx, info.RequestId)
	if errors.Is(err, redis.Nil) {
		return nil
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
		responseHeaders := capture.writer.headerSnap
		if responseHeaders == nil {
			responseHeaders = capture.writer.Header()
		}
		responsePart, responseTruncated := buildInflightTraceHTTPPart(
			"",
			"",
			"",
			"",
			capture.writer.statusCode,
			responseHeaders,
			capture.writer.buf.Bytes(),
			inflightTaskTraceMaxResponseBytes(),
		)
		trace.ClientResponse = responsePart
		trace.Flags.ResponseTruncated = responseTruncated || capture.writer.truncated
		if inProgress {
			trace.Flags.InProgress = true
			trace.Flags.ResponseIncomplete = true
		} else if capture.writer.totalWritten > 0 && capture.writer.buf.Len() == 0 && capture.writer.statusCode == 0 {
			trace.Flags.ResponseIncomplete = true
		} else if capture.writer.buf.Len() == 0 && capture.writer.totalWritten == 0 && capture.writer.statusCode == 0 {
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
	if errors.Is(err, redis.Nil) {
		return nil, errInflightTaskTraceNotFound
	}
	if err != nil {
		return nil, err
	}
	var item InflightTask
	if err = common.UnmarshalJsonStr(itemRaw, &item); err != nil {
		return nil, err
	}
	if item.UserID != userID {
		return nil, errInflightTaskTraceForbidden
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
	return &trace, nil
}

func IsInflightTaskTraceNotFound(err error) bool {
	return errors.Is(err, errInflightTaskTraceNotFound)
}

func IsInflightTaskTraceForbidden(err error) bool {
	return errors.Is(err, errInflightTaskTraceForbidden)
}
