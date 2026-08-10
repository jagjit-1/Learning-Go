package main

// ============================================================
// CHECKER for 11_select_timeouts — run with:  go test -race
// ============================================================

import (
	"bytes"
	"errors"
	"io"
	"os"
	"regexp"
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
		t.Fatalf("%s did not finish within %v — that's a deadlock, or a select "+
			"with no case that can ever fire", what, d)
	}
}

var (
	_ func(a, b <-chan string) string                   = first
	_ func(<-chan int, time.Duration) (int, error)      = recvWithTimeout
	_ func(chan<- int, int) bool                        = trySend
	_ func(<-chan int) (int, bool)                      = tryRecv
	_ func(a, b <-chan int, done <-chan struct{}) []int = merge2
)

// --- TODO 1: first ----------------------------------------------------

func TestFirstPicksWhicheverArrives(t *testing.T) {
	for _, winner := range []string{"a", "b"} {
		a := make(chan string, 1)
		b := make(chan string, 1)
		if winner == "a" {
			a <- "from-a"
		} else {
			b <- "from-b"
		}

		var got string
		mustFinish(t, 5*time.Second, "first", func() { got = first(a, b) })

		want := "from-" + winner
		if got != want {
			t.Errorf("TODO 1: with only %s ready, first() = %q, want %q", winner, got, want)
		}
	}
}

func TestFirstWaitsForTheSlowOne(t *testing.T) {
	// Neither is ready yet; select must block rather than spin or return "".
	a := make(chan string)
	b := make(chan string)
	go func() {
		time.Sleep(50 * time.Millisecond)
		b <- "late"
	}()

	var got string
	mustFinish(t, 5*time.Second, "first", func() { got = first(a, b) })
	if got != "late" {
		t.Errorf("TODO 1: first() = %q, want %q — select should block until one "+
			"of the channels is ready", got, "late")
	}
}

// --- TODO 2: recvWithTimeout -----------------------------------------

func TestRecvWithTimeoutSucceeds(t *testing.T) {
	ch := make(chan int)
	go func() {
		time.Sleep(20 * time.Millisecond)
		ch <- 42
	}()

	var v int
	var err error
	mustFinish(t, 5*time.Second, "recvWithTimeout", func() {
		v, err = recvWithTimeout(ch, 3*time.Second)
	})

	if err != nil {
		t.Fatalf("TODO 2: a value arrived well inside the deadline but got error %v", err)
	}
	if v != 42 {
		t.Errorf("TODO 2: recvWithTimeout = %d, want 42", v)
	}
}

