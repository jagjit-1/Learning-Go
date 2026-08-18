package main

// ============================================================
// CHECKER for 21_embedding — run with:  go test
// ============================================================
// The test that matters most is TestNoVirtualDispatch. If it fails because
// you got the "dog" version back, your Go is behaving like C# — which it
// doesn't, and the difference is the whole exercise.

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
	// Interface satisfaction, checked at compile time.
	_ Entity                         = Animal{}
	_ Entity                         = Dog{}
	_ Namer                          = Dog{}
	_ Describer                      = Dog{}
	_ io.Writer                      = (*LoggingWriter)(nil)
	_ func(Entity) string            = Explain
	_ func(Wrapper) (string, int)    = IDs
	_ func(Employee) (string, error) = EmployeeJSON
)

func testDog() Dog {
	return Dog{Animal: Animal{Name: "Rex"}, Breed: "Lab"}
}

// --- TODO 1: Animal ------------------------------------------------------

func TestAnimal(t *testing.T) {
	a := Animal{Name: "Rex"}
	if got := a.Describe(); got != "animal Rex" {
		t.Errorf("TODO 1: Animal.Describe() = %q, want %q", got, "animal Rex")
	}
	if got := a.GetName(); got != "Rex" {
		t.Errorf("TODO 1: Animal.GetName() = %q, want %q", got, "Rex")
	}
	if got := a.Intro(); got != "I am animal Rex" {
		t.Errorf("TODO 1: Animal.Intro() = %q, want %q", got, "I am animal Rex")
	}
}

// --- TODO 2: promotion and shadowing --------------------------------------

func TestFieldPromotion(t *testing.T) {
	d := testDog()
	if d.Name != "Rex" {
		t.Errorf("TODO 2: dog.Name = %q, want %q — an embedded struct's fields "+
			"are reachable directly on the outer type", d.Name, "Rex")
	}
	if d.Animal.Name != "Rex" {
		t.Errorf("TODO 2: dog.Animal.Name = %q, want %q", d.Animal.Name, "Rex")
	}
}

func TestMethodPromotion(t *testing.T) {
	d := testDog()
	// GetName is not defined on Dog, so this can only be the promoted one.
	if got := d.GetName(); got != "Rex" {
		t.Errorf("TODO 2: dog.GetName() = %q, want %q (promoted from Animal)", got, "Rex")
	}
}

func TestShadowing(t *testing.T) {
	d := testDog()
	if got := d.Describe(); got != "dog Rex (Lab)" {
		t.Errorf("TODO 2: dog.Describe() = %q, want %q — Dog's own method must "+
			"shadow the promoted one", got, "dog Rex (Lab)")
	}
	if got := d.Animal.Describe(); got != "animal Rex" {
		t.Errorf("TODO 2: dog.Animal.Describe() = %q, want %q — the shadowed method "+
			"is still reachable through the embedded field", got, "animal Rex")
	}
}

// --- The lesson -----------------------------------------------------------

func TestNoVirtualDispatch(t *testing.T) {
	d := testDog()

	got := d.Intro()
	if got == "I am dog Rex (Lab)" {
		t.Fatalf("TODO 1/2: dog.Intro() = %q.\n"+
			"  That is what a language with virtual methods would give you, and it\n"+
			"  is not what Go does. Intro has an ANIMAL receiver; inside it, `a` is\n"+
			"  a copy of the embedded Animal and knows nothing about the Dog around\n"+
			"  it, so a.Describe() can only be Animal's.\n"+
			"  Check Intro is declared as `func (a Animal) Intro() string` and calls\n"+
			"  a.Describe().", got)
	}
	if got != "I am animal Rex" {
		t.Errorf("TODO 1/2: dog.Intro() = %q, want %q", got, "I am animal Rex")
	}
}

// --- TODO 3: interface embedding ------------------------------------------

func TestExplainUsesTheDynamicType(t *testing.T) {
	got := Explain(testDog())
	want := "Rex: dog Rex (Lab)"
	if got != want {
		if got == "Rex: animal Rex" {
			t.Fatalf("TODO 3: Explain(dog) = %q, want %q.\n"+
				"  Through an interface you DO get the dynamic type's method. If you\n"+
				"  see the animal version, Explain is probably taking an Animal (or a\n"+
				"  Dog's embedded Animal) rather than the Entity interface.", got, want)
		}
		t.Errorf("TODO 3: Explain(dog) = %q, want %q", got, want)
	}
}

