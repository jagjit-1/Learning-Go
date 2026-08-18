package main

// ============================================================
// CHECKER for 22_testing_benchmarks — run with:  go test
// ============================================================
// This one checks your TABLE as well as your code: every Want in JoinCases
// must be right, and the table has to actually cover the awkward inputs.
// It also runs your benchmarks and looks at what they measured.

import (
	"bytes"
	"io"
	"os"
	"regexp"
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
				t.Errorf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

var (
	_ func([]string, string) string = JoinNaive
	_ func([]string, string) string = JoinBuilder
	_ func() []JoinCase             = JoinCases
	_ func(*testing.B)              = BenchmarkJoinNaive
	_ func(*testing.B)              = BenchmarkJoinBuilder
	_                               = JoinCase{Name: "n", Parts: nil, Sep: "-", Want: ""}
)

// --- TODOs 1 & 2: both implementations, against a fixed table ------------

func TestJoinImplementations(t *testing.T) {
	cases := []struct {
		name  string
		parts []string
		sep   string
		want  string
	}{
		{"no parts", nil, "-", ""},
		{"empty slice", []string{}, "-", ""},
		{"single", []string{"a"}, "-", "a"},
		{"three", []string{"a", "b", "c"}, "-", "a-b-c"},
		{"empty separator", []string{"a", "b", "c"}, "", "abc"},
		{"multi-char separator", []string{"a", "b"}, " -> ", "a -> b"},
		{"empty part", []string{"a", "", "c"}, "-", "a--c"},
		{"all empty parts", []string{"", "", ""}, ",", ",,"},
		{"separator inside a part", []string{"a-b", "c"}, "-", "a-b-c"},
	}

	impls := []struct {
		todo string
		name string
		fn   func([]string, string) string
	}{
		{"TODO 1", "JoinNaive", JoinNaive},
		{"TODO 2", "JoinBuilder", JoinBuilder},
	}

	for _, impl := range impls {
		t.Run(impl.name, func(t *testing.T) {
			for _, c := range cases {
				t.Run(c.name, func(t *testing.T) {
					if got := impl.fn(c.parts, c.sep); got != c.want {
						t.Errorf("%s: %s(%q, %q) = %q, want %q",
							impl.todo, impl.name, c.parts, c.sep, got, c.want)
					}
				})
			}
		})
	}
}

func TestBothImplementationsAgree(t *testing.T) {
	inputs := [][]string{
		nil, {"x"}, {"x", "y"}, {"", "a", ""}, {"one", "two", "three", "four"},
	}
	for _, sep := range []string{"", "-", "::"} {
		for _, in := range inputs {
			a, b := JoinNaive(in, sep), JoinBuilder(in, sep)
			if a != b {
				t.Errorf("TODO 2: the two implementations disagree on (%q, %q): "+
					"naive gave %q, builder gave %q", in, sep, a, b)
			}
		}
	}
}

// --- TODO 3: the table itself --------------------------------------------

func TestJoinCasesAreCorrect(t *testing.T) {
	cases := JoinCases()
	if len(cases) < 5 {
		t.Fatalf("TODO 3: JoinCases returned %d cases, want at least 5", len(cases))
	}

	for _, c := range cases {
		if c.Name == "" {
			t.Errorf("TODO 3: a case has an empty Name — the name is what shows up " +
				"in the output when a subtest fails, so it has to identify the case")
		}
		if got := JoinBuilder(c.Parts, c.Sep); got != c.Want {
			t.Errorf("TODO 3: case %q expects %q, but JoinBuilder(%q, %q) = %q.\n"+
				"  Either the Want in your table is wrong or your implementation is — "+
				"work out which before changing anything.",
				c.Name, c.Want, c.Parts, c.Sep, got)
		}
	}
}

func TestJoinCasesCoverTheEdges(t *testing.T) {
	cases := JoinCases()

	var (
		hasNoParts    bool
		hasSinglePart bool
		hasEmptySep   bool
		hasMultiSep   bool
		hasEmptyPart  bool
	)
	for _, c := range cases {
		switch {
		case len(c.Parts) == 0:
			hasNoParts = true
		case len(c.Parts) == 1:
			hasSinglePart = true
		}
		if c.Sep == "" {
			hasEmptySep = true
		}
		if len(c.Sep) > 1 {
			hasMultiSep = true
		}
		for _, p := range c.Parts {
			if p == "" {
				hasEmptyPart = true
			}
		}
	}

	missing := []string{}
	if !hasNoParts {
		missing = append(missing, "a case with no parts at all")
	}
	if !hasSinglePart {
		missing = append(missing, "a case with exactly one part (no separator should appear)")
	}
	if !hasEmptySep {
		missing = append(missing, "a case with an empty separator")
	}
	if !hasMultiSep {
		missing = append(missing, "a case with a separator longer than one character")
	}
	if !hasEmptyPart {
		missing = append(missing, "a case where one of the parts is the empty string")
	}

	if len(missing) > 0 {
		t.Errorf("TODO 3: your table is missing:\n    - %s\n"+
			"  A table-driven test is only as good as its rows. The boundaries are "+
			"where the bugs live — a Join that forgets the len==1 case still passes "+
			"every three-element example you throw at it.",
			strings.Join(missing, "\n    - "))
	}
}

// --- TODOs 4 & 5: the benchmarks ------------------------------------------

func TestBenchmarksUseBN(t *testing.T) {
	if testing.Short() {
		t.Skip("benchmarks take a second each; skipped under -short")
	}

	res := testing.Benchmark(BenchmarkJoinNaive)

	if res.N <= 0 {
		t.Fatalf("TODO 4: the benchmark reported N = %d", res.N)
	}
	if res.NsPerOp() < 100 {
		t.Errorf("TODO 4: BenchmarkJoinNaive reports %d ns/op over N = %d.\n"+
			"  Joining 200 strings the slow way cannot possibly take that little.\n"+
			"  The usual cause is not looping: the body must be\n"+
			"      for i := 0; i < b.N; i++ { JoinNaive(parts, \"-\") }\n"+
			"  A benchmark that does the work once and ignores b.N divides that one\n"+
			"  measurement by a huge N and reports near zero.", res.NsPerOp(), res.N)
	}
}

func TestBuilderAllocatesLess(t *testing.T) {
	if testing.Short() {
		t.Skip("benchmarks take a second each; skipped under -short")
	}

	naive := testing.Benchmark(BenchmarkJoinNaive)
	builder := testing.Benchmark(BenchmarkJoinBuilder)

	if naive.AllocsPerOp() == 0 || builder.AllocsPerOp() == 0 {
		t.Fatalf("TODO 4/5: allocs/op came back as naive=%d builder=%d. At least "+
			"one benchmark isn't running the join inside the b.N loop.",
			naive.AllocsPerOp(), builder.AllocsPerOp())
	}

	if builder.AllocsPerOp()*5 > naive.AllocsPerOp() {
		t.Errorf("TODO 2/5: JoinBuilder does %d allocs/op against JoinNaive's %d.\n"+
			"  Expected the builder to be dramatically cheaper — roughly one\n"+
			"  allocation rather than one per element. Check you called b.Grow(size)\n"+
			"  with the exact final length before writing anything.",
			builder.AllocsPerOp(), naive.AllocsPerOp())
	}

	t.Logf("naive: %d ns/op, %d allocs/op | builder: %d ns/op, %d allocs/op",
		naive.NsPerOp(), naive.AllocsPerOp(), builder.NsPerOp(), builder.AllocsPerOp())
}

// --- main()'s narration ----------------------------------------------------

func TestOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("main() runs the benchmarks; skipped under -short")
	}

	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 6")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^a-b-c a-b-c$`), "TODO 6", "\"a-b-c a-b-c\" — both agree"},
		{regexp.MustCompile(`(?m)^([5-9]|\d{2,})$`), "TODO 7", "the number of cases (5 or more)"},
		{regexp.MustCompile(`(?m)^true$`), "TODO 8", "true — the builder allocates less"},
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
