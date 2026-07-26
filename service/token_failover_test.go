package service

import (
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// walkFailoverSequence replays the relay retry loop against the cursor and records
// where every attempt lands. It mirrors selectFailoverChannel: the retry index only
// ever moves forward, so the caller's cap is the real ceiling on attempts.
func walkFailoverSequence(groups []string, levels map[string]int, retryTimes int) []string {
	cursor := &failoverCursor{Groups: groups}
	levelsOf := func(group string) int { return levels[group] }

	landings := make([]string, 0, retryTimes+1)
	for retry := 0; retry <= retryTimes; retry++ {
		group, priorityRetry, ok := cursor.nextTarget(retry, levelsOf)
		if !ok {
			break
		}
		landings = append(landings, fmt.Sprintf("%s:%d", group, priorityRetry))
	}
	return landings
}

func TestFailoverCursorLandingSequence(t *testing.T) {
	cases := []struct {
		name       string
		groups     []string
		levels     map[string]int
		retryTimes int
		expected   []string
	}{
		{
			name:       "exhausts each group's priorities before moving on",
			groups:     []string{"a", "b"},
			levels:     map[string]int{"a": 2, "b": 2},
			retryTimes: 3,
			expected:   []string{"a:0", "a:1", "b:0", "b:1"},
		},
		{
			name:       "retry budget stops the walk before the sequence ends",
			groups:     []string{"a", "b", "c"},
			levels:     map[string]int{"a": 2, "b": 2, "c": 2},
			retryTimes: 2,
			expected:   []string{"a:0", "a:1", "b:0"},
		},
		{
			name:       "a group with no usable channel is skipped without spending a retry",
			groups:     []string{"a", "b", "c"},
			levels:     map[string]int{"a": 1, "b": 0, "c": 2},
			retryTimes: 3,
			expected:   []string{"a:0", "c:0", "c:1"},
		},
		{
			name:       "the walk ends when the sequence runs out, even with budget left",
			groups:     []string{"a", "b"},
			levels:     map[string]int{"a": 1, "b": 1},
			retryTimes: 5,
			expected:   []string{"a:0", "b:0"},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t,
				testCase.expected,
				walkFailoverSequence(testCase.groups, testCase.levels, testCase.retryTimes),
			)
		})
	}
}

// The auto cross-group path resets the retry counter on every group switch, which makes
// the real attempt count groupCount * (RetryTimes+1). Failover must not do that.
func TestFailoverCursorNeverExceedsRetryBudget(t *testing.T) {
	const retryTimes = 3
	groups := []string{"a", "b", "c", "d"}
	levels := map[string]int{"a": 3, "b": 3, "c": 3, "d": 3}

	assert.Len(t, walkFailoverSequence(groups, levels, retryTimes), retryTimes+1)
}

func newFailoverTestContext(t *testing.T) *gin.Context {
	t.Helper()

	gin.SetMode(gin.TestMode)
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
	return ctx
}

func TestEffectiveRetryTimesClampsToSystemCap(t *testing.T) {
	previousRetryTimes := common.RetryTimes
	common.RetryTimes = 3
	t.Cleanup(func() { common.RetryTimes = previousRetryTimes })

	cases := []struct {
		name     string
		enabled  bool
		tokenMax int
		expected int
	}{
		{name: "failover off follows the system cap", enabled: false, tokenMax: 1, expected: 3},
		{name: "token may lower the cap", enabled: true, tokenMax: 1, expected: 1},
		{name: "token may not raise the cap", enabled: true, tokenMax: 9, expected: 3},
		{name: "unset token budget follows the system cap", enabled: true, tokenMax: 0, expected: 3},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			ctx := newFailoverTestContext(t)
			common.SetContextKey(ctx, constant.ContextKeyTokenFailoverEnabled, testCase.enabled)
			common.SetContextKey(ctx, constant.ContextKeyTokenFailoverMaxRetry, testCase.tokenMax)

			assert.Equal(t, testCase.expected, EffectiveRetryTimes(ctx))
		})
	}
}

func TestGetFailoverGroupsIgnoresUnsetAndMistypedContext(t *testing.T) {
	ctx := newFailoverTestContext(t)
	assert.Nil(t, GetFailoverGroups(ctx))

	common.SetContextKey(ctx, constant.ContextKeyTokenFailoverGroups, "default,vip")
	assert.Nil(t, GetFailoverGroups(ctx))

	common.SetContextKey(ctx, constant.ContextKeyTokenFailoverGroups, []string{"default", "vip"})
	require.Equal(t, []string{"default", "vip"}, GetFailoverGroups(ctx))
}
