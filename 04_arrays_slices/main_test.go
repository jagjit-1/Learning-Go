package main

// ============================================================
// CHECKER for 04_arrays_slices — run with:  go test
// ============================================================
// Your song titles are yours to pick, so the playlist checks look at
// structure ("1: something"). The sharing experiment has exact expected
// numbers, so those are checked literally.

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

// --- TODO 5: removeSong, tested directly ----------------------------

func TestRemoveSong(t *testing.T) {
	cases := []struct {
		name  string
		in    []string
		index int
		want  []string
	}{
		{"middle", []string{"a", "b", "c", "d"}, 1, []string{"a", "c", "d"}},
		{"first", []string{"a", "b", "c"}, 0, []string{"b", "c"}},
		{"last", []string{"a", "b", "c"}, 2, []string{"a", "b"}},
		{"only element", []string{"a"}, 0, []string{}},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// fresh copy each time: the idiomatic append-based removal
			// mutates the input slice's backing array.
			in := append([]string(nil), c.in...)
			got := removeSong(in, c.index)

			if len(got) != len(c.want) {
				t.Fatalf("TODO 5: removeSong(%v, %d) = %v, want %v",
					c.in, c.index, got, c.want)
			}
			for i := range c.want {
				if got[i] != c.want[i] {
					t.Fatalf("TODO 5: removeSong(%v, %d) = %v, want %v\n"+
						"  hint: append(pl[:index], pl[index+1:]...) — don't forget the ...",
						c.in, c.index, got, c.want)
				}
			}
		})
	}
}

// --- TODOs 1-4: checked through main()'s output ---------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 1")
	}

	// TODO 1: the initial 3-song playlist printed with fmt.Println
	// (a []string prints as [a b c])
	initial := regexp.MustCompile(`\[[^\[\]]+ [^\[\]]+ [^\[\]]+\]`)
	if !initial.MatchString(out) {
		t.Errorf("TODO 1: expected fmt.Println(playlist) to print 3 songs like " +
			"[Song1 Song2 Song3]")
	}

	// TODO 3: five 1-indexed lines
	for i := 1; i <= 5; i++ {
		n := strconv.Itoa(i)
		if !regexp.MustCompile(`(?m)^\s*` + n + `:\s+\S.*$`).MatchString(out) {
			t.Errorf("TODO 2/3: expected a line %q (1-indexed, so position 1 is the "+
				"first song, and there should be 5 songs after the appends)", n+": <song>")
		}
	}

	// TODO 4 b): len/cap after each of the two appends into make([]int, 0, 2)
	lenCap1 := regexp.MustCompile(`len\D*1\D+cap\D*2`)
	lenCap2 := regexp.MustCompile(`len\D*2\D+cap\D*2`)
	if !lenCap1.MatchString(out) {
		t.Errorf("TODO 4b: expected a line reporting len 1 / cap 2 after the first append")
	}
	if !lenCap2.MatchString(out) {
		t.Errorf("TODO 4b: expected a line reporting len 2 / cap 2 after the second append")
	}

	// TODO 4 e): sub[0] = 999 leaked into nums, because they still share
	// the same backing array.
	shared := regexp.MustCompile(`\[999 20\]`)
	if !shared.MatchString(out) {
		t.Errorf("TODO 4e: expected to see [999 20].\n" +
			"  This is the point of the experiment: sub := nums[0:1] shares nums'\n" +
			"  backing array, so sub[0] = 999 is visible through nums.\n" +
			"  (Make sure you appended 10 then 20 as the two elements.)")
	}

	// TODO 4 f/h): appending past cap 2 moves nums onto a NEW array, so the
	// later sub[0] = 111 does not show up in nums.
	if !regexp.MustCompile(`\[999 20 30\]`).MatchString(out) {
		t.Errorf("TODO 4f/4h: expected [999 20 30] after appending 30 and then " +
			"setting sub[0] = 111")
	}
	if strings.Contains(out, "[111") {
		t.Errorf("TODO 4h: found [111 ... in your output.\n" +
			"  After the append that exceeded cap 2, nums points at a freshly\n" +
			"  allocated array and sub still points at the old one — so sub[0] = 111\n" +
			"  must NOT be visible through nums. Seeing it means your append didn't\n" +
			"  actually grow past the capacity.")
	}
}
