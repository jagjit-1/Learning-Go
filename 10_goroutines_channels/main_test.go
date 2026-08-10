package main

// ============================================================
// CHECKER for 10_goroutines_channels — run with:  go test -race
// ============================================================
// Every check here has a deadline. If something blocks forever you get a
// named failure within a few seconds instead of a hung terminal.

import (
	"bytes"
	"io"
	"os"
	"reflect"
	"regexp"
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

// mustFinish fails the test if fn hasn't returned within d.
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
			"  That's a deadlock. Usual causes: an unbuffered send with nobody\n"+
			"  receiving, a range over a channel that never gets closed, or work\n"+
			"  that should have been started with `go` running inline instead.", what, d)
	}
}

// --- Signatures, including channel direction -------------------------
// These fail to compile if a channel parameter isn't directional. A plain
// `chan int` is NOT assignable to `chan<- int` in a function type.
var (
	_ func(int, chan<- int)       = produce
	_ func(<-chan int) []int      = collect
	_ func(<-chan int) <-chan int = square
	_ func([]int) int             = sumConcurrent
	_ func() (int, bool)          = recvClosed
)

// --- TODO 1 & 2: produce and collect ---------------------------------

func TestProduceAndCollect(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "produce + collect", func() {
		ch := make(chan int)
		go produce(5, ch)
		got = collect(ch)
	})

	want := []int{1, 2, 3, 4, 5}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TODO 1/2: produce(5) then collect = %v, want %v", got, want)
	}
}

func TestProduceClosesTheChannel(t *testing.T) {
	// collect only returns when the channel is closed, so if this finishes
	// at all, produce closed it. Check the state directly too.
	mustFinish(t, 5*time.Second, "produce closing its channel", func() {
		ch := make(chan int, 10)
		produce(3, ch) // buffered, so no goroutine needed
		for i := 0; i < 3; i++ {
			<-ch
		}
		if _, ok := <-ch; ok {
			t.Error("TODO 1: produce must close(out) when it's done sending — " +
				"otherwise a `range` over it blocks forever")
		}
	})
}

func TestProduceZero(t *testing.T) {
	mustFinish(t, 5*time.Second, "produce(0)", func() {
		ch := make(chan int, 1)
		produce(0, ch)
		if _, ok := <-ch; ok {
			t.Error("TODO 1: produce(0, ...) should send nothing and close")
		}
	})
}

func TestCollectOnClosedChannel(t *testing.T) {
	mustFinish(t, 5*time.Second, "collect on a closed channel", func() {
		ch := make(chan int)
		close(ch)
		if got := collect(ch); len(got) != 0 {
			t.Errorf("TODO 2: collect on an already-closed channel = %v, want empty", got)
		}
	})
}

// --- TODO 3: square ---------------------------------------------------

func TestSquare(t *testing.T) {
	var got []int
	mustFinish(t, 5*time.Second, "produce -> square -> collect", func() {
		in := make(chan int)
		go produce(5, in)
		got = collect(square(in))
	})

	want := []int{1, 4, 9, 16, 25}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("TODO 3: squares = %v, want %v", got, want)
	}
}

func TestSquareReturnsBeforeDoingAnyWork(t *testing.T) {
	// The input is unbuffered and nothing has been sent yet. If square did its
	// work inline instead of in a goroutine, this call would block forever.
	in := make(chan int)
	returned := make(chan (<-chan int), 1)

	go func() { returned <- square(in) }()

	var out <-chan int
	select {
	case out = <-returned:
	case <-time.After(2 * time.Second):
		t.Fatal("TODO 3: square did not return until work was available.\n" +
			"  It must create the output channel, start a goroutine, and return\n" +
			"  the channel straight away — the work happens later, in the goroutine.")
	}

	mustFinish(t, 5*time.Second, "square draining its input", func() {
		go func() {
			in <- 7
			close(in)
		}()
		if v, ok := <-out; !ok || v != 49 {
			t.Errorf("TODO 3: got (%d, %v) from the squares channel, want (49, true)", v, ok)
		}
	})
}

func TestSquareClosesItsOutput(t *testing.T) {
	mustFinish(t, 5*time.Second, "square closing its output", func() {
		in := make(chan int)
		close(in)
		if _, ok := <-square(in); ok {
			t.Error("TODO 3: when the input channel is exhausted, square must close " +
				"the channel it created (defer close(out) inside the goroutine)")
		}
	})
}

// --- TODO 4: sumConcurrent -------------------------------------------

func TestSumConcurrent(t *testing.T) {
	cases := []struct {
		in   []int
		want int
	}{
		{[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, 55},
		{[]int{1, 2, 3}, 6}, // odd length: the halves are uneven
		{[]int{42}, 42},     // one half is empty
		{[]int{}, 0},
		{nil, 0},
		{[]int{-5, 5, -5, 5}, 0},
	}
	for _, c := range cases {
		var got int
		mustFinish(t, 5*time.Second, "sumConcurrent", func() { got = sumConcurrent(c.in) })
		if got != c.want {
			t.Errorf("TODO 4: sumConcurrent(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestSumConcurrentIsStable(t *testing.T) {
	// Under -race, repeating this shakes out a shared accumulator.
	nums := make([]int, 1000)
	for i := range nums {
		nums[i] = i + 1
	}
	for i := 0; i < 50; i++ {
		var got int
		mustFinish(t, 5*time.Second, "sumConcurrent", func() { got = sumConcurrent(nums) })
		if got != 500500 {
			t.Fatalf("TODO 4: sumConcurrent over 1..1000 = %d, want 500500 (run %d)", got, i)
		}
	}
}

// --- TODO 5: receiving from a closed channel -------------------------

func TestRecvClosed(t *testing.T) {
	var v int
	var ok bool
	mustFinish(t, 5*time.Second, "recvClosed", func() { v, ok = recvClosed() })

	if v != 0 || ok {
		t.Errorf("TODO 5: recvClosed() = (%d, %v), want (0, false).\n"+
			"  A receive on a closed channel returns the ZERO VALUE immediately and\n"+
			"  never blocks — the bool is the only way to tell that apart from a real 0.",
			v, ok)
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 10*time.Second, "main()", func() {
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
		{regexp.MustCompile(`\[1 2 3 4 5\]`), "TODO 6", "[1 2 3 4 5]"},
		{regexp.MustCompile(`\[1 4 9 16 25\]`), "TODO 7", "[1 4 9 16 25]"},
		{regexp.MustCompile(`(?m)^55$`), "TODO 8", "55"},
		{regexp.MustCompile(`(?m)^0 false$`), "TODO 9", "0 false"},
		{regexp.MustCompile(`len=2 cap=3`), "TODO 10", "len=2 cap=3"},
		{regexp.MustCompile(`len=0 cap=3`), "TODO 10", "len=0 cap=3 (after draining)"},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s in the output.\n  your output was:\n%s",
				c.todo, c.want, indent(out))
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
