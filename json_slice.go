package rulesengine

import (
	"encoding/json"
)

// JSONSlice is a wrapper type that ensures slices are serialized as empty
// arrays instead of null when they are nil. Strictly-typed JSON consumers
// (e.g. Fern-generated SDKs) reject `null` for list-typed fields, so wire
// types declared with []T cannot use Go's default nil-slice marshaling
// without back-compat workarounds in every SDK.
type JSONSlice[T any] []T

func NewJSONSlice[T any](slice []T) JSONSlice[T] {
	if slice == nil {
		return JSONSlice[T]{}
	}
	return JSONSlice[T](slice)
}

func (s JSONSlice[T]) MarshalJSON() ([]byte, error) {
	if s == nil {
		return json.Marshal([]T{})
	}
	return json.Marshal([]T(s))
}

func (s *JSONSlice[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		*s = JSONSlice[T]{}
		return nil
	}

	var slice []T
	if err := json.Unmarshal(data, &slice); err != nil {
		return err
	}

	*s = JSONSlice[T](slice)
	return nil
}

func (s JSONSlice[T]) Slice() []T {
	if s == nil {
		return []T{}
	}

	return []T(s)
}
