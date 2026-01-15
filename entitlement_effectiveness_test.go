package rulesengine

import (
	"testing"
)

func ptr[T any](v T) *T {
	return &v
}

func TestNormalizeAllocationToDailyRate(t *testing.T) {
	tests := []struct {
		name       string
		allocation *int64
		period     *MetricPeriod
		want       *float64
	}{
		{
			name:       "nil allocation returns nil (unlimited)",
			allocation: nil,
			period:     ptr(MetricPeriodCurrentDay),
			want:       nil,
		},
		{
			name:       "nil period returns nil",
			allocation: ptr(int64(100)),
			period:     nil,
			want:       nil,
		},
		{
			name:       "all_time period returns nil",
			allocation: ptr(int64(100)),
			period:     ptr(MetricPeriodAllTime),
			want:       nil,
		},
		{
			name:       "current_day returns allocation as-is",
			allocation: ptr(int64(100)),
			period:     ptr(MetricPeriodCurrentDay),
			want:       ptr(float64(100)),
		},
		{
			name:       "current_week divides by 7",
			allocation: ptr(int64(70)),
			period:     ptr(MetricPeriodCurrentWeek),
			want:       ptr(float64(10)),
		},
		{
			name:       "current_month divides by 30",
			allocation: ptr(int64(300)),
			period:     ptr(MetricPeriodCurrentMonth),
			want:       ptr(float64(10)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeAllocationToDailyRate(tt.allocation, tt.period)
			if tt.want == nil {
				if got != nil {
					t.Errorf("NormalizeAllocationToDailyRate() = %v, want nil", *got)
				}
			} else {
				if got == nil {
					t.Errorf("NormalizeAllocationToDailyRate() = nil, want %v", *tt.want)
				} else if *got != *tt.want {
					t.Errorf("NormalizeAllocationToDailyRate() = %v, want %v", *got, *tt.want)
				}
			}
		})
	}
}

func TestIsAllocationMoreGenerous(t *testing.T) {
	daily := MetricPeriodCurrentDay
	weekly := MetricPeriodCurrentWeek
	monthly := MetricPeriodCurrentMonth

	tests := []struct {
		name    string
		alloc1  *int64
		period1 *MetricPeriod
		alloc2  *int64
		period2 *MetricPeriod
		want    bool
	}{
		{
			name:    "unlimited (nil) is more generous than limited",
			alloc1:  nil,
			period1: &daily,
			alloc2:  ptr(int64(100)),
			period2: &daily,
			want:    true,
		},
		{
			name:    "limited is not more generous than unlimited",
			alloc1:  ptr(int64(100)),
			period1: &daily,
			alloc2:  nil,
			period2: &daily,
			want:    false,
		},
		{
			name:    "same period: higher allocation is more generous",
			alloc1:  ptr(int64(200)),
			period1: &daily,
			alloc2:  ptr(int64(100)),
			period2: &daily,
			want:    true,
		},
		{
			name:    "same period: lower allocation is not more generous",
			alloc1:  ptr(int64(50)),
			period1: &daily,
			alloc2:  ptr(int64(100)),
			period2: &daily,
			want:    false,
		},
		{
			name:    "cross-period: weekly 70 (10/day) vs daily 8 - weekly is more generous",
			alloc1:  ptr(int64(70)),
			period1: &weekly,
			alloc2:  ptr(int64(8)),
			period2: &daily,
			want:    true,
		},
		{
			name:    "cross-period: daily 15 vs weekly 70 (10/day) - daily is more generous",
			alloc1:  ptr(int64(15)),
			period1: &daily,
			alloc2:  ptr(int64(70)),
			period2: &weekly,
			want:    true,
		},
		{
			name:    "cross-period: monthly 300 (10/day) vs daily 10 - equal, not more generous",
			alloc1:  ptr(int64(300)),
			period1: &monthly,
			alloc2:  ptr(int64(10)),
			period2: &daily,
			want:    false,
		},
		{
			name:    "no period (trait-based): compare raw values - higher wins",
			alloc1:  ptr(int64(200)),
			period1: nil,
			alloc2:  ptr(int64(100)),
			period2: nil,
			want:    true,
		},
		{
			name:    "no period (trait-based): lower is not more generous",
			alloc1:  ptr(int64(50)),
			period1: nil,
			alloc2:  ptr(int64(100)),
			period2: nil,
			want:    false,
		},
		{
			name:    "e1 has period, e2 doesn't - e1 is more generous",
			alloc1:  ptr(int64(100)),
			period1: &daily,
			alloc2:  ptr(int64(100)),
			period2: nil,
			want:    true,
		},
		{
			name:    "e1 has no period, e2 has period - e1 is not more generous",
			alloc1:  ptr(int64(100)),
			period1: nil,
			alloc2:  ptr(int64(100)),
			period2: &daily,
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsAllocationMoreGenerous(tt.alloc1, tt.period1, tt.alloc2, tt.period2)
			if got != tt.want {
				t.Errorf("IsAllocationMoreGenerous() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldBooleanOverrideWin(t *testing.T) {
	tests := []struct {
		name         string
		newType      EntitlementType
		existingType EntitlementType
		want         bool
	}{
		{
			name:         "company override wins over plan entitlement",
			newType:      EntitlementTypeCompanyOverride,
			existingType: EntitlementTypePlanEntitlement,
			want:         true,
		},
		{
			name:         "plan entitlement does not win over plan entitlement",
			newType:      EntitlementTypePlanEntitlement,
			existingType: EntitlementTypePlanEntitlement,
			want:         false,
		},
		{
			name:         "company override does not win over company override",
			newType:      EntitlementTypeCompanyOverride,
			existingType: EntitlementTypeCompanyOverride,
			want:         false,
		},
		{
			name:         "plan entitlement does not win over company override",
			newType:      EntitlementTypePlanEntitlement,
			existingType: EntitlementTypeCompanyOverride,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldBooleanOverrideWin(tt.newType, tt.existingType)
			if got != tt.want {
				t.Errorf("ShouldBooleanOverrideWin() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldBooleanPlanLose(t *testing.T) {
	tests := []struct {
		name         string
		newType      EntitlementType
		existingType EntitlementType
		want         bool
	}{
		{
			name:         "plan entitlement loses to company override",
			newType:      EntitlementTypePlanEntitlement,
			existingType: EntitlementTypeCompanyOverride,
			want:         true,
		},
		{
			name:         "company override does not lose to plan entitlement",
			newType:      EntitlementTypeCompanyOverride,
			existingType: EntitlementTypePlanEntitlement,
			want:         false,
		},
		{
			name:         "plan entitlement does not lose to plan entitlement",
			newType:      EntitlementTypePlanEntitlement,
			existingType: EntitlementTypePlanEntitlement,
			want:         false,
		},
		{
			name:         "company override does not lose to company override",
			newType:      EntitlementTypeCompanyOverride,
			existingType: EntitlementTypeCompanyOverride,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldBooleanPlanLose(tt.newType, tt.existingType)
			if got != tt.want {
				t.Errorf("ShouldBooleanPlanLose() = %v, want %v", got, tt.want)
			}
		})
	}
}
