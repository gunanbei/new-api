package storage

import (
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeneratedAssetURLRejectsUnsafeTargets(t *testing.T) {
	for _, rawURL := range []string{"http://example.com/a.png", "https://user@example.com/a.png", "https://127.0.0.1/a.png", "https://[::1]/a.png"} {
		parsed, err := url.Parse(rawURL)
		require.NoError(t, err)
		assert.Error(t, validateGeneratedAssetURL(parsed), rawURL)
	}
	for _, rawIP := range []string{"10.0.0.1", "100.64.0.1", "192.0.2.1", "198.18.0.1", "203.0.113.1", "127.0.0.1", "::1", "2001:db8::1"} {
		assert.False(t, allowedGeneratedAssetIP(net.ParseIP(rawIP)), rawIP)
	}
}
