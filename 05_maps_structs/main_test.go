package main

// ============================================================
// CHECKER for 05_maps_structs — run with:  go test
// ============================================================
// Your three contacts are yours to pick, so those checks look at shape.
// The word-frequency sentence is fixed by the exercise, so its counts
// are checked exactly.

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
				t.Fatalf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

// --- TODO 3: the Contact struct -------------------------------------
// This won't compile unless Contact has exactly these three string fields.
var _ = Contact{Name: "n", Phone: "p", Email: "e"}

// --- TODO 6: findContact --------------------------------------------

func TestFindContactFound(t *testing.T) {
	book := []Contact{
		{Name: "Alice", Phone: "111-1111", Email: "alice@example.com"},
		{Name: "Bob", Phone: "222-2222", Email: "bob@example.com"},
		{Name: "Carol", Phone: "333-3333", Email: "carol@example.com"},
	}

	c, ok := findContact(book, "Bob")
	if !ok {
		t.Fatal("TODO 6: findContact(book, \"Bob\") reported not-found, but Bob is in the book")
	}
	if c.Name != "Bob" || c.Phone != "222-2222" || c.Email != "bob@example.com" {
		t.Errorf("TODO 6: findContact returned the wrong contact: %+v", c)
	}
}

func TestFindContactNotFound(t *testing.T) {
	book := []Contact{
		{Name: "Alice", Phone: "111-1111", Email: "alice@example.com"},
	}

	c, ok := findContact(book, "Dave")
	if ok {
		t.Fatal("TODO 6: findContact(book, \"Dave\") reported found, but Dave is not in the book")
	}
	if c != (Contact{}) {
		t.Errorf("TODO 6: on a miss, return the zero value Contact{} — got %+v", c)
	}
}

func TestFindContactEmptyBook(t *testing.T) {
	if _, ok := findContact(nil, "Anyone"); ok {
		t.Error("TODO 6: searching an empty/nil book should always report not-found")
	}
	if _, ok := findContact([]Contact{}, "Anyone"); ok {
		t.Error("TODO 6: searching an empty/nil book should always report not-found")
	}
}

func TestFindContactIsExactMatch(t *testing.T) {
	book := []Contact{{Name: "Alice", Phone: "111-1111", Email: "a@example.com"}}
	if _, ok := findContact(book, "Ali"); ok {
		t.Error("TODO 6: findContact should match the full Name, not a prefix")
	}
}

// --- TODOs 1, 2, 5: checked through main()'s output -----------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 1")
	}

	// TODO 1: word frequencies for
	// "the quick brown fox jumps over the lazy dog the fox runs"
	want := map[string]string{
		"the": "3", "quick": "1", "brown": "1", "fox": "2", "jumps": "1",
		"over": "1", "lazy": "1", "dog": "1", "runs": "1",
	}
	for word, count := range want {
		re := regexp.MustCompile(`(?m)^\s*` + word + `:\s*` + count + `\s*$`)
		if !re.MatchString(out) {
			t.Errorf("TODO 1: expected a line %q (order doesn't matter — map "+
				"iteration order is randomised on purpose)", word+": "+count)
		}
	}
	// A word that isn't in the sentence must not appear as a count line.
	if regexp.MustCompile(`(?m)^\s*cat:\s*\d`).MatchString(out) {
		t.Error("TODO 1: \"cat\" is not in the sentence, it should not have a count")
	}

	// TODO 2: comma-ok lookups
	if !regexp.MustCompile(`fox appears 2 times`).MatchString(out) {
		t.Error("TODO 2: expected the line \"fox appears 2 times\"")
	}
	if !regexp.MustCompile(`cat not found`).MatchString(out) {
		t.Error("TODO 2: expected the line \"cat not found\" — this is the whole point " +
			"of comma-ok: freq[\"cat\"] returns 0, which you cannot distinguish from a " +
			"real count of 0 without the second return value")
	}

	// TODOs 4 & 5: three contacts printed in the required shape
	contactLine := regexp.MustCompile(`(?m)^Name: \S.*, Phone: \S.*$`)
	if n := len(contactLine.FindAllString(out, -1)); n < 3 {
		t.Errorf("TODO 4/5: expected 3 lines shaped like \"Name: <name>, Phone: <phone>\", "+
			"found %d", n)
	}
}
