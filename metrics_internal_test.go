package rulesengine

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// oracleAnniversary is an independent formulation of the clamp: walk down from the
// anchor day until the date stays inside the requested month.
func oracleAnniversary(year int, month time.Month, anchor time.Time) time.Time {
	for day := anchor.Day(); day >= 1; day-- {
		candidate := time.Date(year, month, day, anchor.Hour(), anchor.Minute(), anchor.Second(), anchor.Nanosecond(), time.UTC)
		if candidate.Month() == month {
			return candidate
		}
	}
	panic("unreachable: day 1 exists in every month")
}

// oracleResets lists every reset instant of a subscription from its period start
// through the horizon, computed independently of the code under test.
func oracleResets(periodStart, horizon time.Time) []time.Time {
	resets := []time.Time{periodStart}
	cursor := time.Date(periodStart.Year(), periodStart.Month()+1, 1, 0, 0, 0, 0, time.UTC)
	for cursor.Before(horizon) {
		resets = append(resets, oracleAnniversary(cursor.Year(), cursor.Month(), periodStart))
		cursor = cursor.AddDate(0, 1, 0)
	}
	return resets
}

func TestBillingAnniversaryInMonth(t *testing.T) {
	anchor := func(year int, month time.Month, day int) time.Time {
		return time.Date(year, month, day, 9, 30, 15, 250, time.UTC)
	}
	cases := []struct {
		name  string
		year  int
		month time.Month
		start time.Time
		want  time.Time
	}{
		{"day exists: 15th in September", 2026, time.September, anchor(2026, time.January, 15), anchor(2026, time.September, 15)},
		{"31st clamps to the 30th in September", 2026, time.September, anchor(2026, time.August, 31), anchor(2026, time.September, 30)},
		{"31st clamps to the 30th in April", 2026, time.April, anchor(2026, time.March, 31), anchor(2026, time.April, 30)},
		{"31st clamps to the 28th in a non-leap February", 2026, time.February, anchor(2026, time.January, 31), anchor(2026, time.February, 28)},
		{"30th clamps to the 29th in a leap February", 2028, time.February, anchor(2028, time.January, 30), anchor(2028, time.February, 29)},
		{"29th fits in a leap February", 2028, time.February, anchor(2028, time.January, 29), anchor(2028, time.February, 29)},
		{"29th clamps to the 28th in a century non-leap February", 2100, time.February, anchor(2100, time.January, 29), anchor(2100, time.February, 28)},
		{"31st fits in a 31-day month", 2026, time.October, anchor(2026, time.August, 31), anchor(2026, time.October, 31)},
		{"time of day is preserved", 2026, time.June, anchor(2026, time.May, 31), anchor(2026, time.June, 30)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := billingAnniversaryInMonth(tc.year, tc.month, tc.start)
			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.month, got.Month(), "the reset must stay in the requested month")
		})
	}
}

// Every anchor day in every month of leap, non-leap, and century years agrees with
// the independent oracle and never leaves its month.
func TestBillingAnniversaryInMonth_Exhaustive(t *testing.T) {
	for _, year := range []int{2023, 2024, 2026, 2028, 2100, 2400} {
		for month := time.January; month <= time.December; month++ {
			for day := 1; day <= 31; day++ {
				anchor := time.Date(2020, time.January, day, 23, 59, 59, 999_999_999, time.UTC)
				got := billingAnniversaryInMonth(year, month, anchor)
				want := oracleAnniversary(year, month, anchor)
				if !assert.Equal(t, want, got, "%d-%s anchor day %d", year, month, day) {
					return
				}
				assert.Equal(t, year, got.Year())
				assert.Equal(t, month, got.Month())
			}
		}
	}
}

