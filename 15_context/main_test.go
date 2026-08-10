package main

// ============================================================
// CHECKER for 15_context — run with:  go test -race
// ============================================================

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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
			"  Cancellation in Go is COOPERATIVE — nothing stops a goroutine from\n"+
			"  the outside. If you never select on ctx.Done(), the context expiring\n"+
			"  changes nothing about what your code is doing.", what, d)
	}
}

var (
	_ func(context.Context, time.Duration) error                                                 = doWork
	_ func(context.Context, []int, time.Duration) (int, error)                                   = sumWithContext
	_ func(context.Context, string) context.Context                                              = WithRequestID
	_ func(context.Context) (string, bool)                                                       = RequestID
	_ func(context.Context, []int, func(context.Context, int) (string, error)) ([]string, error) = fetchAll
)

// --- TODO 1: doWork ---------------------------------------------------

func TestDoWorkFinishesInTime(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var err error
	mustFinish(t, 5*time.Second, "doWork", func() { err = doWork(ctx, 20*time.Millisecond) })

	if err != nil {
		t.Errorf("TODO 1: 20ms of work under a 2s deadline returned %v, want nil", err)
	}
}

func TestDoWorkDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	var err error
	mustFinish(t, 5*time.Second, "doWork", func() { err = doWork(ctx, 5*time.Second) })
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 1: 5s of work under a 50ms deadline returned nil, want an error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("TODO 1: returned %v, want context.DeadlineExceeded.\n"+
			"  Return ctx.Err() rather than an error of your own — callers use "+
			"errors.Is against the context sentinels.", err)
	}
	if elapsed > time.Second {
		t.Errorf("TODO 1: took %v to notice a 50ms deadline — it waited for the work "+
			"to finish instead of racing it against ctx.Done()", elapsed)
	}
}

func TestDoWorkCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before we start

	var err error
	mustFinish(t, 5*time.Second, "doWork", func() { err = doWork(ctx, 5*time.Second) })

	if !errors.Is(err, context.Canceled) {
		t.Errorf("TODO 1: returned %v, want context.Canceled — a cancelled context "+
			"and an expired one are different errors, and callers care which", err)
	}
}

// --- TODO 2: sumWithContext -------------------------------------------

func TestSumWithContextCompletes(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var sum int
	var err error
	mustFinish(t, 10*time.Second, "sumWithContext", func() {
		sum, err = sumWithContext(ctx, []int{1, 2, 3, 4, 5}, time.Millisecond)
	})

	if err != nil {
		t.Fatalf("TODO 2: plenty of time available, but got %v", err)
	}
	if sum != 15 {
		t.Errorf("TODO 2: sum = %d, want 15", sum)
	}
}

func TestSumWithContextReturnsPartialWork(t *testing.T) {
	nums := make([]int, 200)
	for i := range nums {
		nums[i] = 1 // every element adds exactly 1, so the sum IS the count
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	var sum int
	var err error
	mustFinish(t, 10*time.Second, "sumWithContext", func() {
		sum, err = sumWithContext(ctx, nums, 2*time.Millisecond)
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 2: 200 elements at 2ms each cannot finish inside 50ms, " +
			"so this should have returned an error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("TODO 2: returned %v, want context.DeadlineExceeded", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("TODO 2: took %v — check the context BETWEEN elements, not just "+
			"once at the start", elapsed)
	}
	if sum <= 0 || sum >= 200 {
		t.Errorf("TODO 2: partial sum = %d, want somewhere between 1 and 199 — "+
			"return the work already done alongside the error, not 0", sum)
	}
}

func TestSumWithContextEmpty(t *testing.T) {
	mustFinish(t, 5*time.Second, "sumWithContext on an empty slice", func() {
		sum, err := sumWithContext(context.Background(), nil, time.Millisecond)
		if sum != 0 || err != nil {
			t.Errorf("TODO 2: got (%d, %v), want (0, nil)", sum, err)
		}
	})
}

// --- TODO 3: request IDs ----------------------------------------------

func TestRequestIDRoundTrip(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-42")

	id, ok := RequestID(ctx)
	if !ok {
		t.Fatal("TODO 3: RequestID reported not-found on a context that has one")
	}
	if id != "req-42" {
		t.Errorf("TODO 3: RequestID = %q, want %q", id, "req-42")
	}
}

func TestRequestIDMissing(t *testing.T) {
	id, ok := RequestID(context.Background())
	if ok || id != "" {
		t.Errorf("TODO 3: RequestID on a bare context = (%q, %v), want (\"\", false).\n"+
			"  ctx.Value returns nil for a missing key, and a failed type assertion "+
			"gives you the zero value plus false — use the two-value form.", id, ok)
	}
}

func TestRequestIDSurvivesDerivedContexts(t *testing.T) {
	ctx := WithRequestID(context.Background(), "req-7")
	child, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	if id, ok := RequestID(child); !ok || id != "req-7" {
		t.Errorf("TODO 3: a derived context lost the value: got (%q, %v)", id, ok)
	}
}

func TestRequestIDKeyIsNotAPlainString(t *testing.T) {
	// If the key were the string "requestID", any package could plant a value
	// under it and RequestID would happily return it.
	ctx := context.WithValue(context.Background(), "requestID", "planted")
	if id, ok := RequestID(ctx); ok {
		t.Errorf("TODO 3: RequestID picked up %q from a plain string key.\n"+
			"  Use an unexported key type (`type requestIDKey struct{}`) so no other "+
			"package can collide with it.", id)
	}
}

// --- TODO 4: fetchAll -------------------------------------------------

func TestFetchAllReturnsResultsInInputOrder(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5}

	var got []string
	var err error
	mustFinish(t, 10*time.Second, "fetchAll", func() {
		got, err = fetchAll(context.Background(), ids, func(ctx context.Context, id int) (string, error) {
			// Later ids finish first, so completion order is reversed.
			time.Sleep(time.Duration(20-id*3) * time.Millisecond)
			return fmt.Sprintf("item-%d", id), nil
		})
	})

	if err != nil {
		t.Fatalf("TODO 4: every fetch succeeded but got error %v", err)
	}
	if len(got) != len(ids) {
		t.Fatalf("TODO 4: got %d results for %d ids", len(got), len(ids))
	}
	for i, id := range ids {
		if want := fmt.Sprintf("item-%d", id); got[i] != want {
			t.Fatalf("TODO 4: position %d holds %q, want %q — results must come back "+
				"in the order of `ids`, not the order they finished.\n  got: %v",
				i, got[i], want, got)
		}
	}
}

