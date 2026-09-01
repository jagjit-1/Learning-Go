package main

import (
	"fmt"
	"strings"
	"testing"
)

// ============================================================
// CONCEPT: testing and benchmarking
// ============================================================
//
// You have been reading test files for 21 exercises. Now you write them.
//
// A test lives in a file ending _test.go, in the same package, and is a
// function taking *testing.T:
//
//   func TestThing(t *testing.T) {
//       got := Thing()
//       if got != "want" {
//           t.Errorf("Thing() = %q, want %q", got, "want")
//       }
//   }
//
//   t.Errorf   record a failure, KEEP GOING
//   t.Fatalf   record a failure and stop this test now
//
// Use Fatal when continuing would panic or produce noise (a nil you're about
// to dereference); Errorf otherwise, so one run reports every problem.
//
// The message convention is "got X, want Y" — and include the INPUT, because
// a failure you can't reproduce from the message costs you a debugging round
// trip. Compare `Join() = "a-b"` with `Join([a b], "-") = "a-b", want "a|b"`.
//
// TABLE-DRIVEN TESTS are the house style. One slice of cases, one loop:
//
//   cases := []struct {
//       name string
//       in   []string
//       want string
//   }{
//       {"empty", nil, ""},
//       {"single", []string{"a"}, "a"},
//   }
//   for _, c := range cases {
//       t.Run(c.name, func(t *testing.T) {
//           if got := Join(c.in); got != c.want {
//               t.Errorf("Join(%v) = %q, want %q", c.in, got, c.want)
//           }
//       })
//   }
//
// t.Run makes each case a SUBTEST: it gets its own name in the output, a
// Fatal in one doesn't kill the rest, and you can run just one with
// `go test -run 'TestJoin/empty'`.
//
// t.Helper() marks a function as a helper, so failures report the CALLER's
// line number instead of the line inside the helper. Any assertion function
// you write should start with it.
//
// testing.TB is the interface both *testing.T and *testing.B satisfy — take
// it when a helper should work from tests and benchmarks alike.
//
// BENCHMARKS take *testing.B and loop b.N times:
//
//   func BenchmarkJoin(b *testing.B) {
//       parts := makeParts(200)     // setup, not measured if you reset
//       b.ReportAllocs()
//       b.ResetTimer()
//       for i := 0; i < b.N; i++ {
//           Join(parts, "-")
//       }
//   }
//
// The framework picks b.N, raising it until the run takes about a second,
// then reports ns/op. YOU MUST USE b.N — a benchmark that does the work once
// and ignores b.N reports a meaningless near-zero time.
//
//   b.ResetTimer()    discard setup time already spent
//   b.StopTimer()/b.StartTimer()   exclude work inside the loop
//   b.ReportAllocs()  add allocs/op and B/op to the output
//
// Running them:
//
//   go test                       tests only; benchmarks are skipped
//   go test -bench .              tests plus every benchmark
//   go test -bench Join -benchmem allocation columns too
//   go test -cover                statement coverage
//   go test -run TestJoin/empty   one subtest
//   go test -count=1              defeat the result cache
//
// ALLOCATIONS are usually the thing to optimise. Building a string with
// `s += x` in a loop allocates a whole new string every iteration — O(n²)
// bytes copied. strings.Builder writes into one growing buffer, and Grow
// pre-sizes it so there is essentially one allocation for the lot. That
// difference is what you're going to measure.

// TODO 1: write `func JoinNaive(parts []string, sep string) string` using
// repeated string concatenation (`out += ...`) — deliberately the slow way.
// No separator before the first part or after the last.
func JoinNaive(parts []string, sep string) string {
	res := ""
	for idx, val := range parts {
		if idx == len(parts)-1 {
			res += val
			continue
		}

		res += val + sep
	}

	return res
}

