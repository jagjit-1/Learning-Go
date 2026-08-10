package main

// ============================================================
// CHECKER for 09_capstone_game — run with:  go test
// ============================================================
// Two layers:
//
//   1. Unit tests over the suggested API (GameOverError, Game, NewGame,
//      Guess). If you deviated from the suggested structure these won't
//      compile — that's expected; delete or adapt them, and rely on the
//      end-to-end test below, which only cares about observable behaviour.
//
//   2. TestEndToEnd builds your program and plays it for real over stdin:
//      it feeds one piece of garbage input, then binary-searches 1..100.
//      Binary search needs exactly 7 guesses in the worst case, so if your
//      program counted "abc" as a guess, this test runs out of tries and
//      fails. That is the point.

import (
	"errors"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============================================================
// 1. Unit tests over the game API
// ============================================================

var _ error = &GameOverError{}

func TestNewGame(t *testing.T) {
	g := NewGame(7)
	if g == nil {
		t.Fatal("NewGame should return a *Game, got nil")
	}
	if g.MaxTries != 7 {
		t.Errorf("NewGame(7).MaxTries = %d, want 7", g.MaxTries)
	}
	if g.Attempts != 0 {
		t.Errorf("a fresh game should have Attempts = 0, got %d", g.Attempts)
	}
	if g.Target < 1 || g.Target > 100 {
		t.Errorf("Target = %d, want it in 1..100 (rand.Intn(100) gives 0..99, "+
			"so remember the +1)", g.Target)
	}
}

func TestNewGameTargetIsRandom(t *testing.T) {
	seen := make(map[int]bool)
	for i := 0; i < 200; i++ {
		n := NewGame(7).Target
		if n < 1 || n > 100 {
			t.Fatalf("Target = %d, want it in 1..100", n)
		}
		seen[n] = true
	}
	if len(seen) < 10 {
		t.Errorf("200 new games produced only %d distinct targets — the target "+
			"should be random per game", len(seen))
	}
}

func TestGuessFeedback(t *testing.T) {
	cases := []struct {
		guess int
		want  string
	}{
		{25, "low"},
		{49, "low"},
		{75, "high"},
		{51, "high"},
		{50, "correct"},
	}
	for _, c := range cases {
		g := NewGame(7)
		g.Target = 50

		result, err := g.Guess(c.guess)
		if err != nil {
			t.Errorf("Guess(%d) on attempt 1 of 7 returned an error: %v", c.guess, err)
			continue
		}
		if !strings.Contains(strings.ToLower(result), c.want) {
			t.Errorf("with Target 50, Guess(%d) = %q, expected it to say %q",
				c.guess, result, c.want)
		}
	}
}

func TestGuessCountsAttempts(t *testing.T) {
	g := NewGame(7)
	g.Target = 50

	for i := 1; i <= 3; i++ {
		if _, err := g.Guess(1); err != nil {
			t.Fatalf("guess %d of 7 returned an error: %v", i, err)
		}
		if g.Attempts != i {
			t.Fatalf("after %d guesses, Attempts = %d, want %d", i, g.Attempts, i)
		}
	}
}

func TestGuessRunsOutOfTries(t *testing.T) {
	g := NewGame(3)
	g.Target = 50

	for i := 1; i <= 3; i++ {
		if _, err := g.Guess(1); err != nil {
			t.Fatalf("guess %d should still be allowed with MaxTries 3, got: %v", i, err)
		}
	}

	_, err := g.Guess(1)
	if err == nil {
		t.Fatal("the 4th guess with MaxTries 3 should return a non-nil error")
	}

	var over *GameOverError
	if !errors.As(err, &over) {
		t.Fatalf("expected a *GameOverError, got %T: %v", err, err)
	}
	if over.Target != 50 {
		t.Errorf("GameOverError.Target = %d, want 50 — the error should reveal "+
			"the number", over.Target)
	}
	if over.Attempts <= 0 {
		t.Errorf("GameOverError.Attempts = %d, want the number of attempts made",
			over.Attempts)
	}
	if msg := err.Error(); msg == "" || !strings.Contains(msg, "50") {
		t.Errorf("GameOverError.Error() = %q — it should mention the target", msg)
	}
}

func TestCorrectGuessAtTheLimitStillWins(t *testing.T) {
	g := NewGame(3)
	g.Target = 50

	g.Guess(1)
	g.Guess(2)

	result, err := g.Guess(50)
	if err != nil {
		t.Fatalf("the 3rd guess of 3 should be allowed, got error: %v", err)
	}
	if !strings.Contains(strings.ToLower(result), "correct") {
		t.Errorf("guessing the target on the last allowed try = %q, want \"correct\"", result)
	}
}

// ============================================================
// 2. End-to-end: build the program and actually play it
// ============================================================

// outBuf collects the program's stdout+stderr in a goroutine-safe way.
type outBuf struct {
	mu sync.Mutex
	b  []byte
}

func (o *outBuf) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.b = append(o.b, p...)
	return len(p), nil
}

