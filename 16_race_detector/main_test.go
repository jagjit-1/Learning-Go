package main

// ============================================================
// CHECKER for 16_race_detector — run with:  go test -race
// ============================================================
// Nothing in here calls racyCount. It is racy on purpose, and calling it
// under -race would fail the run by design. To see it misbehave, use:
//
//   SHOW_RACE=1 go run -race .

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
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
		t.Fatalf("%s did not finish within %v", what, d)
	}
}

// Signature checks only — racyCount is referenced, never called.
var (
	_ func(int) int    = racyCount
	_ func(int) int    = countMutex
	_ func(int) int    = countAtomic
	_ func(int) int    = countChannel
	_ func(int) []int  = appendConcurrent
	_ func() *Registry = NewRegistry
)

// --- TODOs 2, 3, 4: the three correct counters ------------------------

func TestCountersAreCorrect(t *testing.T) {
	impls := []struct {
		todo string
		name string
		fn   func(int) int
	}{
		{"TODO 2", "countMutex", countMutex},
		{"TODO 3", "countAtomic", countAtomic},
		{"TODO 4", "countChannel", countChannel},
	}

	for _, impl := range impls {
		t.Run(impl.name, func(t *testing.T) {
			for _, n := range []int{0, 1, 100, 1000} {
				var got int
				mustFinish(t, 20*time.Second, impl.name, func() { got = impl.fn(n) })
				if got != n {
					t.Errorf("%s: %s(%d) = %d, want %d", impl.todo, impl.name, n, got, n)
				}
			}
		})
	}
}

func TestCountersAreStableUnderRepetition(t *testing.T) {
	// A lost update is intermittent. Repeating raises the odds of catching one
	// even without -race; with -race, the first collision is enough.
	impls := map[string]func(int) int{
		"countMutex":   countMutex,
		"countAtomic":  countAtomic,
		"countChannel": countChannel,
	}

	for name, fn := range impls {
		t.Run(name, func(t *testing.T) {
			for i := 0; i < 20; i++ {
				var got int
				mustFinish(t, 20*time.Second, name, func() { got = fn(500) })
				if got != 500 {
					t.Fatalf("%s returned %d on run %d, want 500 — an increment was lost, "+
						"so the counter isn't actually protected", name, got, i)
				}
			}
		})
	}
}

// --- TODO 5: appendConcurrent ----------------------------------------

func TestAppendConcurrent(t *testing.T) {
	for _, n := range []int{0, 1, 500} {
		var got []int
		mustFinish(t, 20*time.Second, "appendConcurrent", func() { got = appendConcurrent(n) })

		if len(got) != n {
			t.Fatalf("TODO 5: appendConcurrent(%d) returned %d values, want %d.\n"+
				"  append reads the slice header, may allocate a new array, and writes\n"+
				"  the header back — two goroutines doing that at once lose elements.",
				n, len(got), n)
		}

		seen := make(map[int]bool, n)
		for _, v := range got {
			seen[v] = true
		}
		for i := 0; i < n; i++ {
			if !seen[i] {
				t.Fatalf("TODO 5: value %d is missing from the result (order doesn't "+
					"matter, but every index must be there exactly once)", i)
			}
		}
	}
}

// --- TODO 6: Registry -------------------------------------------------

func TestRegistryBasics(t *testing.T) {
	r := NewRegistry()

	if got := r.Count(); got != 0 {
		t.Errorf("TODO 6: a new Registry has Count() = %d, want 0", got)
	}
	if got := r.Names(); len(got) != 0 {
		t.Errorf("TODO 6: a new Registry has Names() = %v, want empty", got)
	}

	r.Register("beta")
	r.Register("alpha")
	r.Register("beta") // same name again: not a new entry

	if got := r.Count(); got != 2 {
		t.Errorf("TODO 6: Count() = %d, want 2 (distinct names, not registrations)", got)
	}

	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "beta" {
		t.Errorf("TODO 6: Names() = %v, want [alpha beta] — sorted", names)
	}
}

func TestRegistryNamesReturnsACopy(t *testing.T) {
	r := NewRegistry()
	r.Register("alpha")
	r.Register("beta")

	names := r.Names()
	names[0] = "tampered"

	if again := r.Names(); again[0] == "tampered" {
		t.Error("TODO 6: mutating the slice Names() returned changed the registry.\n" +
			"  The lock is released the moment Names returns, so handing back a slice\n" +
			"  that shares state with the guarded data means the caller can modify it\n" +
			"  with no lock held at all. Build and return a copy.")
	}
}

func TestRegistryUnderConcurrency(t *testing.T) {
	r := NewRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Register(fmt.Sprintf("name-%d", i))
		}()
	}
	// Readers running alongside the writers. Without a lock this is not a
	// race but a hard crash: "fatal error: concurrent map writes".
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Count()
			r.Names()
		}()
	}

	mustFinish(t, 30*time.Second, "concurrent Registry access", wg.Wait)

	if got := r.Count(); got != 200 {
		t.Errorf("TODO 6: Count() = %d after 200 distinct registrations, want 200", got)
	}

	names := r.Names()
	if len(names) != 200 {
		t.Fatalf("TODO 6: Names() returned %d entries, want 200", len(names))
	}
	if !sort.StringsAreSorted(names) {
		t.Errorf("TODO 6: Names() came back unsorted: %v", names[:5])
	}
}

func TestRegistryCountsRepeats(t *testing.T) {
	r := NewRegistry()

	var wg sync.WaitGroup
	for i := 0; i < 500; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Register("same")
		}()
	}
	mustFinish(t, 20*time.Second, "500 registrations of one name", wg.Wait)

	if got := r.Count(); got != 1 {
		t.Errorf("TODO 6: Count() = %d, want 1 — one distinct name", got)
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	// main() must not run the racy path unless asked.
	if v := os.Getenv("SHOW_RACE"); v != "" {
		t.Skipf("SHOW_RACE is set to %q, so main() takes the deliberately racy "+
			"path — unset it to check the normal output", v)
	}

	var out string
	mustFinish(t, 30*time.Second, "main()", func() {
		out = captureStdout(t, main)
	})
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 8")
	}

	if n := len(regexp.MustCompile(`(?m)^1000$`).FindAllString(out, -1)); n < 3 {
		t.Errorf("TODO 8: expected 1000 three times (mutex, atomic, channel) — "+
			"found %d.\n  your output was:\n%s", n, indent(out))
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^500$`), "TODO 9", "500 (length of appendConcurrent)"},
		{regexp.MustCompile(`(?m)^200$`), "TODO 10", "200 (distinct registered names)"},
		{regexp.MustCompile(`(?m)^name-0$`), "TODO 10", "name-0 (first entry, sorted)"},
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
