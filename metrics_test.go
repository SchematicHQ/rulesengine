package rulesengine_test

import (
	"testing"
	"time"

	"github.com/schematichq/rulesengine"
	"github.com/stretchr/testify/assert"
)

func TestGetCurrentMetricPeriodStartForCalendarMetricPeriod(t *testing.T) {
	t.Run("MetricPeriodCurrentDay", func(t *testing.T) {
		result := rulesengine.GetCurrentMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentDay)
		assert.NotNil(t, result)

		expected := time.Now().UTC().Truncate(24 * time.Hour)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
	})

	t.Run("MetricPeriodCurrentWeek", func(t *testing.T) {
		result := rulesengine.GetCurrentMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentWeek)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		daysSinceSunday := int(now.Weekday())
		expected := now.Truncate(24 * time.Hour).Add(-time.Duration(daysSinceSunday) * 24 * time.Hour)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
		assert.Equal(t, time.Sunday, result.Weekday())
	})

	t.Run("MetricPeriodCurrentMonth", func(t *testing.T) {
		result := rulesengine.GetCurrentMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentMonth)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("MetricPeriodAllTime", func(t *testing.T) {
		result := rulesengine.GetCurrentMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodAllTime)
		assert.Nil(t, result)
	})
}

func TestGetCurrentMetricPeriodStartForCompanyBillingSubscription(t *testing.T) {
	t.Run("Company is nil", func(t *testing.T) {
		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(nil)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("Company subscription is nil", func(t *testing.T) {
		company := createTestCompany()
		company.Subscription = nil
		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("Subscription period start is in future", func(t *testing.T) {
		company := createTestCompany()
		company.Subscription.PeriodStart = time.Now().Add(7 * 24 * time.Hour)

		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		expected := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("Current month reset date has not passed yet", func(t *testing.T) {
		now := time.Now().UTC()
		company := createTestCompany()

		// Set subscription to start on a day later in the month than today
		futureDay := now.Day() + 5
		if futureDay > 28 {
			futureDay = 28 // Avoid month boundary issues
		}

		company.Subscription.PeriodStart = time.Date(
			now.Year()-1,
			now.Month(),
			futureDay,
			12, 0, 0, 0,
			time.UTC,
		)

		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// In this case, the result should be last month's reset date
		expectedMonth := now.Month() - 1
		expectedYear := now.Year()
		if now.Month() == time.January {
			expectedMonth = time.December
			expectedYear = now.Year() - 1
		}

		expected := time.Date(
			expectedYear,
			expectedMonth,
			futureDay,
			12, 0, 0, 0,
			time.UTC,
		)

		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, expected.Hour(), result.Hour())
	})

	t.Run("Current month reset date has passed", func(t *testing.T) {
		now := time.Now().UTC()
		company := createTestCompany()

		// Set subscription to start on a day earlier in the month than today
		pastDay := now.Day() - 5
		if pastDay < 1 {
			pastDay = 1
		}

		company.Subscription.PeriodStart = time.Date(
			now.Year()-1,
			now.Month(),
			pastDay,
			12, 0, 0, 0,
			time.UTC,
		)

		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// In this case, the result should be this month's reset date
		expected := time.Date(
			now.Year(),
			now.Month(),
			pastDay,
			12, 0, 0, 0,
			time.UTC,
		)

		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, expected.Hour(), result.Hour())
	})

	t.Run("Reset date is before subscription period start", func(t *testing.T) {
		now := time.Now().UTC()
		company := createTestCompany()

		// Set a recent subscription start date (10 days ago)
		company.Subscription.PeriodStart = time.Date(
			now.Year(),
			now.Month(),
			now.Day()-10,
			12, 0, 0, 0,
			time.UTC,
		)

		// Set the subscription to have started recently, so any computed reset date
		// that's earlier than 10 days ago should be replaced with the period start
		result := rulesengine.GetCurrentMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// The result should be the period start date
		assert.Equal(t, company.Subscription.PeriodStart.Year(), result.Year())
		assert.Equal(t, company.Subscription.PeriodStart.Month(), result.Month())
		assert.Equal(t, company.Subscription.PeriodStart.Day(), result.Day())
		assert.Equal(t, company.Subscription.PeriodStart.Hour(), result.Hour())
	})
}

func TestGetNextMetricPeriodStartForCalendarMetricPeriod(t *testing.T) {
	t.Run("MetricPeriodCurrentDay", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentDay)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		expected := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
	})

	t.Run("MetricPeriodCurrentWeek", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentWeek)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		daysUntilSunday := (7 - int(now.Weekday())) % 7
		if daysUntilSunday == 0 {
			daysUntilSunday = 7
		}
		expected := now.Truncate(24 * time.Hour).Add(time.Duration(daysUntilSunday) * 24 * time.Hour)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
		assert.Equal(t, time.Sunday, result.Weekday())
	})

	t.Run("MetricPeriodCurrentMonth", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodCurrentMonth)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		expected := firstDayOfCurrentMonth.AddDate(0, 1, 0)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, 0, result.Hour())
		assert.Equal(t, 0, result.Minute())
		assert.Equal(t, 0, result.Second())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("MetricPeriodAllTime", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartForCalendarMetricPeriod(rulesengine.MetricPeriodAllTime)
		assert.Nil(t, result)
	})
}

