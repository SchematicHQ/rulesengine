package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"
	"github.com/schematichq/rulesengine/typeconvert"
	"github.com/stretchr/testify/assert"
)

const spendBalanceID = "bcrd_spendpolicy"

// spendPolicyFlag builds a flag whose only rule is a credit-balance condition on
// creditID, so a check exercises exactly the spend-policy gate.
func spendPolicyFlag(creditID string, consumptionRate float64) *rulesengine.Flag {
	condition := createTestCondition(rulesengine.ConditionTypeCredit)
	condition.Operator = typeconvert.ComparableOperatorGte
	condition.CreditID = &creditID
	condition.ConsumptionRate = null.Nullable(consumptionRate)

	rule := createTestRule()
	rule.Conditions = []*rulesengine.Condition{condition}

	flag := createTestFlag()
	flag.DefaultValue = false
	flag.Rules = []*rulesengine.Rule{rule}
	return flag
}

func perDrawPolicy(limit float64, scope rulesengine.CreditSpendPolicyScope) *rulesengine.CreditSpendPolicy {
	return &rulesengine.CreditSpendPolicy{
		ID:       "csp_" + string(scope),
		CreditID: spendBalanceID,
		Kind:     rulesengine.CreditSpendPolicyKindPerDraw,
		Scope:    scope,
		Limit:    limit,
	}
}

func windowPolicy(limit float64, consumed float64) *rulesengine.CreditSpendPolicy {
	return &rulesengine.CreditSpendPolicy{
		ID:       "csp_window",
		CreditID: spendBalanceID,
		Kind:     rulesengine.CreditSpendPolicyKindWindow,
		Scope:    rulesengine.CreditSpendPolicyScopeCompany,
		Limit:    limit,
		Consumed: consumed,
		Window:   &rulesengine.CreditSpendWindow{Unit: "day", Count: 1},
	}
}

// companyWith returns a funded company carrying the given policies.
func companyWith(policies ...*rulesengine.CreditSpendPolicy) *rulesengine.Company {
	company := createTestCompany()
	company.CreditBalances = map[string]float64{spendBalanceID: 10000}
	company.CreditSpendPolicies = policies
	return company
}

// TestCreditSpendPolicyAllows pins the per-kind verdict in isolation, including
// the fail-open contract for a kind this engine does not recognise.
func TestCreditSpendPolicyAllows(t *testing.T) {
	tests := []struct {
		name          string
		policy        *rulesengine.CreditSpendPolicy
		cost          float64
		wantAllowed   bool
		wantEvaluated bool
	}{
		{"per draw under the limit", perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany), 5, true, true},
		{"per draw at the limit", perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany), 10, true, true},
		{"per draw over the limit", perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany), 11, false, true},
		{"per draw ignores nothing spent", perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany), 10, true, true},
		{"window with room", windowPolicy(100, 60), 40, true, true},
		{"window exactly full", windowPolicy(100, 60), 41, false, true},
		{"window already exhausted", windowPolicy(100, 100), 1, false, true},
		// A draw that would fit a per-draw cap of the same size is still
		// refused once the period is spent — the two kinds are not
		// interchangeable, which is why one number cannot represent both.
		{"window refuses a draw a per-draw cap would allow", windowPolicy(100, 95), 50, false, true},
		{
			name:          "an unrecognised kind is skipped rather than enforced",
			policy:        &rulesengine.CreditSpendPolicy{CreditID: spendBalanceID, Kind: "rolling_average", Limit: 1},
			cost:          1000,
			wantAllowed:   true,
			wantEvaluated: false,
		},
		{"a nil policy is skipped", nil, 1000, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, evaluated := tt.policy.Allows(tt.cost)
			assert.Equal(t, tt.wantAllowed, allowed)
			assert.Equal(t, tt.wantEvaluated, evaluated)
		})
	}
}

