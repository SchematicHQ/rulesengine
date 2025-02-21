package rulesengine

// Get the find element of a slice that satisfies a predicate.
func find[T any](list []T, predicate func(T) bool) (T, bool) {
	var val T
	for _, item := range list {
		if predicate(item) {
			return item, true
		}
	}

	return val, false
}

// Group elements of a slice by a function.
func groupBy[T any, M comparable](a []T, f func(T) M) map[M][]T {
	n := make(map[M][]T)

	for _, e := range a {
		val := f(e)
		if _, ok := n[val]; !ok {
			n[val] = make([]T, 0)
		}
		n[f(e)] = append(n[f(e)], e)
	}

	return n
}
