package typeconvert_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/schematichq/rulesengine/typeconvert"
	"github.com/stretchr/testify/assert"
)

func TestStringToDate(t *testing.T) {
	t.Run("When the string is empty", func(t *testing.T) {
		result := typeconvert.StringToDate("")
		assert.Nil(t, result)
	})

	t.Run("ISO string", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 21, 59, 40, 162000000, time.UTC)
		dateStr := "2024-01-15T21:59:40.162Z"

		result := typeconvert.StringToDate(dateStr)
		assert.NotNil(t, result)
		assert.Equal(t, date.Unix(), result.Unix())
	})

	t.Run("YYYY-MM-DD", func(t *testing.T) {
		date := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
		dateStr := date.Format("2006-01-02")

		result := typeconvert.StringToDate(dateStr)
		assert.NotNil(t, result)
		assert.Equal(t, date.Unix(), result.Unix())
	})

	t.Run("JavaScript toString", func(t *testing.T) {
		utcDiff := getTzUtcDiff("America/New_York")
		date := time.Date(2024, time.January, 16, 12-utcDiff, 44, 18, 0, time.UTC)
		loc, _ := time.LoadLocation("America/New_York")
		date = date.In(loc)
		dateStr := fmt.Sprintf("Tue Jan 16 2024 12:44:18 GMT-0%d00 (Eastern Standard Time)", utcDiff*-1)

		result := typeconvert.StringToDate(dateStr)
		assert.NotNil(t, result)
		assert.Equal(t, date.Unix(), result.Unix())
	})

	t.Run("YYYY-MM-DD HH:MM:SS UTC", func(t *testing.T) {
		dateStr := "2023-09-18 13:52:16 UTC"
		date := time.Date(2023, 9, 18, 13, 52, 16, 0, time.UTC)

		result := typeconvert.StringToDate(dateStr)
		assert.NotNil(t, result)
		assert.Equal(t, date.Unix(), result.Unix())
	})
}

func getTzUtcDiff(tz string) int {
	tzLoc, err := time.LoadLocation(tz)
	if err != nil {
		return 0
	}

	_, offset := time.Now().In(tzLoc).Zone()
	return offset / 60 / 60
}
