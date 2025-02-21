package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
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
}
