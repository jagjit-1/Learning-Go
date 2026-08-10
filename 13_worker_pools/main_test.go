package main

// ============================================================
// CHECKER for 13_worker_pools — run with:  go test -race
// ============================================================
// The interesting checks measure real concurrency: how many of your workers
// were inside fn at the same moment, and whether abandoned work actually
// stopped running.

import (
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
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
			"  Deadlock. In a worker pool it's nearly always one of:\n"+
			"    - jobs channel never closed, so workers range forever\n"+
			"    - results channel never closed, so the collector ranges forever\n"+
			"    - feeding jobs on the same goroutine that drains results", what, d)
	}
}

// gauge records the high-water mark of concurrent fn calls.
type gauge struct {
	cur   atomic.Int64
	max   atomic.Int64
	calls atomic.Int64
}

func (g *gauge) enter() {
	g.calls.Add(1)
	c := g.cur.Add(1)
	for {
		old := g.max.Load()
		if c <= old || g.max.CompareAndSwap(old, c) {
			return
		}
	}
}

func (g *gauge) exit() { g.cur.Add(-1) }

func oneToN(n int) []int {
	s := make([]int, n)
	for i := range s {
		s[i] = i + 1
	}
	return s
}

var (
	_ func([]int, int, func(int) (int, error)) []Result = RunPool
	_ func([]int, int, func(int) int) []int             = RunPoolOrdered
	_ func([]int, int, func(int) error) error           = FirstError
	_                                                   = Result{Job: 1, Value: 2, Err: nil}
)

// --- TODO 2: RunPool --------------------------------------------------

func TestRunPoolReturnsEveryJob(t *testing.T) {
	jobs := oneToN(50)

	var got []Result
	mustFinish(t, 15*time.Second, "RunPool", func() {
		got = RunPool(jobs, 5, func(n int) (int, error) { return n * n, nil })
	})

	if len(got) != len(jobs) {
		t.Fatalf("TODO 2: RunPool returned %d results for %d jobs", len(got), len(jobs))
	}

	sort.Slice(got, func(i, j int) bool { return got[i].Job < got[j].Job })
	for i, r := range got {
		want := i + 1
		if r.Job != want {
			t.Fatalf("TODO 2: results are missing job %d (each Result must carry its "+
				"own Job)", want)
		}
		if r.Value != want*want {
			t.Errorf("TODO 2: job %d has Value %d, want %d", r.Job, r.Value, want*want)
		}
		if r.Err != nil {
			t.Errorf("TODO 2: job %d has unexpected Err %v", r.Job, r.Err)
		}
	}
}

func TestRunPoolCarriesErrors(t *testing.T) {
	boom := errors.New("boom")

	var got []Result
	mustFinish(t, 15*time.Second, "RunPool with errors", func() {
		got = RunPool(oneToN(20), 4, func(n int) (int, error) {
			if n%2 == 0 {
				return 0, boom
			}
			return n, nil
		})
	})

	if len(got) != 20 {
		t.Fatalf("TODO 2: got %d results, want 20 — a failing job still produces a "+
			"Result, it just carries the error", len(got))
	}
	failed := 0
	for _, r := range got {
		if r.Err != nil {
			failed++
			if !errors.Is(r.Err, boom) {
				t.Errorf("TODO 2: job %d carries %v, want the error fn returned", r.Job, r.Err)
			}
		}
	}
	if failed != 10 {
		t.Errorf("TODO 2: %d results carry an error, want 10", failed)
	}
}

func TestRunPoolRespectsTheWorkerLimit(t *testing.T) {
	for _, workers := range []int{1, 2, 5} {
		g := &gauge{}
		mustFinish(t, 20*time.Second, "RunPool", func() {
			RunPool(oneToN(40), workers, func(n int) (int, error) {
				g.enter()
				time.Sleep(5 * time.Millisecond)
				g.exit()
				return n, nil
			})
		})

		if max := g.max.Load(); max > int64(workers) {
			t.Errorf("TODO 2: with workers=%d, %d fn calls were running at once.\n"+
				"  Start exactly `workers` goroutines up front and have them all range "+
				"the same jobs channel — don't spawn one per job.", workers, max)
		} else if workers > 1 && max < 2 {
			t.Errorf("TODO 2: with workers=%d, never more than %d fn call ran at a "+
				"time — the jobs aren't actually running concurrently", workers, max)
		}
	}
}

func TestRunPoolEdgeCases(t *testing.T) {
	mustFinish(t, 10*time.Second, "RunPool on an empty job list", func() {
		if got := RunPool(nil, 4, func(n int) (int, error) { return n, nil }); len(got) != 0 {
			t.Errorf("TODO 2: RunPool(nil, ...) returned %v, want empty", got)
		}
	})

	mustFinish(t, 10*time.Second, "RunPool with workers=0", func() {
		got := RunPool(oneToN(5), 0, func(n int) (int, error) { return n, nil })
		if len(got) != 5 {
			t.Errorf("TODO 2: workers=0 should behave as 1 worker, not zero workers "+
				"(zero workers means nothing ever drains the jobs channel) — got %d results",
				len(got))
		}
	})
}

// --- TODO 3: RunPoolOrdered ------------------------------------------

