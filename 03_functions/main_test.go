package main

// ============================================================
// CHECKER for 03_functions — run with:  go test
// ============================================================
// This one calls your functions directly, so it also checks their
// signatures. If you see "undefined: add" or "too many arguments",
// that's the checker telling you the function isn't written yet (or
// doesn't have the shape the TODO asked for).

import "testing"

func TestAdd(t *testing.T) {
	cases := []struct{ a, b, want int }{
		{5, 3, 8},
		{-2, 2, 0},
		{0, 0, 0},
		{100, -250, -150},
	}
	for _, c := range cases {
		if got := add(c.a, c.b); got != c.want {
			t.Errorf("TODO 1: add(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestDivmod(t *testing.T) {
	cases := []struct{ a, b, wantQ, wantR int }{
		{17, 5, 3, 2},
		{10, 3, 3, 1},
		{9, 3, 3, 0},
		{4, 10, 0, 4},
	}
	for _, c := range cases {
		q, r := divmod(c.a, c.b)
		if q != c.wantQ || r != c.wantR {
			t.Errorf("TODO 2: divmod(%d, %d) = (%d, %d), want (%d, %d)",
				c.a, c.b, q, r, c.wantQ, c.wantR)
		}
	}
}

func TestDivmodNamed(t *testing.T) {
	cases := []struct{ a, b, wantQ, wantR int }{
		{17, 5, 3, 2},
		{10, 3, 3, 1},
		{9, 3, 3, 0},
		{4, 10, 0, 4},
	}
	for _, c := range cases {
		q, r := divmodNamed(c.a, c.b)
		if q != c.wantQ || r != c.wantR {
			t.Errorf("TODO 3: divmodNamed(%d, %d) = (%d, %d), want (%d, %d)\n"+
				"  hint: with named returns you ASSIGN (q = ...) then use a bare `return`",
				c.a, c.b, q, r, c.wantQ, c.wantR)
		}
	}
}

func TestSumIsVariadic(t *testing.T) {
	// The zero-argument call is the important one — it proves `nums` is a
	// variadic parameter and not a []int parameter.
	if got := sum(); got != 0 {
		t.Errorf("TODO 4: sum() = %d, want 0 (empty slice sums to 0)", got)
	}
	if got := sum(1, 2, 3); got != 6 {
		t.Errorf("TODO 4: sum(1, 2, 3) = %d, want 6", got)
	}
	if got := sum(1, 2, 3, 4, 5); got != 15 {
		t.Errorf("TODO 4: sum(1, 2, 3, 4, 5) = %d, want 15", got)
	}
	if got := sum(-5, 5); got != 0 {
		t.Errorf("TODO 4: sum(-5, 5) = %d, want 0", got)
	}

	// A variadic function can also be fed a slice with `...`
	nums := []int{10, 20, 30}
	if got := sum(nums...); got != 60 {
		t.Errorf("TODO 4: sum(nums...) = %d, want 60", got)
	}
}

func TestMakeCounterCountsUp(t *testing.T) {
	c := makeCounter()
	for want := 1; want <= 3; want++ {
		if got := c(); got != want {
			t.Fatalf("TODO 5: call #%d of a counter returned %d, want %d\n"+
				"  hint: increment the captured variable BEFORE returning it",
				want, got, want)
		}
	}
}

func TestMakeCounterInstancesAreIndependent(t *testing.T) {
	a := makeCounter()
	b := makeCounter()

	a()
	a()
	a() // a is now at 3

	if got := b(); got != 1 {
		t.Errorf("TODO 5: a second counter's first call returned %d, want 1\n"+
			"  hint: `count` must be declared INSIDE makeCounter so each call to "+
			"makeCounter gets its own. If you declared it at package level, all "+
			"counters share one.", got)
	}
	if got := a(); got != 4 {
		t.Errorf("TODO 5: counter A's 4th call returned %d, want 4 "+
			"(counter B must not have disturbed it)", got)
	}
}