func TestCreditSpendPolicyRemaining(t *testing.T) {
	assert.Equal(t, float64(10), perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany).Remaining())
	assert.Equal(t, float64(40), windowPolicy(100, 60).Remaining())
	// Over-consumption clamps at zero rather than reporting negative headroom.
	assert.Equal(t, float64(0), windowPolicy(100, 140).Remaining())
}

func TestCreditSpendPolicyCheckFlag(t *testing.T) {
	ctx := context.Background()

	t.Run("no policy leaves the check on the balance alone", func(t *testing.T) {
		result, err := rulesengine.CheckFlag(ctx, companyWith(), nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 50))

		assert.NoError(t, err)
		assert.True(t, result.Value)
		assert.Nil(t, result.CreditSpendPolicy)
	})

	t.Run("a per-draw policy refuses an over-limit draw despite a funded balance", func(t *testing.T) {
		policy := perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany)

		result, err := rulesengine.CheckFlag(ctx, companyWith(policy), nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 50))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		if assert.NotNil(t, result.CreditSpendPolicy) {
			assert.Equal(t, float64(50), result.CreditSpendPolicy.Cost)
			assert.Equal(t, policy, result.CreditSpendPolicy.Policy)
		}
		assert.Equal(t, rulesengine.ReasonCreditSpendPolicyExceeded(50, policy), result.Reason)
	})

	t.Run("a window policy refuses once the period is spent", func(t *testing.T) {
		result, err := rulesengine.CheckFlag(ctx, companyWith(windowPolicy(100, 95)), nil,
			spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 10))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		if assert.NotNil(t, result.CreditSpendPolicy) {
			assert.Equal(t, rulesengine.CreditSpendPolicyKindWindow, result.CreditSpendPolicy.Policy.Kind)
		}
	})

	t.Run("a draw fitting every policy passes", func(t *testing.T) {
		result, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(50, rulesengine.CreditSpendPolicyScopeCompany), windowPolicy(100, 20)),
			nil, spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 40))

		assert.NoError(t, err)
		assert.True(t, result.Value)
		assert.Nil(t, result.CreditSpendPolicy)
	})

	t.Run("any refusal blocks, whichever policy it is", func(t *testing.T) {
		// The per-draw cap allows 40; the window has only 5 left. Neither is
		// "tighter" as a single number, so the engine refuses on the one that
		// actually fails.
		result, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(50, rulesengine.CreditSpendPolicyScopeCompany), windowPolicy(100, 95)),
			nil, spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 40))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		if assert.NotNil(t, result.CreditSpendPolicy) {
			assert.Equal(t, rulesengine.CreditSpendPolicyKindWindow, result.CreditSpendPolicy.Policy.Kind)
		}
	})

	t.Run("a user policy binds alongside the company's", func(t *testing.T) {
		user := createTestUser()
		user.CreditSpendPolicies = []*rulesengine.CreditSpendPolicy{
			perDrawPolicy(5, rulesengine.CreditSpendPolicyScopeUser),
		}

		result, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(500, rulesengine.CreditSpendPolicyScopeCompany)), user,
			spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 10))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		if assert.NotNil(t, result.CreditSpendPolicy) {
			assert.Equal(t, rulesengine.CreditSpendPolicyScopeUser, result.CreditSpendPolicy.Policy.Scope)
		}
	})

	t.Run("a user policy does not bind a check with no user", func(t *testing.T) {
		result, err := rulesengine.CheckFlag(ctx, companyWith(), nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 10))

		assert.NoError(t, err)
		assert.True(t, result.Value)
	})

	t.Run("a policy on another credit does not bind", func(t *testing.T) {
		other := perDrawPolicy(1, rulesengine.CreditSpendPolicyScopeCompany)
		other.CreditID = "bcrd_other"

		result, err := rulesengine.CheckFlag(ctx, companyWith(other), nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 50))

		assert.NoError(t, err)
		assert.True(t, result.Value)
	})

	t.Run("WithUsage draws are costed at quantity x consumption rate", func(t *testing.T) {
		// 4 units x 2 credits = 8, under the limit of 10.
		allowed, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany)), nil,
			spendPolicyFlag(spendBalanceID, 2), rulesengine.WithUsage(4))
		assert.NoError(t, err)
		assert.True(t, allowed.Value)

		// 6 units x 2 credits = 12, over it.
		refused, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany)), nil,
			spendPolicyFlag(spendBalanceID, 2), rulesengine.WithUsage(6))
		assert.NoError(t, err)
		assert.False(t, refused.Value)
		if assert.NotNil(t, refused.CreditSpendPolicy) {
			assert.Equal(t, float64(12), refused.CreditSpendPolicy.Cost)
		}
	})

	t.Run("a single-unit check is refused by a limit below the consumption rate", func(t *testing.T) {
		result, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(1, rulesengine.CreditSpendPolicyScopeCompany)), nil,
			spendPolicyFlag(spendBalanceID, 5))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		if assert.NotNil(t, result.CreditSpendPolicy) {
			assert.Equal(t, float64(5), result.CreditSpendPolicy.Cost)
		}
	})

	t.Run("a zero limit refuses every non-free draw", func(t *testing.T) {
		refused, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(0, rulesengine.CreditSpendPolicyScopeCompany)), nil,
			spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 1))
		assert.NoError(t, err)
		assert.False(t, refused.Value)

		free, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(0, rulesengine.CreditSpendPolicyScopeCompany)), nil,
			spendPolicyFlag(spendBalanceID, 1), rulesengine.WithCreditCost(spendBalanceID, 0))
		assert.NoError(t, err)
		assert.True(t, free.Value)
	})

	t.Run("an unrecognised kind lets the draw through", func(t *testing.T) {
		unknown := &rulesengine.CreditSpendPolicy{CreditID: spendBalanceID, Kind: "rolling_average", Limit: 1}

		result, err := rulesengine.CheckFlag(ctx, companyWith(unknown), nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 500))

		assert.NoError(t, err)
		assert.True(t, result.Value)
		assert.Nil(t, result.CreditSpendPolicy)
	})

	t.Run("an insufficient balance still reports no rules matched", func(t *testing.T) {
		company := companyWith(perDrawPolicy(100, rulesengine.CreditSpendPolicyScopeCompany))
		company.CreditBalances = map[string]float64{spendBalanceID: 1}

		result, err := rulesengine.CheckFlag(ctx, company, nil, spendPolicyFlag(spendBalanceID, 1),
			rulesengine.WithCreditCost(spendBalanceID, 50))

		assert.NoError(t, err)
		assert.False(t, result.Value)
		assert.Nil(t, result.CreditSpendPolicy)
		assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
	})

	t.Run("a later matching rule wins over an earlier refusal", func(t *testing.T) {
		flag := spendPolicyFlag(spendBalanceID, 1)
		override := createTestRule()
		override.RuleType = rulesengine.RuleTypeGlobalOverride
		override.Value = true
		flag.Rules = append(flag.Rules, override)

		result, err := rulesengine.CheckFlag(ctx,
			companyWith(perDrawPolicy(1, rulesengine.CreditSpendPolicyScopeCompany)), nil, flag,
			rulesengine.WithCreditCost(spendBalanceID, 50))

		assert.NoError(t, err)
		assert.True(t, result.Value)
		assert.Nil(t, result.CreditSpendPolicy)
	})
}

func TestCreditSpendPolicyDescribe(t *testing.T) {
	assert.Equal(t, "company limit of 10 credits per request",
		perDrawPolicy(10, rulesengine.CreditSpendPolicyScopeCompany).Describe())
	assert.Equal(t, "company limit of 100 credits per day", windowPolicy(100, 0).Describe())

	multi := windowPolicy(100, 0)
	multi.Window = &rulesengine.CreditSpendWindow{Unit: "hour", Count: 6}
	assert.Equal(t, "company limit of 100 credits per 6 hours", multi.Describe())
}
