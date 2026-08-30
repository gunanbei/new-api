package common

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type StreamEndReason string

const (
	StreamEndReasonNone                StreamEndReason = ""
	StreamEndReasonDone                StreamEndReason = "done"
	StreamEndReasonTimeout             StreamEndReason = "timeout"
	StreamEndReasonClientGone          StreamEndReason = "client_gone"
	StreamEndReasonScannerErr          StreamEndReason = "scanner_error"
	StreamEndReasonHandlerStop         StreamEndReason = "handler_stop"
	StreamEndReasonEOF                 StreamEndReason = "eof"
	StreamEndReasonPanic               StreamEndReason = "panic"
	StreamEndReasonPingFail            StreamEndReason = "ping_fail"
	StreamEndReasonFirstContentTimeout StreamEndReason = "first_content_timeout"
)

const maxStreamErrorEntries = 20

type StreamErrorEntry struct {
	Message   string
	Timestamp time.Time
}

type StreamStatus struct {
	EndReason StreamEndReason
	EndError  error
	EndDetail string
	// LastEventType records the last parsed provider event. It is deliberately
	// metadata only (never the event payload) so failure logs retain the
	// upstream lifecycle reason without leaking model output.
	LastEventType string
	endOnce       sync.Once

	mu         sync.Mutex
	Errors     []StreamErrorEntry
	ErrorCount int
}

func (s *StreamStatus) SetLastEventType(eventType string) {
	if s == nil || eventType == "" {
		return
	}
	s.mu.Lock()
	s.LastEventType = eventType
	s.mu.Unlock()
}

func (s *StreamStatus) SetEndDetail(detail string) {
	if s == nil || detail == "" {
		return
	}
	s.mu.Lock()
	s.EndDetail = detail
	s.mu.Unlock()
}

func NewStreamStatus() *StreamStatus {
	return &StreamStatus{}
}

func (s *StreamStatus) SetEndReason(reason StreamEndReason, err error) {
	if s == nil {
		return
	}
	s.endOnce.Do(func() {
		s.EndReason = reason
		s.EndError = err
	})
}

func (s *StreamStatus) RecordError(msg string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ErrorCount++
	if len(s.Errors) < maxStreamErrorEntries {
		s.Errors = append(s.Errors, StreamErrorEntry{
			Message:   msg,
			Timestamp: time.Now(),
		})
	}
}

func (s *StreamStatus) HasErrors() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount > 0
}

func (s *StreamStatus) TotalErrorCount() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ErrorCount
}

// FirstError returns the first recorded stream error for backend diagnostics.
func (s *StreamStatus) FirstError() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.Errors) == 0 || s.Errors[0].Message == "" {
		return nil
	}
	return fmt.Errorf("%s", s.Errors[0].Message)
}

func (s *StreamStatus) IsNormalEnd() bool {
	if s == nil {
		return true
	}
	if s.EndReason == StreamEndReasonEOF {
		// A body EOF without an explicit terminal event is an interrupted
		// provider stream, not a successful completion.
		return s.EndError == nil
	}
	return s.EndReason == StreamEndReasonDone || s.EndReason == StreamEndReasonHandlerStop
}

func (s *StreamStatus) Summary() string {
	if s == nil {
		return "StreamStatus<nil>"
	}
	b := &strings.Builder{}
	fmt.Fprintf(b, "reason=%s", s.EndReason)
	if s.EndError != nil {
		fmt.Fprintf(b, " end_error=%q", s.EndError.Error())
	}
	s.mu.Lock()
	if s.EndDetail != "" {
		fmt.Fprintf(b, " end_detail=%q", s.EndDetail)
	}
	if s.ErrorCount > 0 {
		fmt.Fprintf(b, " soft_errors=%d", s.ErrorCount)
	}
	if s.LastEventType != "" {
		fmt.Fprintf(b, " last_event=%s", s.LastEventType)
	}
	s.mu.Unlock()
	return b.String()
}
