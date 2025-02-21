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
}