func TestGetNextMetricPeriodStartForCompanyBillingSubscription(t *testing.T) {
	t.Run("Company is nil", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(nil)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		expected := firstDayOfCurrentMonth.AddDate(0, 1, 0)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("Company subscription is nil", func(t *testing.T) {
		company := createTestCompany()
		company.Subscription = nil
		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		now := time.Now().UTC()
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		expected := firstDayOfCurrentMonth.AddDate(0, 1, 0)
		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, 1, result.Day())
	})

	t.Run("Subscription period start is in future", func(t *testing.T) {
		company := createTestCompany()
		now := time.Now().UTC()

		// Set subscription to start 7 days from now
		company.Subscription.PeriodStart = now.AddDate(0, 0, 7)

		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// If period start is sooner than next month, period start should be used
		firstDayOfCurrentMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		startOfNextMonth := firstDayOfCurrentMonth.AddDate(0, 1, 0)

		if company.Subscription.PeriodStart.After(startOfNextMonth) {
			// Period start is after next month, so next month should be used
			assert.Equal(t, startOfNextMonth.Unix(), result.Unix())
		} else {
			// Period start is before next month, so period start should be used
			assert.Equal(t, company.Subscription.PeriodStart.Unix(), result.Unix())
		}
	})

	t.Run("Next reset date is after subscription end", func(t *testing.T) {
		company := createTestCompany()
		now := time.Now().UTC()

		// Set subscription to have started some time ago
		company.Subscription.PeriodStart = now.AddDate(0, -6, 0)

		// But set it to end soon (before the next monthly reset)
		company.Subscription.PeriodEnd = now.AddDate(0, 0, 10)

		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// The result should be the period end date
		assert.Equal(t, company.Subscription.PeriodEnd.Unix(), result.Unix())
	})

	t.Run("Current month reset date has passed", func(t *testing.T) {
		now := time.Now().UTC()
		company := createTestCompany()

		// Set subscription to start on a day earlier in the month than today
		pastDay := now.Day() - 5
		if pastDay < 1 {
			pastDay = 1
		}

		company.Subscription.PeriodStart = time.Date(
			now.Year()-1,
			now.Month(),
			pastDay,
			12, 0, 0, 0,
			time.UTC,
		)

		company.Subscription.PeriodEnd = now.AddDate(1, 0, 0) // Set end date far in the future

		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// In this case, the result should be next month's reset date
		expected := time.Date(
			now.Year(),
			now.Month()+1,
			pastDay,
			12, 0, 0, 0,
			time.UTC,
		)
		// Handle December to January transition
		if now.Month() == time.December {
			expected = time.Date(
				now.Year()+1,
				time.January,
				pastDay,
				12, 0, 0, 0,
				time.UTC,
			)
		}

		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, expected.Hour(), result.Hour())
	})

	t.Run("Current month reset date has not passed yet", func(t *testing.T) {
		now := time.Now().UTC()
		company := createTestCompany()

		// Set subscription to start on a day later in the month than today
		futureDay := now.Day() + 5
		if futureDay > 28 {
			futureDay = 28 // Avoid month boundary issues
		}

		company.Subscription.PeriodStart = time.Date(
			now.Year()-1,
			now.Month(),
			futureDay,
			12, 0, 0, 0,
			time.UTC,
		)

		company.Subscription.PeriodEnd = now.AddDate(1, 0, 0) // Set end date far in the future

		result := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)
		assert.NotNil(t, result)

		// In this case, the result should be this month's reset date
		expected := time.Date(
			now.Year(),
			now.Month(),
			futureDay,
			12, 0, 0, 0,
			time.UTC,
		)

		assert.Equal(t, expected.Year(), result.Year())
		assert.Equal(t, expected.Month(), result.Month())
		assert.Equal(t, expected.Day(), result.Day())
		assert.Equal(t, expected.Hour(), result.Hour())
	})
}

