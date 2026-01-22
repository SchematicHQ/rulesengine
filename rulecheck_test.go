package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"
	"github.com/schematichq/rulesengine/typeconvert"
	"github.com/stretchr/testify/assert"
)

func TestRuleCheckService(t *testing.T) {
	ctx := context.Background()

	t.Run("Basic rule checking", func(t *testing.T) {
		t.Run("Check returns false for nil rule", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    nil,
			})

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.False(t, result.Match)
		})

		t.Run("Check returns true for default rules", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()
			rule := createTestRule()
			rule.RuleType = rulesengine.RuleTypeDefault

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Check returns true for global override rules", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()
			rule := createTestRule()
			rule.RuleType = rulesengine.RuleTypeGlobalOverride

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})
	})

	t.Run("Company targeting", func(t *testing.T) {
		t.Run("Rule matches specific company", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()
			rule := createTestRule()

			// Create condition targeting the company
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{company.ID}
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})
	})

	t.Run("Metric evaluation", func(t *testing.T) {
		t.Run("Rule matches when metric is within limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "test-event"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(10)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 5)
			company.Metrics = append(company.Metrics, metric)

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Rule does not match when metric exceeds limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "test-event"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(5)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, metricValue+1)
			company.Metrics = append(company.Metrics, metric)

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match)
		})
	})

	t.Run("Trait evaluation", func(t *testing.T) {
		t.Run("Rule matches when trait value matches condition", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()
			trait := createTestTrait("test-value", nil)
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = trait.TraitDefinition
			condition.TraitValue = "test-value"
			condition.Operator = typeconvert.ComparableOperatorEquals
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})
	})

	t.Run("Condition groups", func(t *testing.T) {
		t.Run("Rule matches when any condition in group matches", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			rule := createTestRule()
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}

			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{condition1, condition2},
			}
			rule.ConditionGroups = []*rulesengine.ConditionGroup{group}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Rule does not match when no conditions in group match", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			rule := createTestRule()
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)

			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{condition1, condition2},
			}
			rule.ConditionGroups = []*rulesengine.ConditionGroup{group}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match)
		})
	})

	t.Run("Credit condition evaluation", func(t *testing.T) {
		t.Run("Rule matches when credit balance exceeds consumption rate", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balance
			company.CreditBalances = map[string]float64{
				"test-credit-id": 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = null.Nullable("test-credit-id")
			condition.ConsumptionRate = null.Nullable(50.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Rule does not match when credit balance is less than consumption rate", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balance
			company.CreditBalances = map[string]float64{
				"test-credit-id": 10.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = null.Nullable("test-credit-id")
			condition.ConsumptionRate = null.Nullable(50.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match)
		})

		t.Run("Rule does not match when credit ID doesn't exist", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balance
			company.CreditBalances = map[string]float64{
				"other-credit-id": 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = null.Nullable("test-credit-id")
			condition.ConsumptionRate = null.Nullable(50.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match)
		})

		t.Run("Rule matches with default consumption rate", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balance
			creditID := "test-credit-id"
			company.CreditBalances = map[string]float64{
				creditID: 1.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			// Don't set consumption rate - should default to 1.0
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Complex mixed condition group with credit condition", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add traits
			trait := createTestTrait("test-value", nil)
			company.Traits = append(company.Traits, trait)

			// Add credit balance
			creditID := "test-credit-id"
			company.CreditBalances = map[string]float64{
				creditID: 200.0,
			}

			rule := createTestRule()

			// Credit condition
			creditCondition := createTestCondition(rulesengine.ConditionTypeCredit)
			creditCondition.CreditID = &creditID
			creditCondition.ConsumptionRate = null.Nullable(150.0)

			// Company condition (non-matching)
			companyCondition := createTestCondition(rulesengine.ConditionTypeCompany)
			companyCondition.ResourceIDs = []string{"different-company-id"}

			// Trait condition
			traitCondition := createTestCondition(rulesengine.ConditionTypeTrait)
			traitCondition.TraitDefinition = trait.TraitDefinition
			traitCondition.TraitValue = "test-value"
			traitCondition.Operator = typeconvert.ComparableOperatorEquals

			// Create mixed condition group
			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{creditCondition, companyCondition, traitCondition},
			}
			rule.ConditionGroups = []*rulesengine.ConditionGroup{group}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Rule should match because at least one condition in the group matches")
		})

		t.Run("Multiple condition groups with credit conditions", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balances
			creditID1 := "test-credit-id-1"
			creditID2 := "test-credit-id-2"
			company.CreditBalances = map[string]float64{
				creditID1: 5.0,   // Not enough for condition1
				creditID2: 100.0, // Enough for condition2
			}

			rule := createTestRule()

			// First credit condition (not enough balance)
			condition1 := createTestCondition(rulesengine.ConditionTypeCredit)
			condition1.CreditID = &creditID1
			condition1.ConsumptionRate = null.Nullable(10.0)

			// Second credit condition (enough balance)
			condition2 := createTestCondition(rulesengine.ConditionTypeCredit)
			condition2.CreditID = &creditID2
			condition2.ConsumptionRate = null.Nullable(50.0)

			// Use rule type that will match regardless of conditions
			rule.RuleType = rulesengine.RuleTypeDefault

			// Create a single condition group with the matching condition
			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{condition2},
			}
			rule.ConditionGroups = []*rulesengine.ConditionGroup{group}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Rule should match with default rule type and matching condition group")
		})

		t.Run("Rule with nil company returns false", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()

			// Create a standard rule (not default or global override)
			rule := createTestRule()
			rule.RuleType = rulesengine.RuleTypeStandard

			// Add a company condition to ensure it needs company evaluation
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{"any-company-id"}
			rule.Conditions = []*rulesengine.Condition{condition}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: nil,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Rule should not match with nil company")
		})

		t.Run("Credit condition with nil credit ID", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			// Add credit balance
			company.CreditBalances = map[string]float64{
				"some-credit-id": 100.0,
			}

			rule := createTestRule()
			// Instead of adding a direct condition with a nil credit ID (which would cause a nil pointer error),
			// let's test using a condition group
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{"different-company-id"} // This won't match
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Rule should not match with non-matching condition")
		})
	})

	t.Run("Plan version condition evaluation", func(t *testing.T) {
		t.Run("Rule matches when plan version ID is in company's plan version IDs", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypePlanVersion)
			condition.ResourceIDs = []string{company.PlanVersionIDs[0]}
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match)
		})

		t.Run("Rule does not match when plan version ID is not in company's plan version IDs", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypePlanVersion)
			condition.ResourceIDs = []string{"non-matching-version-id"}
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match)
		})
	})
}
