package set

import "encoding/json"

type Set[E comparable] map[E]struct{}

func NewSet[E comparable](vals ...E) Set[E] {
	s := Set[E]{}
	for _, v := range vals {
		s[v] = struct{}{}
	}
	return s
}

func (s Set[E]) Add(e ...E) {
	for _, v := range e {
		s[v] = struct{}{}
	}
}

func (s Set[E]) Contains(e E) bool {
	_, ok := s[e]
	return ok
}

func (s Set[E]) Difference(other Set[E]) Set[E] {
	difference := Set[E]{}
	for e := range s {
		if !other.Contains(e) {
			difference.Add(e)
		}
	}
	return difference
}

func (s Set[E]) Eq(other Set[E]) bool {
	if s.Len() != other.Len() {
		return false
	}
	for e := range s {
		if !other.Contains(e) {
			return false
		}
	}
	for e := range other {
		if !s.Contains(e) {
			return false
		}
	}
	return true
}

func (s Set[E]) Intersection(other Set[E]) Set[E] {
	intersection := Set[E]{}
	for e := range s {
		if other.Contains(e) {
			intersection.Add(e)
		}
	}
	for e := range other {
		if s.Contains(e) {
			intersection.Add(e)
		}
	}
	return intersection
}

func (s Set[E]) Len() int {
	return len(s)
}

func (s Set[E]) Remove(e E) {
	delete(s, e)
}

func (s Set[E]) Slice() []E {
	slice := make([]E, 0, len(s))
	for e := range s {
		slice = append(slice, e)
	}
	return slice
}

func (s Set[E]) Union(other Set[E]) Set[E] {
	union := Set[E]{}
	for e := range s {
		union.Add(e)
	}
	for e := range other {
		union.Add(e)
	}
	return union
}

func (s Set[E]) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.Slice())
}

func (s *Set[E]) UnmarshalJSON(data []byte) error {
	var elements []E
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}
	if *s == nil {
		*s = NewSet[E]()
	}
	for _, e := range elements {
		s.Add(e)
	}
	return nil
}
