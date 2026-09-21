package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"
	"github.com/schematichq/rulesengine/typeconvert"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A credit burndown entitlement is written by the API as a plan_entitlement
// rule (true, lt) and a plan_entitlement_usage_exceeded rule (false, gte), both
// carrying the same credit condition. The engine used to ignore the operator, so
// the exceeded rule could never match and a drained balance fell through to the
// flag default with no rule_type.
//
// Mirrors the credit entitlement tests in rulesengine-rust; the two engines must
// agree (SCHY-515) for as long as both are in use.
func TestCreditEntitlementExceededRule(t *testing.T) {
	ctx := context.Background()

	const creditID = "test-credit-id"

	creditCondition := func(operator typeconvert.ComparableOperator) *rulesengine.Condition {
		condition := createTestCondition(rulesengine.ConditionTypeCredit)
		condition.Operator = operator
		condition.CreditID = null.Nullable(creditID)
		condition.ConsumptionRate = null.Nullable(1.0)
		return condition
	}

	entitlementFlag := func() (*rulesengine.Flag, *rulesengine.Rule, *rulesengine.Rule) {
		entitled := createTestRule()
		entitled.RuleType = rulesengine.RuleTypePlanEntitlement
		entitled.Value = true
		entitled.Conditions = []*rulesengine.Condition{creditCondition(typeconvert.ComparableOperatorLt)}

		exceeded := createTestRule()
		exceeded.RuleType = rulesengine.RuleTypePlanEntitlementUsageExceeded
		exceeded.Value = false
		exceeded.Conditions = []*rulesengine.Condition{creditCondition(typeconvert.ComparableOperatorGte)}

		flag := createTestFlag()
		// A false default would hide a drained balance falling through to it.
		flag.DefaultValue = true
		flag.Rules = []*rulesengine.Rule{entitled, exceeded}
		return flag, entitled, exceeded
	}

	companyWith := func(balance float64) *rulesengine.Company {
		company := createTestCompany()
		company.CreditBalances = map[string]float64{creditID: balance}
		return company
	}

	t.Run("entitles while the balance covers the cost", func(t *testing.T) {
		flag, entitled, _ := entitlementFlag()

		result, err := rulesengine.CheckFlag(ctx, companyWith(5), nil, flag)

		require.NoError(t, err)
		assert.True(t, result.Value)
		assert.Equal(t, &entitled.ID, result.RuleID)
		assert.Equal(t, rulesengine.RuleTypePlanEntitlement, *result.RuleType)
	})

	t.Run("denies through the exceeded rule when drained", func(t *testing.T) {
		flag, _, exceeded := entitlementFlag()

		result, err := rulesengine.CheckFlag(ctx, companyWith(0.5), nil, flag)

		require.NoError(t, err)
		assert.False(t, result.Value, "a drained balance must deny, not fall through to the flag default")
		assert.Equal(t, &exceeded.ID, result.RuleID)
		assert.Equal(t, rulesengine.RuleTypePlanEntitlementUsageExceeded, *result.RuleType)
	})

	t.Run("denies through the exceeded rule with no balance row", func(t *testing.T) {
		flag, _, exceeded := entitlementFlag()

		result, err := rulesengine.CheckFlag(ctx, createTestCompany(), nil, flag)

		require.NoError(t, err)
		assert.False(t, result.Value)
		assert.Equal(t, &exceeded.ID, result.RuleID)
	})

	t.Run("exceeded rule honors a preflight credit cost", func(t *testing.T) {
		// Balance 10 covers one unit at rate 1 but not a 50-credit call.
		flag, _, exceeded := entitlementFlag()

		result, err := rulesengine.CheckFlag(ctx, companyWith(10), nil, flag, rulesengine.WithCreditCost(creditID, 50))

		require.NoError(t, err)
		assert.False(t, result.Value)
		assert.Equal(t, &exceeded.ID, result.RuleID)
	})

	// Unbounded postpaid removes the balance gate, so the exceeded rule must stay quiet
	// even at zero.
	t.Run("exceeded rule does not fire with unbounded postpaid", func(t *testing.T) {
		flag, entitled, _ := entitlementFlag()
		company := companyWith(0)
		company.CreditPostpaid = map[string]rulesengine.CreditPostpaidConfig{creditID: {}}

		result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

		require.NoError(t, err)
		assert.True(t, result.Value)
		assert.Equal(t, &entitled.ID, result.RuleID)
	})

	// A capped postpaid moves the floor to -cap rather than removing it, so the
	// exceeded rule fires once the cap is spent, the same as a drained balance
	// with postpaid off.
	t.Run("exceeded rule fires once a capped postpaid is spent", func(t *testing.T) {
		flag, entitled, exceeded := entitlementFlag()
		overdraftLimit := 10.0

		postpaid := map[string]rulesengine.CreditPostpaidConfig{creditID: {OverdraftLimit: &overdraftLimit}}

		company := companyWith(-5)
		company.CreditPostpaid = postpaid
		result, err := rulesengine.CheckFlag(ctx, company, nil, flag)
		require.NoError(t, err)
		assert.True(t, result.Value, "still inside the cap")
		assert.Equal(t, &entitled.ID, result.RuleID)

		company = companyWith(-10)
		company.CreditPostpaid = postpaid
		result, err = rulesengine.CheckFlag(ctx, company, nil, flag)
		require.NoError(t, err)
		assert.False(t, result.Value, "the cap is spent")
		assert.Equal(t, &exceeded.ID, result.RuleID)
		assert.Equal(t, rulesengine.RuleTypePlanEntitlementUsageExceeded, *result.RuleType)
	})
}
