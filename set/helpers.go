package set

func Dedupe[T comparable](s []T) []T {
	return NewSet(s...).Slice()
}