func (o *outBuf) since(n int) string {
	o.mu.Lock()
	defer o.mu.Unlock()
	if n > len(o.b) {
		return ""
	}
	return string(o.b[n:])
}

func (o *outBuf) len() int {
	o.mu.Lock()
	defer o.mu.Unlock()
	return len(o.b)
}

func (o *outBuf) String() string { return o.since(0) }

// waitForOutput polls until new output past `mark` satisfies pred.
func (o *outBuf) waitForOutput(mark int, pred func(string) bool, timeout time.Duration) (string, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if s := o.since(mark); pred(s) {
			return s, true
		}
		time.Sleep(20 * time.Millisecond)
	}
	return o.since(mark), false
}

func TestEndToEnd(t *testing.T) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not on PATH; skipping the end-to-end play-through")
	}

	bin := filepath.Join(t.TempDir(), "game")
	if out, err := exec.Command(goBin, "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("your program does not build:\n%s", out)
	}

	ob := &outBuf{}
	cmd := exec.Command(bin)
	cmd.Stdout = ob
	cmd.Stderr = ob
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("could not open stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("could not start your program: %v", err)
	}

	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	defer func() {
		stdin.Close()
		select {
		case <-finished:
		case <-time.After(3 * time.Second):
			cmd.Process.Kill()
			<-finished
		}
	}()

	send := func(s string) {
		if _, err := io.WriteString(stdin, s+"\n"); err != nil {
			t.Fatalf("your program stopped reading input after %q — full output:\n%s",
				s, indent(ob.String()))
		}
	}

	// It should greet the player before asking for anything.
	if _, ok := ob.waitForOutput(0, func(s string) bool {
		return strings.TrimSpace(s) != ""
	}, 5*time.Second); !ok {
		t.Fatal("the program printed nothing at startup — expected a prompt")
	}

	// Spec step 3: garbage input must not crash and must not burn a guess.
	mark := ob.len()
	send("abc")
	if got, ok := ob.waitForOutput(mark, func(s string) bool {
		return strings.TrimSpace(s) != ""
	}, 3*time.Second); !ok {
		t.Fatalf("after the input \"abc\" the program said nothing — expected a "+
			"friendly \"that's not a valid number\" and another prompt.\nOutput so far:\n%s",
			indent(ob.String()))
	} else if strings.Contains(strings.ToLower(got), "too low") ||
		strings.Contains(strings.ToLower(got), "too high") {
		t.Fatalf("the program treated \"abc\" as a guess: %q", strings.TrimSpace(got))
	}

	// Binary search over 1..100 — worst case is exactly 7 guesses.
	lo, hi := 1, 100
	won := false
	for i := 1; i <= 7 && lo <= hi; i++ {
		guess := (lo + hi) / 2
		mark = ob.len()
		send(strconv.Itoa(guess))

		resp, ok := ob.waitForOutput(mark, func(s string) bool {
			s = strings.ToLower(s)
			return strings.Contains(s, "low") || strings.Contains(s, "high") ||
				strings.Contains(s, "correct") || strings.Contains(s, "game over")
		}, 5*time.Second)
		if !ok {
			t.Fatalf("guess #%d (%d) got no usable response within 5s.\n"+
				"  new output was: %q\n  full output:\n%s",
				i, guess, strings.TrimSpace(resp), indent(ob.String()))
		}

		lower := strings.ToLower(resp)
		switch {
		case strings.Contains(lower, "correct"):
			won = true
		case strings.Contains(lower, "low"):
			lo = guess + 1
		case strings.Contains(lower, "high"):
			hi = guess - 1
		default:
			t.Fatalf("the game ended on guess #%d of 7: %q\n"+
				"  Binary search needs all 7 tries in the worst case, so this usually "+
				"means an invalid input was counted as a guess, or MaxTries is < 7.",
				i, strings.TrimSpace(resp))
		}
		if won {
			break
		}
	}

	if !won {
		t.Fatalf("binary search over 1..100 did not win within 7 guesses.\nFull output:\n%s",
			indent(ob.String()))
	}

	// Spec step 6: report how many guesses it took.
	final := strings.ToLower(ob.String())
	if !strings.Contains(final, "correct") {
		t.Errorf("expected a congratulation mentioning \"correct\"")
	}
	if !hasDigit(final[strings.LastIndex(final, "correct"):]) {
		t.Error("the winning message should say how many guesses it took")
	}
}

func hasDigit(s string) bool {
	return strings.ContainsAny(s, "0123456789")
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
