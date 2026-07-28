package types

// TokenFailoverRules controls which failures may advance a failover token to
// another channel or group. Enabled is request-local and is not persisted.
type TokenFailoverRules struct {
	HTTPStatusCodes string `json:"http_status_codes"`
	// A zero timeout disables only the corresponding token failover trigger.
	// It does not disable or extend the system-wide upstream timeout.
	// Legacy timeout fields are kept as fallbacks for tokens saved before
	// stream and non-stream timeouts became independently configurable.
	ResponseHeaderTimeoutMS          int  `json:"response_header_timeout_ms,omitempty"`
	FirstContentTimeoutMS            int  `json:"first_content_timeout_ms,omitempty"`
	StreamResponseHeaderTimeoutMS    *int `json:"stream_response_header_timeout_ms,omitempty"`
	StreamFirstContentTimeoutMS      *int `json:"stream_first_content_timeout_ms,omitempty"`
	NonStreamResponseHeaderTimeoutMS *int `json:"non_stream_response_header_timeout_ms,omitempty"`
	NonStreamFirstContentTimeoutMS   *int `json:"non_stream_first_content_timeout_ms,omitempty"`
	RetryOnTransportError            bool `json:"retry_on_transport_error"`
	RetryOnEmptyResponse             bool `json:"retry_on_empty_response"`
	RetryOnInvalidResponse           bool `json:"retry_on_invalid_response"`
	RetryOnStreamError               bool `json:"retry_on_stream_error"`
	Enabled                          bool `json:"-"`
}

func (r TokenFailoverRules) ResponseHeaderTimeout(stream bool) int {
	if stream && r.StreamResponseHeaderTimeoutMS != nil {
		return *r.StreamResponseHeaderTimeoutMS
	}
	if !stream && r.NonStreamResponseHeaderTimeoutMS != nil {
		return *r.NonStreamResponseHeaderTimeoutMS
	}
	return r.ResponseHeaderTimeoutMS
}

func (r TokenFailoverRules) FirstContentTimeout(stream bool) int {
	if stream && r.StreamFirstContentTimeoutMS != nil {
		return *r.StreamFirstContentTimeoutMS
	}
	if !stream && r.NonStreamFirstContentTimeoutMS != nil {
		return *r.NonStreamFirstContentTimeoutMS
	}
	return r.FirstContentTimeoutMS
}

func DefaultTokenFailoverRules() TokenFailoverRules {
	return TokenFailoverRules{
		RetryOnTransportError:  true,
		RetryOnEmptyResponse:   true,
		RetryOnInvalidResponse: true,
		RetryOnStreamError:     true,
	}
}
