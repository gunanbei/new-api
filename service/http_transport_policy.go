package service

import (
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
)

type HTTPTransportPolicy struct {
	Protocol string
	Shards   int
}

var httpTransportPolicyWarnings sync.Map

func NormalizeHTTPTransportPolicy(settings dto.ChannelSettings) HTTPTransportPolicy {
	policy := HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 1}
	switch protocol := strings.ToLower(strings.TrimSpace(settings.HTTPProtocol)); protocol {
	case "", dto.HTTPProtocolAuto:
	case dto.HTTPProtocolHTTP1:
		policy.Protocol = dto.HTTPProtocolHTTP1
	default:
		warnHTTPTransportPolicyOnce("http_protocol", settings.HTTPProtocol)
	}
	if settings.HTTP2ConnectionShards > 1 {
		policy.Shards = settings.HTTP2ConnectionShards
	}
	if policy.Shards > dto.MaxHTTP2ConnectionShards {
		warnHTTPTransportPolicyOnce("http2_connection_shards", fmt.Sprint(policy.Shards))
		policy.Shards = dto.MaxHTTP2ConnectionShards
	}
	if policy.Protocol == dto.HTTPProtocolHTTP1 {
		policy.Shards = 1
	}
	return policy
}

func (p HTTPTransportPolicy) String() string { return fmt.Sprintf("%s|%d", p.Protocol, p.Shards) }

func warnHTTPTransportPolicyOnce(field, value string) {
	key := field + "=" + value
	if _, loaded := httpTransportPolicyWarnings.LoadOrStore(key, struct{}{}); !loaded {
		logger.LogWarn(nil, fmt.Sprintf("invalid channel http transport setting clamped: %s=%q", field, value))
	}
}
