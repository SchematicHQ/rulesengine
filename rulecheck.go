package rulesengine

import (
	"context"
	"fmt"

	"github.com/schematichq/rulesengine/set"
	"github.com/schematichq/rulesengine/typeconvert"
)

type CheckScope struct {
	Company    *Company
	Rule       *Rule
	User       *User
	Usage      *int64
	EventUsage map[string]int64
}

type CheckResult struct {
	*CheckScope
	Match bool
}

type RuleCheckService struct {
}

func NewRuleCheckService() *RuleCheckService {
	return &RuleCheckService{}
}

func (s *RuleCheckService) Check(ctx context.Context, scope *CheckScope) (res *CheckResult, err error) {
	res = &CheckResult{
		CheckScope: scope,
	}

	if scope.Rule == nil {
		return
	}

	if scope.Rule.RuleType == RuleTypeDefault || scope.Rule.RuleType == RuleTypeGlobalOverride {
		res.Match = true
		return
	}

	var match bool
	for _, condition := range scope.Rule.Conditions {
		match, err = s.checkCondition(ctx, scope, condition)
		if err != nil || !match {
			return
		}
	}

	for _, group := range scope.Rule.ConditionGroups {
		match, err = s.checkConditionGroup(ctx, scope, group)
		if err != nil || !match {
			return
		}
	}

	res.Match = true
	return
}

func (s *RuleCheckService) checkCondition(ctx context.Context, scope *CheckScope, condition *Condition) (match bool, err error) {
	if condition == nil {
		return false, nil
	}

	switch condition.ConditionType {
	case ConditionTypeCompany:
		return s.checkCompanyCondition(ctx, scope.Company, condition)
	case ConditionTypeMetric:
		return s.checkMetricCondition(ctx, scope, condition)
	case ConditionTypeBasePlan:
		return s.checkBasePlanCondition(ctx, scope.Company, condition)
	case ConditionTypePlan:
		return s.checkPlanCondition(ctx, scope.Company, condition)
	case ConditionTypeTrait:
		return s.checkTraitCondition(ctx, scope, condition)
	case ConditionTypeUser:
		return s.checkUserCondition(ctx, scope.User, condition)
	case ConditionTypeBillingProduct:
		return s.checkBillingProductCondition(ctx, scope.Company, condition)
	case ConditionTypeCrmProduct:
		return s.checkCrmProductCondition(ctx, scope.Company, condition)
	case ConditionTypeCredit:
		return s.checkCreditBalanceCondition(ctx, scope, condition)
	}

	return
}

func (s *RuleCheckService) checkConditionGroup(ctx context.Context, scope *CheckScope, group *ConditionGroup) (bool, error) {
	if group == nil {
		return false, nil
	}

	// Condition groups are OR'd together, so we return true if any condition matches
	for _, condition := range group.Conditions {
		match, err := s.checkCondition(ctx, scope, condition)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}

	// If no condition in the group matches, return false
	return false, nil
}

func (s *RuleCheckService) checkCompanyCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeCompany || company == nil {
		return false, nil
	}

	resourceMatch := set.NewSet(condition.ResourceIDs...).Contains(company.ID)
	if condition.Operator == typeconvert.ComparableOperatorNotEquals {
		return !resourceMatch, nil
	}

	return resourceMatch, nil
}

func (s *RuleCheckService) checkCreditBalanceCondition(ctx context.Context, scope *CheckScope, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeCredit || scope.Company == nil || condition.CreditID == nil {
		return false, nil
	}

	var consumptionRate = float64(1)
	if condition.ConsumptionRate != nil {
		consumptionRate = *condition.ConsumptionRate
	}

	var creditBalance float64
	for creditID, balance := range scope.Company.CreditBalances {
		if creditID == *condition.CreditID {
			creditBalance = balance
			break
		}
	}

	// WithUsage: Check if there are enough credits for generic usage
	if scope.Usage != nil && *scope.Usage > 0 {
		creditsNeeded := float64(*scope.Usage) * consumptionRate
		return creditBalance >= creditsNeeded, nil
	}

	// WithEventUsage: Check if there are enough credits for event-specific usage
	if condition.EventSubtype != nil && scope.EventUsage != nil {
		if eventUsage, ok := scope.EventUsage[*condition.EventSubtype]; ok && eventUsage > 0 {
			creditsNeeded := float64(eventUsage) * consumptionRate
			return creditBalance >= creditsNeeded, nil
		}
	}

	// Check against current metric usage if EventSubtype is specified
	if condition.EventSubtype != nil {
		usage := int64(0)
		metric := scope.Company.Metrics.Find(*condition.EventSubtype, condition.MetricPeriod, condition.MetricPeriodMonthReset)
		if metric != nil {
			usage = metric.Value
		}

		creditsNeeded := float64(usage) * consumptionRate
		return creditBalance >= creditsNeeded, nil
	}

	return creditBalance >= consumptionRate, nil
}

func (s *RuleCheckService) checkBillingProductCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeBillingProduct || company == nil {
		return false, nil
	}

	companyBillingProductIDs := set.NewSet(company.BillingProductIDs...)
	resourceMatch := set.NewSet(condition.ResourceIDs...).Intersection(companyBillingProductIDs).Len() > 0
	if condition.Operator == typeconvert.ComparableOperatorNotEquals {
		return !resourceMatch, nil
	}

	return resourceMatch, nil
}

func (s *RuleCheckService) checkCrmProductCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeCrmProduct || company == nil {
		return false, nil
	}

	companyCrmProductIDs := set.NewSet(company.CRMProductIDs...)
	resourceMatch := set.NewSet(condition.ResourceIDs...).Intersection(companyCrmProductIDs).Len() > 0
	if condition.Operator == typeconvert.ComparableOperatorNotEquals {
		return !resourceMatch, nil
	}

	return resourceMatch, nil
}

