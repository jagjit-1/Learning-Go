package main

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func JoinNaive(parts []string, sep string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += sep // a brand new string every iteration
		}
		out += p
	}
	return out
}

func JoinBuilder(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}

	size := len(sep) * (len(parts) - 1)
	for _, p := range parts {
		size += len(p)
	}

	var b strings.Builder
	b.Grow(size) // one allocation for the whole result
	for i, p := range parts {
		if i > 0 {
			b.WriteString(sep)
		}
		b.WriteString(p)
	}
	return b.String()
}

type JoinCase struct {
	Name  string
	Parts []string
	Sep   string
	Want  string
}

func JoinCases() []JoinCase {
	return []JoinCase{
		{"no parts", nil, "-", ""},
		{"single part", []string{"a"}, "-", "a"},
		{"several parts", []string{"a", "b", "c"}, "-", "a-b-c"},
		{"empty separator", []string{"a", "b", "c"}, "", "abc"},
		{"multi-char separator", []string{"a", "b"}, " -> ", "a -> b"},
		{"empty part", []string{"a", "", "c"}, "-", "a--c"},
	}
}

func benchParts(n int) []string {
	parts := make([]string, n)
	for i := range parts {
		parts[i] = "part-" + strconv.Itoa(i)
	}
	return parts
}

func BenchmarkJoinNaive(b *testing.B) {
	parts := benchParts(200)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JoinNaive(parts, "-")
	}
}

func BenchmarkJoinBuilder(b *testing.B) {
	parts := benchParts(200)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JoinBuilder(parts, "-")
	}
}

func main() {
	parts := []string{"a", "b", "c"}
	fmt.Println(JoinNaive(parts, "-"), JoinBuilder(parts, "-"))

	fmt.Println(len(JoinCases()))

	naive := testing.Benchmark(BenchmarkJoinNaive)
	builder := testing.Benchmark(BenchmarkJoinBuilder)
	fmt.Println(builder.AllocsPerOp() < naive.AllocsPerOp())
}
