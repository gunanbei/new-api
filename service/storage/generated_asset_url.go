package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

const maxGeneratedRedirects = 3

func OpenGeneratedAssetURL(ctx context.Context, rawURL string, maxBytes int64) (io.ReadCloser, string, error) {
	if maxBytes <= 0 {
		return nil, "", errors.New("invalid generated asset size limit")
	}
	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			if len(via) >= maxGeneratedRedirects {
				return errors.New("too many generated asset redirects")
			}
			return validateGeneratedAssetURL(request.URL)
		},
		Transport: &http.Transport{DialContext: generatedAssetDialContext},
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, "", errors.New("invalid generated asset URL")
	}
	if err := validateGeneratedAssetURL(parsed); err != nil {
		return nil, "", err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return nil, "", err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, "", err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		response.Body.Close()
		return nil, "", fmt.Errorf("generated asset download returned status %d", response.StatusCode)
	}
	if response.ContentLength > maxBytes {
		response.Body.Close()
		return nil, "", errors.New("generated asset exceeds size limit")
	}
	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if !validGeneratedMime(mimeType) {
		response.Body.Close()
		return nil, "", errors.New("generated asset MIME type is not allowed")
	}
	return &generatedAssetURLReader{ReadCloser: response.Body, remaining: maxBytes}, mimeType, nil
}

type generatedAssetURLReader struct {
	io.ReadCloser
	remaining int64
}

func (reader *generatedAssetURLReader) Read(buffer []byte) (int, error) {
	if reader.remaining == 0 {
		var probe [1]byte
		count, err := reader.ReadCloser.Read(probe[:])
		if count > 0 {
			return 0, errors.New("generated asset exceeds size limit")
		}
		return 0, err
	}
	if int64(len(buffer)) > reader.remaining {
		buffer = buffer[:reader.remaining]
	}
	count, err := reader.ReadCloser.Read(buffer)
	reader.remaining -= int64(count)
	return count, err
}

func validateGeneratedAssetURL(target *url.URL) error {
	if target.Scheme != "https" || target.User != nil || target.Hostname() == "" {
		return errors.New("generated asset URL is not allowed")
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(context.Background(), target.Hostname())
	if err != nil || len(addresses) == 0 {
		return errors.New("generated asset host cannot be resolved")
	}
	for _, address := range addresses {
		if !allowedGeneratedAssetIP(address.IP) {
			return errors.New("generated asset URL resolves to a private address")
		}
	}
	return nil
}

func generatedAssetDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	addresses, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: 10 * time.Second}
	for _, candidate := range addresses {
		if !allowedGeneratedAssetIP(candidate.IP) {
			continue
		}
		connection, err := dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
		if err == nil {
			return connection, nil
		}
	}
	return nil, errors.New("generated asset host has no public address")
}

func allowedGeneratedAssetIP(address net.IP) bool {
	if address == nil || address.IsLoopback() || address.IsLinkLocalUnicast() || address.IsLinkLocalMulticast() || address.IsMulticast() || address.IsUnspecified() {
		return false
	}
	parsed, ok := netip.AddrFromSlice(address)
	if !ok {
		return false
	}
	if parsed.Is4In6() {
		parsed = parsed.Unmap()
	}
	for _, reserved := range generatedAssetBlockedPrefixes {
		if reserved.Contains(parsed) {
			return false
		}
	}
	return parsed.IsGlobalUnicast()
}

var generatedAssetBlockedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("10.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"),
	netip.MustParsePrefix("172.16.0.0/12"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.168.0.0/16"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("224.0.0.0/4"),
	netip.MustParsePrefix("fc00::/7"),
	netip.MustParsePrefix("fe80::/10"),
	netip.MustParsePrefix("2001:db8::/32"),
}
