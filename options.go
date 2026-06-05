package rulesengine

// CheckFlagOption configures a CheckFlag invocation. Functional options let
// callers express preflight semantics ("simulate this much additional usage"
// or "this is the exact credit cost") without expanding the positional
// signature of CheckFlag.
//
// Zero quantities passed to WithUsage or WithEventUsage are treated as no-ops
// on the condition types where they would apply, and the engine falls through
// to the next precedence tier. Negative values are rejected by CheckFlag with
// ErrorNegativePreflightUsage / ErrorNegativePreflightCreditCost — they
// indicate a caller-side bug rather than a meaningful preflight intent.
//
// Preflight options influence whether a rule matches, but they do not
// rewrite the usage figures reported back on the result. CheckFlagResult's
// FeatureUsage still reflects the company's current (un-simulated) value —
// callers know their own pending quantity and can add it if they need a
// post-preflight projection.
type CheckFlagOption func(*checkFlagOptions)

// checkFlagOptions is the resolved bag of optional inputs threaded through to
// each condition check. Empty/zero values == legacy behavior (the engine
// evaluates each condition using only the condition's own configuration).
//
// Callers can provide multiple options; the engine picks the most specific
// one for each condition it evaluates. For credit-balance conditions:
// CreditCost (keyed by credit_id) wins over EventUsage (when its subtype
// matches) wins over Usage (generic). For metric conditions: EventUsage
// wins over Usage when the subtype matches. Trait conditions only ever
// use Usage.
type checkFlagOptions struct {
	// creditCost is keyed by billing_credit_id → a per-call cost in credits
	// the caller has already computed. When a credit-balance condition
	// matches one of these keys, the engine gates on `balance >= cost`
	// directly, with no quantity or consumption_rate math.
	creditCost map[string]float64

	// usage is a generic simulated quantity applied to whatever numeric
	// condition is being evaluated. For metric and trait-int conditions,
	// the quantity is added to the current value. For credit-balance
	// conditions, the credit cost compared against the balance is
	// `quantity × consumption_rate`. Use when the caller doesn't need to
	// disambiguate across event subtypes.
	usage *int64

	// eventUsage is a single (event_subtype, quantity) pair. Applied to
	// metric conditions whose event_subtype matches (quantity is added to
	// the current metric value) and to credit-balance conditions whose
	// event_subtype matches (compared as `quantity × consumption_rate`
	// against the balance). Deliberately singular: one check preflights one
	// action, and per-condition gating doesn't aggregate costs across
	// subtypes, so multiple pairs against the same credit balance could
	// each pass while the combined cost overdraws.
	eventUsage *eventUsage
}

// eventUsage pairs an event_subtype with a simulated quantity for preflight.
type eventUsage struct {
	eventSubtype string
	quantity     int64
}

// newCheckFlagOptions returns a zero-valued checkFlagOptions with its maps
// initialized, so option setters don't need to nil-check before writing.
func newCheckFlagOptions() *checkFlagOptions {
	return &checkFlagOptions{
		creditCost: make(map[string]float64),
	}
}

// validate rejects negative preflight values at the CheckFlag boundary. Zero
// remains a valid no-op for usage/event_usage and the "this call is free"
// semantic for credit_cost; negatives are programming errors and surfaced as
// such so callers can fix the source bug instead of silently misbehaving.
func (o *checkFlagOptions) validate() error {
	if o.usage != nil && *o.usage < 0 {
		return ErrorNegativePreflightUsage
	}
	if o.eventUsage != nil && o.eventUsage.quantity < 0 {
		return ErrorNegativePreflightUsage
	}
	for _, c := range o.creditCost {
		if c < 0 {
			return ErrorNegativePreflightCreditCost
		}
	}
	return nil
}

// WithCreditCost gates a credit-balance condition on `balance >= cost` when
// the condition's credit_id matches. Lets callers supply an already-
// computed per-call cost in credits, bypassing the engine's default
// quantity × consumption_rate math. Highest precedence on credit-balance
// conditions when supplied. Call multiple times to attach costs for
// several credit types in the same check.
//
// Unlike WithUsage / WithEventUsage, a zero cost is not treated as a no-op:
// cost is gated as-is, so `WithCreditCost(id, 0)` passes whenever the balance
// is non-negative (the "this call is free" semantic). Callers who want to
// skip the override should omit the option entirely. Negative costs are
// rejected by CheckFlag with ErrorNegativePreflightCreditCost.
func WithCreditCost(creditID string, cost float64) CheckFlagOption {
	return func(o *checkFlagOptions) {
		o.creditCost[creditID] = cost
	}
}

// WithUsage simulates additional usage of a generic quantity for any numeric
// condition encountered while evaluating rules. For metric conditions, the
// quantity is added to the current metric value. For trait conditions with
// an int-comparable trait, it's added to the trait value. For credit-balance
// conditions, the credit cost compared against the balance is
// `quantity × consumption_rate`. Zero is a no-op; negative quantities are
// rejected by CheckFlag with ErrorNegativePreflightUsage.
func WithUsage(quantity int64) CheckFlagOption {
	return func(o *checkFlagOptions) {
		o.usage = &quantity
	}
}

// WithEventUsage simulates additional usage of a specific event_subtype.
// Applied to metric conditions whose event_subtype matches (quantity is
// added to the current metric value) and to credit-balance conditions whose
// event_subtype matches (compared as `quantity × consumption_rate` against
// the balance). Preferred over WithUsage on those conditions when the
// caller knows the specific subtype. Zero is a no-op; negative quantities
// are rejected by CheckFlag with ErrorNegativePreflightUsage. Calling this
// more than once replaces the previous pair (last write wins, matching
// WithUsage).
func WithEventUsage(eventSubtype string, quantity int64) CheckFlagOption {
	return func(o *checkFlagOptions) {
		o.eventUsage = &eventUsage{eventSubtype: eventSubtype, quantity: quantity}
	}
}