func TestFetchAllIsConcurrent(t *testing.T) {
	ids := []int{1, 2, 3, 4, 5, 6, 7, 8}

	start := time.Now()
	mustFinish(t, 10*time.Second, "fetchAll", func() {
		fetchAll(context.Background(), ids, func(ctx context.Context, id int) (string, error) {
			time.Sleep(80 * time.Millisecond)
			return "ok", nil
		})
	})
	elapsed := time.Since(start)

	if elapsed > 400*time.Millisecond {
		t.Errorf("TODO 4: 8 fetches of 80ms each took %v. Run sequentially that's "+
			"640ms; concurrently it's about 80ms — these aren't running at the "+
			"same time.", elapsed)
	}
}

func TestFetchAllReportsTheError(t *testing.T) {
	boom := errors.New("fetch 3 failed")

	var err error
	mustFinish(t, 10*time.Second, "fetchAll with a failure", func() {
		_, err = fetchAll(context.Background(), []int{1, 2, 3, 4, 5},
			func(ctx context.Context, id int) (string, error) {
				if id == 3 {
					return "", boom
				}
				return "ok", nil
			})
	})

	if err == nil {
		t.Fatal("TODO 4: id 3 failed but fetchAll returned nil")
	}
	if !errors.Is(err, boom) {
		t.Errorf("TODO 4: returned %v, want the error the fetch produced", err)
	}
}

func TestFetchAllCancelsTheSiblings(t *testing.T) {
	var cancelled atomic.Int64

	start := time.Now()
	var err error
	mustFinish(t, 10*time.Second, "fetchAll cancelling siblings", func() {
		_, err = fetchAll(context.Background(), []int{1, 2, 3, 4, 5},
			func(ctx context.Context, id int) (string, error) {
				if id == 3 {
					return "", errors.New("boom")
				}
				select {
				case <-time.After(3 * time.Second):
					return "slow", nil
				case <-ctx.Done():
					cancelled.Add(1)
					return "", ctx.Err()
				}
			})
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 4: expected an error")
	}
	if cancelled.Load() == 0 {
		t.Errorf("TODO 4: none of the sibling fetches saw a cancelled context.\n" +
			"  Derive a child context inside fetchAll (context.WithCancel), pass THAT\n" +
			"  to fetch, and call cancel() when a fetch fails. Passing the caller's\n" +
			"  context straight through gives you nothing to cancel.")
	}
	if elapsed > 2*time.Second {
		t.Errorf("TODO 4: took %v — it waited out the slow siblings instead of "+
			"cancelling them", elapsed)
	}
}

func TestFetchAllRespectsTheCallersContext(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Millisecond)
	defer cancel()

	start := time.Now()
	var err error
	mustFinish(t, 10*time.Second, "fetchAll under a caller deadline", func() {
		_, err = fetchAll(ctx, []int{1, 2, 3}, func(ctx context.Context, id int) (string, error) {
			select {
			case <-time.After(3 * time.Second):
				return "slow", nil
			case <-ctx.Done():
				return "", ctx.Err()
			}
		})
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 4: the caller's context expired, so this should have failed")
	}
	if elapsed > 2*time.Second {
		t.Errorf("TODO 4: took %v — the child context must be derived FROM the "+
			"caller's, so the caller's deadline still applies", elapsed)
	}
}

func TestFetchAllEmpty(t *testing.T) {
	mustFinish(t, 5*time.Second, "fetchAll with no ids", func() {
		got, err := fetchAll(context.Background(), nil, func(ctx context.Context, id int) (string, error) {
			return "", errors.New("should never be called")
		})
		if err != nil {
			t.Errorf("TODO 4: no ids means no work and no error, got %v", err)
		}
		if len(got) != 0 {
			t.Errorf("TODO 4: got %v, want empty", got)
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
		{regexp.MustCompile(`(?m)^<nil>$`), "TODO 5", "<nil> — the work beat the deadline"},
		{regexp.MustCompile(`context deadline exceeded`), "TODO 6", "\"context deadline exceeded\""},
		{regexp.MustCompile(`context canceled`), "TODO 7", "\"context canceled\" (note: one l)"},
		{regexp.MustCompile(`(?m)^\d+ context deadline exceeded$`), "TODO 8",
			"a partial sum followed by the deadline error"},
		{regexp.MustCompile(`(?m)^req-42 true$`), "TODO 9", "\"req-42 true\""},
		{regexp.MustCompile(`(?m)^ false$`), "TODO 9", "\" false\" — empty id, ok is false"},
		{regexp.MustCompile(`\[item-1 item-2 item-3 item-4 item-5\]`), "TODO 10",
			"[item-1 item-2 item-3 item-4 item-5] in id order"},
		{regexp.MustCompile(`(?mi)^.*fetch.*3.*fail`), "TODO 10", "the error from the failing fetch"},
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
