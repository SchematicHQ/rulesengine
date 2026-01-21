package rulesengine

import (
	"sync"
	"time"

	"github.com/schematichq/rulesengine/typeconvert"
)

type TraitDefinition struct {
	ID             string                     `json:"id"`
	ComparableType typeconvert.ComparableType `json:"comparable_type" binding:"oneof=bool date int string"`
	EntityType     EntityType                 `json:"entity_type" binding:"oneof=user company"`
}

type Flag struct {
	ID            string  `json:"id"`
	AccountID     string  `json:"account_id"`
	EnvironmentID string  `json:"environment_id"`
	Key           string  `json:"key"`
	Rules         []*Rule `json:"rules"`
	DefaultValue  bool    `json:"default_value"`
}

type Rule struct {
	ID              string            `json:"id"`
	FlagID          *string           `json:"flag_id"`
	AccountID       string            `json:"account_id"`
	EnvironmentID   string            `json:"environment_id"`
	RuleType        RuleType          `json:"rule_type" binding:"oneof=default global_override company_override company_override_usage_exceeded plan_entitlement plan_entitlement_usage_exceeded standard"`
	Name            string            `json:"name"`
	Priority        int64             `json:"priority"`
	Conditions      []*Condition      `json:"conditions"`
	ConditionGroups []*ConditionGroup `json:"condition_groups"`
	Value           bool              `json:"value"`
}

type Condition struct {
	ID            string                         `json:"id"`
	AccountID     string                         `json:"account_id"`
	EnvironmentID string                         `json:"environment_id"`
	ConditionType ConditionType                  `json:"condition_type" binding:"oneof=base_plan billing_product company credit metric plan trait user"`
	Operator      typeconvert.ComparableOperator `json:"operator" binding:"oneof=eq ne gt lt gte lte is_empty not_empty"`

	// Fields relevant when ConditionType is one of Company, User, Plan, Base Plan, Billing Product, or Billing Credit
	ResourceIDs []string `json:"resource_ids"`

	// Fields relevant when ConditionType = Event
	EventSubtype           *string                 `json:"event_subtype"`
	MetricValue            *int64                  `json:"metric_value"`
	MetricPeriod           *MetricPeriod           `json:"metric_period" binding:"oneof=all_time current_day current_month current_week"`
	MetricPeriodMonthReset *MetricPeriodMonthReset `json:"metric_period_month_reset" binding:"oneof=first_of_month billing_cycle"`

	// Fields relevant when ConditionType = Billing Credit
	CreditID        *string  `json:"credit_id"`
	ConsumptionRate *float64 `json:"consumption_rate"`

	// Fields relevant when ConditionType = Trait
	TraitDefinition *TraitDefinition `json:"trait_definition"`
	TraitValue      string           `json:"trait_value"`

	// Relevant when ConditionType is either Event or Trait
	ComparisonTraitDefinition *TraitDefinition `json:"comparison_trait_definition"`
}

type ConditionGroup struct {
	Conditions []*Condition `json:"conditions"`
}

// Evaluation objects

type CompanyMetric struct {
	AccountID     string                 `json:"account_id"`
	EnvironmentID string                 `json:"environment_id"`
	CompanyID     string                 `json:"company_id"`
	EventSubtype  string                 `json:"event_subtype"`
	Period        MetricPeriod           `json:"period" binding:"oneof=all_time current_day current_month current_week"`
	MonthReset    MetricPeriodMonthReset `json:"month_reset" binding:"oneof=first_of_month billing_cycle"`
	Value         int64                  `json:"value"`
	CreatedAt     time.Time              `json:"created_at"`
	ValidUntil    *time.Time             `json:"valid_until"`
}

type CompanyMetricCollection []*CompanyMetric

func (c CompanyMetricCollection) Find(
	eventSubtype string,
	period *MetricPeriod,
	monthReset *MetricPeriodMonthReset,
) *CompanyMetric {
	if len(c) == 0 {
		return nil
	}

	if period == nil {
		p := MetricPeriodAllTime
		period = &p
	}

	if monthReset == nil {
		r := MetricPeriodMonthResetFirst
		monthReset = &r
	}

	item, found := find(c, func(item *CompanyMetric) bool {
		return item.EventSubtype == eventSubtype && item.Period == *period && item.MonthReset == *monthReset
	})
	if !found {
		return nil
	}

	return item
}

