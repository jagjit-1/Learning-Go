package main

// ============================================================
// CHECKER for 18_generics — run with:  go test
// ============================================================
// The var block below instantiates each generic at concrete types. That is a
// compile-time check on your type parameters and constraints: if Sum's
// constraint forgets the ~, the Celsius line stops compiling.

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = old
			w.Close()
			if rec := recover(); rec != nil {
				t.Errorf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

// A named type over float64 — only satisfies Number if the constraint uses ~.
type Celsius float64

var (
	_ func([]int, func(int) string) []string               = Map[int, string]
	_ func([]string, func(string) bool) []string           = Filter[string]
	_ func([]int, string, func(string, int) string) string = Reduce[int, string]
	_ func([]int) int                                      = Sum[int]
	_ func([]Celsius) Celsius                              = Sum[Celsius]
	_ func(map[string]int) []string                        = Keys[string, int]
	_ func([]float64) (float64, bool)                      = MaxOf[float64]
	_ func() *Set[string]                                  = NewSet[string]
)

// --- TODO 1: Map -------------------------------------------------------

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(n int) string { return strconv.Itoa(n * 2) })
	want := []string{"2", "4", "6"}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("TODO 1: Map = %v, want %v", got, want)
	}
}

func TestMapChangesType(t *testing.T) {
	// The point of two type parameters: input and output types differ.
	got := Map([]string{"a", "bb", "ccc"}, func(s string) int { return len(s) })
	if len(got) != 3 || got[0] != 1 || got[1] != 2 || got[2] != 3 {
		t.Errorf("TODO 1: Map string->int = %v, want [1 2 3]", got)
	}
}

func TestMapEmpty(t *testing.T) {
	got := Map(nil, func(n int) int { return n })
	if got == nil {
		t.Error("TODO 1: Map on empty input returned nil; return an empty slice so " +
			"callers can range over it and append to it without a nil check")
	}
	if len(got) != 0 {
		t.Errorf("TODO 1: Map on empty input = %v, want empty", got)
	}
}

// --- TODO 2: Filter ----------------------------------------------------

func TestFilter(t *testing.T) {
	got := Filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
	want := []int{2, 4, 6}
	if len(got) != len(want) {
		t.Fatalf("TODO 2: Filter = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 2: Filter = %v, want %v (order must be preserved)", got, want)
		}
	}
}

func TestFilterKeepsNothing(t *testing.T) {
	got := Filter([]int{1, 3, 5}, func(n int) bool { return n%2 == 0 })
	if len(got) != 0 {
		t.Errorf("TODO 2: Filter = %v, want empty", got)
	}
}

// --- TODO 3: Reduce ----------------------------------------------------

func TestReduce(t *testing.T) {
	if got := Reduce([]int{1, 2, 3, 4, 5}, 0, func(acc, n int) int { return acc + n }); got != 15 {
		t.Errorf("TODO 3: Reduce sum = %d, want 15", got)
	}
	if got := Reduce([]int{1, 2, 3}, 100, func(acc, n int) int { return acc + n }); got != 106 {
		t.Errorf("TODO 3: Reduce with init 100 = %d, want 106 — start from init, not zero", got)
	}
}

func TestReduceAccumulatorTypeDiffers(t *testing.T) {
	got := Reduce([]int{1, 2, 3}, "", func(acc string, n int) string {
		return acc + strconv.Itoa(n)
	})
	if got != "123" {
		t.Errorf("TODO 3: Reduce int->string = %q, want %q", got, "123")
	}
}

func TestReduceEmpty(t *testing.T) {
	if got := Reduce([]int{}, 42, func(acc, n int) int { return acc + n }); got != 42 {
		t.Errorf("TODO 3: Reduce over an empty slice = %d, want the init value 42", got)
	}
}

// --- TODO 4: Sum and the Number constraint -----------------------------

func TestSum(t *testing.T) {
	if got := Sum([]int{1, 2, 3, 4}); got != 10 {
		t.Errorf("TODO 4: Sum([]int) = %d, want 10", got)
	}
	if got := Sum([]float64{2.5, 5.0}); got != 7.5 {
		t.Errorf("TODO 4: Sum([]float64) = %v, want 7.5", got)
	}
	if got := Sum([]int{}); got != 0 {
		t.Errorf("TODO 4: Sum of an empty slice = %d, want 0", got)
	}
}

func TestSumAcceptsNamedTypes(t *testing.T) {
	// If Number said `float64` rather than `~float64`, this would not compile.
	if got := Sum([]Celsius{10, 20.5}); got != Celsius(30.5) {
		t.Errorf("TODO 4: Sum([]Celsius) = %v, want 30.5", got)
	}
}

// --- TODO 5: Keys ------------------------------------------------------

