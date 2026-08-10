package main

// ============================================================
// CHECKER for 14_pipelines_fanin_fanout — run with:  go test -race
// ============================================================
// TestDoneAwarePipelineDoesNotLeak is the one that matters. It counts live
// goroutines before and after abandoning a pipeline 20 times over. A leaky
// stage adds two stuck goroutines per run and shows up unmistakably.

import (
	"bytes"
	"io"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strings"
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
			"  A pipeline stage that never closes its output channel leaves the next\n"+
			"  stage's `range` blocked forever. Every stage that creates a channel\n"+
			"  must `defer close(out)`.", what, d)
	}
}

// goroutinesBelow waits until the live goroutine count drops to `target`,
// and returns the last count it saw.
func goroutinesBelow(target int, d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		runtime.Gosched()
		n := runtime.NumGoroutine()
		if n <= target || time.Now().After(deadline) {
			return n
		}
		time.Sleep(10 * time.Millisecond)
	}
}

var (
	_ func(...int) <-chan int                      = gen
	_ func(<-chan int) <-chan int                  = sq
	_ func(...<-chan int) <-chan int               = merge
	_ func([]int, int) []int                       = pipeline
	_ func(<-chan struct{}, ...int) <-chan int     = genDone
	_ func(<-chan struct{}, <-chan int) <-chan int = sqDone
	_ func(<-chan int, int, chan struct{}) []int   = firstN
)

func drain(ch <-chan int) []int {
	out := []int{}
	for v := range ch {
		out = append(out, v)
	}
	return out
}

// --- TODO 1: gen ------------------------------------------------------

func TestGen(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "gen", func() { got = drain(gen(1, 2, 3)) })

	want := []int{1, 2, 3}
	if len(got) != 3 || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
		t.Errorf("TODO 1: gen(1,2,3) yielded %v, want %v (in order — a single "+
			"generator goroutine sends sequentially)", got, want)
	}
}

func TestGenWithNoArgs(t *testing.T) {
	mustFinish(t, 5*time.Second, "gen with no arguments", func() {
		if got := drain(gen()); len(got) != 0 {
			t.Errorf("TODO 1: gen() yielded %v, want nothing (but it must still "+
				"close the channel)", got)
		}
	})
}

// --- TODO 2: sq -------------------------------------------------------

func TestSq(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "gen -> sq", func() { got = drain(sq(gen(1, 2, 3, 4))) })

	want := []int{1, 4, 9, 16}
	if len(got) != len(want) {
		t.Fatalf("TODO 2: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 2: got %v, want %v", got, want)
		}
	}
}

func TestStagesCompose(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "sq(sq(gen(...)))", func() {
		got = drain(sq(sq(gen(2, 3))))
	})
	if len(got) != 2 || got[0] != 16 || got[1] != 81 {
		t.Errorf("TODO 2: sq(sq(gen(2,3))) = %v, want [16 81]", got)
	}
}

// --- TODO 3: merge ----------------------------------------------------

func TestMerge(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "merge", func() {
		got = drain(merge(gen(1, 2, 3), gen(4, 5), gen(6)))
	})

	sort.Ints(got)
	want := []int{1, 2, 3, 4, 5, 6}
	if len(got) != len(want) {
		t.Fatalf("TODO 3: merge yielded %v, want the 6 values %v in some order", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 3: merge yielded %v, want %v in some order", got, want)
		}
	}
}

func TestMergeClosesOnlyWhenAllInputsAreDone(t *testing.T) {
	slow := make(chan int)
	go func() {
		defer close(slow)
		time.Sleep(100 * time.Millisecond)
		slow <- 99
	}()

	var got []int
	mustFinish(t, 5*time.Second, "merge with one slow input", func() {
		got = drain(merge(gen(1), slow))
	})

	if len(got) != 2 {
		t.Errorf("TODO 3: merge yielded %v, want both values.\n"+
			"  Closing the output as soon as the first input finishes drops the rest — "+
			"wait on a WaitGroup covering every input first.", got)
	}
}

func TestMergeWithNoChannels(t *testing.T) {
	mustFinish(t, 5*time.Second, "merge with no channels", func() {
		if got := drain(merge()); len(got) != 0 {
			t.Errorf("TODO 3: merge() yielded %v, want nothing", got)
		}
	})
	// Reaching here means merge() closed its output rather than blocking forever.
}

// --- TODO 4: pipeline -------------------------------------------------

func TestPipeline(t *testing.T) {
	for _, fanOut := range []int{1, 3, 8, 0, -2} {
		var got []int
		mustFinish(t, 10*time.Second, "pipeline", func() {
			got = pipeline([]int{1, 2, 3, 4, 5, 6, 7, 8}, fanOut)
		})

		sort.Ints(got)
		want := []int{1, 4, 9, 16, 25, 36, 49, 64}
		if len(got) != len(want) {
			t.Fatalf("TODO 4: fanOut=%d yielded %d values, want %d.\n"+
				"  Every stage must read from the SAME source channel — if you built a "+
				"separate gen per stage you'd get duplicates, and if merge closes early "+
				"you lose some.", fanOut, len(got), len(want))
		}
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("TODO 4: fanOut=%d yielded %v (sorted), want %v", fanOut, got, want)
			}
		}
	}
}

