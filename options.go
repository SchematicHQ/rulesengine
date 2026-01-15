package rulesengine

type CheckFlagOption func(*checkFlagOptions)

type checkFlagOptions struct {
	usage      *int64
	eventUsage map[string]int64
}

// WithUsage increments the "usage" value for any numeric condition encountered while checking rules
// (trait, metric, credits). This is best for cases where the flag has only one type of rules with
// one type of trait, metric, or credit.
func WithUsage(quantity int64) CheckFlagOption {
	return func(o *checkFlagOptions) {
		o.usage = &quantity
	}
}

// WithEventUsage specifies a specific event subtype, and for credit or metric conditions we check
// as if this additional usage had occurred.
func WithEventUsage(eventSubtype string, quantity int64) CheckFlagOption {
	return func(o *checkFlagOptions) {
		if o.eventUsage == nil {
			o.eventUsage = make(map[string]int64)
		}
		o.eventUsage[eventSubtype] = quantity
	}
}
