package service

import (
	"math/rand"
	"sort"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"
)

func GetFailoverRules(c *gin.Context) types.TokenFailoverRules {
	if c == nil {
		return types.TokenFailoverRules{}
	}
	rules, _ := common.GetContextKeyType[types.TokenFailoverRules](c, constant.ContextKeyTokenFailoverRules)
	return rules
}

// ResolveFailoverGroups turns a token's saved group list into the sequence the request
// will actually walk: entries the user may no longer use, or groups that were since
// deprecated, are skipped instead of failing the request, because a token configured
// months ago should still fail over across whatever is left. An empty result means the
// token has no usable group at all and the caller must reject the request.
//
// The order is decided here once per request so every retry keeps the same sequence,
// which matters for the random strategy.
func ResolveFailoverGroups(userGroup string, groups []string, strategy string) []string {
	usable := GetUserUsableGroups(userGroup)
	resolved := make([]string, 0, len(groups))
	seen := make(map[string]struct{}, len(groups))
	for _, group := range groups {
		if _, duplicate := seen[group]; duplicate {
			continue
		}
		if _, ok := usable[group]; !ok {
			continue
		}
		if !ratio_setting.ContainsGroupRatio(group) {
			continue
		}
		seen[group] = struct{}{}
		resolved = append(resolved, group)
	}

	switch strategy {
	case constant.FailoverStrategyLowestRatio:
		sort.SliceStable(resolved, func(i, j int) bool {
			return GetUserGroupRatio(userGroup, resolved[i]) < GetUserGroupRatio(userGroup, resolved[j])
		})
	case constant.FailoverStrategyRandom:
		rand.Shuffle(len(resolved), func(i, j int) {
			resolved[i], resolved[j] = resolved[j], resolved[i]
		})
	}
	return resolved
}

// GetFailoverGroups returns the group sequence resolved for the current request, or nil
// when the token is not in failover mode.
func GetFailoverGroups(c *gin.Context) []string {
	if c == nil {
		return nil
	}
	value, exists := common.GetContextKey(c, constant.ContextKeyTokenFailoverGroups)
	if !exists {
		return nil
	}
	groups, ok := value.([]string)
	if !ok {
		return nil
	}
	return groups
}

// EffectiveRetryTimes reports how many retries the current request may spend. A failover
// token can lower the system cap but never raise it; the clamp is applied here rather
// than trusted from the saved value because an admin can reduce RetryTimes at any time.
func EffectiveRetryTimes(c *gin.Context) int {
	if c == nil || !common.GetContextKeyBool(c, constant.ContextKeyTokenFailoverEnabled) {
		return common.RetryTimes
	}
	tokenRetry := common.GetContextKeyInt(c, constant.ContextKeyTokenFailoverMaxRetry)
	if tokenRetry > 0 && tokenRetry < common.RetryTimes {
		return tokenRetry
	}
	return common.RetryTimes
}
