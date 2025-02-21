package rulesengine_test

import (
	"context"
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/typeconvert"
	"github.com/stretchr/testify/assert"
)

func TestCheckFlag(t *testing.T) {
	ctx := context.Background()

	t.Run("Basic flag checks", func(t *testing.T) {
		t.Run("Returns error result when flag is nil", func(t *testing.T) {
			company := createTestCompany()

			result, err := rulesengine.CheckFlag(ctx, company, nil, nil)

			assert.NoError(t, err)
			assert.Equal(t, rulesengine.ReasonFlagNotFound, result.Reason)
			assert.Equal(t, rulesengine.ErrorFlagNotFound, result.Err)
		})

		t.Run("Returns default value when no rules match", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = true

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
			assert.True(t, result.Value)
			assert.Equal(t, &company.ID, result.CompanyID)
		})

		t.Run("Returns first matching rule's value", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = false

			rule1 := createTestRule()
			rule1.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{company.ID}
			rule1.Conditions = append(rule1.Conditions, condition)

			flag.Rules = append(flag.Rules, rule1)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Contains(t, result.Reason, "Matched standard rule")
			assert.Equal(t, &rule1.ID, result.RuleID)
		})
	})

	t.Run("Rule prioritization", func(t *testing.T) {
		t.Run("Global override takes precedence", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create a standard rule that matches
			standardRule := createTestRule()
			standardRule.Value = false
			standardCondition := createTestCondition(rulesengine.ConditionTypeCompany)
			standardCondition.ResourceIDs = []string{company.ID}
			standardRule.Conditions = append(standardRule.Conditions, standardCondition)

			// Create a global override rule
			overrideRule := createTestRule()
			overrideRule.RuleType = rulesengine.RuleTypeGlobalOverride
			overrideRule.Value = true

			flag.Rules = append(flag.Rules, standardRule, overrideRule)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &overrideRule.ID, result.RuleID)
		})

		t.Run("Rules evaluated in priority order", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create two matching rules with different priorities
			rule1 := createTestRule()
			rule1.Priority = 2
			rule1.Value = false
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			rule1.Conditions = append(rule1.Conditions, condition1)

			rule2 := createTestRule()
			rule2.Priority = 1 // Lower priority number = higher priority
			rule2.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}
			rule2.Conditions = append(rule2.Conditions, condition2)

			flag.Rules = append(flag.Rules, rule1, rule2)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule2.ID, result.RuleID)
		})
	})

	t.Run("Condition groups", func(t *testing.T) {
		t.Run("Matches when any condition in group matches", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			rule := createTestRule()
			rule.Value = true

			// Create condition group with two conditions
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{"non-matching-id"}

			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}

			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{condition1, condition2},
			}

			rule.ConditionGroups = append(rule.ConditionGroups, group)
			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule.ID, result.RuleID)
		})
	})

	t.Run("Entitlement rules", func(t *testing.T) {
		t.Run("Sets usage and allocation for metric condition", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create entitlement rule with metric condition
			eventSubtype := "test-event"
			rule := createTestRule()
			rule.RuleType = rulesengine.RuleTypePlanEntitlement
			rule.Value = true

			condition := createTestCondition(rulesengine.ConditionTypeMetric)
			condition.EventSubtype = &eventSubtype
			metricValue := int64(10)
			condition.MetricValue = &metricValue
			condition.Operator = typeconvert.ComparableOperatorLte

			rule.Conditions = append(rule.Conditions, condition)
			flag.Rules = append(flag.Rules, rule)

			// Create company metric
			metric := createTestMetric(company, eventSubtype, *condition.MetricPeriod, 5)
			metric.EventSubtype = eventSubtype
			company.Metrics = append(company.Metrics, metric)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule.ID, result.RuleID)
			assert.NotNil(t, result.FeatureUsage)
			assert.Equal(t, int64(5), *result.FeatureUsage)
			assert.NotNil(t, result.FeatureAllocation)
			assert.Equal(t, int64(10), *result.FeatureAllocation)
		})

		t.Run("Sets usage and allocation for trait condition", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create trait
			traitDef := createTestTraitDefinition(typeconvert.ComparableTypeInt, rulesengine.EntityTypeCompany)
			trait := createTestTrait("5", traitDef)
			company.Traits = append(company.Traits, trait)

			// Create entitlement rule with trait condition
			rule := createTestRule()
			rule.RuleType = rulesengine.RuleTypePlanEntitlement
			rule.Value = true

			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "10"
			condition.Operator = typeconvert.ComparableOperatorLte

			rule.Conditions = append(rule.Conditions, condition)
			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule.ID, result.RuleID)
			assert.NotNil(t, result.FeatureUsage)
			assert.Equal(t, int64(5), *result.FeatureUsage)
			assert.NotNil(t, result.FeatureAllocation)
			assert.Equal(t, int64(10), *result.FeatureAllocation)
		})
	})

	t.Run("User context", func(t *testing.T) {
		t.Run("Matches user-specific conditions", func(t *testing.T) {
			user := createTestUser()
			flag := createTestFlag()

			rule := createTestRule()
			rule.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeUser)
			condition.ResourceIDs = []string{user.ID}
			rule.Conditions = append(rule.Conditions, condition)

			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &user.ID, result.UserID)
			assert.Equal(t, &rule.ID, result.RuleID)
		})

		t.Run("Checks user traits", func(t *testing.T) {
			user := createTestUser()
			traitDef := createTestTraitDefinition(typeconvert.ComparableTypeString, rulesengine.EntityTypeUser)
			trait := createTestTrait("test-value", traitDef)
			user.Traits = append(user.Traits, trait)

			flag := createTestFlag()
			rule := createTestRule()
			rule.Value = true

			condition := createTestCondition(rulesengine.ConditionTypeTrait)
			condition.TraitDefinition = traitDef
			condition.TraitValue = "test-value"
			condition.Operator = typeconvert.ComparableOperatorEquals

			rule.Conditions = append(rule.Conditions, condition)
			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule.ID, result.RuleID)
		})
	})

	t.Run("Complex scenarios", func(t *testing.T) {
		t.Run("Handles multiple condition types and groups", func(t *testing.T) {
			company := createTestCompany()
			trait := createTestTrait("test-value", nil)
			company.Traits = append(company.Traits, trait)

			flag := createTestFlag()
			rule := createTestRule()
			rule.Value = true

			// Add direct conditions
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			rule.Conditions = append(rule.Conditions, condition1)

			condition2 := createTestCondition(rulesengine.ConditionTypeTrait)
			condition2.TraitDefinition = trait.TraitDefinition
			condition2.TraitValue = "test-value"
			condition2.Operator = typeconvert.ComparableOperatorEquals
			rule.Conditions = append(rule.Conditions, condition2)

			// Add condition group
			group := &rulesengine.ConditionGroup{
				Conditions: []*rulesengine.Condition{
					createTestCondition(rulesengine.ConditionTypePlan),
					createTestCondition(rulesengine.ConditionTypeBasePlan),
				},
			}
			group.Conditions[0].ResourceIDs = []string{company.PlanIDs[0]}
			if company.BasePlanID != nil {
				group.Conditions[1].ResourceIDs = []string{*company.BasePlanID}
			}

			rule.ConditionGroups = append(rule.ConditionGroups, group)
			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &rule.ID, result.RuleID)
		})

		t.Run("Handles missing or invalid data gracefully", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			rule := createTestRule()

			// Add condition with nil fields
			condition := &rulesengine.Condition{
				ConditionType: rulesengine.ConditionTypeMetric,
			}
			rule.Conditions = append(rule.Conditions, condition)

			// Add empty condition group
			group := &rulesengine.ConditionGroup{}
			rule.ConditionGroups = append(rule.ConditionGroups, group)

			flag.Rules = append(flag.Rules, rule)

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.Equal(t, flag.DefaultValue, result.Value)
			assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
		})
	})
}
