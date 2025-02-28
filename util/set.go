package util

import (
	"cmp"
	"slices"
)

// QADI of sets that have a stable sort
type Set[T cmp.Ordered] map[T]struct{}

// newSetFromSlices takes slices creates a set with their contents
func NewSetFromSlices[T cmp.Ordered](vals ...[]T) Set[T] {
	s := Set[T]{}
	for _, innerVals := range vals {
		for _, v := range innerVals {
			s[v] = struct{}{}
		}
	}
	return s
}

func (s Set[T]) Add(vals ...T) {
	for _, v := range vals {
		s[v] = struct{}{}
	}
}

func (s Set[T]) Members() []T {
	result := make([]T, 0, len(s))
	for v := range s {
		result = append(result, v)
	}
	slices.Sort(result)
	return result
}

func (s Set[T]) Has(val T) bool {
	_, found := s[val]
	return found
}
