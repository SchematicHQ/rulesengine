package typeconvert

import (
	"strconv"
	"strings"
	"time"

	"github.com/schematichq/rulesengine/null"
)

func Compare(
	a string,
	b string,
	comparableType ComparableType,
	operator ComparableOperator,
) bool {
	return TypeComparableString(a).Compare(
		TypeComparableString(b),
		comparableType,
		operator,
	)
}

func BoolToString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func Int64ToBool(v int64) bool {
	return v != 0
}

func Int64ToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func StringToBool(v string) bool {
	return v == "true"
}

func StringToInt64(v string) int64 {
	i, _ := strconv.ParseInt(v, 10, 0)

	return i
}

func StringToDate(v string) *time.Time {
	formats := []string{
		"2006-01-02",
		"2006-01-02 15:04:05 MST",
		"2006-01-02T15:04:05.999Z07:00",
		"Mon Jan 02 2006",
		"Mon Jan 02 2006 15:04:05 GMT-0700 (MST)",
		"Mon Jan 02 2006 15:04:05 MST",
		"Mon Jan 02 2006 15:04:05 GMT-0700",
	}
	for _, format := range formats {
		if date, err := time.Parse(format, v); err == nil {
			return null.Nullable(date.UTC())
		}
	}

	// TODO: Expand this list
	tzMap := map[string]string{
		"Alaska Standard Time":   "AKST",
		"Central Standard Time":  "CST",
		"Eastern Standard Time":  "EST",
		"Hawaii Standard Time":   "HST",
		"Mountain Standard Time": "MST",
		"Pacific Standard Time":  "PST",
	}
	for tzName, tzAbbr := range tzMap {
		v = strings.ReplaceAll(v, tzName, tzAbbr)
		for _, format := range formats {
			if date, err := time.Parse(format, v); err == nil {
				return &date
			}
		}
	}

	return nil
}