type Trait struct {
	TraitDefinition *TraitDefinition `json:"trait_definition"`
	Value           string           `json:"value"`
}

type Subscription struct {
	ID          string    `json:"id"`
	PeriodStart time.Time `json:"period_start"`
	PeriodEnd   time.Time `json:"period_end"`
}

type FeatureEntitlement struct {
	FeatureID       string                  `json:"feature_id" desc:"The ID of the feature"`
	FeatureKey      string                  `json:"feature_key" desc:"The key of the feature"`
	ValueType       EntitlementValueType    `json:"value_type" binding:"oneof=boolean credit numeric trait unknown unlimited" desc:"The type of the entitlement value"`
	Allocation      *int64                  `json:"allocation" desc:"If a numeric feature entitlement rule was matched, its allocation"`
	Usage           *int64                  `json:"usage" desc:"If a numeric feature entitlement rule was matched, the company's usage"`
	MetricPeriod    *MetricPeriod           `json:"metric_period" binding:"oneof=all_time current_day current_month current_week" desc:"For event-based feature entitlement rules, the period over which usage is tracked (current_month, current_day, current_week, all_time)"`
	MonthReset      *MetricPeriodMonthReset `json:"month_reset" binding:"oneof=first_of_month billing_cycle" desc:"For event-based feature entitlement rules, when the usage period resets (first_of_month or billing_cycle)"`
	MetricResetAt   *time.Time              `json:"metric_reset_at" desc:"For event-based feature entitlement rules, when the usage period will reset"`
	CreditID        *string                 `json:"credit_id" desc:"If a credit-based feature entitlement rule was matched, the ID of the credit"`
	CreditTotal     *float64                `json:"credit_total" desc:"If a credit-based feature entitlement rule was matched, the total credit amount"`
	CreditUsed      *float64                `json:"credit_used" desc:"If a credit-based feature entitlement rule was matched, the amount of credit used"`
	CreditRemaining *float64                `json:"credit_remaining" desc:"If a credit-based feature entitlement rule was matched, the remaining credit amount"`
}

type Company struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	EnvironmentID string `json:"environment_id"`

	BasePlanID        *string                 `json:"base_plan_id"`
	BillingProductIDs []string                `json:"billing_product_ids"`
	Keys              map[string]string       `json:"keys"`
	PlanIDs           []string                `json:"plan_ids"`
	Metrics           CompanyMetricCollection `json:"metrics"`
	CreditBalances    map[string]float64      `json:"credit_balances"`
	Subscription      *Subscription           `json:"subscription"`
	Traits            []*Trait                `json:"traits"`
	Rules             []*Rule                 `json:"rules"`
	Entitlements      []*FeatureEntitlement   `json:"entitlements,omitempty"`

	mu sync.Mutex `json:"-"` // mutex for thread safety
}

func (c *Company) getTraitByDefinitionID(traitDefinitionID string) *Trait {
	if c == nil {
		return nil
	}

	if len(c.Traits) == 0 {
		return nil
	}

	for _, trait := range c.Traits {
		if trait.TraitDefinition != nil && trait.TraitDefinition.ID == traitDefinitionID {
			return trait
		}
	}

	return nil
}

// AddMetric adds a new metric to the company's metrics collection or replaces an existing one
// that matches the same unique constraint (eventSubtype, period, and monthReset).
// It uses a mutex to ensure thread safety.
func (c *Company) AddMetric(metric *CompanyMetric) {
	if c == nil || metric == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.Metrics) == 0 {
		c.Metrics = CompanyMetricCollection{metric}
		return
	}

	// Loop through once, either replace an existing metric or append a new one
	for i, m := range c.Metrics {
		if m.EventSubtype == metric.EventSubtype &&
			m.Period == metric.Period &&
			m.MonthReset == metric.MonthReset {
			// Found a match, replace it
			c.Metrics[i] = metric
			return
		}
	}

	// If we get here, no match was found, so append the new metric
	c.Metrics = append(c.Metrics, metric)
}

type User struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	EnvironmentID string `json:"environment_id"`

	Keys   map[string]string `json:"keys"`
	Traits []*Trait          `json:"traits"`
	Rules  []*Rule           `json:"rules"`
}
