package rulesengine

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/schematichq/rulesengine/typeconvert"
)

type CheckFlagResult struct {
	CompanyID           *string             `json:"company_id,omitempty"`
	Err                 error               `json:"err,omitempty"`
	Entitlement         *FeatureEntitlement `json:"entitlement,omitempty"`
	FeatureAllocation   *int64              `json:"feature_allocation,omitempty"`
	FeatureUsage        *int64              `json:"feature_usage,omitempty"`
	FeatureUsageEvent   *string             `json:"feature_usage_event,omitempty"`
	FeatureUsagePeriod  *MetricPeriod       `json:"feature_usage_period,omitempty" binding:"oneof=all_time current_day current_month current_week"`
	FeatureUsageResetAt *time.Time          `json:"feature_usage_reset_at,omitempty"`
	FlagID              *string             `json:"flag_id,omitempty"`
	FlagKey             string              `json:"flag_key"`
	Reason              string              `json:"reason"`
	RuleID              *string             `json:"rule_id,omitempty"`
	RuleType            *RuleType           `json:"rule_type,omitempty" binding:"oneof=default global_override company_override company_override_usage_exceeded plan_entitlement plan_entitlement_usage_exceeded standard"`
	UserID              *string             `json:"user_id,omitempty"`
	Value               bool                `json:"value"`
}

const (
	ReasonNoCompanyOrUser     = "No company or user context; default value for flag"
	ReasonCompanyNotFound     = "Company not found"
	ReasonCompanyNotSpecified = "Must specify a company"
	ReasonFlagNotFound        = "Flag not found"
	ReasonNoRulesMatched      = "No rules matched; default value for flag"
	ReasonServerError         = "Server error; Schematic has been notified"
	ReasonUserNotFound        = "User not found"
)

func (r *CheckFlagResult) setRuleFields(company *Company, rule *Rule) {
	if rule == nil {
		return
	}

	r.RuleID = &rule.ID
	r.RuleType = &rule.RuleType

	if company == nil {
		return
	}

	// only set entitlement fields if the matched rule is an entitlement rule
	if !r.RuleType.isEntitlement() {
		return
	}

	// for a numeric entitlement rule, there will be a metric or trait condition; for a boolean or unlimited entitlement rule, we don't need to set these fields
	usageCondition, ok := find(rule.Conditions, func(c *Condition) bool {
		return c != nil && (c.ConditionType == ConditionTypeMetric || c.ConditionType == ConditionTypeTrait)
	})
	if !ok || usageCondition == nil {
		return
	}

	// set usage, allocation, and other usage-related fields
	var usage int64
	var allocation int64
	switch usageCondition.ConditionType {
	case ConditionTypeMetric:
		if usageCondition.EventSubtype != nil {
			r.FeatureUsageEvent = usageCondition.EventSubtype
			usageMetric := company.Metrics.Find(*usageCondition.EventSubtype, usageCondition.MetricPeriod, usageCondition.MetricPeriodMonthReset)
			if usageMetric != nil {
				usage = usageMetric.Value
			}
		}

		if usageCondition.MetricValue != nil {
			allocation = *usageCondition.MetricValue
		}

		metricPeriod := MetricPeriodAllTime
		if usageCondition.MetricPeriod != nil {
			metricPeriod = *usageCondition.MetricPeriod
		}
		r.FeatureUsagePeriod = &metricPeriod
		r.FeatureUsageResetAt = GetNextMetricPeriodStartFromCondition(usageCondition, company)
	case ConditionTypeTrait:
		if usageCondition.TraitDefinition != nil {
			companyUsageTrait := company.getTraitByDefinitionID(usageCondition.TraitDefinition.ID)
			if companyUsageTrait != nil {
				usage = typeconvert.StringToInt64(companyUsageTrait.Value)
			}
		}

		allocation = typeconvert.StringToInt64(usageCondition.TraitValue)
	}

	// if there is a comparison trait, this takes precedence for allocation over the numeric value
	if usageCondition.ComparisonTraitDefinition != nil {
		companyAllocationTrait := company.getTraitByDefinitionID(usageCondition.ComparisonTraitDefinition.ID)
		if companyAllocationTrait != nil {
			allocation = typeconvert.StringToInt64(companyAllocationTrait.Value)
		}
	}

	r.FeatureUsage = &usage
	r.FeatureAllocation = &allocation
}

