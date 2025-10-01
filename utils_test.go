package rulesengine_test

import (
	"testing"

	"github.com/schematichq/rulesengine"
	"github.com/stretchr/testify/assert"
)

func TestGroupRulesByPriority(t *testing.T) {
	t.Run("When there are no rules", func(t *testing.T) {
		rules := []*rulesengine.Rule{}
		grouped := rulesengine.GroupRulesByPriority(rules)
		assert.Empty(t, grouped)
	})

	t.Run("When there are company override and plan entitlement rules", func(t *testing.T) {
		rules := []*rulesengine.Rule{
			{
				ID:       "rule_foo",
				RuleType: rulesengine.RuleTypePlanEntitlement,
			},
			{
				ID:       "rule_bar",
				RuleType: rulesengine.RuleTypeCompanyOverride,
			},
		}

		grouped := rulesengine.GroupRulesByPriority(rules)

		// Check overall structure
		assert.Len(t, grouped, 2)

		// Check first group (company override)
		assert.Len(t, grouped[0], 1)
		assert.Equal(t, rulesengine.RuleTypeCompanyOverride, grouped[0][0].RuleType)

		// Check second group (plan entitlement)
		assert.Len(t, grouped[1], 1)
		assert.Equal(t, rulesengine.RuleTypePlanEntitlement, grouped[1][0].RuleType)
	})

	t.Run("When multiple rule slices are provided", func(t *testing.T) {
		flagRules := []*rulesengine.Rule{
			{
				ID:       "flag_rule_1",
				RuleType: rulesengine.RuleTypeStandard,
				Priority: 2,
			},
		}

		companyRules := []*rulesengine.Rule{
			{
				ID:       "company_rule_1",
				RuleType: rulesengine.RuleTypeCompanyOverride,
			},
		}

		userRules := []*rulesengine.Rule{
			{
				ID:       "user_rule_1",
				RuleType: rulesengine.RuleTypeStandard,
				Priority: 1,
			},
		}

		grouped := rulesengine.GroupRulesByPriority(flagRules, companyRules, userRules)

		// Check overall structure - should have 2 groups (company override and standard)
		assert.Len(t, grouped, 2)

		// Check first group (company override)
		assert.Len(t, grouped[0], 1)
		assert.Equal(t, rulesengine.RuleTypeCompanyOverride, grouped[0][0].RuleType)
		assert.Equal(t, "company_rule_1", grouped[0][0].ID)

		// Check second group (standard) - should have both standard rules sorted by priority
		assert.Len(t, grouped[1], 2)
		assert.Equal(t, rulesengine.RuleTypeStandard, grouped[1][0].RuleType)
		assert.Equal(t, rulesengine.RuleTypeStandard, grouped[1][1].RuleType)
		// User rule has priority 1, flag rule has priority 2
		assert.Equal(t, "user_rule_1", grouped[1][0].ID)
		assert.Equal(t, "flag_rule_1", grouped[1][1].ID)
	})

	t.Run("When nil rule slices are provided", func(t *testing.T) {
		flagRules := []*rulesengine.Rule{
			{
				ID:       "flag_rule_1",
				RuleType: rulesengine.RuleTypeStandard,
			},
		}

		var nilRules []*rulesengine.Rule

		grouped := rulesengine.GroupRulesByPriority(flagRules, nilRules, nilRules)

		// Should only contain the flag rule
		assert.Len(t, grouped, 1)
		assert.Len(t, grouped[0], 1)
		assert.Equal(t, "flag_rule_1", grouped[0][0].ID)
	})
}
