package rulesengine

import (
	"time"

	"github.com/schematichq/rulesengine/null"
)

type MetricPeriod string

const (
	MetricPeriodAllTime      MetricPeriod = "all_time"
	MetricPeriodCurrentDay   MetricPeriod = "current_day"
	MetricPeriodCurrentMonth MetricPeriod = "current_month"
	MetricPeriodCurrentWeek  MetricPeriod = "current_week"
)

// For MetricPeriodMonth, there's an additional option indicating when the month should reset
type MetricPeriodMonthReset string

const (
	MetricPeriodMonthResetFirst   MetricPeriodMonthReset = "first_of_month"
	MetricPeriodMonthResetBilling MetricPeriodMonthReset = "billing_cycle"
)

// Given a calendar-based metric period, return the next metric period reset time
// Will return nil for non-calendar-based metric periods such as all-time or billing cycle
func GetNextMetricPeriodStartForCalendarMetricPeriod(metricPeriod MetricPeriod) *time.Time {
	switch metricPeriod {
	case MetricPeriodCurrentDay:
		// UTC midnight for upcoming day
		tomorrow := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return null.Nullable(tomorrow)
	case MetricPeriodCurrentWeek:
		// UTC midnight for upcoming Sunday
		now := time.Now().UTC()
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			// if it is currently sunday, we want to look forward to the next sunday
			daysUntilSunday = 7
		}
		upcomingSunday := now.Truncate(24 * time.Hour).Add(time.Duration(daysUntilSunday) * 24 * time.Hour)
		return null.Nullable(upcomingSunday)
	case MetricPeriodCurrentMonth:
		// UTC midnight for the first day of next month
		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		nextMonth := firstDayOfCurrentMonth.AddDate(0, 1, 0)
		return null.Nullable(nextMonth)
	}

	return nil
}

// Given a company, determine the next metric period start based on the company's billing GetNextMetricPeriodStartForCompanyBillingSubscription
func GetNextMetricPeriodStartForCompanyBillingSubscription(company *Company) *time.Time {
	// if no subscription exists, we use calendar month reset
	if company == nil || company.Subscription == nil {
		return GetNextMetricPeriodStartForCalendarMetricPeriod(MetricPeriodCurrentMonth)
	}

	now := time.Now().UTC()
	periodEnd := company.Subscription.PeriodEnd
	periodStart := company.Subscription.PeriodStart

	// if the start period is in the future, the metric period is from the start of the current calendar month until either
	// the end of the current calendar month or the start of the billing period, whichever comes first
	if periodStart.After(now) {
		startOfNextMonth := GetNextMetricPeriodStartForCalendarMetricPeriod(MetricPeriodCurrentMonth)
		if periodStart.After(*startOfNextMonth) {
			return startOfNextMonth
		}

		return &periodStart
	}

	// month metric period will reset on the same day/hour/minute/second as the susbcription started every month; get that timestamp for the current month
	nextReset := time.Date(
		now.Year(),
		now.Month(),
		periodStart.Day(),
		periodStart.Hour(),
		periodStart.Minute(),
		periodStart.Second(),
		periodStart.Nanosecond(),
		time.UTC,
	)

	// if we've already passed this month's reset date, move to next month
	if !nextReset.After(now) {
		nextReset = nextReset.AddDate(0, 1, 0)
	}

	// if the next reset is after the end of the billing period, use the end of the billing period instead
	if nextReset.After(periodEnd) {
		return &periodEnd
	}

	return &nextReset
}

// Given a rule condition and a company, determine the next metric period start
// Will return nil if the condition is not a metric condition
func GetNextMetricPeriodStartFromCondition(
	condition *Condition,
	company *Company,
) *time.Time {
	// Only metric conditions have a metric period that can reset
	if condition == nil || condition.ConditionType != ConditionTypeMetric {
		return nil
	}

	// If the metric period is all-time, no reset
	if condition.MetricPeriod == nil {
		return nil
	}

	// Metric period current month with billing cycle reset
	monthReset := condition.MetricPeriodMonthReset
	if *condition.MetricPeriod == MetricPeriodCurrentMonth && monthReset != nil && *monthReset == MetricPeriodMonthResetBilling {
		return GetNextMetricPeriodStartForCompanyBillingSubscription(company)
	}

	// Calendar-based metric periods
	return GetNextMetricPeriodStartForCalendarMetricPeriod(*condition.MetricPeriod)
}
