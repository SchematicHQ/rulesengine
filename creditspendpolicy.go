package rulesengine

import (
	"fmt"
	"strconv"
)

// CreditSpendPolicyKind is the extension point: a new way to limit spending
// arrives as a new kind here, not as another field on Company and User.
type CreditSpendPolicyKind string

const (
	CreditSpendPolicyKindPerDraw CreditSpendPolicyKind = "per_draw"
	CreditSpendPolicyKindWindow  CreditSpendPolicyKind = "window"
)

type CreditSpendPolicyScope string

const (
	CreditSpendPolicyScopeCompany CreditSpendPolicyScope = "company"
	CreditSpendPolicyScopeUser    CreditSpendPolicyScope = "user"
	CreditSpendPolicyScopeGroup   CreditSpendPolicyScope = "group"
)

type CreditSpendWindow struct {
	Unit  string `json:"unit" desc:"The period the limit accumulates over"`
	Count int    `json:"count" desc:"How many units make up one period"`
}

type CreditSpendPolicy struct {
	ID       string                 `json:"id" desc:"The ID of the policy"`
	CreditID string                 `json:"credit_id" desc:"The credit the policy limits"`
	Kind     CreditSpendPolicyKind  `json:"kind" desc:"How the limit is applied"`
	Scope    CreditSpendPolicyScope `json:"scope" desc:"Whether the policy limits the company or one user"`
	Label    *string                `json:"label,omitempty" desc:"The name the account gave the policy"`
	Limit    float64                `json:"limit" desc:"The ceiling, in credits"`
	Consumed float64                `json:"consumed,omitempty" desc:"How much of the limit is already spent in the current period"`
	Window   *CreditSpendWindow     `json:"window,omitempty" desc:"For a windowed limit, the period it accumulates over"`
}

func (p *CreditSpendPolicy) Remaining() float64 {
	if p == nil {
		return 0
	}

	return max(p.Limit-p.Consumed, 0)
}

// Equal compares Window and Label by value: both are pointers rebuilt on every
// projection, so a pointer compare would call every rebuild a change.
func (p *CreditSpendPolicy) Equal(other *CreditSpendPolicy) bool {
	if p == nil || other == nil {
		return p == other
	}

	if p.ID != other.ID ||
		p.CreditID != other.CreditID ||
		p.Kind != other.Kind ||
		p.Scope != other.Scope ||
		p.Limit != other.Limit ||
		p.Consumed != other.Consumed {
		return false
	}

	if (p.Label == nil) != (other.Label == nil) {
		return false
	}
	if p.Label != nil && *p.Label != *other.Label {
		return false
	}

	if (p.Window == nil) != (other.Window == nil) {
		return false
	}
	if p.Window != nil && *p.Window != *other.Window {
		return false
	}

	return true
}

// Allows reports whether a draw costing cost fits inside this policy. evaluated
// is false for a kind this engine does not recognise; callers treat that as
// absent, so an engine older than the kind lets the draw through rather than
// blocking every check against the credit.
func (p *CreditSpendPolicy) Allows(cost float64) (allowed bool, evaluated bool) {
	if p == nil {
		return true, false
	}

	switch p.Kind {
	case CreditSpendPolicyKindPerDraw:
		return cost <= p.Limit, true
	case CreditSpendPolicyKindWindow:
		return p.Consumed+cost <= p.Limit, true
	default:
		return true, false
	}
}

func (p *CreditSpendPolicy) Describe() string {
	if p == nil {
		return ""
	}

	limit := formatCreditAmount(p.Limit)
	switch p.Kind {
	case CreditSpendPolicyKindWindow:
		if p.Window == nil {
			return fmt.Sprintf("%s limit of %s credits per period", p.Scope, limit)
		}
		if p.Window.Count == 1 {
			return fmt.Sprintf("%s limit of %s credits per %s", p.Scope, limit, p.Window.Unit)
		}
		return fmt.Sprintf("%s limit of %s credits per %d %ss", p.Scope, limit, p.Window.Count, p.Window.Unit)
	default:
		return fmt.Sprintf("%s limit of %s credits per request", p.Scope, limit)
	}
}

func formatCreditAmount(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// creditSpendPolicyRefusal returns the first policy refusing a draw of cost, or
// nil when every applicable policy allows it. Any refusal blocks: kinds are not
// comparable by a single number, so there is no "tightest" one to pick.
func creditSpendPolicyRefusal(scope *CheckScope, creditID string, cost float64) *CreditSpendPolicy {
	if scope == nil {
		return nil
	}

	var policies []*CreditSpendPolicy
	if scope.Company != nil {
		policies = append(policies, scope.Company.CreditSpendPolicies...)
	}
	if scope.User != nil {
		policies = append(policies, scope.User.CreditSpendPolicies...)
	}

	for _, policy := range policies {
		if policy == nil || policy.CreditID != creditID {
			continue
		}
		allowed, evaluated := policy.Allows(cost)
		if evaluated && !allowed {
			return policy
		}
	}

	return nil
}
