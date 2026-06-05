package rulesengine

import (
	"encoding/json"
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
	ID            string           `json:"id"`
	AccountID     string           `json:"account_id"`
	EnvironmentID string           `json:"environment_id"`
	Key           string           `json:"key"`
	Rules         JSONSlice[*Rule] `json:"rules"`
	DefaultValue  bool             `json:"default_value"`
}

type Rule struct {
	ID              string                     `json:"id"`
	FlagID          *string                    `json:"flag_id"`
	AccountID       string                     `json:"account_id"`
	EnvironmentID   string                     `json:"environment_id"`
	RuleType        RuleType                   `json:"rule_type" binding:"oneof=default global_override company_override company_override_usage_exceeded plan_entitlement plan_entitlement_usage_exceeded standard"`
	Name            string                     `json:"name"`
	Priority        int64                      `json:"priority"`
	Conditions      JSONSlice[*Condition]      `json:"conditions"`
	ConditionGroups JSONSlice[*ConditionGroup] `json:"condition_groups"`
	Value           bool                       `json:"value"`
}

type Condition struct {
	ID            string                         `json:"id"`
	AccountID     string                         `json:"account_id"`
	EnvironmentID string                         `json:"environment_id"`
	ConditionType ConditionType                  `json:"condition_type" binding:"oneof=base_plan billing_product company credit metric plan plan_version trait user"`
	Operator      typeconvert.ComparableOperator `json:"operator" binding:"oneof=eq ne gt lt gte lte is_empty not_empty"`

	// Fields relevant when ConditionType is one of Company, User, Plan, Plan Version, Base Plan, Billing Product, or Billing Credit
	ResourceIDs JSONSlice[string] `json:"resource_ids"`

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
	Conditions JSONSlice[*Condition] `json:"conditions"`
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

// MarshalJSON ensures a nil collection serializes as `[]` rather than
// `null`, matching the JSONSlice contract that the rest of the wire types
// in this package follow.
func (c CompanyMetricCollection) MarshalJSON() ([]byte, error) {
	if c == nil {
		return []byte("[]"), nil
	}
	return json.Marshal([]*CompanyMetric(c))
}

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
	FeatureKey      string                  `json:"feature_key" desc:"The key of the flag associated with the feature"`
	ValueType       EntitlementValueType    `json:"value_type" binding:"oneof=boolean credit numeric trait unknown unlimited" desc:"The type of the entitlement value"`
	Allocation      *int64                  `json:"allocation" desc:"If the company has a numeric entitlement for this feature, the allocated amount"`
	SoftLimit       *int64                  `json:"soft_limit" desc:"For usage-based pricing, the soft limit for overage charges or the next tier boundary"`
	Usage           *int64                  `json:"usage" desc:"If the company has a numeric entitlement for this feature, the current usage amount"`
	EventName       *string                 `json:"event_name" desc:"If the feature is event-based, the name of the event tracked for usage"`
	MetricPeriod    *MetricPeriod           `json:"metric_period" binding:"oneof=all_time current_day current_month current_week" desc:"For event-based feature entitlements, the period over which usage is tracked"`
	MonthReset      *MetricPeriodMonthReset `json:"month_reset" binding:"oneof=first_of_month billing_cycle" desc:"For event-based feature entitlements that have a monthly period, whether that monthly reset is based on the calendar month or a billing cycle"`
	MetricResetAt   *time.Time              `json:"metric_reset_at" desc:"For event-based feature entitlements, when the usage period will reset"`
	CreditID        *string                 `json:"credit_id" desc:"If the company has a credit-based entitlement for this feature, the ID of the credit"`
	CreditTotal     *float64                `json:"credit_total" desc:"If the company has a credit-based entitlement for this feature, the total credit amount"`
	CreditUsed      *float64                `json:"credit_used" desc:"If the company has a credit-based entitlement for this feature, the amount of credit used"`
	CreditRemaining *float64                `json:"credit_remaining" desc:"If the company has a credit-based entitlement for this feature, the credit available to fund new consumption or a new lease hold — open lease holds are excluded. Clients that hold a lease should gate on this plus their own unspent hold; clients with no lease awareness should use credit_settled instead"`
	CreditReserved  *float64                `json:"credit_reserved,omitempty" desc:"If the company has a credit-based entitlement for this feature, the unspent amount held by an open credit lease. Returns to credit_remaining when the lease is released"`
	CreditSettled   *float64                `json:"credit_settled,omitempty" desc:"If the company has a credit-based entitlement for this feature, the balance net of actual consumption, unaffected by open lease holds (credit_remaining plus credit_reserved). The number to display to end users"`
}

type Company struct {
	ID            string `json:"id"`
	AccountID     string `json:"account_id"`
	EnvironmentID string `json:"environment_id"`

	BasePlanID        *string                        `json:"base_plan_id"`
	BillingProductIDs JSONSlice[string]              `json:"billing_product_ids"`
	CreditBalances    map[string]float64             `json:"credit_balances"`
	Entitlements      JSONSlice[*FeatureEntitlement] `json:"entitlements,omitempty"`
	Keys              map[string]string              `json:"keys"`
	Metrics           CompanyMetricCollection        `json:"metrics"`
	PlanIDs           JSONSlice[string]              `json:"plan_ids"`
	PlanVersionIDs    JSONSlice[string]              `json:"plan_version_ids"`
	Rules             JSONSlice[*Rule]               `json:"rules"`
	Subscription      *Subscription                  `json:"subscription"`
	Traits            JSONSlice[*Trait]              `json:"traits"`

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
	Traits JSONSlice[*Trait] `json:"traits"`
	Rules  JSONSlice[*Rule]  `json:"rules"`
}
