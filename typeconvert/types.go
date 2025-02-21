package typeconvert

import (
	"errors"
	"time"
)

type ComparableType string

const (
	ComparableTypeBool   ComparableType = "bool"
	ComparableTypeDate   ComparableType = "date"
	ComparableTypeInt    ComparableType = "int"
	ComparableTypeString ComparableType = "string"
)

type ComparableOperator string

const (
	ComparableOperatorEquals    ComparableOperator = "eq"
	ComparableOperatorNotEquals ComparableOperator = "ne"
	ComparableOperatorGt        ComparableOperator = "gt"
	ComparableOperatorLt        ComparableOperator = "lt"
	ComparableOperatorGte       ComparableOperator = "gte"
	ComparableOperatorLte       ComparableOperator = "lte"
	ComparableOperatorIsEmpty   ComparableOperator = "is_empty"
	ComparableOperatorNotEmpty  ComparableOperator = "not_empty"
)

type TypeComparableString string

func (s TypeComparableString) Bool() bool {
	return StringToBool(string(s))
}

func (s TypeComparableString) Date() *time.Time {
	return StringToDate(string(s))
}

func (s TypeComparableString) Int64() int64 {
	return StringToInt64(string(s))
}

func (o ComparableOperator) Sql() (string, error) {
	switch o {
	case ComparableOperatorEquals:
		return "=", nil
	case ComparableOperatorNotEquals:
		return "!=", nil
	case ComparableOperatorLt:
		return "<", nil
	case ComparableOperatorLte:
		return "<=", nil
	case ComparableOperatorGt:
		return ">", nil
	case ComparableOperatorGte:
		return ">=", nil
	case ComparableOperatorIsEmpty:
		return "IS NULL", nil
	case ComparableOperatorNotEmpty:
		return "IS NOT NULL", nil
	}

	return "=", errors.New("invalid operator")
}

func (s TypeComparableString) String() string {
	return string(s)
}

func (s TypeComparableString) Compare(other TypeComparableString, comparableType ComparableType, operator ComparableOperator) bool {
	switch comparableType {
	case ComparableTypeString:
		return CompareString(s.String(), other.String(), operator)
	case ComparableTypeInt:
		return CompareInt64(s.Int64(), other.Int64(), operator)
	case ComparableTypeBool:
		return CompareBool(s.Bool(), other.Bool(), operator)
	case ComparableTypeDate:
		return CompareDate(s.Date(), other.Date(), operator)
	}

	return false
}

func CompareBool(a bool, b bool, operator ComparableOperator) bool {
	switch operator {
	case ComparableOperatorEquals:
		return a == b
	case ComparableOperatorNotEquals:
		return a != b
	case ComparableOperatorIsEmpty:
		return false
	case ComparableOperatorNotEmpty:
		return true
	}

	return false
}

func CompareDate(a *time.Time, b *time.Time, operator ComparableOperator) bool {
	switch operator {
	case ComparableOperatorEquals:
		if a == nil && b == nil {
			return true
		} else if a == nil || b == nil {
			return false
		} else {
			return a.Equal(*b)
		}
	case ComparableOperatorNotEquals:
		return !CompareDate(a, b, ComparableOperatorEquals)
	case ComparableOperatorGt:
		if a == nil {
			return false
		} else if b == nil {
			return true
		} else {
			return a.After(*b)
		}
	case ComparableOperatorLt:
		if b == nil {
			return false
		} else if a == nil {
			return true
		} else {
			return a.Before(*b)
		}
	case ComparableOperatorGte:
		if CompareDate(a, b, ComparableOperatorEquals) {
			return true
		} else {
			return CompareDate(a, b, ComparableOperatorGt)
		}
	case ComparableOperatorLte:
		if CompareDate(a, b, ComparableOperatorEquals) {
			return true
		} else {
			return CompareDate(a, b, ComparableOperatorLt)
		}
	case ComparableOperatorIsEmpty:
		return a == nil
	case ComparableOperatorNotEmpty:
		return a != nil
	}

	return false
}

func CompareInt64(a int64, b int64, operator ComparableOperator) bool {
	switch operator {
	case ComparableOperatorEquals:
		return a == b
	case ComparableOperatorNotEquals:
		return a != b
	case ComparableOperatorGt:
		return a > b
	case ComparableOperatorLt:
		return a < b
	case ComparableOperatorGte:
		return a >= b
	case ComparableOperatorLte:
		return a <= b
	case ComparableOperatorIsEmpty:
		return a == 0
	case ComparableOperatorNotEmpty:
		return a > 0
	}

	return false
}

func CompareString(a string, b string, operator ComparableOperator) bool {
	switch operator {
	case ComparableOperatorEquals:
		return a == b
	case ComparableOperatorNotEquals:
		return a != b
	case ComparableOperatorGt:
		return a > b
	case ComparableOperatorLt:
		return a < b
	case ComparableOperatorGte:
		return a >= b
	case ComparableOperatorLte:
		return a <= b
	case ComparableOperatorIsEmpty:
		return a == ""
	case ComparableOperatorNotEmpty:
		return a != ""
	}

	return false
}
