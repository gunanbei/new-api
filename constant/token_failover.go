package constant

// Token failover strategies decide the order in which the token's configured groups
// are tried when failover mode is enabled.
const (
	// FailoverStrategyOrder follows the group order the user configured.
	FailoverStrategyOrder = "order"
	// FailoverStrategyLowestRatio tries the cheapest group for this user first.
	FailoverStrategyLowestRatio = "lowest_ratio"
	// FailoverStrategyRandom shuffles the groups once per request.
	FailoverStrategyRandom = "random"
)

// MaxFailoverGroups bounds how many groups a single token may chain together.
const MaxFailoverGroups = 10

// MinFailoverGroups is the smallest list that can actually fail over.
const MinFailoverGroups = 2

func IsValidFailoverStrategy(strategy string) bool {
	switch strategy {
	case FailoverStrategyOrder, FailoverStrategyLowestRatio, FailoverStrategyRandom:
		return true
	}
	return false
}
