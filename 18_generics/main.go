package main

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
)

// ============================================================
// CONCEPT: generics — type parameters and constraints
// ============================================================
//
// Before Go 1.18 you had two options for "works with any type": write it
// again per type, or use interface{} and lose all type safety. Generics give
// you a third: write it once, keep the types.
//
//   func Map[T, U any](xs []T, f func(T) U) []U
//        ^^^^^^^^^^^^ TYPE PARAMETERS, in square brackets, before the args
//
// `any` is a constraint meaning "no restrictions" — it's an alias for
// interface{}, added because it reads better here.
//
// CALLING one: usually you don't write the types at all, inference does it.
//
//   Map([]int{1,2,3}, strconv.Itoa)    // T=int, U=string, inferred
//   Map[int, string](xs, f)            // explicit, only when inference can't
//
// CONSTRAINTS are interfaces used as a type set rather than a method set.
// A union lists the permitted types:
//
//   type Number interface { ~int | ~int64 | ~float64 }
//
// THE TILDE matters. `int` means exactly int. `~int` means "any type whose
// UNDERLYING type is int" — which is what you want, because
//
//   type Celsius float64
//
// is not float64, but its underlying type is. Without the tilde, Sum would
// reject []Celsius. Almost always write ~.
//
// TWO BUILT-IN CONSTRAINTS you'll use constantly:
//
//   any          — anything
//   comparable   — usable with == and !=, so valid as a map key
//
// And one from the standard library (package `cmp`, Go 1.21+):
//
//   cmp.Ordered  — supports < <= >= >, i.e. every numeric type plus string
//
// GENERIC TYPES take parameters too, and their methods carry them:
//
//   type Stack[T any] struct { items []T }
//   func (s *Stack[T]) Push(v T) { s.items = append(s.items, v) }
//
// Note the receiver is *Stack[T], not *Stack.
//
// THE ZERO VALUE of an unknown type is `var zero T`. You cannot write
// `return nil` or `return 0` — you don't know which one it is.
//
// WHAT YOU CANNOT DO:
//   - methods cannot have their own type parameters. This is illegal:
//         func (s *Stack[T]) MapTo[U any](f func(T) U) []U
//     Make it a package-level function taking the Stack instead. This trips up
//     everyone arriving from C# or Java.
//   - you can't switch on a type parameter's operations beyond what the
//     constraint promises.
//
// WHEN NOT TO. Generics are for containers and algorithms that genuinely
// don't care about the type. If your function only ever gets called with one
// type, or an interface with a method would express it better, use that
// instead — the Go proverb "a little copying is better than a little
// dependency" still applies.
//
// The stdlib already ships generic `slices` and `maps` packages (Go 1.21+):
// slices.Sort, slices.Contains, slices.Max, maps.Keys. You're writing some of
// these by hand here to learn the mechanics — reach for the stdlib in real code.
// (Note: `constraints` is NOT stdlib; it lives in golang.org/x/exp/constraints.
// `cmp.Ordered` covers the common case without the dependency.)

// TODO 1: write `func Map[T, U any](xs []T, f func(T) U) []U` — apply f to
// every element. Return an empty (non-nil) slice for empty input.
func Map[T, U any](xs []T, f func(T) U) []U {
	res := []U{}
	if len(xs) == 0 {
		return res
	}
	for _, val := range xs {
		res = append(res, f(val))
	}

	return res
}

// TODO 2: write `func Filter[T any](xs []T, keep func(T) bool) []T`.
func Filter[T any](xs []T, keep func(T) bool) []T {
	res := []T{}

	for _, val := range xs {
		if keep(val) {
			res = append(res, val)
		}
	}

	return res
}

// TODO 3: write `func Reduce[T, U any](xs []T, init U, f func(U, T) U) U` —
// fold the slice down to a single value. Note U is the ACCUMULATOR type and
// need not match T; that's the whole point of having two parameters.
func Reduce[T, U any](xs []T, init U, f func(U, T) U) U {
	acc := init

	for _, x := range xs {
		acc = f(acc, x)
	}
	return acc
}

// TODO 4: declare `type Number interface { ~int | ~int64 | ~float64 }` and
// write `func Sum[T Number](xs []T) T`. Use `var total T` for the zero value.
// The checker calls it with a named type whose underlying type is float64, so
// the tilde is not optional.
type Number interface{ ~int | ~int64 | ~float64 }

