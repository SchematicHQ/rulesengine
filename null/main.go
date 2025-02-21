package null

// Return a pointer to a value of any type.
func Nullable[T any](value T) *T {
	return &value
}
