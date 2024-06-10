package policy

import (
	"cmp"
	"slices"
)

// QADI of sets that have a stable sort
type set[T cmp.Ordered] map[T]struct{}

// newSetFromSlices takes slices creates a set with their contents
func newSetFromSlices[T cmp.Ordered](vals ...[]T) set[T] {
	s := set[T]{}
	for _, innerVals := range vals {
		for _, v := range innerVals {
			s[v] = struct{}{}
		}
	}
	return s
}

func (s set[T]) Add(vals ...T) {
	for _, v := range vals {
		s[v] = struct{}{}
	}
}

func (s set[T]) Members() []T {
	result := make([]T, 0, len(s))
	for v := range s {
		result = append(result, v)
	}
	slices.Sort(result)
	return result
}
