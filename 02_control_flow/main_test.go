package main

// ============================================================
// CHECKER for 02_control_flow — run with:  go test
// ============================================================
// Checks the five TODOs by reading what main() prints. The FizzBuzz and
// countdown blocks must appear as consecutive lines somewhere in your
// output (extra headers/labels around them are fine).

import (
	"bytes"
	"io"
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

func nonEmptyLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

// findSequence returns the index in lines (at or after `from`) where all of
// `want` appears as consecutive lines, or -1.
func findSequence(lines, want []string, from int) int {
	for i := from; i+len(want) <= len(lines); i++ {
		ok := true
		for j, w := range want {
			if lines[i+j] != w {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}

// findLine returns the index of the first line at/after `from` satisfying pred, or -1.
func findLine(lines []string, from int, pred func(string) bool) int {
	for i := from; i < len(lines); i++ {
		if pred(lines[i]) {
			return i
		}
	}
	return -1
}

func TestOutput(t *testing.T) {
	lines := nonEmptyLines(captureStdout(t, main))
	if len(lines) == 0 {
		t.Fatal("main() printed nothing — start with TODO 1")
	}

	// --- TODO 1: FizzBuzz 1..20 -------------------------------------
	var fizz []string
	for i := 1; i <= 20; i++ {
		switch {
		case i%15 == 0:
			fizz = append(fizz, "FizzBuzz")
		case i%3 == 0:
			fizz = append(fizz, "Fizz")
		case i%5 == 0:
			fizz = append(fizz, "Buzz")
		default:
			fizz = append(fizz, strconv.Itoa(i))
		}
	}
	fizzAt := findSequence(lines, fizz, 0)
	if fizzAt < 0 {
		t.Fatalf("TODO 1: could not find FizzBuzz 1..20 as consecutive lines.\n"+
			"  want:\n    %s\n  your output was:\n    %s",
			strings.Join(fizz, "\n    "), strings.Join(lines, "\n    "))
	}
	cursor := fizzAt + len(fizz)

	// --- TODO 2: countdown 10..1 ------------------------------------
	var down []string
	for i := 10; i >= 1; i-- {
		down = append(down, strconv.Itoa(i))
	}
	downAt := findSequence(lines, down, cursor)
	if downAt < 0 {
		t.Fatalf("TODO 2: could not find the countdown 10,9,...,1 as consecutive "+
			"lines after FizzBuzz.\n  your output after FizzBuzz was:\n    %s",
			strings.Join(lines[cursor:], "\n    "))
	}
	cursor = downAt + len(down)

	// --- TODO 3: sum 1..n until it exceeds 1000 ---------------------
	// 1+2+...+45 == 1035, and 1..44 == 990, so the answer is 45 numbers / 1035.
	has45 := regexp.MustCompile(`\b45\b`)
	has1035 := regexp.MustCompile(`\b1035\b`)
	sumAt := findLine(lines, cursor, func(l string) bool {
		return has45.MatchString(l) && has1035.MatchString(l)
	})
	if sumAt < 0 {
		t.Fatalf("TODO 3: expected a line reporting that it took 45 numbers to reach "+
			"a total of 1035.\n  hint: 1+2+...+44 == 990 (not > 1000), 1+2+...+45 == 1035.\n"+
			"  your output after the countdown was:\n    %s",
			strings.Join(lines[cursor:], "\n    "))
	}
	cursor = sumAt + 1

	// --- TODO 4: grade for score 78 ---------------------------------
	gradeAt := findLine(lines, cursor, func(l string) bool { return l == "C" })
	if gradeAt < 0 {
		t.Fatalf("TODO 4: expected a line %q (score 78 is in the 70-79 band).\n"+
			"  your output after the sum line was:\n    %s",
			"C", strings.Join(lines[cursor:], "\n    "))
	}
	cursor = gradeAt + 1

	// --- TODO 5: day classification ---------------------------------
	if findLine(lines, cursor, func(l string) bool { return l == "Weekday" }) < 0 {
		t.Fatalf("TODO 5: expected a line %q (\"Wed\" is a weekday).\n"+
			"  your output after the grade was:\n    %s",
			"Weekday", strings.Join(lines[cursor:], "\n    "))
	}
}
