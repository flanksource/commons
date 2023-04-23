package set

type Set[E comparable] map[E]struct{}

func New[T comparable](elems ...T) Set[T] {
	s := Set[T]{}
	s.Add(elems...)
	return s
}

func (s Set[T]) Add(items ...T) {
	for _, item := range items {
		s[item] = struct{}{}
	}
}

func (s Set[T]) Contains(v T) bool {
	_, ok := s[v]
	return ok
}

func (s Set[T]) Remove(v T) {
	delete(s, v)
}

func (s Set[T]) ToSlice() []T {
	var result []T
	for v := range s {
		result = append(result, v)
	}
	return result
}

func (s Set[T]) Union(s2 Set[T]) Set[T] {
	u := New(s.ToSlice()...)
	u.Add(s2.ToSlice()...)
	return u
}

func (s Set[T]) Intersection(s2 Set[T]) Set[T] {
	i := New[T]()
	for _, item := range s2.ToSlice() {
		if s.Contains(item) {
			i.Add(item)
		}
	}
	return i
}