func TestRunPoolOrderedKeepsInputOrder(t *testing.T) {
	jobs := oneToN(30)

	var got []int
	mustFinish(t, 20*time.Second, "RunPoolOrdered", func() {
		got = RunPoolOrdered(jobs, 6, func(n int) int {
			// Uneven durations, so completion order is definitely not input order.
			time.Sleep(time.Duration(30-n) * time.Millisecond)
			return n * 2
		})
	})

	if len(got) != len(jobs) {
		t.Fatalf("TODO 3: got %d values for %d jobs", len(got), len(jobs))
	}
	for i, v := range got {
		if want := (i + 1) * 2; v != want {
			t.Fatalf("TODO 3: position %d holds %d, want %d.\n"+
				"  Results came back in completion order. Write into a preallocated "+
				"slice at the job's own index instead of appending as results arrive.",
				i, v, want)
		}
	}
}

func TestRunPoolOrderedRespectsTheWorkerLimit(t *testing.T) {
	g := &gauge{}
	mustFinish(t, 20*time.Second, "RunPoolOrdered", func() {
		RunPoolOrdered(oneToN(40), 4, func(n int) int {
			g.enter()
			time.Sleep(5 * time.Millisecond)
			g.exit()
			return n
		})
	})

	if max := g.max.Load(); max > 4 {
		t.Errorf("TODO 3: %d fn calls ran at once with workers=4", max)
	} else if max < 2 {
		t.Errorf("TODO 3: never more than %d fn call ran at a time — this isn't "+
			"running concurrently at all", max)
	}
}

func TestRunPoolOrderedEmpty(t *testing.T) {
	mustFinish(t, 10*time.Second, "RunPoolOrdered on an empty job list", func() {
		if got := RunPoolOrdered(nil, 4, func(n int) int { return n }); len(got) != 0 {
			t.Errorf("TODO 3: got %v, want empty", got)
		}
	})
}

// --- TODO 4: FirstError -----------------------------------------------

func TestFirstErrorReturnsNilWhenAllSucceed(t *testing.T) {
	var calls atomic.Int64
	var err error
	mustFinish(t, 15*time.Second, "FirstError", func() {
		err = FirstError(oneToN(50), 5, func(n int) error {
			calls.Add(1)
			return nil
		})
	})

	if err != nil {
		t.Errorf("TODO 4: no job failed, but FirstError returned %v", err)
	}
	if got := calls.Load(); got != 50 {
		t.Errorf("TODO 4: fn ran %d times, want 50 — every job must run when none fail", got)
	}
}

func TestFirstErrorReportsTheFailure(t *testing.T) {
	boom := errors.New("boom")

	var err error
	mustFinish(t, 15*time.Second, "FirstError", func() {
		err = FirstError(oneToN(20), 4, func(n int) error {
			if n == 3 {
				return boom
			}
			return nil
		})
	})

	if err == nil {
		t.Fatal("TODO 4: job 3 failed but FirstError returned nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("TODO 4: FirstError returned %v, want the error fn produced", err)
	}
}

func TestFirstErrorAbandonsTheRest(t *testing.T) {
	// 200 jobs at 20ms each with 4 workers is ~1s of work. Job 1 fails
	// immediately, so a pool that really stops should finish almost at once.
	var calls atomic.Int64

	start := time.Now()
	var err error
	mustFinish(t, 20*time.Second, "FirstError", func() {
		err = FirstError(oneToN(200), 4, func(n int) error {
			calls.Add(1)
			if n == 1 {
				return errors.New("failed early")
			}
			time.Sleep(20 * time.Millisecond)
			return nil
		})
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 4: expected an error")
	}
	if n := calls.Load(); n > 60 {
		t.Errorf("TODO 4: fn ran %d times out of 200 after an early failure.\n"+
			"  Collecting the error isn't enough — the remaining jobs must stop being\n"+
			"  handed out. Close a `done` channel (guarded by sync.Once) and select "+
			"on it in the feeder and the workers.", n)
	}
	if elapsed > 800*time.Millisecond {
		t.Errorf("TODO 4: took %v; running all 200 jobs would take about 1s, so the "+
			"pool is still working through them after the failure", elapsed)
	}
}

func TestFirstErrorEmpty(t *testing.T) {
	mustFinish(t, 10*time.Second, "FirstError on an empty job list", func() {
		if err := FirstError(nil, 4, func(n int) error { return errors.New("x") }); err != nil {
			t.Errorf("TODO 4: no jobs means no errors, got %v", err)
		}
	})
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 30*time.Second, "main()", func() {
		out = captureStdout(t, main)
	})
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 5")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^385$`), "TODO 5", "385 (sum of squares 1..10)"},
		{regexp.MustCompile(`\[2 4 6 8 10 12 14 16 18 20\]`), "TODO 6",
			"[2 4 6 8 10 12 14 16 18 20] in input order"},
		{regexp.MustCompile(`(?mi)^.*\b7\b.*fail`), "TODO 7", "an error mentioning job 7"},
		{regexp.MustCompile(`(?m)^<nil>$`), "TODO 8", "<nil> when no job fails"},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s.\n  your output was:\n%s",
				c.todo, c.want, indent(out))
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
