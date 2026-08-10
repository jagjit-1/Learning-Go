package main

// ============================================================
// CHECKER for 12_sync_primitives — run with:  go test -race
// ============================================================
// -race is not optional here. Most of these tests would pass on a broken
// implementation without it — a missing lock usually still produces the
// right answer most of the time, which is exactly what makes it dangerous.

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
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

func mustFinish(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v.\n"+
			"  Deadlock. With mutexes the usual cause is a missing Unlock on some\n"+
			"  path (use `defer mu.Unlock()`), or calling a method that Locks while\n"+
			"  already holding the same lock — Go's Mutex is NOT reentrant.", what, d)
	}
}

var (
	_ func([]int, int) int = parallelSum
	_ func() string        = LoadConfig
	_ func() int           = ConfigLoadCount
	_ func() *SafeMap      = NewSafeMap
)

// --- TODO 1: parallelSum ---------------------------------------------

func TestParallelSum(t *testing.T) {
	oneToN := func(n int) []int {
		s := make([]int, n)
		for i := range s {
			s[i] = i + 1
		}
		return s
	}

	cases := []struct {
		name    string
		nums    []int
		workers int
		want    int
	}{
		{"1..100 / 4 workers", oneToN(100), 4, 5050},
		{"1..100 / 1 worker", oneToN(100), 1, 5050},
		{"1..100 / 7 workers (uneven split)", oneToN(100), 7, 5050},
		{"1..1000 / 16 workers", oneToN(1000), 16, 500500},
		{"more workers than items", oneToN(3), 10, 6},
		{"workers = 0 is treated as 1", oneToN(10), 0, 55},
		{"negative workers", oneToN(10), -4, 55},
		{"empty", nil, 4, 0},
		{"negatives", []int{-5, 5, -5, 5}, 2, 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got int
			mustFinish(t, 10*time.Second, "parallelSum", func() {
				got = parallelSum(c.nums, c.workers)
			})
			if got != c.want {
				t.Errorf("TODO 1: parallelSum(%d items, %d workers) = %d, want %d.\n"+
					"  If you're short, check your chunk boundaries — the last chunk is "+
					"usually the one that gets clipped.", len(c.nums), c.workers, got, c.want)
			}
		})
	}
}

func TestParallelSumIsStable(t *testing.T) {
	nums := make([]int, 500)
	for i := range nums {
		nums[i] = i + 1
	}
	for i := 0; i < 30; i++ {
		var got int
		mustFinish(t, 10*time.Second, "parallelSum", func() {
			got = parallelSum(nums, 8)
		})
		if got != 125250 {
			t.Fatalf("TODO 1: run %d gave %d, want 125250 — an unguarded `total += sum` "+
				"loses updates when two goroutines interleave", i, got)
		}
	}
}

// --- TODO 2: Counter --------------------------------------------------

func TestCounterUnderConcurrency(t *testing.T) {
	c := &Counter{}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Inc()
			}
		}()
	}

	mustFinish(t, 15*time.Second, "100 goroutines incrementing a Counter", wg.Wait)

	if got := c.Value(); got != 10000 {
		t.Errorf("TODO 2: after 100 goroutines x 100 increments, Value() = %d, want 10000.\n"+
			"  n++ compiles to read-add-write. Two goroutines can read the same value,\n"+
			"  both add one, and both write back — two increments, one gained.", got)
	}
}

func TestCounterValueAlsoLocks(t *testing.T) {
	// A concurrent read against a write is a race even though the read
	// "can't corrupt anything". -race is what catches an unlocked Value().
	c := &Counter{}

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				c.Inc()
			}
		}()
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				_ = c.Value()
			}
		}()
	}

	mustFinish(t, 15*time.Second, "concurrent Inc and Value", wg.Wait)

	if got := c.Value(); got != 10000 {
		t.Errorf("TODO 2: Value() = %d, want 10000", got)
	}
}

// --- TODO 3: SafeMap --------------------------------------------------