func TestRecvWithTimeoutExpires(t *testing.T) {
	ch := make(chan int) // nothing will ever be sent

	start := time.Now()
	var v int
	var err error
	mustFinish(t, 5*time.Second, "recvWithTimeout", func() {
		v, err = recvWithTimeout(ch, 50*time.Millisecond)
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 2: nothing was ever sent, so recvWithTimeout should have timed out")
	}
	if !errors.Is(err, ErrTimeout) {
		t.Errorf("TODO 2: got error %v, want it to match ErrTimeout via errors.Is.\n"+
			"  Return the sentinel itself (or wrap it with %%w) rather than a fresh "+
			"errors.New each time — a fresh error is never Is-equal to anything.", err)
	}
	if v != 0 {
		t.Errorf("TODO 2: on timeout, return the zero value alongside the error — got %d", v)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("TODO 2: returned after only %v for a 50ms deadline — it isn't "+
			"actually waiting", elapsed)
	}
	if elapsed > 2*time.Second {
		t.Errorf("TODO 2: took %v for a 50ms deadline", elapsed)
	}
}

// --- TODO 3: trySend --------------------------------------------------

func TestTrySend(t *testing.T) {
	mustFinish(t, 5*time.Second, "trySend", func() {
		ch := make(chan int, 1)

		if !trySend(ch, 1) {
			t.Error("TODO 3: trySend into a channel with room = false, want true")
		}
		if trySend(ch, 2) {
			t.Error("TODO 3: trySend into a FULL channel = true, want false — " +
				"that's what the default case is for")
		}
		if got := <-ch; got != 1 {
			t.Errorf("TODO 3: the channel holds %d, want 1 (the failed send must not "+
				"have overwritten anything)", got)
		}
	})
}

func TestTrySendUnbufferedWithNoReceiver(t *testing.T) {
	mustFinish(t, 5*time.Second, "trySend on an unbuffered channel", func() {
		if trySend(make(chan int), 1) {
			t.Error("TODO 3: an unbuffered channel with no receiver waiting can never " +
				"accept a send, so trySend should return false")
		}
	})
}

// --- TODO 4: tryRecv --------------------------------------------------

func TestTryRecv(t *testing.T) {
	mustFinish(t, 5*time.Second, "tryRecv", func() {
		ch := make(chan int, 1)

		if v, ok := tryRecv(ch); ok || v != 0 {
			t.Errorf("TODO 4: tryRecv on an empty channel = (%d, %v), want (0, false)", v, ok)
		}

		ch <- 7
		if v, ok := tryRecv(ch); !ok || v != 7 {
			t.Errorf("TODO 4: tryRecv on a channel holding 7 = (%d, %v), want (7, true)", v, ok)
		}
	})
}

func TestTryRecvOnClosedChannel(t *testing.T) {
	mustFinish(t, 5*time.Second, "tryRecv on a closed channel", func() {
		ch := make(chan int)
		close(ch)

		v, ok := tryRecv(ch)
		if ok || v != 0 {
			t.Errorf("TODO 4: tryRecv on a CLOSED channel = (%d, %v), want (0, false).\n"+
				"  A closed channel is always ready, so `case v := <-ch:` fires and hands\n"+
				"  you a zero that looks like a real value. Use `case v, ok := <-ch:`.", v, ok)
		}
	})
}

// --- TODO 5: merge2 ---------------------------------------------------

func TestMerge2CollectsBothChannels(t *testing.T) {
	a := make(chan int)
	b := make(chan int)
	go func() {
		defer close(a)
		for i := 1; i <= 3; i++ {
			a <- i
		}
	}()
	go func() {
		defer close(b)
		for i := 4; i <= 6; i++ {
			b <- i
		}
	}()

	var got []int
	mustFinish(t, 5*time.Second, "merge2", func() {
		got = merge2(a, b, make(chan struct{}))
	})

	sort.Ints(got)
	want := []int{1, 2, 3, 4, 5, 6}
	if len(got) != len(want) {
		t.Fatalf("TODO 5: merge2 returned %v, want the 6 values %v (in any order).\n"+
			"  If you got extra zeros, a closed channel is still firing in your select "+
			"— set the local variable to nil when ok is false.", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 5: merge2 returned %v, want %v (in any order)", got, want)
		}
	}
}

func TestMerge2UnevenChannels(t *testing.T) {
	a := make(chan int, 5)
	b := make(chan int)
	for i := 1; i <= 5; i++ {
		a <- i
	}
	close(a)
	close(b) // b closes immediately and delivers nothing

	var got []int
	mustFinish(t, 5*time.Second, "merge2 with one empty channel", func() {
		got = merge2(a, b, make(chan struct{}))
	})

	if len(got) != 5 {
		t.Errorf("TODO 5: with one channel closed-and-empty, merge2 returned %v, "+
			"want the 5 values from the other one", got)
	}
}

func TestMerge2StopsOnDone(t *testing.T) {
	a := make(chan int) // never sends, never closes
	b := make(chan int)
	done := make(chan struct{})
	close(done)

	mustFinish(t, 5*time.Second, "merge2 with done already closed", func() {
		merge2(a, b, done)
	})
	// Reaching here is the whole assertion: neither input will ever close, so
	// only the done case can end the loop.
}

func TestMerge2BothClosedImmediately(t *testing.T) {
	a := make(chan int)
	b := make(chan int)
	close(a)
	close(b)

	var got []int
	mustFinish(t, 5*time.Second, "merge2 with both closed", func() {
		got = merge2(a, b, make(chan struct{}))
	})
	if len(got) != 0 {
		t.Errorf("TODO 5: both channels closed and empty, but merge2 returned %v.\n"+
			"  Those are zero values from receiving on closed channels — check the "+
			"`ok` before appending.", got)
	}
}

// --- main()'s narration ----------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 15*time.Second, "main()", func() {
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
		{regexp.MustCompile(`(?m)^fast$`), "TODO 6", "\"fast\" — the shorter sleep wins"},
		{regexp.MustCompile(`(?m)^42$`), "TODO 7", "42 from the successful receive"},
		{regexp.MustCompile(`(?m)^timed out$`), "TODO 7", "\"timed out\" from the expired one"},
		{regexp.MustCompile(`(?m)^true$`), "TODO 8", "true from the first trySend"},
		{regexp.MustCompile(`(?m)^false$`), "TODO 8", "false from the second trySend"},
		{regexp.MustCompile(`(?m)^0 false$`), "TODO 9", "\"0 false\" from tryRecv"},
		{regexp.MustCompile(`(?m)^6$`), "TODO 10", "6 merged values"},
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
