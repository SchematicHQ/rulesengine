package rulesengine

import (
	"context"
	"fmt"

	"github.com/schematichq/rulesengine/set"
	"github.com/schematichq/rulesengine/typeconvert"
)

type CheckScope struct {
	Company *Company
	Rule    *Rule
	User    *User
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
		match, err = s.checkCondition(ctx, scope.Company, scope.User, condition)
		if err != nil || !match {
			return
		}
	}

	for _, group := range scope.Rule.ConditionGroups {
		match, err = s.checkConditionGroup(ctx, scope.Company, scope.User, group)
		if err != nil || !match {
			return
		}
	}

	res.Match = true
	return
}

func (s *RuleCheckService) checkCondition(ctx context.Context, company *Company, user *User, condition *Condition) (match bool, err error) {
	if condition == nil {
		return false, nil
	}

	switch condition.ConditionType {
	case ConditionTypeCompany:
		return s.checkCompanyCondition(ctx, company, condition)
	case ConditionTypeMetric:
		return s.checkMetricCondition(ctx, company, condition)
	case ConditionTypeBasePlan:
		return s.checkBasePlanCondition(ctx, company, condition)
	case ConditionTypePlan:
		return s.checkPlanCondition(ctx, company, condition)
	case ConditionTypePlanVersion:
		return s.checkPlanVersionCondition(ctx, company, condition)
	case ConditionTypeTrait:
		return s.checkTraitCondition(ctx, company, user, condition)
	case ConditionTypeUser:
		return s.checkUserCondition(ctx, user, condition)
	case ConditionTypeBillingProduct:
		return s.checkBillingProductCondition(ctx, company, condition)
	case ConditionTypeCredit:
		return s.checkCreditBalanceCondition(ctx, company, condition)
	}

	return
}

func (s *RuleCheckService) checkConditionGroup(ctx context.Context, company *Company, user *User, group *ConditionGroup) (bool, error) {
	if group == nil {
		return false, nil
	}

	// Condition groups are OR'd together, so we return true if any condition matches
	for _, condition := range group.Conditions {
		match, err := s.checkCondition(ctx, company, user, condition)
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

func (s *RuleCheckService) checkCreditBalanceCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypeCredit || company == nil || condition.CreditID == nil {
		return false, nil
	}

	var consumptionCost = float64(1)
	if condition.ConsumptionRate != nil {
		consumptionCost = *condition.ConsumptionRate
	}

	var creditBalance float64
	for creditID, balance := range company.CreditBalances {
		if creditID == *condition.CreditID {
			creditBalance = balance
			break
		}
	}

	return creditBalance >= consumptionCost, nil
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

func (s *RuleCheckService) checkPlanVersionCondition(ctx context.Context, company *Company, condition *Condition) (bool, error) {
	if condition.ConditionType != ConditionTypePlanVersion || company == nil {
		return false, nil
	}

	companyPlanVersionIDs := set.NewSet(company.PlanVersionIDs...)
	resourceMatch := set.NewSet(condition.ResourceIDs...).Intersection(companyPlanVersionIDs).Len() > 0

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
	company *Company,
	condition *Condition,
) (bool, error) {
	if condition == nil || condition.ConditionType != ConditionTypeMetric || company == nil || condition.EventSubtype == nil {
		return false, nil
	}

	leftVal := int64(0)
	metric := company.Metrics.Find(*condition.EventSubtype, condition.MetricPeriod, condition.MetricPeriodMonthReset)
	if metric != nil {
		leftVal = metric.Value
	}

	if condition.MetricValue == nil {
		return false, fmt.Errorf("expected metric value for condition: %s, but received nil ", condition.ID)
	}

	rightVal := *condition.MetricValue
	if condition.ComparisonTraitDefinition != nil {
		comparisonTrait := s.findTrait(ctx, condition.ComparisonTraitDefinition, company.Traits)
		if comparisonTrait == nil {
			rightVal = 0
		} else {
			rightVal = typeconvert.StringToInt64(comparisonTrait.Value)
		}
	}

	return typeconvert.CompareInt64(leftVal, rightVal, condition.Operator), nil

}

func (s *RuleCheckService) checkTraitCondition(ctx context.Context, company *Company, user *User, condition *Condition) (bool, error) {
	if condition == nil || condition.ConditionType != ConditionTypeTrait || condition.TraitDefinition == nil {
		return false, nil
	}

	traitDef := condition.TraitDefinition
	var trait *Trait
	var comparisonTrait *Trait
	if traitDef.EntityType == EntityTypeCompany && company != nil {
		trait = s.findTrait(ctx, traitDef, company.Traits)
		comparisonTrait = s.findTrait(ctx, condition.ComparisonTraitDefinition, company.Traits)
	} else if traitDef.EntityType == EntityTypeUser && user != nil {
		trait = s.findTrait(ctx, traitDef, user.Traits)
		comparisonTrait = s.findTrait(ctx, condition.ComparisonTraitDefinition, user.Traits)
	} else {
		return false, nil
	}

	return s.compareTraits(ctx, condition, trait, comparisonTrait), nil
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

func (s *RuleCheckService) compareTraits(ctx context.Context, condition *Condition, trait *Trait, comparisonTrait *Trait) bool {
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
