package service

import (
	"crypto/tls"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

type shardedRoundTripper struct {
	shards   []http.RoundTripper
	n        uint32
	counters sync.Map
}

func newShardedRoundTripper(policy HTTPTransportPolicy, factory func() *http.Transport) *shardedRoundTripper {
	n := policy.Shards
	if n < 1 {
		n = 1
	}
	shards := make([]http.RoundTripper, n)
	for i := range shards {
		shards[i] = factory()
	}
	return &shardedRoundTripper{shards: shards, n: uint32(n)}
}

func (s *shardedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	origin := ""
	if req != nil && req.URL != nil {
		origin = strings.ToLower(req.URL.Scheme) + "://" + req.URL.Host
	}
	v, _ := s.counters.LoadOrStore(origin, &atomic.Uint32{})
	idx := (v.(*atomic.Uint32).Add(1) - 1) % s.n
	return s.shards[idx].RoundTrip(req)
}

func (s *shardedRoundTripper) CloseIdleConnections() {
	for _, rt := range s.shards {
		if c, ok := rt.(interface{ CloseIdleConnections() }); ok {
			c.CloseIdleConnections()
		}
	}
}

func applyHTTP1Force(transport *http.Transport) {
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = make(map[string]func(string, *tls.Conn) http.RoundTripper)
	if transport.TLSClientConfig != nil {
		cfg := transport.TLSClientConfig.Clone()
		cfg.NextProtos = nil
		transport.TLSClientConfig = cfg
	}
}