func TestExplainWorksForAnimalToo(t *testing.T) {
	if got := Explain(Animal{Name: "Generic"}); got != "Generic: animal Generic" {
		t.Errorf("TODO 3: Explain(animal) = %q, want %q", got, "Generic: animal Generic")
	}
}

// --- TODO 4: LoggingWriter -------------------------------------------------

func TestLoggingWriter(t *testing.T) {
	var buf bytes.Buffer
	lw := &LoggingWriter{Writer: &buf}

	if _, err := io.WriteString(lw, "hello "); err != nil {
		t.Fatalf("TODO 4: Write returned %v", err)
	}
	if _, err := io.WriteString(lw, "world"); err != nil {
		t.Fatalf("TODO 4: Write returned %v", err)
	}

	if buf.String() != "hello world" {
		t.Errorf("TODO 4: the wrapped writer holds %q, want %q — after recording, "+
			"delegate to the embedded Writer", buf.String(), "hello world")
	}
	if len(lw.Records) != 2 {
		t.Fatalf("TODO 4: Records has %d entries, want 2", len(lw.Records))
	}
	if lw.Records[0] != "hello " || lw.Records[1] != "world" {
		t.Errorf("TODO 4: Records = %q, want [\"hello \" \"world\"]", lw.Records)
	}
}

func TestLoggingWriterSatisfiesWriterByEmbedding(t *testing.T) {
	// io.Copy only needs io.Writer; embedding the interface is what makes
	// LoggingWriter usable anywhere a Writer is wanted.
	var buf bytes.Buffer
	lw := &LoggingWriter{Writer: &buf}
	if _, err := io.Copy(lw, strings.NewReader("streamed")); err != nil {
		t.Fatalf("TODO 4: io.Copy into a LoggingWriter failed: %v", err)
	}
	if buf.String() != "streamed" {
		t.Errorf("TODO 4: buffer = %q, want %q", buf.String(), "streamed")
	}
}

// --- TODO 5: field shadowing ------------------------------------------------

func TestIDs(t *testing.T) {
	outer, inner := IDs(Wrapper{Base: Base{ID: 7}, ID: "outer"})
	if outer != "outer" {
		t.Errorf("TODO 5: outer ID = %q, want %q — the shallower field wins for "+
			"the bare name w.ID", outer, "outer")
	}
	if inner != 7 {
		t.Errorf("TODO 5: inner ID = %d, want 7 — reach it as w.Base.ID", inner)
	}
}

// --- TODO 6: embedding and JSON ----------------------------------------------

func TestEmployeeJSONIsFlattened(t *testing.T) {
	got, err := EmployeeJSON(Employee{
		Person: Person{Name: "Ada", Email: "ada@example.com"},
		Salary: 100,
	})
	if err != nil {
		t.Fatalf("TODO 6: EmployeeJSON returned %v", err)
	}

	want := `{"name":"Ada","email":"ada@example.com","salary":100}`
	if got != want {
		if strings.Contains(got, `"Person"`) {
			t.Fatalf("TODO 6: got\n  %s\nwant\n  %s\n"+
				"  A nested \"Person\" object means Person is a NAMED field rather than "+
				"embedded. Drop the field name.", got, want)
		}
		t.Errorf("TODO 6: got\n  %s\nwant\n  %s", got, want)
	}
}

// --- main()'s narration -------------------------------------------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 7")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^Rex dog Rex \(Lab\)$`), "TODO 7", "\"Rex dog Rex (Lab)\""},
		{regexp.MustCompile(`(?m)^I am animal Rex$`), "TODO 8", "\"I am animal Rex\" from Intro()"},
		{regexp.MustCompile(`(?m)^animal Rex$`), "TODO 8", "\"animal Rex\" from dog.Animal.Describe()"},
		{regexp.MustCompile(`(?m)^Rex: dog Rex \(Lab\)$`), "TODO 9", "\"Rex: dog Rex (Lab)\" from Explain"},
		{regexp.MustCompile(`(?m)^hello world 2$`), "TODO 10", "\"hello world 2\""},
		{regexp.MustCompile(`(?m)^outer 7$`), "TODO 11", "\"outer 7\""},
		{regexp.MustCompile(`\{"name":"[^"]+","email":"[^"]+","salary":\d+\}`), "TODO 12",
			"flattened employee JSON"},
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
