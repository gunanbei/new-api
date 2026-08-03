package service

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/setting/system_setting"

	"golang.org/x/net/proxy"
)

var (
	httpClient              *http.Client
	ssrfProtectedHTTPClient *http.Client
	proxyClientLock         sync.Mutex
	proxyClients            = make(map[string]*http.Client)
)

func checkRedirect(req *http.Request, via []*http.Request) error {
	urlStr := req.URL.String()
	if err := validateURLWithCurrentFetchSetting(urlStr, true); err != nil {
		return fmt.Errorf("redirect to %s blocked: %v", urlStr, err)
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return nil
}

func checkProtectedFetchRedirect(req *http.Request, via []*http.Request) error {
	urlStr := req.URL.String()
	if err := ValidateSSRFProtectedFetchURL(urlStr); err != nil {
		return fmt.Errorf("redirect to %s blocked: %v", urlStr, err)
	}
	if len(via) >= 10 {
		return fmt.Errorf("stopped after 10 redirects")
	}
	return nil
}

func validateURLWithCurrentFetchSetting(urlStr string, applyDomainIPFilter bool) error {
	fetchSetting := system_setting.GetFetchSetting()
	return common.ValidateURLWithFetchSetting(urlStr, fetchSetting.EnableSSRFProtection, fetchSetting.AllowPrivateIp, fetchSetting.DomainFilterMode, fetchSetting.IpFilterMode, fetchSetting.DomainList, fetchSetting.IpList, fetchSetting.AllowedPorts, applyDomainIPFilter && fetchSetting.ApplyIPFilterForDomain)
}

func ValidateSSRFProtectedFetchURL(urlStr string) error {
	return validateURLWithCurrentFetchSetting(urlStr, true)
}

func InitHttpClient() {
	httpClient, _ = newHTTPClient(dto.ChannelSettings{}, nil)
	ssrfProtectedHTTPClient = newProtectedFetchHTTPClient()
}

// GetHttpClient returns the general outbound client used by relay/provider
// integrations. Do not attach the SSRF-protected dialer here: provider base URLs
// are root/operator-managed deployment targets, not arbitrary user-controlled
// input, and may legitimately point at private networks, private-link endpoints,
// self-hosted services, or local proxies. Code paths that fetch arbitrary
// user-controlled URLs must use GetSSRFProtectedHTTPClient or
// ValidateSSRFProtectedFetchURL instead.
func GetHttpClient() *http.Client {
	return httpClient
}

// GetSSRFProtectedHTTPClient 返回带拨号时 SSRF 校验的客户端。
// ssrfProtectedHTTPClient 由 InitHttpClient 在启动时初始化，运行期只读。
func GetSSRFProtectedHTTPClient() *http.Client {
	if fetchSetting := system_setting.GetFetchSetting(); fetchSetting != nil && !fetchSetting.EnableSSRFProtection {
		return GetHttpClient()
	}
	return ssrfProtectedHTTPClient
}

// GetHttpClientWithProxy returns the default client or a proxy-enabled one when proxyURL is provided.
func GetHttpClientWithProxy(proxyURL string) (*http.Client, error) {
	return GetHttpClientWithProxySettings(proxyURL, dto.ChannelSettings{})
}

func GetHttpClientWithProxySettings(proxyURL string, settings dto.ChannelSettings) (*http.Client, error) {
	policy := NormalizeHTTPTransportPolicy(settings)
	key := proxyURL + "\x00" + policy.String()
	if proxyURL == "" && policy.String() == (HTTPTransportPolicy{Protocol: dto.HTTPProtocolAuto, Shards: 1}).String() {
		if client := GetHttpClient(); client != nil {
			return client, nil
		}
	}
	proxyClientLock.Lock()
	if client, ok := proxyClients[key]; ok {
		proxyClientLock.Unlock()
		return client, nil
	}
	proxyClientLock.Unlock()
	var parsedURL *url.URL
	var err error
	if proxyURL != "" {
		parsedURL, err = url.Parse(proxyURL)
		if err != nil {
			return nil, err
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" && parsedURL.Scheme != "socks5" && parsedURL.Scheme != "socks5h" {
			return nil, fmt.Errorf("unsupported proxy scheme: %s, must be http, https, socks5 or socks5h", parsedURL.Scheme)
		}
	}
	client, err := newHTTPClient(settings, parsedURL)
	if err != nil {
		return nil, err
	}
	proxyClientLock.Lock()
	proxyClients[key] = client
	proxyClientLock.Unlock()
	return client, nil
}

// ResetProxyClientCache 清空代理客户端缓存，确保下次使用时重新初始化
func ResetProxyClientCache() {
	proxyClientLock.Lock()
	defer proxyClientLock.Unlock()
	for _, client := range proxyClients {
		client.CloseIdleConnections()
	}
	proxyClients = make(map[string]*http.Client)
}

// NewProxyHttpClient 创建支持代理的 HTTP 客户端
func NewProxyHttpClient(proxyURL string) (*http.Client, error) {
	return GetHttpClientWithProxy(proxyURL)
}

func newHTTPClient(settings dto.ChannelSettings, proxyURL *url.URL) (*http.Client, error) {
	policy := NormalizeHTTPTransportPolicy(settings)
	factory := func() *http.Transport {
		transport := &http.Transport{MaxIdleConns: common.RelayMaxIdleConns, MaxIdleConnsPerHost: common.RelayMaxIdleConnsPerHost, IdleConnTimeout: time.Duration(common.RelayIdleConnTimeout) * time.Second, ForceAttemptHTTP2: true}
		if proxyURL == nil {
			transport.Proxy = http.ProxyFromEnvironment
		} else if proxyURL.Scheme == "http" || proxyURL.Scheme == "https" {
			transport.Proxy = http.ProxyURL(proxyURL)
		} else {
			var auth *proxy.Auth
			if proxyURL.User != nil {
				auth = &proxy.Auth{User: proxyURL.User.Username()}
				auth.Password, _ = proxyURL.User.Password()
			}
			dialer, err := proxy.SOCKS5("tcp", proxyURL.Host, auth, proxy.Direct)
			if err != nil {
				return nil
			}
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) { return dialer.Dial(network, addr) }
		}
		if common.TLSInsecureSkipVerify {
			transport.TLSClientConfig = common.InsecureTLSConfig
		}
		if policy.Protocol == dto.HTTPProtocolHTTP1 {
			applyHTTP1Force(transport)
		}
		return transport
	}
	if proxyURL != nil && (proxyURL.Scheme == "socks5" || proxyURL.Scheme == "socks5h") {
		if test := factory(); test == nil {
			return nil, fmt.Errorf("failed to create SOCKS5 proxy dialer")
		}
	}
	var rt http.RoundTripper
	if policy.Shards > 1 && policy.Protocol != dto.HTTPProtocolHTTP1 {
		rt = newShardedRoundTripper(policy, factory)
	} else {
		rt = factory()
	}
	if rt == nil {
		return nil, fmt.Errorf("failed to create HTTP transport")
	}
	client := &http.Client{Transport: rt, CheckRedirect: checkRedirect}
	if common.RelayTimeout != 0 {
		client.Timeout = time.Duration(common.RelayTimeout) * time.Second
	}
	return client, nil
}