// TODO 2: write `func JoinBuilder(parts []string, sep string) string` with a
// strings.Builder. Call Grow first with the exact final size so it allocates
// once. It must return exactly what JoinNaive returns, for every input.
func JoinBuilder(parts []string, sep string) string {
	builder := strings.Builder{}

	size := 0

	for _, val := range parts {
		size += len(val)
	}

	nsep := max(len(parts) - 1, 0)
	size += len(sep) * nsep

	builder.Grow(size)

	for idx, val := range parts {
		builder.WriteString(val)

		if idx != len(parts)-1 {
			builder.WriteString(sep)
		}
	}

	return builder.String()
}

// TODO 3: define
//
//	type JoinCase struct { Name string; Parts []string; Sep string; Want string }
//
// and `func JoinCases() []JoinCase` returning a TABLE of at least 5 cases.
// The checker runs your table against both implementations, so every Want
// must be right. It also insists you covered the awkward ones:
//   - no parts at all
//   - exactly one part (so: no separator anywhere)
//   - an empty separator
//   - a separator more than one character long
//   - a part that is itself the empty string
type JoinCase struct {
	Name  string
	Parts []string
	Sep   string
	Want  string
}

func JoinCases() []JoinCase {
	cases := []JoinCase{}

	cases = append(cases, JoinCase{Name: "No parts", Parts: []string{}, Sep: ",", Want: ""})
	cases = append(cases, JoinCase{Name: "One part", Parts: []string{"one"}, Sep: ",", Want: "one"})
	cases = append(cases, JoinCase{Name: "No separator", Parts: []string{"one", "two"}, Sep: "", Want: "onetwo"})
	cases = append(cases, JoinCase{Name: "Separator with more than one length", Parts: []string{"one", "two"}, Sep: "tt", Want: "onetttwo"})
	cases = append(cases, JoinCase{Name: "Empty part string", Parts: []string{"one", "", "two"}, Sep: ",", Want: "one,,two"})

	return cases
}

// TODO 4: write `func BenchmarkJoinNaive(b *testing.B)`. Build a 200-element
// []string as setup, call b.ReportAllocs() and b.ResetTimer(), then loop
// b.N times calling JoinNaive.
// (Benchmarks normally live in _test.go. They're in main.go here so the
// checker can run them for you with testing.Benchmark.)
func BenchmarkJoinNaive(b *testing.B) {
	setup := []string{}
	sep := ","
	for range 100 {
		setup = append(setup, "bruh")
	}
	b.ReportAllocs()

	for b.Loop() {
		JoinNaive(setup, sep)
	}
}

// TODO 5: write `func BenchmarkJoinBuilder(b *testing.B)`, the same but for
// JoinBuilder.
func BenchmarkJoinBuilder(b *testing.B) {
	setup := []string{}
	sep := ","
	for range 100 {
		setup = append(setup, "bruh")
	}
	b.ReportAllocs()

	for b.Loop() {
		JoinBuilder(setup, sep)
	}
}

func main() {
	// TODO 6: print JoinNaive and JoinBuilder of ["a","b","c"] with "-".
	// They must agree.
	fmt.Println(JoinNaive([]string{"a", "b", "c"}, "-"), JoinBuilder([]string{"a", "b", "c"}, "-"))

	// TODO 7: print how many cases JoinCases() returns.
	fmt.Println(len(JoinCases()))
	// TODO 8: run both benchmarks with testing.Benchmark and print whether
	// the builder version allocates fewer times per operation:
	//   naive := testing.Benchmark(BenchmarkJoinNaive)
	//   builder := testing.Benchmark(BenchmarkJoinBuilder)
	//   fmt.Println(builder.AllocsPerOp() < naive.AllocsPerOp())
	naive := testing.Benchmark(BenchmarkJoinNaive)
	builder := testing.Benchmark(BenchmarkJoinBuilder)
	fmt.Println(builder.AllocsPerOp() < naive.AllocsPerOp())
}

// EXPECTED OUTPUT:
// a-b-c a-b-c
// 5            <- or more, depending on how many cases you wrote
// true
