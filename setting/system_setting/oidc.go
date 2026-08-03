package system_setting

import (
	"strings"

	"github.com/QuantumNous/new-api/setting/config"
)

type OIDCSettings struct {
	Enabled               bool   `json:"enabled"`
	DisplayName           string `json:"display_name"`
	ClientId              string `json:"client_id"`
	ClientSecret          string `json:"client_secret"`
	WellKnown             string `json:"well_known"`
	AuthorizationEndpoint string `json:"authorization_endpoint"`
	TokenEndpoint         string `json:"token_endpoint"`
	UserInfoEndpoint      string `json:"user_info_endpoint"`
}

// 默认配置
var defaultOIDCSettings = OIDCSettings{}

func init() {
	// 注册到全局配置管理器
	config.GlobalConfig.Register("oidc", &defaultOIDCSettings)
}

func GetOIDCSettings() *OIDCSettings {
	return &defaultOIDCSettings
}

// GetEffectiveDisplayName returns the configured display name, falling back to
// the protocol label used by older installations.
func (s *OIDCSettings) GetEffectiveDisplayName() string {
	if s == nil {
		return "OIDC"
	}
	if trimmed := strings.TrimSpace(s.DisplayName); trimmed != "" {
		return trimmed
	}
	return "OIDC"
}