func TestPipelineEmpty(t *testing.T) {
	mustFinish(t, 10*time.Second, "pipeline with no input", func() {
		if got := pipeline(nil, 4); len(got) != 0 {
			t.Errorf("TODO 4: pipeline(nil, 4) = %v, want empty", got)
		}
	})
}

// --- TODOs 5 & 6: the done-aware pipeline -----------------------------

func TestFirstN(t *testing.T) {
	done := make(chan struct{})

	var got []int
	mustFinish(t, 5*time.Second, "firstN", func() {
		got = firstN(genDone(done, 10, 20, 30, 40, 50), 3, done)
	})

	if len(got) != 3 || got[0] != 10 || got[1] != 20 || got[2] != 30 {
		t.Errorf("TODO 6: firstN(..., 3, ...) = %v, want [10 20 30]", got)
	}

	select {
	case <-done:
	default:
		t.Error("TODO 6: firstN must close `done` before returning — that's the " +
			"signal the stages upstream are waiting for")
	}
}

func TestFirstNWhenInputRunsOutFirst(t *testing.T) {
	done := make(chan struct{})

	var got []int
	mustFinish(t, 5*time.Second, "firstN asking for more than exists", func() {
		got = firstN(genDone(done, 1, 2), 10, done)
	})

	if len(got) != 2 {
		t.Errorf("TODO 6: asked for 10 but only 2 were available, got %v — return "+
			"what you have when the channel closes", got)
	}
}

func TestFirstNZero(t *testing.T) {
	done := make(chan struct{})
	// Nothing will ever be sent on this channel and it never closes, so
	// receiving even once would hang.
	stuck := make(chan int)

	var got []int
	mustFinish(t, 5*time.Second, "firstN with n=0", func() {
		got = firstN(stuck, 0, done)
	})
	if len(got) != 0 {
		t.Errorf("TODO 6: firstN(..., 0, ...) = %v, want empty", got)
	}
}

func TestDoneAwarePipelineProducesTheRightValues(t *testing.T) {
	done := make(chan struct{})
	big := make([]int, 1000)
	for i := range big {
		big[i] = i + 1
	}

	var got []int
	mustFinish(t, 10*time.Second, "done-aware pipeline", func() {
		got = firstN(sqDone(done, genDone(done, big...)), 3, done)
	})

	if len(got) != 3 || got[0] != 1 || got[1] != 4 || got[2] != 9 {
		t.Errorf("TODO 5/6: got %v, want [1 4 9]", got)
	}
}

func TestDoneAwarePipelineDoesNotLeak(t *testing.T) {
	big := make([]int, 1000)
	for i := range big {
		big[i] = i + 1
	}

	// Quiesce, then take a baseline.
	runtime.GC()
	baseline := goroutinesBelow(0, 500*time.Millisecond)

	const runs = 20
	mustFinish(t, 20*time.Second, "20 abandoned pipelines", func() {
		for i := 0; i < runs; i++ {
			done := make(chan struct{})
			firstN(sqDone(done, genDone(done, big...)), 3, done)
		}
	})

	// Correct stages unwind as soon as done is closed. Leaky ones are parked
	// on a send forever: two per run, so 40 extra goroutines.
	after := goroutinesBelow(baseline+5, 5*time.Second)

	if after > baseline+5 {
		t.Errorf("TODO 5: after abandoning %d pipelines, %d goroutines are still "+
			"alive (started from %d).\n"+
			"  Each abandoned run left stages parked on a send nobody will ever\n"+
			"  receive. Those goroutines are never collected — the runtime cannot\n"+
			"  tell a blocked goroutine from a busy one.\n"+
			"  Every send in genDone and sqDone needs to be a select against done:\n"+
			"      select {\n"+
			"      case out <- v:\n"+
			"      case <-done:\n"+
			"          return\n"+
			"      }",
			runs, after, baseline)
	}
}

func TestDoneAwareStagesStillCloseNormally(t *testing.T) {
	// With done never closed, the stages must behave like the plain versions.
	done := make(chan struct{})

	var got []int
	mustFinish(t, 5*time.Second, "genDone -> sqDone drained fully", func() {
		got = drain(sqDone(done, genDone(done, 1, 2, 3)))
	})

	want := []int{1, 4, 9}
	if len(got) != len(want) {
		t.Fatalf("TODO 5: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 5: got %v, want %v", got, want)
		}
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 20*time.Second, "main()", func() {
		out = captureStdout(t, main)
	})
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 7")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`\[1 4 9 16 25 36 49 64\]`), "TODO 7",
			"[1 4 9 16 25 36 49 64] (sorted — fan-in order is not input order)"},
		{regexp.MustCompile(`\[1 4 9\]`), "TODO 8", "[1 4 9] from the first-3 pipeline"},
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
