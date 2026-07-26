package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupFailoverValidationContext gives normalizeTokenFailover the state it reads:
// a user row so GetUserGroup resolves a group, and a system retry cap.
func setupFailoverValidationContext(t *testing.T, systemRetryTimes int) {
	t.Helper()

	initModelListColumnNames(t)
	db := openTokenControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Token{}))
	require.NoError(t, db.Create(&model.User{
		Id: 1, Username: "failover-user", Password: "password",
		AffCode: "failover-aff", Status: 1, Group: "default",
	}).Error)

	previousRetryTimes := common.RetryTimes
	common.RetryTimes = systemRetryTimes
	t.Cleanup(func() { common.RetryTimes = previousRetryTimes })
}

func TestNormalizeTokenFailoverRejectsInvalidConfigurations(t *testing.T) {
	// The default usable groups are "default" and "vip"; "premium" is out of reach.
	cases := []struct {
		name  string
		token model.Token
	}{
		{
			name: "retry budget above the system cap",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","vip"]`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 4,
			},
		},
		{
			name: "retry budget below one",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","vip"]`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 0,
			},
		},
		{
			name: "group the user may not access",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","premium"]`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 2,
			},
		},
		{
			name: "auto group nested inside the sequence",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","auto"]`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 2,
			},
		},
		{
			name: "fewer than two distinct groups",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","default"]`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 2,
			},
		},
		{
			name: "unknown strategy",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `["default","vip"]`,
				FailoverStrategy: "cheapest",
				FailoverMaxRetry: 2,
			},
		},
		{
			name: "malformed group list",
			token: model.Token{
				FailoverEnabled:  true,
				FailoverGroups:   `not json`,
				FailoverStrategy: "order",
				FailoverMaxRetry: 2,
			},
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			setupFailoverValidationContext(t, 3)
			ctx, _ := newAuthenticatedContext(t, http.MethodPost, "/api/token/", nil, 1)

			token := testCase.token
			assert.Error(t, normalizeTokenFailover(ctx, &token))
		})
	}
}

func TestNormalizeTokenFailoverRejectsWhenSystemRetriesAreOff(t *testing.T) {
	setupFailoverValidationContext(t, 0)
	ctx, _ := newAuthenticatedContext(t, http.MethodPost, "/api/token/", nil, 1)

	token := model.Token{
		FailoverEnabled:  true,
		FailoverGroups:   `["default","vip"]`,
		FailoverStrategy: "order",
		FailoverMaxRetry: 1,
	}
	assert.Error(t, normalizeTokenFailover(ctx, &token))
}

func TestNormalizeTokenFailoverCanonicalizesValidConfiguration(t *testing.T) {
	setupFailoverValidationContext(t, 3)
	ctx, _ := newAuthenticatedContext(t, http.MethodPost, "/api/token/", nil, 1)

	token := model.Token{
		Group:            "auto",
		CrossGroupRetry:  true,
		FailoverEnabled:  true,
		FailoverGroups:   `[" default ","vip","default"]`,
		FailoverStrategy: "lowest_ratio",
		FailoverMaxRetry: 3,
	}
	require.NoError(t, normalizeTokenFailover(ctx, &token))

	assert.Equal(t, `["default","vip"]`, token.FailoverGroups)
	// Failover supersedes the auto group's cross-group retry, but the single group
	// stays on the record so turning failover off restores the previous setup.
	assert.False(t, token.CrossGroupRetry)
	assert.Equal(t, "auto", token.Group)
}

func TestNormalizeTokenFailoverClearsSettingsWhenDisabled(t *testing.T) {
	setupFailoverValidationContext(t, 3)
	ctx, _ := newAuthenticatedContext(t, http.MethodPost, "/api/token/", nil, 1)

	token := model.Token{
		FailoverEnabled:  false,
		FailoverGroups:   `["default","vip"]`,
		FailoverStrategy: "order",
		FailoverMaxRetry: 2,
	}
	require.NoError(t, normalizeTokenFailover(ctx, &token))

	assert.Empty(t, token.FailoverGroups)
	assert.Empty(t, token.FailoverStrategy)
	assert.Zero(t, token.FailoverMaxRetry)
}
