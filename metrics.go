package rulesengine

import (
	"fmt"
	"time"
)

type MetricPeriod string

const (
	MetricPeriodAllTime      MetricPeriod = "all_time"
	MetricPeriodCurrentDay   MetricPeriod = "current_day"
	MetricPeriodCurrentMonth MetricPeriod = "current_month"
	MetricPeriodCurrentWeek  MetricPeriod = "current_week"
)

// ToInt converts MetricPeriod to its integer representation
func (mp MetricPeriod) ToInt() int {
	switch mp {
	case MetricPeriodAllTime:
		return 0
	case MetricPeriodCurrentDay:
		return 1
	case MetricPeriodCurrentWeek:
		return 2
	case MetricPeriodCurrentMonth:
		return 3
	default:
		return 0
	}
}

// Format implements fmt.Formatter interface
func (mp MetricPeriod) Format(f fmt.State, verb rune) {
	switch verb {
	case 'd':
		fmt.Fprintf(f, "%d", mp.ToInt())
	case 's':
		fmt.Fprintf(f, "%s", string(mp))
	case 'v':
		if f.Flag('#') {
			fmt.Fprintf(f, "MetricPeriod(%s)", string(mp))
		} else {
			fmt.Fprintf(f, "%s", string(mp))
		}
	default:
		fmt.Fprintf(f, "%%!%c(MetricPeriod=%s)", verb, string(mp))
	}
}

// MetricPeriodFromInt converts an integer to MetricPeriod
func MetricPeriodFromInt(i int) MetricPeriod {
	switch i {
	case 0:
		return MetricPeriodAllTime
	case 1:
		return MetricPeriodCurrentDay
	case 2:
		return MetricPeriodCurrentWeek
	case 3:
		return MetricPeriodCurrentMonth
	default:
		return MetricPeriodAllTime
	}
}

// For MetricPeriodMonth, there's an additional option indicating when the month should reset
type MetricPeriodMonthReset string

const (
	MetricPeriodMonthResetFirst   MetricPeriodMonthReset = "first_of_month"
	MetricPeriodMonthResetBilling MetricPeriodMonthReset = "billing_cycle"
)

// ToInt converts MetricPeriodMonthReset to its integer representation
func (mr MetricPeriodMonthReset) ToInt() int {
	switch mr {
	case MetricPeriodMonthResetFirst:
		return 0
	case MetricPeriodMonthResetBilling:
		return 1
	default:
		return 0
	}
}

// Format implements fmt.Formatter interface
func (mr MetricPeriodMonthReset) Format(f fmt.State, verb rune) {
	switch verb {
	case 'd':
		fmt.Fprintf(f, "%d", mr.ToInt())
	case 's':
		fmt.Fprintf(f, "%s", string(mr))
	case 'v':
		if f.Flag('#') {
			fmt.Fprintf(f, "MetricPeriodMonthReset(%s)", string(mr))
		} else {
			fmt.Fprintf(f, "%s", string(mr))
		}
	default:
		fmt.Fprintf(f, "%%!%c(MetricPeriodMonthReset=%s)", verb, string(mr))
	}
}

// MetricPeriodMonthResetFromInt converts an integer to MetricPeriodMonthReset
func MetricPeriodMonthResetFromInt(i int) MetricPeriodMonthReset {
	switch i {
	case 0:
		return MetricPeriodMonthResetFirst
	case 1:
		return MetricPeriodMonthResetBilling
	default:
		return MetricPeriodMonthResetFirst
	}
}

// Given a calendar-based metric period, return the beginning of the current metric period
// Will return nil for non-calendar-based metric periods such as all-time or billing cycle
func GetCurrentMetricPeriodStartForCalendarMetricPeriod(metricPeriod MetricPeriod) *time.Time {
	switch metricPeriod {
	case MetricPeriodCurrentDay:
		// UTC midnight for the current day
		today := time.Now().UTC().Truncate(24 * time.Hour)
		return &today
	case MetricPeriodCurrentWeek:
		// UTC midnight for the most recent Sunday
		now := time.Now().UTC()
		daysSinceSunday := int(now.Weekday())
		currentSunday := now.Truncate(24 * time.Hour).Add(-time.Duration(daysSinceSunday) * 24 * time.Hour)
		return &currentSunday
	case MetricPeriodCurrentMonth:
		// UTC midnight for the first day of current month
		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return &firstDayOfCurrentMonth
	}

	return nil
}

// Given a company, determine the beginning of the current metric period based on the company's billing subscription
func GetCurrentMetricPeriodStartForCompanyBillingSubscription(company *Company) *time.Time {
	// if no subscription exists, we use calendar month reset
	if company == nil || company.Subscription == nil {
		return GetCurrentMetricPeriodStartForCalendarMetricPeriod(MetricPeriodCurrentMonth)
	}

	now := time.Now().UTC()
	periodStart := company.Subscription.PeriodStart

	// if the start period is in the future, the metric period is from the start of the current calendar month
	if periodStart.After(now) {
		return GetCurrentMetricPeriodStartForCalendarMetricPeriod(MetricPeriodCurrentMonth)
	}

	// find the most recent reset date based on subscription start date
	currentReset := time.Date(
		now.Year(),
		now.Month(),
		periodStart.Day(),
		periodStart.Hour(),
		periodStart.Minute(),
		periodStart.Second(),
		periodStart.Nanosecond(),
		time.UTC,
	)

	// if the reset date for current month is in the future, use previous month's reset date
	if currentReset.After(now) {
		currentReset = currentReset.AddDate(0, -1, 0)
	}

	// if the current reset is before the subscription period start, use the period start instead
	if currentReset.Before(periodStart) {
		return &periodStart
	}

	return &currentReset
}

// Given a calendar-based metric period, return the next metric period reset time
// Will return nil for non-calendar-based metric periods such as all-time or billing cycle
func GetNextMetricPeriodStartForCalendarMetricPeriod(metricPeriod MetricPeriod) *time.Time {
	switch metricPeriod {
	case MetricPeriodCurrentDay:
		// UTC midnight for upcoming day
		tomorrow := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)
		return &tomorrow
	case MetricPeriodCurrentWeek:
		// UTC midnight for upcoming Sunday
		now := time.Now().UTC()
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			// if it is currently sunday, we want to look forward to the next sunday
			daysUntilSunday = 7
		}
		upcomingSunday := now.Truncate(24 * time.Hour).Add(time.Duration(daysUntilSunday) * 24 * time.Hour)
		return &upcomingSunday
	case MetricPeriodCurrentMonth:
		// UTC midnight for the first day of next month
		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		nextMonth := firstDayOfCurrentMonth.AddDate(0, 1, 0)
		return &nextMonth
	}

	return nil
}

// Given a company, determine the next metric period start based on the company's billing subscription
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