func CheckFlag(
	ctx context.Context,
	company *Company,
	user *User,
	flag *Flag,
) (*CheckFlagResult, error) {
	resp := &CheckFlagResult{Reason: ReasonNoRulesMatched}

	if flag == nil {
		resp.Reason = ReasonFlagNotFound
		resp.Err = ErrorFlagNotFound
		return resp, nil
	}

	resp.FlagID = &flag.ID
	resp.FlagKey = flag.Key
	resp.Value = flag.DefaultValue

	if company != nil {
		resp.CompanyID = &company.ID

		// Find matching entitlement from company.Entitlements by feature key
		for _, ent := range company.Entitlements {
			if ent != nil && ent.FeatureKey == flag.Key {
				resp.Entitlement = ent
				break
			}
		}
	}
	if user != nil {
		resp.UserID = &user.ID
	}

	ruleChecker := NewRuleCheckService()
	var companyRules, userRules []*Rule
	if company != nil {
		for _, rule := range company.Rules {
			if rule != nil && rule.FlagID != nil && *rule.FlagID == flag.ID {
				companyRules = append(companyRules, rule)
			}
		}
	}
	if user != nil {
		for _, rule := range user.Rules {
			if rule != nil && rule.FlagID != nil && *rule.FlagID == flag.ID {
				userRules = append(userRules, rule)
			}
		}
	}
	for _, group := range GroupRulesByPriority(flag.Rules, companyRules, userRules) {
		for _, rule := range group {
			if rule == nil {
				continue
			}

			checkRuleResp, err := ruleChecker.Check(ctx, &CheckScope{
				Company: company,
				Rule:    rule,
				User:    user,
			})
			if err != nil {
				resp.Err = err
				return resp, err
			}

			if checkRuleResp == nil {
				resp.Err = err
				return resp, ErrorUnexpected
			}

			if checkRuleResp.Match {
				resp.Value = rule.Value
				resp.Reason = fmt.Sprintf("Matched %s rule \"%s\" (%s)", rule.RuleType.DisplayName(), rule.Name, rule.ID)
				resp.setRuleFields(company, rule)
				return resp, nil
			}
		}
	}

	return resp, nil
}

// Given a list of rules, group by type, then sort each group as appropriate to the type
func GroupRulesByPriority(ruleSlices ...[]*Rule) [][]*Rule {
	allRules := []*Rule{}
	for _, rules := range ruleSlices {
		allRules = append(allRules, rules...)
	}

	// Group rules by their type
	grouped := groupBy(allRules, func(rule *Rule) RuleType {
		return rule.RuleType
	})

	// Prioritize rules within each type group
	for ruleType, rules := range grouped {
		switch ruleType.PrioritizationMethod() {
		case RulePrioritizationMethodPriority:
			sort.Slice(rules, func(i, j int) bool {
				// Sort by ascending priority int
				return rules[i].Priority < rules[j].Priority
			})
		case RulePrioritizationMethodOptimistic:
			sort.Slice(rules, func(i, j int) bool {
				// Don't really care about order, just move all rules with true value to the front
				return rules[i].Value
			})
		}
	}

	// Prioritize type groups relative to one another
	prioritizedGroups := [][]*Rule{}
	for _, ruleType := range RuleTypePriority {
		if rules, ok := grouped[ruleType]; ok {
			prioritizedGroups = append(prioritizedGroups, rules)
		}
	}

	return prioritizedGroups
}
