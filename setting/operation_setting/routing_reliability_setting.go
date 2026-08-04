package operation_setting

import (
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/config"
)

type CrossGroupRetryRule struct {
	Group           string `json:"group"`
	HTTPStatusCodes string `json:"http_status_codes"`
}

type GroupAutoDisableRule struct {
	Group                   string  `json:"group"`
	Enabled                 bool    `json:"enabled"`
	DisableThresholdSeconds float64 `json:"disable_threshold_seconds"`
	HTTPStatusCodes         string  `json:"http_status_codes"`
	FailureKeywords         string  `json:"failure_keywords"`
}

type RoutingReliabilitySetting struct {
	CrossGroupRetryRules  []CrossGroupRetryRule  `json:"cross_group_retry_rules"`
	GroupAutoDisableRules []GroupAutoDisableRule `json:"group_auto_disable_rules"`
}

var routingReliabilitySetting = RoutingReliabilitySetting{
	CrossGroupRetryRules:  []CrossGroupRetryRule{},
	GroupAutoDisableRules: []GroupAutoDisableRule{},
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

func ParseGroupAutoDisableRules(value string) ([]GroupAutoDisableRule, error) {
	var rules []GroupAutoDisableRule
	if err := common.UnmarshalJsonStr(value, &rules); err != nil {
		return nil, errors.New("invalid group auto-disable rules")
	}
	if rules == nil {
		return []GroupAutoDisableRule{}, nil
	}

	normalized := make([]GroupAutoDisableRule, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		group := strings.TrimSpace(rule.Group)
		if group == "" {
			return nil, errors.New("group auto-disable group cannot be empty")
		}
		if _, exists := seen[group]; exists {
			return nil, fmt.Errorf("duplicate group auto-disable group: %s", group)
		}
		if rule.DisableThresholdSeconds < 0 || math.IsNaN(rule.DisableThresholdSeconds) || math.IsInf(rule.DisableThresholdSeconds, 0) {
			return nil, fmt.Errorf("invalid group auto-disable threshold for group %s", group)
		}

		ranges, err := ParseHTTPStatusCodeRanges(rule.HTTPStatusCodes)
		if err != nil {
			return nil, fmt.Errorf("invalid group auto-disable HTTP status codes for group %s", group)
		}
		seen[group] = struct{}{}
		normalized = append(normalized, GroupAutoDisableRule{
			Group:                   group,
			Enabled:                 rule.Enabled,
			DisableThresholdSeconds: rule.DisableThresholdSeconds,
			HTTPStatusCodes:         statusCodeRangesToString(ranges),
			FailureKeywords:         normalizeAutoDisableKeywords(rule.FailureKeywords),
		})
	}
	return normalized, nil
}

func FindGroupAutoDisableRule(group string) (GroupAutoDisableRule, bool) {
	group = strings.TrimSpace(group)
	for _, rule := range routingReliabilitySetting.GroupAutoDisableRules {
		if rule.Group == group {
			return rule, true
		}
	}
	return GroupAutoDisableRule{}, false
}

func IsGroupAutoDisableEnabled(group string) bool {
	rule, configured := FindGroupAutoDisableRule(group)
	return !configured || rule.Enabled
}

func ShouldDisableInGroup(group string, errStatusCode int, isChannelError, keywordMatched bool) bool {
	rule, configured := FindGroupAutoDisableRule(group)
	if configured && !rule.Enabled {
		return false
	}
	if isChannelError || keywordMatched {
		return true
	}
	if !configured {
		return ShouldDisableByStatusCode(errStatusCode)
	}
	ranges, err := ParseHTTPStatusCodeRanges(rule.HTTPStatusCodes)
	return err == nil && shouldMatchStatusCodeRanges(ranges, errStatusCode)
}

func ShouldDisableInGroupWithMessage(group string, errStatusCode int, isChannelError, globalKeywordMatched bool, message string) bool {
	rule, configured := FindGroupAutoDisableRule(group)
	if configured && !rule.Enabled {
		return false
	}
	if isChannelError || globalKeywordMatched {
		return true
	}
	if configured {
		if matchesAutoDisableKeywords(message, rule.FailureKeywords) {
			return true
		}
		ranges, err := ParseHTTPStatusCodeRanges(rule.HTTPStatusCodes)
		return err == nil && shouldMatchStatusCodeRanges(ranges, errStatusCode)
	}
	return ShouldDisableByStatusCode(errStatusCode) || matchesAutoDisableKeywords(message, strings.Join(AutomaticDisableKeywords, "\n"))
}

func EffectiveAutoDisableThreshold(groups []string, fallback float64) float64 {
	if len(groups) == 0 {
		return fallback
	}

	threshold := 0.0
	for _, group := range groups {
		rule, configured := FindGroupAutoDisableRule(group)
		if configured && !rule.Enabled {
			continue
		}

		candidate := fallback
		if configured {
			candidate = rule.DisableThresholdSeconds
		}
		if candidate > 0 && (threshold == 0 || candidate < threshold) {
			threshold = candidate
		}
	}
	return threshold
}

func normalizeAutoDisableKeywords(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	keywords := make([]string, 0, len(lines))
	for _, line := range lines {
		if keyword := strings.TrimSpace(strings.ToLower(line)); keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	return strings.Join(keywords, "\n")
}

func matchesAutoDisableKeywords(message string, keywords string) bool {
	for _, keyword := range strings.Split(normalizeAutoDisableKeywords(keywords), "\n") {
		if keyword != "" && strings.Contains(message, keyword) {
			return true
		}
	}
	return false
}
