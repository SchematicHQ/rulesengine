package rulesengine_test

import (
	"fmt"
	"strings"
	"time"

	gofakeit "github.com/brianvoe/gofakeit/v6"
	"github.com/schematichq/rulesengine"
	"github.com/schematichq/rulesengine/null"
	"github.com/schematichq/rulesengine/typeconvert"
)

func generateTestID(prefix string) string {
	const randomPartLength = 12
	return fmt.Sprintf("%s_%s", prefix, strings.ToLower(gofakeit.LetterN(randomPartLength)))
}

func createTestCompany() *rulesengine.Company {
	return &rulesengine.Company{
		ID:                generateTestID("comp"),
		AccountID:         generateTestID("acct"),
		EnvironmentID:     generateTestID("env"),
		PlanIDs:           []string{generateTestID("plan"), generateTestID("plan")},
		PlanVersionIDs:    []string{generateTestID("plnv"), generateTestID("plnv")},
		BillingProductIDs: []string{generateTestID("bilp"), generateTestID("bilp")},
		BasePlanID:        null.Nullable(generateTestID("plan")),
		Metrics:           make(rulesengine.CompanyMetricCollection, 0),
		Traits:            make([]*rulesengine.Trait, 0),
		Subscription: &rulesengine.Subscription{
			ID:          generateTestID("bilsub"),
			PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
			PeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
		},
	}
}

func createTestSubscription() *rulesengine.Subscription {
	return &rulesengine.Subscription{
		ID:          generateTestID("bilsub"),
		PeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		PeriodEnd:   time.Now().Add(30 * 24 * time.Hour),
	}
}

func createTestUser() *rulesengine.User {
	return &rulesengine.User{
		ID:            generateTestID("user"),
		AccountID:     generateTestID("acct"),
		EnvironmentID: generateTestID("env"),
		Traits:        make([]*rulesengine.Trait, 0),
	}
}

func createTestRule() *rulesengine.Rule {
	return &rulesengine.Rule{
		ID:              generateTestID("rule"),
		AccountID:       generateTestID("acct"),
		EnvironmentID:   generateTestID("env"),
		RuleType:        rulesengine.RuleTypeStandard,
		Name:            gofakeit.Name(),
		Priority:        1,
		Conditions:      make([]*rulesengine.Condition, 0),
		ConditionGroups: make([]*rulesengine.ConditionGroup, 0),
		Value:           true,
	}
}

func createTestFlag() *rulesengine.Flag {
	return &rulesengine.Flag{
		ID:            generateTestID("flag"),
		AccountID:     generateTestID("acct"),
		EnvironmentID: generateTestID("env"),
		Key:           gofakeit.Word(),
		Rules:         make([]*rulesengine.Rule, 0),
		DefaultValue:  gofakeit.Bool(),
	}
}

func createTestCondition(conditionType rulesengine.ConditionType) *rulesengine.Condition {
	condition := &rulesengine.Condition{
		ID:            generateTestID("cond"),
		AccountID:     generateTestID("acct"),
		EnvironmentID: generateTestID("env"),
		ConditionType: conditionType,
		Operator:      typeconvert.ComparableOperatorEquals,
	}

	switch conditionType {
	case rulesengine.ConditionTypeMetric:
		subtype := gofakeit.Word()
		value := gofakeit.Int64()
		period := rulesengine.MetricPeriodAllTime
		reset := rulesengine.MetricPeriodMonthResetFirst
		condition.EventSubtype = &subtype
		condition.MetricValue = &value
		condition.MetricPeriod = &period
		condition.MetricPeriodMonthReset = &reset
	case rulesengine.ConditionTypeTrait:
		condition.TraitDefinition = createTestTraitDefinition(typeconvert.ComparableTypeInt, rulesengine.EntityTypeCompany)
		condition.TraitValue = gofakeit.Word()
	}

	return condition
}

func createTestMetric(company *rulesengine.Company, eventSubtype string, period rulesengine.MetricPeriod, value int64) *rulesengine.CompanyMetric {
	return &rulesengine.CompanyMetric{
		AccountID:     company.AccountID,
		EnvironmentID: company.EnvironmentID,
		CompanyID:     company.ID,
		EventSubtype:  eventSubtype,
		Period:        period,
		MonthReset:    rulesengine.MetricPeriodMonthResetFirst,
		Value:         value,
		CreatedAt:     time.Now(),
	}
}

func createTestTrait(value string, def *rulesengine.TraitDefinition) *rulesengine.Trait {
	if def == nil {
		def = createTestTraitDefinition(typeconvert.ComparableTypeInt, rulesengine.EntityTypeCompany)
	}

	return &rulesengine.Trait{
		TraitDefinition: def,
		Value:           value,
	}
}

func createTestTraitDefinition(
	comparableType typeconvert.ComparableType,
	entityType rulesengine.EntityType,
) *rulesengine.TraitDefinition {
	return &rulesengine.TraitDefinition{
		ID:             generateTestID("trt"),
		ComparableType: comparableType,
		EntityType:     entityType,
	}
}
