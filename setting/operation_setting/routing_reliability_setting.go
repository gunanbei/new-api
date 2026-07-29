package operation_setting

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type CrossGroupRetryRule struct {
	Group           string `json:"group"`
	HTTPStatusCodes string `json:"http_status_codes"`
}

type RoutingReliabilitySetting struct {
	CrossGroupRetryRules []CrossGroupRetryRule `json:"cross_group_retry_rules"`
}

var routingReliabilitySetting = RoutingReliabilitySetting{
	CrossGroupRetryRules: []CrossGroupRetryRule{},
}

func init() {
	config.GlobalConfig.Register("routing_reliability_setting", &routingReliabilitySetting)
}

func GetRoutingReliabilitySetting() *RoutingReliabilitySetting {
	return &routingReliabilitySetting
}

func ParseCrossGroupRetryRules(value string) ([]CrossGroupRetryRule, error) {
	var rules []CrossGroupRetryRule
	if err := common.UnmarshalJsonStr(value, &rules); err != nil {
		return nil, errors.New("invalid cross-group retry rules")
	}
	if rules == nil {
		return []CrossGroupRetryRule{}, nil
	}

	normalized := make([]CrossGroupRetryRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		group := strings.TrimSpace(rule.Group)
		if group == "" {
			return nil, errors.New("cross-group retry group cannot be empty")
		}
		if _, exists := seen[group]; exists {
			return nil, fmt.Errorf("duplicate cross-group retry group: %s", group)
		}

		ranges, err := ParseHTTPStatusCodeRanges(rule.HTTPStatusCodes)
		if err != nil || len(ranges) == 0 {
			return nil, fmt.Errorf("invalid cross-group retry HTTP status codes for group %s", group)
		}
		seen[group] = struct{}{}
		normalized = append(normalized, CrossGroupRetryRule{
			Group:           group,
			HTTPStatusCodes: statusCodeRangesToString(ranges),
		})
	}
	return normalized, nil
}

func ShouldRetryAcrossGroup(group string, statusCode int) bool {
	if group == "" || IsAlwaysSkipRetryStatusCode(statusCode) {
		return false
	}
	for _, rule := range routingReliabilitySetting.CrossGroupRetryRules {
		if rule.Group != group {
			continue
		}
		ranges, err := ParseHTTPStatusCodeRanges(rule.HTTPStatusCodes)
		return err == nil && shouldMatchStatusCodeRanges(ranges, statusCode)
	}
	return false
}