// Sweep the clock day by day across two years for a subscription anchored on each
// day of the month and check both period functions against the oracle: the current
// reset is the latest one at or before now, the next is the earliest one after now,
// and the two bracket now with a gap of one calendar month.
func TestBillingPeriodStarts_SweepAgainstOracle(t *testing.T) {
	horizon := time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC)
	for day := 1; day <= 31; day++ {
		periodStart := time.Date(2025, time.January, day, 13, 45, 7, 500, time.UTC)
		periodEnd := time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
		sub := &Subscription{ID: fmt.Sprintf("sub_%d", day), PeriodStart: periodStart, PeriodEnd: periodEnd}
		resets := oracleResets(periodStart, horizon.AddDate(0, 2, 0))

		checks := 0
		for now := periodStart; now.Before(horizon); now = now.Add(24 * time.Hour) {
			var wantCurrent, wantNext time.Time
			for _, r := range resets {
				if !r.After(now) {
					wantCurrent = r
				} else {
					wantNext = r
					break
				}
			}

			current := currentBillingPeriodStartAt(sub, now)
			next := nextBillingPeriodStartAt(sub, now)
			require.NotNil(t, current)
			require.NotNil(t, next)
			if !assert.Equal(t, wantCurrent, *current, "anchor %d, now %s: current", day, now) {
				return
			}
			if !assert.Equal(t, wantNext, *next, "anchor %d, now %s: next", day, now) {
				return
			}
			assert.False(t, current.After(now))
			assert.True(t, next.After(now))
			gap := next.Sub(*current)
			assert.True(t, gap >= 28*24*time.Hour && gap <= 31*24*time.Hour, "anchor %d, now %s: gap %s", day, now, gap)
			assert.Equal(t, min(day, daysIn(current.Year(), current.Month())), current.Day(), "anchor %d, now %s", day, now)
			assert.Equal(t, min(day, daysIn(next.Year(), next.Month())), next.Day(), "anchor %d, now %s", day, now)
			checks++
		}
		require.GreaterOrEqual(t, checks, 700)
	}
}

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// The reset instant itself belongs to the new period; a nanosecond earlier still
// belongs to the old one.
func TestBillingPeriodStarts_Boundaries(t *testing.T) {
	periodStart := time.Date(2026, time.January, 31, 8, 0, 0, 0, time.UTC)
	sub := &Subscription{ID: "sub", PeriodStart: periodStart, PeriodEnd: time.Date(2027, time.January, 31, 8, 0, 0, 0, time.UTC)}

	feb := time.Date(2026, time.February, 28, 8, 0, 0, 0, time.UTC)
	mar := time.Date(2026, time.March, 31, 8, 0, 0, 0, time.UTC)
	apr := time.Date(2026, time.April, 30, 8, 0, 0, 0, time.UTC)

	assert.Equal(t, feb, *currentBillingPeriodStartAt(sub, feb), "at the reset instant the new period has begun")
	assert.Equal(t, mar, *nextBillingPeriodStartAt(sub, feb))

	assert.Equal(t, periodStart, *currentBillingPeriodStartAt(sub, feb.Add(-time.Nanosecond)), "a nanosecond before the reset is still the previous period")
	assert.Equal(t, feb, *nextBillingPeriodStartAt(sub, feb.Add(-time.Nanosecond)))

	assert.Equal(t, mar, *currentBillingPeriodStartAt(sub, apr.Add(-time.Nanosecond)))
	assert.Equal(t, apr, *nextBillingPeriodStartAt(sub, apr.Add(-time.Nanosecond)))

	// Never before the subscription started.
	assert.Equal(t, periodStart, *currentBillingPeriodStartAt(sub, periodStart))
	assert.Equal(t, periodStart, *currentBillingPeriodStartAt(sub, periodStart.Add(time.Hour)))
}

// The next reset is capped at the recorded period end, and a subscription that has
// not started yet falls back to calendar months bounded by its start.
func TestBillingPeriodStarts_EdgeCases(t *testing.T) {
	t.Run("next reset never passes the period end", func(t *testing.T) {
		periodStart := time.Date(2026, time.March, 31, 0, 0, 0, 0, time.UTC)
		periodEnd := time.Date(2026, time.April, 15, 0, 0, 0, 0, time.UTC)
		sub := &Subscription{ID: "sub", PeriodStart: periodStart, PeriodEnd: periodEnd}
		got := nextBillingPeriodStartAt(sub, time.Date(2026, time.April, 10, 0, 0, 0, 0, time.UTC))
		assert.Equal(t, periodEnd, *got)
	})

	t.Run("future period start uses calendar months until it begins", func(t *testing.T) {
		now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
		soon := &Subscription{ID: "sub", PeriodStart: now.AddDate(0, 0, 5), PeriodEnd: now.AddDate(0, 1, 5)}
		assert.Equal(t, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), *currentBillingPeriodStartAt(soon, now))
		assert.Equal(t, soon.PeriodStart, *nextBillingPeriodStartAt(soon, now), "the start arrives before the next calendar month")

		later := &Subscription{ID: "sub", PeriodStart: now.AddDate(0, 2, 0), PeriodEnd: now.AddDate(0, 3, 0)}
		assert.Equal(t, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), *nextBillingPeriodStartAt(later, now), "the calendar month ends first")
	})

	t.Run("no subscription uses calendar months", func(t *testing.T) {
		now := time.Date(2026, time.September, 10, 12, 0, 0, 0, time.UTC)
		assert.Equal(t, time.Date(2026, time.September, 1, 0, 0, 0, 0, time.UTC), *currentBillingPeriodStartAt(nil, now))
		assert.Equal(t, time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC), *nextBillingPeriodStartAt(nil, now))
	})
}
