package rulesengine

// NormalizeAllocationToDailyRate converts an allocation to a per-day equivalent for comparison.
// Returns nil for unlimited allocations (nil input) or all-time periods (can't normalize).
//
// Daily rates:
//   - current_day: allocation as-is
//   - current_week: allocation / 7
//   - current_month: allocation / 30
func NormalizeAllocationToDailyRate(allocation *int64, period *MetricPeriod) *float64 {
	if allocation == nil {
		// nil means unlimited
		return nil
	}

	if period == nil || *period == MetricPeriodAllTime {
		// all_time allocations can't be normalized to daily rate
		// treat as unlimited for comparison purposes
		return nil
	}

	allocationFloat := float64(*allocation)
	var dailyRate float64

	switch *period {
	case MetricPeriodCurrentDay:
		dailyRate = allocationFloat
	case MetricPeriodCurrentWeek:
		dailyRate = allocationFloat / 7.0
	case MetricPeriodCurrentMonth:
		dailyRate = allocationFloat / 30.0
	default:
		// Unknown period, treat as the raw value
		dailyRate = allocationFloat
	}

	return &dailyRate
}

// IsAllocationMoreGenerous returns true if allocation1 is more generous than allocation2
// by normalizing both to daily rates and comparing.
//
// Rules:
//   - Unlimited (nil) is always more generous than limited
//   - Higher daily rate is more generous
//   - Period-based allocations are more generous than no-period allocations
func IsAllocationMoreGenerous(alloc1 *int64, period1 *MetricPeriod, alloc2 *int64, period2 *MetricPeriod) bool {
	// If allocation1 is unlimited (nil), it's more generous
	if alloc1 == nil {
		return true
	}

	// If allocation2 is unlimited but allocation1 isn't, allocation1 is less generous
	if alloc2 == nil {
		return false
	}

	// If both have no period (e.g., trait-based features), compare directly
	if period1 == nil && period2 == nil {
		return *alloc1 > *alloc2
	}

	// If one has a period and the other doesn't, prefer the one with a period
	if period1 == nil && period2 != nil {
		return false // Keep period-based
	}
	if period1 != nil && period2 == nil {
		return true // Use period-based
	}

	// Both have periods - normalize to daily rates
	daily1 := NormalizeAllocationToDailyRate(alloc1, period1)
	daily2 := NormalizeAllocationToDailyRate(alloc2, period2)

	// If normalization returned nil (e.g., all_time period), compare raw allocations
	if daily1 == nil || daily2 == nil {
		return *alloc1 > *alloc2
	}

	// Both have comparable daily rates
	return *daily1 > *daily2
}

// ShouldBooleanOverrideWin returns true if a boolean company override should
// take precedence over an existing boolean plan entitlement.
//
// For boolean entitlements, company overrides ALWAYS win to support "negative overrides"
// (e.g., override value=false disabling a plan feature with value=true).
func ShouldBooleanOverrideWin(newType, existingType EntitlementType) bool {
	return newType == EntitlementTypeCompanyOverride && existingType == EntitlementTypePlanEntitlement
}

// ShouldBooleanPlanLose returns true if a boolean plan entitlement should NOT
// override an existing boolean company override.
func ShouldBooleanPlanLose(newType, existingType EntitlementType) bool {
	return newType == EntitlementTypePlanEntitlement && existingType == EntitlementTypeCompanyOverride
}
