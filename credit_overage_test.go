package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// SCHX-582: credit overage. With overage enabled on a credit, consumption
// continues past a zero balance and accrues at a configured rate, so the balance
// stops gating the check.
//
// These mirror credit_overage_tests in rulesengine-rust; the two engines must
// agree (SCHY-515) for as long as both are in use.
func TestCreditOverage(t *testing.T) {
	ctx := context.Background()

	const creditID = "test-credit-id"

	// overage: nil leaves the credit out of the map entirely (off); non-nil puts
	// it in, with the pointed-at value as the cap — nil cap meaning uncapped.
	companyWith := func(balance float64, overage *bool) *rulesengine.Company {
		company := createTestCompany()
		company.CreditBalances = map[string]float64{creditID: balance}
		if overage != nil && *overage {
			company.CreditOverage = map[string]*float64{creditID: nil}
		} else if overage != nil {
			company.CreditOverage = map[string]*float64{}
		}
		return company
	}

	companyWithCap := func(balance float64, cap float64) *rulesengine.Company {
		enabled := true
		company := companyWith(balance, &enabled)
		company.CreditOverage = map[string]*float64{creditID: &cap}
		return company
	}

	creditRule := func() *rulesengine.Rule {
		rule := createTestRule()
		condition := createTestCondition(rulesengine.ConditionTypeCredit)
		condition.CreditID = null.Nullable(creditID)
		condition.ConsumptionRate = null.Nullable(1.0)
		rule.Conditions = []*rulesengine.Condition{condition}
		return rule
	}

	matches := func(t *testing.T, company *rulesengine.Company) bool {
		t.Helper()

		result, err := rulesengine.NewRuleCheckService().Check(ctx, &rulesengine.CheckScope{
			Company: company,
			Rule:    creditRule(),
		})
		require.NoError(t, err)
		return result.Match
	}

	// The existing hard stop, unchanged when nobody has opted in.
	t.Run("denies at zero without overage", func(t *testing.T) {
		assert.False(t, matches(t, companyWith(0, nil)))
	})

	t.Run("allows with balance", func(t *testing.T) {
		assert.True(t, matches(t, companyWith(5, nil)))
	})

	// The point of the feature.
	t.Run("allows past zero with overage enabled", func(t *testing.T) {
		assert.True(t, matches(t, companyWith(0, null.Nullable(true))))
	})

	// A negative balance is legal (SCH-5103); overage is what makes it billable,
	// and an already-overdrafted company must keep working.
	t.Run("allows when already negative", func(t *testing.T) {
		assert.True(t, matches(t, companyWith(-40, null.Nullable(true))))
	})

	// Explicitly false must behave exactly like absent, or switching the setting
	// back off would not restore the hard stop.
	t.Run("explicit false still denies", func(t *testing.T) {
		assert.False(t, matches(t, companyWith(0, null.Nullable(false))))
	})

	// Overage is per credit: enabling it on one must not unblock another.
	t.Run("does not leak across credits", func(t *testing.T) {
		company := companyWith(0, null.Nullable(true))
		company.CreditBalances["other-credit-id"] = 0

		rule := createTestRule()
		condition := createTestCondition(rulesengine.ConditionTypeCredit)
		condition.CreditID = null.Nullable("other-credit-id")
		condition.ConsumptionRate = null.Nullable(1.0)
		rule.Conditions = []*rulesengine.Condition{condition}

		result, err := rulesengine.NewRuleCheckService().Check(ctx, &rulesengine.CheckScope{
			Company: company,
			Rule:    rule,
		})
		require.NoError(t, err)
		assert.False(t, result.Match, "the other credit has no overage and no balance")
	})

	// Overage has to beat the more specific branches too. A caller-supplied
	// credit cost would otherwise re-impose the balance gate it short-circuits.
	t.Run("overrides the credit cost option", func(t *testing.T) {
		flag := createTestFlag()
		// Pinned false: createTestFlag randomizes DefaultValue, and CheckFlag falls
		// back to it when no rule matches — so a true default would let this pass
		// without the rule ever matching.
		flag.DefaultValue = false
		flag.Rules = []*rulesengine.Rule{creditRule()}

		result, err := rulesengine.CheckFlag(
			ctx,
			companyWith(0, null.Nullable(true)),
			nil,
			flag,
			rulesengine.WithCreditCost(creditID, 25),
		)
		require.NoError(t, err)
		assert.True(t, result.Value)
	})

	// SCHX-582 cap: the floor moves from zero to -cap rather than disappearing.
	t.Run("allows while inside the cap", func(t *testing.T) {
		assert.True(t, matches(t, companyWithCap(-40, 100)))
	})

	t.Run("denies once the cap is spent", func(t *testing.T) {
		assert.False(t, matches(t, companyWithCap(-100, 100)))
	})

	t.Run("denies past the cap", func(t *testing.T) {
		assert.False(t, matches(t, companyWithCap(-140, 100)))
	})

	// A cap on one credit must not bound a different one.
	t.Run("cap does not leak across credits", func(t *testing.T) {
		company := companyWithCap(-140, 100)
		otherCap := 100.0
		company.CreditOverage = map[string]*float64{
			creditID:       nil,
			"other-credit": &otherCap,
		}
		assert.True(t, matches(t, company))
	})

	// Absent cap keeps the uncapped behaviour that shipped first.
	t.Run("no cap means uncapped", func(t *testing.T) {
		enabled := true
		assert.True(t, matches(t, companyWith(-10_000, &enabled)))
	})
}
