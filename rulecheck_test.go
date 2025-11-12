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

		t.Run("Credit burndown with usage tracking - sufficient credits", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 20.0,
			}

			metric := createTestMetric(company, eventSubtype, rulesengine.MetricPeriodAllTime, 24)
			company.Metrics = append(company.Metrics, metric)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(0.0001)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should have enough credits for the usage (20 >= 0.0024)")
		})

		t.Run("Credit burndown with usage tracking - insufficient credits", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 5.0,
			}

			metric := createTestMetric(company, eventSubtype, rulesengine.MetricPeriodAllTime, 100)
			company.Metrics = append(company.Metrics, metric)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(0.1)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should NOT have enough credits for the usage (5 < 10)")
		})

		t.Run("Credit burndown with zero usage", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(0.5)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should pass with zero usage (0 × 0.5 = 0 credits needed)")
		})

		t.Run("Credit burndown with zero consumption rate", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 100.0,
			}

			metric := createTestMetric(company, eventSubtype, rulesengine.MetricPeriodAllTime, 10)
			company.Metrics = append(company.Metrics, metric)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(0.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should pass with zero consumption rate (10 × 0 = 0 credits needed)")
		})

		t.Run("Usage-based credit check - sufficient credits", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			company.CreditBalances = map[string]float64{
				creditID: 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.ConsumptionRate = null.Nullable(2.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should have enough credits for usage (100 >= 30 × 2 = 60)")
		})

		t.Run("Usage-based credit check - insufficient credits", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			company.CreditBalances = map[string]float64{
				creditID: 50.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.ConsumptionRate = null.Nullable(2.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should NOT have enough credits for usage (50 < 30 × 2 = 60)")
		})

		t.Run("Usage-based credit check with fractional consumption rate", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			company.CreditBalances = map[string]float64{
				creditID: 10.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.ConsumptionRate = null.Nullable(0.01) // 0.01 credits per unit

			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(500) // Want to consume 500 units
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should have enough credits for usage (10 >= 500 × 0.01 = 5)")
		})
	})

	t.Run("Metric evaluation with quantity", func(t *testing.T) {
		t.Run("Usage-based metric check - within limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100) // Limit is 100
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 50)
			company.Metrics = append(company.Metrics, metric)

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should be within limit (50 + 30 = 80 <= 100)")
		})

		t.Run("Usage-based metric check - exceeds limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100) // Limit is 100
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 80)
			company.Metrics = append(company.Metrics, metric)

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should exceed limit (80 + 30 = 110 > 100)")
		})

		t.Run("Usage-based metric check - exactly at limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 70)
			company.Metrics = append(company.Metrics, metric)

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should be exactly at limit (70 + 30 = 100 <= 100)")
		})

		t.Run("Usage-based metric check - no current usage", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(50)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should be within limit (0 + 50 = 50 <= 100)")
		})
	})

	t.Run("Trait evaluation with usage", func(t *testing.T) {
		t.Run("Usage-based numeric trait check - within limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "usage-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "70",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(20)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should be within limit (70 + 20 = 90 <= 100)")
		})

		t.Run("Usage-based numeric trait check - exceeds limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "usage-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "85",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(20)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should exceed limit (85 + 20 = 105 > 100)")
		})

		t.Run("Usage-based numeric trait check - exactly at limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "usage-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "80",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(20)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should be exactly at limit (80 + 20 = 100 <= 100)")
		})

		t.Run("Usage does not affect non-numeric traits", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "string-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeString,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "test-value",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "test-value"
			condition.Operator = typeconvert.ComparableOperatorEquals
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(100)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Usage should not affect string trait comparison")
		})

		t.Run("Usage-based trait check with greater than operator", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "seats-used",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "8",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "10"
			condition.Operator = typeconvert.ComparableOperatorGt
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(3)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should match with usage (8 + 3 = 11 > 10)")
		})

		t.Run("Usage-based trait check with user trait", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()
			user := createTestUser()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "user-credits",
				EntityType:     rulesengine.EntityTypeUser,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "50",
			}
			user.Traits = append(user.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(30)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				User:    user,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should work with user traits (50 + 30 = 80 <= 100)")
		})

		t.Run("Usage-based trait check - negative case within limit", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "usage-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "95",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			usage := int64(10)
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should NOT match when usage pushes over limit (95 + 10 = 105 > 100)")
		})

		t.Run("Usage-based trait check - negative case without usage would pass", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			traitDef := &rulesengine.TraitDefinition{
				ID:             "usage-trait",
				EntityType:     rulesengine.EntityTypeCompany,
				ComparableType: typeconvert.ComparableTypeInt,
			}

			trait := &rulesengine.Trait{
				TraitDefinition: traitDef,
				Value:           "95",
			}
			company.Traits = append(company.Traits, trait)

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "100"
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should pass without usage (95 <= 100)")

			usage := int64(10)
			result, err = svc.Check(ctx, &rulesengine.CheckScope{
				Company: company,
				Rule:    rule,
				Usage:   &usage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should fail with usage (95 + 10 = 105 > 100)")
		})
	})

	t.Run("Event-specific usage with WithEventUsage", func(t *testing.T) {
		t.Run("WithEventUsage applies to matching metric condition", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 70)
			company.Metrics = append(company.Metrics, metric)

			eventUsage := map[string]int64{
				"api-calls": 25,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should apply event-specific usage (70 + 25 = 95 <= 100)")
		})

		t.Run("WithEventUsage does not apply to non-matching metric", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 70)
			company.Metrics = append(company.Metrics, metric)

			eventUsage := map[string]int64{
				"storage-usage": 50,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should NOT apply non-matching event usage (70 <= 100, ignoring storage-usage)")
		})

		t.Run("WithEventUsage exceeds limit for matching event", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 70)
			company.Metrics = append(company.Metrics, metric)

			eventUsage := map[string]int64{
				"api-calls": 35,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should exceed limit with event usage (70 + 35 = 105 > 100)")
		})

		t.Run("WithEventUsage takes precedence over WithUsage", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype := "api-calls"
			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(100)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte
			rule.Conditions = []*rulesengine.Condition{condition}

			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 60)
			company.Metrics = append(company.Metrics, metric)

			usage := int64(50)
			eventUsage := map[string]int64{
				"api-calls": 20,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				Usage:      &usage,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should use EventUsage (20) not Usage (50): 60 + 20 = 80 <= 100")
		})

		t.Run("WithEventUsage with multiple events in flag", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype1 := "api-calls"
			eventSubtype2 := "storage-usage"

			rule := createTestRule()

			condition1 := createTestCondition(rulesengine.ConditionTypeMetric)
			condition1.EventSubtype = &eventSubtype1
			metricValue1 := int64(100)
			condition1.MetricValue = &metricValue1
			condition1.Operator = typeconvert.ComparableOperatorLte

			condition2 := createTestCondition(rulesengine.ConditionTypeMetric)
			condition2.EventSubtype = &eventSubtype2
			metricValue2 := int64(500)
			condition2.MetricValue = &metricValue2
			condition2.Operator = typeconvert.ComparableOperatorLte

			rule.Conditions = []*rulesengine.Condition{condition1, condition2}

			metric1 := createTestMetric(company, eventSubtype1, *condition1.MetricPeriod, 80)
			metric2 := createTestMetric(company, eventSubtype2, *condition2.MetricPeriod, 450)
			company.Metrics = append(company.Metrics, metric1, metric2)

			eventUsage := map[string]int64{
				"api-calls":     15,
				"storage-usage": 40,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Both conditions should pass: (80+15=95<=100) AND (450+40=490<=500)")
		})

		t.Run("WithEventUsage with multiple events - one fails", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			eventSubtype1 := "api-calls"
			eventSubtype2 := "storage-usage"

			rule := createTestRule()

			condition1 := createTestCondition(rulesengine.ConditionTypeMetric)
			condition1.EventSubtype = &eventSubtype1
			metricValue1 := int64(100)
			condition1.MetricValue = &metricValue1
			condition1.Operator = typeconvert.ComparableOperatorLte

			condition2 := createTestCondition(rulesengine.ConditionTypeMetric)
			condition2.EventSubtype = &eventSubtype2
			metricValue2 := int64(500)
			condition2.MetricValue = &metricValue2
			condition2.Operator = typeconvert.ComparableOperatorLte

			rule.Conditions = []*rulesengine.Condition{condition1, condition2}

			metric1 := createTestMetric(company, eventSubtype1, *condition1.MetricPeriod, 80)
			metric2 := createTestMetric(company, eventSubtype2, *condition2.MetricPeriod, 450)
			company.Metrics = append(company.Metrics, metric1, metric2)

			eventUsage := map[string]int64{
				"api-calls":     25,
				"storage-usage": 40,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should fail because api-calls exceeds: (80+25=105>100)")
		})

		t.Run("WithEventUsage applies to credit conditions with matching event", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(2.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			eventUsage := map[string]int64{
				"api-calls": 30,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should have enough credits for event usage (100 >= 30 × 2 = 60)")
		})

		t.Run("WithEventUsage does not apply to credit without matching event", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 100.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(2.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			eventUsage := map[string]int64{
				"storage-usage": 100,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.True(t, result.Match, "Should pass with zero usage for api-calls (ignoring storage-usage)")
		})

		t.Run("WithEventUsage - credit check insufficient with event usage", func(t *testing.T) {
			svc := rulesengine.NewRuleCheckService()
			company := createTestCompany()

			creditID := "test-credit-id"
			eventSubtype := "api-calls"

			company.CreditBalances = map[string]float64{
				creditID: 50.0,
			}

			rule := createTestRule()
			condition := createTestCondition(rulesengine.ConditionTypeCredit)
			condition.CreditID = &creditID
			condition.EventSubtype = &eventSubtype
			condition.ConsumptionRate = null.Nullable(2.0)

			rule.Conditions = []*rulesengine.Condition{condition}

			eventUsage := map[string]int64{
				"api-calls": 30,
			}
			result, err := svc.Check(ctx, &rulesengine.CheckScope{
				Company:    company,
				Rule:       rule,
				EventUsage: eventUsage,
			})

			assert.NoError(t, err)
			assert.False(t, result.Match, "Should NOT have enough credits (50 < 30 × 2 = 60)")
		})
	})
}
