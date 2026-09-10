package rulesengine

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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