func Sum[T Number](xs []T) T {
	var total T

	for _, x := range xs {
		total += x
	}
	return total
}

// TODO 5: write `func Keys[K comparable, V any](m map[K]V) []K`. Order doesn't
// matter — map iteration is randomised, and the checker sorts before comparing.
func Keys[K comparable, V any](m map[K]V) []K {
	res := []K{}
	for key := range m {
		res = append(res, key)
	}
	return res
}

// TODO 6: write `func MaxOf[T cmp.Ordered](xs []T) (T, bool)` returning the
// largest element, or (zero, false) for an empty slice. Import "cmp".
func MaxOf[T cmp.Ordered](xs []T) (T, bool) {
	var res T
	if len(xs) == 0 {
		return res, false
	}
	res = xs[0]
	for _, x := range xs {
		if res < x {
			res = x
		}
	}

	return res, true
}

// TODO 7: write `type Stack[T any]` with pointer-receiver methods
// `Push(v T)`, `Pop() (T, bool)` and `Len() int`. The zero value must be
// usable straight away — no constructor.
type Stack[T any] struct {
	val []T
}

func (st *Stack[T]) Len() int {
	return len(st.val)
}

func (st *Stack[T]) Pop() (T, bool) {
	var res T
	if len(st.val) == 0 {
		return res, false
	}
	res = st.val[len(st.val)-1]
	st.val = st.val[:len(st.val)-1]
	return res, true
}

func (st *Stack[T]) Push(v T) {
	st.val = append(st.val, v)
}

// TODO 8: write `type Set[T comparable]` with `NewSet[T comparable]() *Set[T]`,
// `Add(v T)`, `Has(v T) bool`, `Len() int`, and `Items() []T`.
// A map[T]struct{} is the idiomatic backing store: struct{} occupies zero
// bytes, so it says "I only care about the keys".
type Set[T comparable] struct {
	val map[T]struct{}
}

func NewSet[T comparable]() *Set[T] {
	return &Set[T]{val: map[T]struct{}{}}
}

func (s *Set[T]) Add(v T) {
	s.val[v] = struct{}{}
}

func (s *Set[T]) Has(v T) bool {
	_, ok := s.val[v]

	return ok
}

func (s *Set[T]) Items() []T {
	res := []T{}
	for key := range s.val {
		res = append(res, key)
	}
	return res
}

func (s *Set[T]) Len() int {
	return len(s.val)
}

func main() {
	// TODO 9: Map [1 2 3] to their doubled strings and print the result.
	fmt.Println(Map([]int{1, 2, 3}, func(v int) string { return strconv.Itoa(v * 2) }))
	// TODO 10: Filter 1..10 down to the even numbers and print them.
	arr := []int{}
	for i := range 10 {
		arr = append(arr, i+1)
	}
	fmt.Println(Filter(arr, func(v int) bool {
		if v%2 == 0 {
			return true
		}
		return false
	}))
	// TODO 11: Reduce 1..5 into a single int total and print it.
	arr = []int{}
	for i := range 5 {
		arr = append(arr, i+1)
	}
	fmt.Println(Reduce(arr, 0, func(acc int, curr int) int { return acc + curr }))
	// TODO 12: print Sum of an []int and Sum of a []float64.
	fmt.Println(Sum(arr), Sum([]float64{1.1, 2.1, 3.1}))
	// TODO 13: print the sorted Keys of a map[string]int.
	val := map[string]int{"alpha": 1, "beta": 2, "gamma": 3}
	keys := Keys(val)
	slices.Sort(keys)
	fmt.Println(keys)
	// TODO 14: print MaxOf on a []string and on an empty []int.
	fmt.Println(MaxOf([]string{"pear", "apple", "banana"}))
	fmt.Println(MaxOf([]int{}))
	// TODO 15: push three values onto a Stack[string], pop one, print the
	// popped value and the remaining Len().
	st := Stack[string]{}
	st.Push("value1")
	st.Push("value2")
	st.Push("value3")
	v, _ := st.Pop()
	fmt.Println(v, st.Len())
	// TODO 16: add "a", "b", "a" to a Set[string] and print its Len().
	s := NewSet[string]()
	s.Add("a")
	s.Add("b")
	s.Add("a")
	fmt.Println(s.Len())
}

// EXPECTED OUTPUT:
// [2 4 6]
// [2 4 6 8 10]
// 15
// 10 7.5
// [alpha beta gamma]
// pear true
// 0 false
// c 2
// 2
