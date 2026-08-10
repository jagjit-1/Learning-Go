package main

// ============================================================
// CHECKER for 07_interfaces — run with:  go test
// ============================================================
// The dimensions you use in main() are yours to pick, so the geometry is
// checked against shapes this file builds itself. main()'s output is only
// checked for structure — plus one real cross-check: your "Total area"
// must actually equal the areas you printed.

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

func approx(got, want float64) bool { return math.Abs(got-want) < 1e-9 }

// --- TODOs 1-5: the interface and the three implementations ---------
// These declarations fail to COMPILE if a type is missing a method — the
// same safety net the concept block describes.
var (
	_ Shape = Circle{}
	_ Shape = Rectangle{}
	_ Shape = Triangle{}
)

func TestCircle(t *testing.T) {
	c := Circle{Radius: 5}
	if got := c.Area(); !approx(got, math.Pi*25) {
		t.Errorf("TODO 2: Circle{5}.Area() = %v, want %v (math.Pi * r * r)", got, math.Pi*25)
	}
	if got := c.Perimeter(); !approx(got, 2*math.Pi*5) {
		t.Errorf("TODO 2: Circle{5}.Perimeter() = %v, want %v (2 * math.Pi * r)", got, 2*math.Pi*5)
	}

	unit := Circle{Radius: 1}
	if got := unit.Area(); !approx(got, math.Pi) {
		t.Errorf("TODO 2: Circle{1}.Area() = %v, want %v", got, math.Pi)
	}
}

func TestRectangle(t *testing.T) {
	r := Rectangle{Width: 6, Height: 4}
	if got := r.Area(); !approx(got, 24) {
		t.Errorf("TODO 3: Rectangle{6,4}.Area() = %v, want 24", got)
	}
	if got := r.Perimeter(); !approx(got, 20) {
		t.Errorf("TODO 3: Rectangle{6,4}.Perimeter() = %v, want 20 (2*(w+h))", got)
	}

	sq := Rectangle{Width: 3, Height: 3}
	if got := sq.Area(); !approx(got, 9) {
		t.Errorf("TODO 3: Rectangle{3,3}.Area() = %v, want 9", got)
	}
}

func TestTriangle(t *testing.T) {
	tri := Triangle{Base: 10, Height: 4, SideA: 3, SideB: 4, SideC: 5}
	if got := tri.Area(); !approx(got, 20) {
		t.Errorf("TODO 4: Triangle{Base:10, Height:4}.Area() = %v, want 20 "+
			"(0.5 * base * height)", got)
	}
	if got := tri.Perimeter(); !approx(got, 12) {
		t.Errorf("TODO 4: Triangle{SideA:3, SideB:4, SideC:5}.Perimeter() = %v, "+
			"want 12 (a+b+c — NOT base+height)", got)
	}
}

// --- TODO 6: Describe -------------------------------------------------

func TestDescribeFormat(t *testing.T) {
	got := captureStdout(t, func() { Describe(Circle{Radius: 5}) })
	want := "Area: 78.54, Perimeter: 31.42\n"
	if got != want {
		t.Errorf("TODO 6: Describe(Circle{5}) printed %q, want %q\n"+
			"  hint: fmt.Printf(\"Area: %%.2f, Perimeter: %%.2f\\n\", ...)", got, want)
	}
}

func TestDescribeAcceptsTheInterface(t *testing.T) {
	// If Describe took a concrete type this would not compile.
	var s Shape = Rectangle{Width: 2, Height: 3}
	got := captureStdout(t, func() { Describe(s) })
	if got != "Area: 6.00, Perimeter: 10.00\n" {
		t.Errorf("TODO 6: Describe(Rectangle{2,3}) printed %q, want %q",
			got, "Area: 6.00, Perimeter: 10.00\n")
	}
}

// --- TODO 8: TotalArea ------------------------------------------------

func TestTotalArea(t *testing.T) {
	shapes := []Shape{
		Circle{Radius: 1},
		Rectangle{Width: 2, Height: 3},
		Triangle{Base: 4, Height: 5, SideA: 1, SideB: 1, SideC: 1},
	}
	want := math.Pi + 6 + 10
	if got := TotalArea(shapes); !approx(got, want) {
		t.Errorf("TODO 8: TotalArea = %v, want %v", got, want)
	}

	if got := TotalArea(nil); !approx(got, 0) {
		t.Errorf("TODO 8: TotalArea of an empty slice = %v, want 0", got)
	}
}

// --- TODO 9: WhatShape ------------------------------------------------

func TestWhatShape(t *testing.T) {
	cases := []struct {
		shape Shape
		word  string
	}{
		{Circle{Radius: 5}, "circle"},
		{Rectangle{Width: 2, Height: 3}, "rectangle"},
		{Triangle{Base: 1, Height: 1, SideA: 1, SideB: 1, SideC: 1}, "triangle"},
	}
	for _, c := range cases {
		out := strings.ToLower(captureStdout(t, func() { WhatShape(c.shape) }))
		if !strings.Contains(out, c.word) {
			t.Errorf("TODO 9: WhatShape(%T) printed %q, expected it to mention %q",
				c.shape, strings.TrimSpace(out), c.word)
		}
	}

	// The radius should come from the type switch's bound variable.
	out := captureStdout(t, func() { WhatShape(Circle{Radius: 5}) })
	if !strings.Contains(out, "5") {
		t.Errorf("TODO 9: the Circle case should report the radius (v.Radius), got %q",
			strings.TrimSpace(out))
	}
}

// --- main()'s narration -----------------------------------------------

var describeLine = regexp.MustCompile(`(?m)^Area: (\d+\.\d{2}), Perimeter: \d+\.\d{2}$`)

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — see TODO 6")
	}

	matches := describeLine.FindAllStringSubmatch(out, -1)
	if len(matches) < 6 {
		t.Fatalf("TODO 6/7: expected at least 6 \"Area: ..., Perimeter: ...\" lines "+
			"(one per shape directly, then one per shape via the []Shape slice) — found %d",
			len(matches))
	}

	// TODO 8: the total must match the three areas printed by the slice loop.
	totalRe := regexp.MustCompile(`(?i)total area:?\s*(\d+\.?\d*)`)
	tm := totalRe.FindStringSubmatch(out)
	if tm == nil {
		t.Fatal("TODO 8: expected a line like \"Total area: 114.54\"")
	}
	reported, err := strconv.ParseFloat(tm[1], 64)
	if err != nil {
		t.Fatalf("TODO 8: could not read the total from %q", tm[0])
	}

	last3 := matches[len(matches)-3:]
	sum := 0.0
	for _, m := range last3 {
		v, err := strconv.ParseFloat(m[1], 64)
		if err != nil {
			t.Fatalf("could not read an area from %q", m[0])
		}
		sum += v
	}
	if math.Abs(reported-sum) > 0.02 {
		t.Errorf("TODO 8: reported total area %v, but the three shapes you described "+
			"add up to %v", reported, sum)
	}

	// TODO 9: all three type-switch branches were exercised.
	lower := strings.ToLower(out)
	for _, word := range []string{"circle", "rectangle", "triangle"} {
		if !strings.Contains(lower, word) {
			t.Errorf("TODO 9: expected WhatShape to be called on a %s too", word)
		}
	}
	if strings.Contains(lower, "unknown shape") {
		t.Error("TODO 9: something hit the default branch — every shape in your " +
			"slice should match a case")
	}
}
