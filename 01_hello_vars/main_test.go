package main

// ============================================================
// CHECKER for 01_hello_vars — run with:  go test
// ============================================================
// This is NOT a solution. It runs your main() and checks the SHAPE of
// what you printed. Your name / age / height are yours to pick; the
// sentence structure and the arithmetic are what get checked.

import (
	"bytes"
	"io"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// captureStdout runs fn and returns everything it printed.
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

// nonEmptyLines splits output into trimmed, non-blank lines.
func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

var introRe = regexp.MustCompile(`^(.+) is (\d+) years old, (\d+(?:\.\d+)?)m tall\. Learning Go: (true|false)$`)

func TestOutput(t *testing.T) {
	lines := nonEmptyLines(captureStdout(t, main))

	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines of output, got %d:\n%s",
			len(lines), strings.Join(lines, "\n"))
	}

	// --- TODO 1-5: the intro sentence -------------------------------
	m := introRe.FindStringSubmatch(lines[0])
	if m == nil {
		t.Fatalf("line 1 does not match the required sentence shape.\n"+
			"  got:  %q\n"+
			"  want: \"<Name> is <age> years old, <height>m tall. Learning Go: <true|false>\"\n"+
			"  hint: one fmt.Printf with %%s %%d %%f-ish and %%t verbs, ending in \\n",
			lines[0])
	}

	name, ageStr, heightStr := m[1], m[2], m[3]

	if name == "" {
		t.Error("TODO 1: name is empty — declare it with := and a real value")
	}

	age, err := strconv.Atoi(ageStr)
	if err != nil || age <= 0 {
		t.Errorf("TODO 2: age should be a positive int, got %q", ageStr)
	}

	height, err := strconv.ParseFloat(heightStr, 64)
	if err != nil || height <= 0 {
		t.Errorf("TODO 3: heightMeters should be a positive float64, got %q", heightStr)
	}

	// --- TODO 6: float64(age) / 2 -----------------------------------
	got, err := strconv.ParseFloat(lines[1], 64)
	if err != nil {
		t.Fatalf("TODO 6: line 2 should be just the number float64(age)/2, got %q", lines[1])
	}
	want := float64(age) / 2
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("TODO 6: printed %v, but float64(%d)/2 == %v\n"+
			"  hint: if you got a whole number when you expected a fraction, you did "+
			"integer division — convert to float64 BEFORE dividing", got, age, want)
	}
	if age%2 == 0 {
		t.Logf("note: your age (%d) is even, so int division and float division give "+
			"the same answer here and this check can't tell them apart. Try an odd "+
			"age for a moment to see the difference the float64() conversion makes.", age)
	}

	// --- TODO 7: the constant ---------------------------------------
	if lines[2] != "There are 7 days in a week" {
		t.Errorf("TODO 7: line 3 should be exactly %q, got %q",
			"There are 7 days in a week", lines[2])
	}
}
