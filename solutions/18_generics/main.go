package main

import (
	"cmp"
	"fmt"
	"sort"
	"strconv"
)

func Map[T, U any](xs []T, f func(T) U) []U {
	out := make([]U, 0, len(xs))
	for _, x := range xs {
		out = append(out, f(x))
	}
	return out
}

func Filter[T any](xs []T, keep func(T) bool) []T {
	out := make([]T, 0, len(xs))
	for _, x := range xs {
		if keep(x) {
			out = append(out, x)
		}
	}
	return out
}

func Reduce[T, U any](xs []T, init U, f func(U, T) U) U {
	acc := init
	for _, x := range xs {
		acc = f(acc, x)
	}
	return acc
}

type Number interface {
	~int | ~int64 | ~float64
}

func Sum[T Number](xs []T) T {
	var total T // the only way to say "zero of an unknown type"
	for _, x := range xs {
		total += x
	}
	return total
}

func Keys[K comparable, V any](m map[K]V) []K {
	out := make([]K, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func MaxOf[T cmp.Ordered](xs []T) (T, bool) {
	var best T
	if len(xs) == 0 {
		return best, false
	}
	best = xs[0]
	for _, x := range xs[1:] {
		if x > best {
			best = x
		}
	}
	return best, true
}

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	v := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return v, true
}

func (s *Stack[T]) Len() int { return len(s.items) }

type Set[T comparable] struct {
	items map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{items: make(map[T]struct{})}
}

func (s *Set[T]) Add(v T) { s.items[v] = struct{}{} }

func (s *Set[T]) Has(v T) bool {
	_, ok := s.items[v]
	return ok
}

func (s *Set[T]) Len() int { return len(s.items) }

func (s *Set[T]) Items() []T {
	out := make([]T, 0, len(s.items))
	for v := range s.items {
		out = append(out, v)
	}
	return out
}

func main() {
	fmt.Println(Map([]int{1, 2, 3}, func(n int) string { return strconv.Itoa(n * 2) }))

	nums := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fmt.Println(Filter(nums, func(n int) bool { return n%2 == 0 }))

	fmt.Println(Reduce([]int{1, 2, 3, 4, 5}, 0, func(acc, n int) int { return acc + n }))

	fmt.Println(Sum([]int{1, 2, 3, 4}), Sum([]float64{2.5, 5.0}))

	m := map[string]int{"beta": 2, "alpha": 1, "gamma": 3}
	keys := Keys(m)
	sort.Strings(keys)
	fmt.Println(keys)

	fruit, ok := MaxOf([]string{"apple", "pear", "banana"})
	fmt.Println(fruit, ok)
	empty, ok := MaxOf([]int{})
	fmt.Println(empty, ok)

	var st Stack[string] // zero value is ready to use
	st.Push("a")
	st.Push("b")
	st.Push("c")
	top, _ := st.Pop()
	fmt.Println(top, st.Len())

	set := NewSet[string]()
	set.Add("a")
	set.Add("b")
	set.Add("a")
	fmt.Println(set.Len())
}
