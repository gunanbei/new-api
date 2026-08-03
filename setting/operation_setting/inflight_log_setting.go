package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

// InflightLogSetting controls the compliance acknowledgement required before
// recording request/response bodies outside self-use mode.
type InflightLogSetting struct {
	ComplianceConfirmed    bool   `json:"compliance_confirmed"`
	ComplianceTermsVersion string `json:"compliance_terms_version"`
	ComplianceConfirmedAt  int64  `json:"compliance_confirmed_at"`
	ComplianceConfirmedBy  int    `json:"compliance_confirmed_by"`
	ComplianceConfirmedIP  string `json:"compliance_confirmed_ip"`
}

const CurrentInflightLogComplianceTermsVersion = "v1"

var inflightLogSetting InflightLogSetting

func init() {
	config.GlobalConfig.Register("inflight_log_setting", &inflightLogSetting)
}

func GetInflightLogSetting() *InflightLogSetting {
	return &inflightLogSetting
}

func IsInflightLogComplianceConfirmed() bool {
	return inflightLogSetting.ComplianceConfirmed &&
		inflightLogSetting.ComplianceTermsVersion == CurrentInflightLogComplianceTermsVersion
}

func InflightLogComplianceRequired() bool {
	return !SelfUseModeEnabled || DemoSiteEnabled
}
