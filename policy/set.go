package policy

// QADI of sets
type set[T comparable] map[T]struct{}

// newSetFromSlices takes slices creates a set with their contents
func newSetFromSlices[T comparable](vals ...[]T) set[T] {
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
    return result
}