func TestKeys(t *testing.T) {
	got := Keys(map[string]int{"beta": 2, "alpha": 1, "gamma": 3})
	sort.Strings(got)
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != 3 {
		t.Fatalf("TODO 5: Keys = %v, want the 3 keys %v in some order", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 5: Keys = %v, want %v in some order", got, want)
		}
	}

	if got := Keys(map[int]bool{}); len(got) != 0 {
		t.Errorf("TODO 5: Keys of an empty map = %v, want empty", got)
	}
}

// --- TODO 6: MaxOf -----------------------------------------------------

func TestMaxOf(t *testing.T) {
	if got, ok := MaxOf([]int{3, 9, 2}); !ok || got != 9 {
		t.Errorf("TODO 6: MaxOf([3 9 2]) = (%d, %v), want (9, true)", got, ok)
	}
	if got, ok := MaxOf([]string{"apple", "pear", "banana"}); !ok || got != "pear" {
		t.Errorf("TODO 6: MaxOf on strings = (%q, %v), want (\"pear\", true) — "+
			"cmp.Ordered covers string too", got, ok)
	}
	if got, ok := MaxOf([]int{-5, -1, -9}); !ok || got != -1 {
		t.Errorf("TODO 6: MaxOf on negatives = (%d, %v), want (-1, true) — don't "+
			"start from the zero value, start from xs[0]", got, ok)
	}
	if got, ok := MaxOf([]float64{}); ok || got != 0 {
		t.Errorf("TODO 6: MaxOf of an empty slice = (%v, %v), want (0, false)", got, ok)
	}
}

// --- TODO 7: Stack -----------------------------------------------------

func TestStack(t *testing.T) {
	var s Stack[string] // zero value must work: no constructor
	if s.Len() != 0 {
		t.Errorf("TODO 7: a zero-value Stack has Len() = %d, want 0", s.Len())
	}
	if _, ok := s.Pop(); ok {
		t.Error("TODO 7: Pop on an empty Stack should report ok == false")
	}

	s.Push("a")
	s.Push("b")
	s.Push("c")
	if s.Len() != 3 {
		t.Errorf("TODO 7: Len() = %d, want 3", s.Len())
	}

	v, ok := s.Pop()
	if !ok || v != "c" {
		t.Errorf("TODO 7: Pop = (%q, %v), want (\"c\", true) — a stack is LIFO", v, ok)
	}
	if s.Len() != 2 {
		t.Errorf("TODO 7: Len() after one Pop = %d, want 2", s.Len())
	}
}

func TestStackZeroValueOfT(t *testing.T) {
	var s Stack[int]
	v, ok := s.Pop()
	if ok || v != 0 {
		t.Errorf("TODO 7: Pop on empty = (%d, %v), want (0, false) — use `var zero T`", v, ok)
	}
}

// --- TODO 8: Set -------------------------------------------------------

func TestSet(t *testing.T) {
	s := NewSet[string]()
	if s.Len() != 0 || s.Has("a") {
		t.Error("TODO 8: a new Set should be empty")
	}

	s.Add("a")
	s.Add("b")
	s.Add("a") // duplicate

	if s.Len() != 2 {
		t.Errorf("TODO 8: Len() = %d, want 2 — adding a duplicate changes nothing", s.Len())
	}
	if !s.Has("a") || !s.Has("b") || s.Has("z") {
		t.Error("TODO 8: Has reported the wrong membership")
	}

	items := s.Items()
	sort.Strings(items)
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Errorf("TODO 8: Items() = %v, want [a b] in some order", items)
	}
}

func TestSetWorksWithOtherComparableTypes(t *testing.T) {
	s := NewSet[int]()
	for _, n := range []int{1, 2, 2, 3, 3, 3} {
		s.Add(n)
	}
	if s.Len() != 3 {
		t.Errorf("TODO 8: Set[int].Len() = %d, want 3", s.Len())
	}
}

// --- main()'s narration ------------------------------------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 9")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`\[2 4 6\]`), "TODO 9", "[2 4 6] from Map"},
		{regexp.MustCompile(`\[2 4 6 8 10\]`), "TODO 10", "[2 4 6 8 10] from Filter"},
		{regexp.MustCompile(`(?m)^15$`), "TODO 11", "15 from Reduce"},
		{regexp.MustCompile(`(?m)^\d+ [\d.]+$`), "TODO 12", "an int Sum and a float Sum on one line"},
		{regexp.MustCompile(`\[alpha beta gamma\]`), "TODO 13", "[alpha beta gamma] from sorted Keys"},
		{regexp.MustCompile(`(?m)^\S+ true$`), "TODO 14", "the max plus true"},
		{regexp.MustCompile(`(?m)^0 false$`), "TODO 14", "\"0 false\" for the empty slice"},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s.\n  your output was:\n%s", c.todo, c.want, indent(out))
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