func TestSafeMapBasics(t *testing.T) {
	m := NewSafeMap()

	if got := m.Len(); got != 0 {
		t.Errorf("TODO 3: a new SafeMap has Len() = %d, want 0", got)
	}
	if _, ok := m.Get("nope"); ok {
		t.Error("TODO 3: Get on a missing key should report ok == false")
	}

	m.Set("a", 1)
	m.Set("b", 2)
	m.Set("a", 3) // overwrite, not a new entry

	if v, ok := m.Get("a"); !ok || v != 3 {
		t.Errorf("TODO 3: Get(\"a\") = (%d, %v), want (3, true)", v, ok)
	}
	if got := m.Len(); got != 2 {
		t.Errorf("TODO 3: Len() = %d, want 2", got)
	}
}

func TestSafeMapUnderConcurrency(t *testing.T) {
	m := NewSafeMap()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Set(fmt.Sprintf("key-%d", i), i)
		}()
	}
	// Readers running against the writers — this is what RLock is for.
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Get(fmt.Sprintf("key-%d", i))
			m.Len()
		}()
	}

	mustFinish(t, 15*time.Second, "concurrent SafeMap access", wg.Wait)

	if got := m.Len(); got != 100 {
		t.Errorf("TODO 3: after 100 distinct Sets, Len() = %d, want 100", got)
	}
	for i := 0; i < 100; i++ {
		if v, ok := m.Get(fmt.Sprintf("key-%d", i)); !ok || v != i {
			t.Fatalf("TODO 3: Get(\"key-%d\") = (%d, %v), want (%d, true)", i, v, ok, i)
		}
	}
}

// --- TODO 4: sync.Once ------------------------------------------------

func TestLoadConfigRunsOnce(t *testing.T) {
	const goroutines = 50

	results := make([]string, goroutines)
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results[i] = LoadConfig()
		}()
	}

	mustFinish(t, 15*time.Second, "50 goroutines calling LoadConfig", wg.Wait)

	if got := ConfigLoadCount(); got != 1 {
		t.Errorf("TODO 4: the init body ran %d times, want exactly 1.\n"+
			"  A plain `if config == \"\" { ... }` guard is not enough — every "+
			"goroutine can pass that check before any of them assigns.", got)
	}

	if results[0] == "" {
		t.Fatal("TODO 4: LoadConfig returned an empty string — set the config " +
			"inside once.Do and return it")
	}
	for i, r := range results {
		if r != results[0] {
			t.Fatalf("TODO 4: goroutine %d saw %q but goroutine 0 saw %q — every caller "+
				"must block until the init finishes, so they all see the same value",
				i, r, results[0])
		}
	}
}

func TestLoadConfigStaysOnceOnLaterCalls(t *testing.T) {
	for i := 0; i < 10; i++ {
		LoadConfig()
	}
	if got := ConfigLoadCount(); got != 1 {
		t.Errorf("TODO 4: after further calls the count is %d, want 1", got)
	}
}

// --- TODO 5: AtomicCounter -------------------------------------------

func TestAtomicCounter(t *testing.T) {
	c := &AtomicCounter{}

	if got := c.Value(); got != 0 {
		t.Errorf("TODO 5: a zero-value AtomicCounter reads %d, want 0 — atomic.Int64 "+
			"is usable straight away, no constructor needed", got)
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				c.Inc()
			}
		}()
	}

	mustFinish(t, 15*time.Second, "100 goroutines on an AtomicCounter", wg.Wait)

	if got := c.Value(); got != 10000 {
		t.Errorf("TODO 5: Value() = %d, want 10000", got)
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 30*time.Second, "main()", func() {
		out = captureStdout(t, main)
	})
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 6")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^5050$`), "TODO 6", "5050 (sum of 1..100)"},
		{regexp.MustCompile(`(?m)^100$`), "TODO 8", "100 (SafeMap length)"},
		{regexp.MustCompile(`(?m)^1$`), "TODO 9", "1 (the init ran once)"},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s.\n  your output was:\n%s", c.todo, c.want, indent(out))
		}
	}

	if n := len(regexp.MustCompile(`(?m)^10000$`).FindAllString(out, -1)); n < 2 {
		t.Errorf("TODO 7/10: expected 10000 twice — once from Counter, once from "+
			"AtomicCounter — found %d.\n  your output was:\n%s", n, indent(out))
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
