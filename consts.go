package rulesengine

import (
	"slices"
	"strings"
)

type ConditionType string

const (
	ConditionTypeUnknown        ConditionType = ""
	ConditionTypeBasePlan       ConditionType = "base_plan"
	ConditionTypeBillingProduct ConditionType = "billing_product"
	ConditionTypeCompany        ConditionType = "company"
	ConditionTypeCrmProduct     ConditionType = "crm_product"
	ConditionTypeMetric         ConditionType = "metric"
	ConditionTypePlan           ConditionType = "plan"
	ConditionTypeTrait          ConditionType = "trait"
	ConditionTypeUser           ConditionType = "user"
)

type EntityType string

const (
	EntityTypeUnknown EntityType = ""
	EntityTypeUser    EntityType = "user"
	EntityTypeCompany EntityType = "company"
)

type RuleType string

func (t RuleType) DisplayName() string {
	return strings.Replace(string(t), "_", " ", -1)
}

func (t *RuleType) isEntitlement() bool {
	return t != nil && *t == RuleTypePlanEntitlement || *t == RuleTypePlanEntitlementUsageExceeded || *t == RuleTypeCompanyOverride || *t == RuleTypeCompanyOverrideUsageExceeded
}

const (
	RuleTypeUnknown                      RuleType = ""
	RuleTypeGlobalOverride               RuleType = "global_override" // Global on/off toggle; will not have any conditions, only a value
	RuleTypeCompanyOverride              RuleType = "company_override"
	RuleTypeCompanyOverrideUsageExceeded RuleType = "company_override_usage_exceeded"
	RuleTypePlanEntitlement              RuleType = "plan_entitlement"
	RuleTypePlanEntitlementUsageExceeded RuleType = "plan_entitlement_usage_exceeded"
	RuleTypeStandard                     RuleType = "standard" // Any other rule type
	RuleTypeDefault                      RuleType = "default"  // Default on/off toggle; will not have any conditions, only a value

	RuleTypePlanAudience RuleType = "plan_audience" // Plan audience rule; should have a plan_id but no flag_id
)

type RulePrioritizationMethod string

const (
	RulePrioritizationMethodUnknown    RulePrioritizationMethod = ""
	RulePrioritizationMethodNone       RulePrioritizationMethod = "none"
	RulePrioritizationMethodPriority   RulePrioritizationMethod = "priority"
	RulePrioritizationMethodOptimistic RulePrioritizationMethod = "optimistic"
)

func (t RuleType) PrioritizationMethod() RulePrioritizationMethod {
	if t == RuleTypeStandard {
		return RulePrioritizationMethodPriority
	}

	if slices.Contains([]RuleType{RuleTypeCompanyOverride, RuleTypePlanEntitlement, RuleTypeCompanyOverrideUsageExceeded, RuleTypePlanEntitlementUsageExceeded}, t) {
		return RulePrioritizationMethodOptimistic
	}

	return RulePrioritizationMethodNone
}

// In a flag context, rules are checked in this order and prioritized within these type groups
var RuleTypePriority = []RuleType{
	RuleTypeGlobalOverride,
	RuleTypeCompanyOverride,
	RuleTypePlanEntitlement,
	RuleTypeCompanyOverrideUsageExceeded,
	RuleTypePlanEntitlementUsageExceeded,
	RuleTypeStandard,
	RuleTypeDefault,
}