func TestGetNextMetricPeriodStartFromCondition(t *testing.T) {
	t.Run("Condition is nil", func(t *testing.T) {
		result := rulesengine.GetNextMetricPeriodStartFromCondition(nil, nil)
		assert.Nil(t, result)
	})

	t.Run("Condition is not metric type", func(t *testing.T) {
		condition := createTestCondition(rulesengine.ConditionTypeTrait)
		result := rulesengine.GetNextMetricPeriodStartFromCondition(condition, nil)
		assert.Nil(t, result)
	})

	t.Run("Metric period is nil", func(t *testing.T) {
		condition := createTestCondition(rulesengine.ConditionTypeMetric)
		condition.MetricPeriod = nil
		result := rulesengine.GetNextMetricPeriodStartFromCondition(condition, nil)
		assert.Nil(t, result)
	})

	t.Run("Metric period is all time", func(t *testing.T) {
		condition := createTestCondition(rulesengine.ConditionTypeMetric)
		allTime := rulesengine.MetricPeriodAllTime
		condition.MetricPeriod = &allTime
		result := rulesengine.GetNextMetricPeriodStartFromCondition(condition, nil)
		assert.Nil(t, result)
	})

	t.Run("Metric period is current month with billing cycle reset", func(t *testing.T) {
		company := createTestCompany()
		condition := createTestCondition(rulesengine.ConditionTypeMetric)
		currentMonth := rulesengine.MetricPeriodCurrentMonth
		billingReset := rulesengine.MetricPeriodMonthResetBilling
		condition.MetricPeriod = &currentMonth
		condition.MetricPeriodMonthReset = &billingReset

		result := rulesengine.GetNextMetricPeriodStartFromCondition(condition, company)
		expected := rulesengine.GetNextMetricPeriodStartForCompanyBillingSubscription(company)

		assert.NotNil(t, result)
		assert.Equal(t, expected.Unix(), result.Unix())
	})

	t.Run("Metric period is calendar-based", func(t *testing.T) {
		condition := createTestCondition(rulesengine.ConditionTypeMetric)
		currentDay := rulesengine.MetricPeriodCurrentDay
		firstReset := rulesengine.MetricPeriodMonthResetFirst
		condition.MetricPeriod = &currentDay
		condition.MetricPeriodMonthReset = &firstReset

		result := rulesengine.GetNextMetricPeriodStartFromCondition(condition, nil)
		expected := rulesengine.GetNextMetricPeriodStartForCalendarMetricPeriod(currentDay)

		assert.NotNil(t, result)
		assert.Equal(t, expected.Unix(), result.Unix())
	})
}