func (s *RuleCheckService) checkPlanCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypePlan || company == nil {
		return false, nil
	}

	companyPlanIDs := set.NewSet(company.PlanIDs...)
	resourceMatch := set.NewSet(condition.ResourceIDs...).Intersection(companyPlanIDs).Len() > 0
	if condition.Operator == typeconvert.ComparableOperatorNotEquals {
		return !resourceMatch, nil
	}

	return resourceMatch, nil
}

func (s *RuleCheckService) checkBasePlanCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeBasePlan || company == nil {
		return false, nil
	}

	conditionPlanIDSet := set.NewSet(condition.ResourceIDs...)

	switch condition.Operator {
	case typeconvert.ComparableOperatorEquals:
		return company.BasePlanID != nil && conditionPlanIDSet.Contains(*company.BasePlanID), nil
	case typeconvert.ComparableOperatorNotEquals:
		return company.BasePlanID == nil || !conditionPlanIDSet.Contains(*company.BasePlanID), nil
	case typeconvert.ComparableOperatorIsEmpty:
		return company.BasePlanID == nil, nil
	case typeconvert.ComparableOperatorNotEmpty:
		return company.BasePlanID != nil, nil
	}

	return false, nil

}

func (s *RuleCheckService) checkMetricCondition(
	ctx context.Context,
	scope *CheckScope,
	condition *Condition,
) (bool, error) {
	if condition == nil || condition.ConditionType != ConditionTypeMetric || scope.Company == nil || condition.EventSubtype == nil {
		return false, nil
	}

	leftVal := int64(0)
	metric := scope.Company.Metrics.Find(*condition.EventSubtype, condition.MetricPeriod, condition.MetricPeriodMonthReset)
	if metric != nil {
		leftVal = metric.Value
	}

	// WithEventUsage: Add event-specific usage if this event matches
	if scope.EventUsage != nil {
		if eventUsage, ok := scope.EventUsage[*condition.EventSubtype]; ok && eventUsage > 0 {
			leftVal += eventUsage
		}
	} else if scope.Usage != nil && *scope.Usage > 0 {
		// WithUsage: Add generic usage to current metric value
		leftVal += *scope.Usage
	}

	if condition.MetricValue == nil {
		return false, fmt.Errorf("expected metric value for condition: %s, but received nil ", condition.ID)
	}

	rightVal := *condition.MetricValue
	if condition.ComparisonTraitDefinition != nil {
		comparisonTrait := s.findTrait(ctx, condition.ComparisonTraitDefinition, scope.Company.Traits)
		if comparisonTrait == nil {
			rightVal = 0
		} else {
			rightVal = typeconvert.StringToInt64(comparisonTrait.Value)
		}
	}

	return typeconvert.CompareInt64(leftVal, rightVal, condition.Operator), nil

}

func (s *RuleCheckService) checkTraitCondition(ctx context.Context, scope *CheckScope, condition *Condition) (bool, error) {
	if condition == nil || condition.ConditionType != ConditionTypeTrait || condition.TraitDefinition == nil {
		return false, nil
	}

	traitDef := condition.TraitDefinition
	var trait *Trait
	var comparisonTrait *Trait
	if traitDef.EntityType == EntityTypeCompany && scope.Company != nil {
		trait = s.findTrait(ctx, traitDef, scope.Company.Traits)
		comparisonTrait = s.findTrait(ctx, condition.ComparisonTraitDefinition, scope.Company.Traits)
	} else if traitDef.EntityType == EntityTypeUser && scope.User != nil {
		trait = s.findTrait(ctx, traitDef, scope.User.Traits)
		comparisonTrait = s.findTrait(ctx, condition.ComparisonTraitDefinition, scope.User.Traits)
	} else {
		return false, nil
	}

	return s.compareTraitsWithUsage(ctx, scope, condition, trait, comparisonTrait), nil
}

func (s *RuleCheckService) checkUserCondition(ctx context.Context, user *User, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeUser || user == nil {
		return false, nil
	}

	resourceMatch := set.NewSet(condition.ResourceIDs...).Contains(user.ID)
	if condition.Operator == typeconvert.ComparableOperatorNotEquals {
		return !resourceMatch, nil
	}

	return resourceMatch, nil
}

func (s *RuleCheckService) compareTraitsWithUsage(ctx context.Context, scope *CheckScope, condition *Condition, trait *Trait, comparisonTrait *Trait) bool {
	var leftVal string
	rightVal := condition.TraitValue
	if trait != nil {
		leftVal = trait.Value
	}
	if comparisonTrait != nil {
		rightVal = comparisonTrait.Value
	}

	comparableType := typeconvert.ComparableTypeString
	if trait != nil && trait.TraitDefinition != nil {
		comparableType = trait.TraitDefinition.ComparableType
	}

	if comparableType == typeconvert.ComparableTypeInt && scope.Usage != nil && *scope.Usage > 0 {
		leftNumeric := typeconvert.StringToInt64(leftVal)
		leftNumeric += *scope.Usage
		leftVal = fmt.Sprintf("%d", leftNumeric)
	}

	return typeconvert.Compare(leftVal, rightVal, comparableType, condition.Operator)
}

func (s *RuleCheckService) findTrait(ctx context.Context, traitDef *TraitDefinition, traits []*Trait) *Trait {
	if traitDef == nil {
		return nil
	}

	// If the company has the trait, return the view for this
	if trait, ok := find(traits, func(trait *Trait) bool {
		return trait.TraitDefinition != nil && trait.TraitDefinition.ID == traitDef.ID
	}); ok {
		return trait
	}

	// Otherwise, return a trait with only the definition
	return &Trait{TraitDefinition: traitDef}
}
