package main

// ============================================================
// CHECKER for 08_error_handling — run with:  go test
// ============================================================
// Note: this tests ParseAge itself (not a "v2"), so fold the custom
// error from TODO 3 into ParseAge rather than writing a second function.

import (
	"bytes"
	"errors"
	"io"
	"math"
	"os"
	"regexp"
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
				t.Fatalf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

// --- TODO 3: NegativeAgeError must satisfy the error interface -------
// This is a compile-time check. If it fails, either the type is missing
// or Error() has a pointer receiver mismatch.
var _ error = &NegativeAgeError{}

// --- TODO 1: Divide ---------------------------------------------------

func TestDivideSucceeds(t *testing.T) {
	cases := []struct{ a, b, want float64 }{
		{10, 2, 5},
		{-10, 4, -2.5},
		{0, 7, 0},
		{1, 3, 1.0 / 3.0},
	}
	for _, c := range cases {
		got, err := Divide(c.a, c.b)
		if err != nil {
			t.Errorf("TODO 1: Divide(%v, %v) returned an unexpected error: %v", c.a, c.b, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("TODO 1: Divide(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestDivideByZeroFails(t *testing.T) {
	got, err := Divide(10, 0)
	if err == nil {
		t.Fatal("TODO 1: Divide(10, 0) should return a non-nil error " +
			"(float division by zero silently gives +Inf, which is why the guard matters)")
	}
	if err.Error() == "" {
		t.Error("TODO 1: the error needs a message, e.g. errors.New(\"division by zero\")")
	}
	if got != 0 {
		t.Errorf("TODO 1: on the error path, return the zero value alongside the "+
			"error — got %v, want 0", got)
	}
}

// --- TODO 2: ParseAge, happy path and range ---------------------------

func TestParseAgeValid(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"25", 25},
		{"0", 0},
		{"150", 150}, // "over 150" is the failure, so 150 itself is fine
		{"1", 1},
	}
	for _, c := range cases {
		got, err := ParseAge(c.in)
		if err != nil {
			t.Errorf("TODO 2: ParseAge(%q) returned an unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("TODO 2: ParseAge(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestParseAgeTooLarge(t *testing.T) {
	for _, in := range []string{"151", "200", "99999"} {
		_, err := ParseAge(in)
		if err == nil {
			t.Errorf("TODO 2: ParseAge(%q) should fail — anything over 150 is rejected", in)
			continue
		}
		var neg *NegativeAgeError
		if errors.As(err, &neg) {
			t.Errorf("TODO 2/3: ParseAge(%q) returned a *NegativeAgeError, but %q "+
				"is not negative — that branch should use fmt.Errorf", in, in)
		}
	}
}

func TestParseAgePassesThroughTheStrconvError(t *testing.T) {
	for _, in := range []string{"abc", "", "12.5", "2 5"} {
		_, err := ParseAge(in)
		if err == nil {
			t.Errorf("TODO 2: ParseAge(%q) should fail — that isn't an integer", in)
			continue
		}
		if !errors.Is(err, strconv.ErrSyntax) {
			t.Errorf("TODO 2: ParseAge(%q) should pass strconv.Atoi's own error back "+
				"(return it as-is, or wrap it with %%w so errors.Is can still see it).\n"+
				"  got: %v", in, err)
		}
	}
}

// --- TODOs 3 & 4: the custom error type -------------------------------

func TestParseAgeReturnsNegativeAgeError(t *testing.T) {
	for _, n := range []int{-1, -5, -10, -99} {
		in := strconv.Itoa(n)
		_, err := ParseAge(in)
		if err == nil {
			t.Errorf("TODO 3: ParseAge(%q) should fail", in)
			continue
		}

		var neg *NegativeAgeError
		if !errors.As(err, &neg) {
			t.Errorf("TODO 3: ParseAge(%q) should return &NegativeAgeError{...} so that "+
				"errors.As can pick it out.\n  got a plain error instead: %v", in, err)
			continue
		}
		if neg.Value != n {
			t.Errorf("TODO 3: NegativeAgeError.Value = %d, want %d", neg.Value, n)
		}
	}
}

func TestNegativeAgeErrorMessage(t *testing.T) {
	e := &NegativeAgeError{Value: -10}
	msg := e.Error()
	if msg == "" {
		t.Fatal("TODO 3: NegativeAgeError.Error() returned an empty string")
	}
	if !strings.Contains(msg, "-10") {
		t.Errorf("TODO 3: Error() = %q — it should mention the offending value (-10), "+
			"otherwise the custom type buys you nothing over errors.New", msg)
	}
}

// --- main()'s narration -----------------------------------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 1")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`5\.00`), "TODO 1", "the result of Divide(10, 2) formatted with %.2f"},
		{regexp.MustCompile(`(?i)division by zero`), "TODO 1", "the Divide(10, 0) error printed"},
		{regexp.MustCompile(`(?i)parsed age:\s*25`), "TODO 2", "\"Parsed age: 25\" for the input \"25\""},
		{regexp.MustCompile(`(?i)invalid syntax`), "TODO 2", "strconv's own error for the input \"abc\""},
		{regexp.MustCompile(`(?i)rejected negative age:\s*-10`), "TODO 4", "\"Rejected negative age: -10\""},
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
