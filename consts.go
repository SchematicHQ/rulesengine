package rulesengine

import (
	"slices"
	"strings"
)

type ConditionType string

const (
	ConditionTypeBasePlan       ConditionType = "base_plan"
	ConditionTypeBillingProduct ConditionType = "billing_product"
	ConditionTypeCredit         ConditionType = "credit"
	ConditionTypeCompany        ConditionType = "company"
	ConditionTypeMetric         ConditionType = "metric"
	ConditionTypePlan           ConditionType = "plan"
	ConditionTypeTrait          ConditionType = "trait"
	ConditionTypeUser           ConditionType = "user"
)

type EntityType string

const (
	EntityTypeUser    EntityType = "user"
	EntityTypeCompany EntityType = "company"
)

// EntitlementType represents whether an entitlement is from a plan or company override.
type EntitlementType string

const (
	EntitlementTypePlanEntitlement EntitlementType = "plan_entitlement"
	EntitlementTypeCompanyOverride EntitlementType = "company_override"
)

type EntitlementValueType string

const (
	EntitlementValueTypeBoolean   EntitlementValueType = "boolean"
	EntitlementValueTypeCredit    EntitlementValueType = "credit"
	EntitlementValueTypeNumeric   EntitlementValueType = "numeric"
	EntitlementValueTypeTrait     EntitlementValueType = "trait"
	EntitlementValueTypeUnknown   EntitlementValueType = "unknown"
	EntitlementValueTypeUnlimited EntitlementValueType = "unlimited"
)

type RuleType string

func (t RuleType) DisplayName() string {
	return strings.Replace(string(t), "_", " ", -1)
}

func (t *RuleType) isEntitlement() bool {
	return t != nil && *t == RuleTypePlanEntitlement || *t == RuleTypePlanEntitlementUsageExceeded || *t == RuleTypeCompanyOverride || *t == RuleTypeCompanyOverrideUsageExceeded
}

const (
	RuleTypeGlobalOverride               RuleType = "global_override" // Global on/off toggle; will not have any conditions, only a value
	RuleTypeCompanyOverride              RuleType = "company_override"
	RuleTypeCompanyOverrideUsageExceeded RuleType = "company_override_usage_exceeded"
	RuleTypePlanEntitlement              RuleType = "plan_entitlement"
	RuleTypePlanEntitlementUsageExceeded RuleType = "plan_entitlement_usage_exceeded"
	RuleTypeStandard                     RuleType = "standard" // Any other rule type
	RuleTypeDefault                      RuleType = "default"  // Default on/off toggle; will not have any conditions, only a value
)

type RulePrioritizationMethod string

const (
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

// VersionKey generates a version key based on the structure of the rules engine models
// This ensures cache invalidation when the model structures change
var VersionKey = GetVersionKey()
