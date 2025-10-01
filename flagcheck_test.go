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
			assert.NotNil(t, result.FeatureUsageEvent)
			assert.Equal(t, eventSubtype, *result.FeatureUsageEvent)
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

	t.Run("Company-provided rules", func(t *testing.T) {
		t.Run("Company rule is evaluated along with flag rules", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create a company-provided rule that matches
			companyRule := createTestRule()
			companyRule.FlagID = &flag.ID
			companyRule.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{company.ID}
			companyRule.Conditions = append(companyRule.Conditions, condition)

			company.Rules = []*rulesengine.Rule{companyRule}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &companyRule.ID, result.RuleID)
		})

		t.Run("Company rule respects priority ordering", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create flag rule with lower priority
			flagRule := createTestRule()
			flagRule.Priority = 2
			flagRule.Value = false
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			flagRule.Conditions = append(flagRule.Conditions, condition1)

			// Create company rule with higher priority
			companyRule := createTestRule()
			companyRule.FlagID = &flag.ID
			companyRule.Priority = 1
			companyRule.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}
			companyRule.Conditions = append(companyRule.Conditions, condition2)

			flag.Rules = []*rulesengine.Rule{flagRule}
			company.Rules = []*rulesengine.Rule{companyRule}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &companyRule.ID, result.RuleID)
		})

		t.Run("Company rule with global override type takes precedence", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()

			// Create standard flag rule
			flagRule := createTestRule()
			flagRule.Value = false
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			flagRule.Conditions = append(flagRule.Conditions, condition1)

			// Create company rule with global override
			companyRule := createTestRule()
			companyRule.FlagID = &flag.ID
			companyRule.RuleType = rulesengine.RuleTypeGlobalOverride
			companyRule.Value = true

			flag.Rules = []*rulesengine.Rule{flagRule}
			company.Rules = []*rulesengine.Rule{companyRule}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &companyRule.ID, result.RuleID)
		})

		t.Run("Multiple company rules are all evaluated", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create two company rules, only one matches
			companyRule1 := createTestRule()
			companyRule1.FlagID = &flag.ID
			companyRule1.Priority = 1
			companyRule1.Value = true
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{"non-matching-id"}
			companyRule1.Conditions = append(companyRule1.Conditions, condition1)

			companyRule2 := createTestRule()
			companyRule2.FlagID = &flag.ID
			companyRule2.Priority = 2
			companyRule2.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}
			companyRule2.Conditions = append(companyRule2.Conditions, condition2)

			company.Rules = []*rulesengine.Rule{companyRule1, companyRule2}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &companyRule2.ID, result.RuleID)
		})
	})

	t.Run("User-provided rules", func(t *testing.T) {
		t.Run("User rule is evaluated along with flag rules", func(t *testing.T) {
			user := createTestUser()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create a user-provided rule that matches
			userRule := createTestRule()
			userRule.FlagID = &flag.ID
			userRule.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeUser)
			condition.ResourceIDs = []string{user.ID}
			userRule.Conditions = append(userRule.Conditions, condition)

			user.Rules = []*rulesengine.Rule{userRule}

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &userRule.ID, result.RuleID)
		})

		t.Run("User rule respects priority ordering", func(t *testing.T) {
			user := createTestUser()
			flag := createTestFlag()

			// Create flag rule with lower priority
			flagRule := createTestRule()
			flagRule.Priority = 2
			flagRule.Value = false
			condition1 := createTestCondition(rulesengine.ConditionTypeUser)
			condition1.ResourceIDs = []string{user.ID}
			flagRule.Conditions = append(flagRule.Conditions, condition1)

			// Create user rule with higher priority
			userRule := createTestRule()
			userRule.FlagID = &flag.ID
			userRule.Priority = 1
			userRule.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeUser)
			condition2.ResourceIDs = []string{user.ID}
			userRule.Conditions = append(userRule.Conditions, condition2)

			flag.Rules = []*rulesengine.Rule{flagRule}
			user.Rules = []*rulesengine.Rule{userRule}

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &userRule.ID, result.RuleID)
		})

		t.Run("User rule with global override type takes precedence", func(t *testing.T) {
			user := createTestUser()
			flag := createTestFlag()

			// Create standard flag rule
			flagRule := createTestRule()
			flagRule.Value = false
			condition1 := createTestCondition(rulesengine.ConditionTypeUser)
			condition1.ResourceIDs = []string{user.ID}
			flagRule.Conditions = append(flagRule.Conditions, condition1)

			// Create user rule with global override
			userRule := createTestRule()
			userRule.FlagID = &flag.ID
			userRule.RuleType = rulesengine.RuleTypeGlobalOverride
			userRule.Value = true

			flag.Rules = []*rulesengine.Rule{flagRule}
			user.Rules = []*rulesengine.Rule{userRule}

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &userRule.ID, result.RuleID)
		})
	})

	t.Run("Combined company and user rules", func(t *testing.T) {
		t.Run("Both company and user rules are evaluated", func(t *testing.T) {
			company := createTestCompany()
			user := createTestUser()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create company rule that doesn't match
			companyRule := createTestRule()
			companyRule.FlagID = &flag.ID
			companyRule.Priority = 1
			companyRule.Value = true
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{"non-matching-id"}
			companyRule.Conditions = append(companyRule.Conditions, condition1)

			// Create user rule that matches
			userRule := createTestRule()
			userRule.FlagID = &flag.ID
			userRule.Priority = 2
			userRule.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeUser)
			condition2.ResourceIDs = []string{user.ID}
			userRule.Conditions = append(userRule.Conditions, condition2)

			company.Rules = []*rulesengine.Rule{companyRule}
			user.Rules = []*rulesengine.Rule{userRule}

			result, err := rulesengine.CheckFlag(ctx, company, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.Equal(t, &userRule.ID, result.RuleID)
		})

		t.Run("All three rule sources evaluated with correct priority", func(t *testing.T) {
			company := createTestCompany()
			user := createTestUser()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create rules from all three sources - all matching their respective conditions
			flagRule := createTestRule()
			flagRule.Priority = 2
			flagRule.Value = true
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			flagRule.Conditions = append(flagRule.Conditions, condition1)

			companyRule := createTestRule()
			companyRule.FlagID = &flag.ID
			companyRule.Priority = 3
			companyRule.Value = true
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}
			companyRule.Conditions = append(companyRule.Conditions, condition2)

			userRule := createTestRule()
			userRule.FlagID = &flag.ID
			userRule.Priority = 1 // Highest priority
			userRule.Value = true
			condition3 := createTestCondition(rulesengine.ConditionTypeUser)
			condition3.ResourceIDs = []string{user.ID}
			userRule.Conditions = append(userRule.Conditions, condition3)

			flag.Rules = []*rulesengine.Rule{flagRule}
			company.Rules = []*rulesengine.Rule{companyRule}
			user.Rules = []*rulesengine.Rule{userRule}

			result, err := rulesengine.CheckFlag(ctx, company, user, flag)

			assert.NoError(t, err)
			assert.True(t, result.Value)
			assert.NotNil(t, result.RuleID)
			// Should match the user rule since it has highest priority (lowest number)
			assert.Equal(t, &userRule.ID, result.RuleID)
		})

		t.Run("Company rules for different flag are not evaluated", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = false

			otherFlagID := "other-flag-id"

			// Create a company rule for a different flag
			companyRuleForOtherFlag := createTestRule()
			companyRuleForOtherFlag.FlagID = &otherFlagID
			companyRuleForOtherFlag.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{company.ID}
			companyRuleForOtherFlag.Conditions = append(companyRuleForOtherFlag.Conditions, condition)

			company.Rules = []*rulesengine.Rule{companyRuleForOtherFlag}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			// Should use default value since the company rule is for a different flag
			assert.False(t, result.Value)
			assert.Nil(t, result.RuleID)
			assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
		})

		t.Run("User rules for different flag are not evaluated", func(t *testing.T) {
			user := createTestUser()
			flag := createTestFlag()
			flag.DefaultValue = false

			otherFlagID := "other-flag-id"

			// Create a user rule for a different flag
			userRuleForOtherFlag := createTestRule()
			userRuleForOtherFlag.FlagID = &otherFlagID
			userRuleForOtherFlag.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeUser)
			condition.ResourceIDs = []string{user.ID}
			userRuleForOtherFlag.Conditions = append(userRuleForOtherFlag.Conditions, condition)

			user.Rules = []*rulesengine.Rule{userRuleForOtherFlag}

			result, err := rulesengine.CheckFlag(ctx, nil, user, flag)

			assert.NoError(t, err)
			// Should use default value since the user rule is for a different flag
			assert.False(t, result.Value)
			assert.Nil(t, result.RuleID)
			assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
		})

		t.Run("Rules with nil FlagID are not evaluated", func(t *testing.T) {
			company := createTestCompany()
			flag := createTestFlag()
			flag.DefaultValue = false

			// Create a company rule with nil FlagID (legacy rule before FlagID was added)
			companyRuleWithoutFlagID := createTestRule()
			companyRuleWithoutFlagID.FlagID = nil
			companyRuleWithoutFlagID.Value = true
			condition := createTestCondition(rulesengine.ConditionTypeCompany)
			condition.ResourceIDs = []string{company.ID}
			companyRuleWithoutFlagID.Conditions = append(companyRuleWithoutFlagID.Conditions, condition)

			company.Rules = []*rulesengine.Rule{companyRuleWithoutFlagID}

			result, err := rulesengine.CheckFlag(ctx, company, nil, flag)

			assert.NoError(t, err)
			// Should use default value since the company rule has nil FlagID
			assert.False(t, result.Value)
			assert.Nil(t, result.RuleID)
			assert.Equal(t, rulesengine.ReasonNoRulesMatched, result.Reason)
		})

		t.Run("Correct flag rule is selected when company has multiple flag rules", func(t *testing.T) {
			company := createTestCompany()
			flag1 := createTestFlag()
			flag2 := createTestFlag()

			// Create rules for two different flags
			ruleForFlag1 := createTestRule()
			ruleForFlag1.FlagID = &flag1.ID
			ruleForFlag1.Value = true
			condition1 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition1.ResourceIDs = []string{company.ID}
			ruleForFlag1.Conditions = append(ruleForFlag1.Conditions, condition1)

			ruleForFlag2 := createTestRule()
			ruleForFlag2.FlagID = &flag2.ID
			ruleForFlag2.Value = false
			condition2 := createTestCondition(rulesengine.ConditionTypeCompany)
			condition2.ResourceIDs = []string{company.ID}
			ruleForFlag2.Conditions = append(ruleForFlag2.Conditions, condition2)

			company.Rules = []*rulesengine.Rule{ruleForFlag1, ruleForFlag2}

			// Check flag1 - should use ruleForFlag1
			result1, err := rulesengine.CheckFlag(ctx, company, nil, flag1)
			assert.NoError(t, err)
			assert.True(t, result1.Value)
			assert.Equal(t, &ruleForFlag1.ID, result1.RuleID)

			// Check flag2 - should use ruleForFlag2
			result2, err := rulesengine.CheckFlag(ctx, company, nil, flag2)
			assert.NoError(t, err)
			assert.False(t, result2.Value)
			assert.Equal(t, &ruleForFlag2.ID, result2.RuleID)
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
