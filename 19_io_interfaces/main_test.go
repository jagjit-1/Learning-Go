package main

// ============================================================
// CHECKER for 19_io_interfaces — run with:  go test
// ============================================================

import (
	"bytes"
	"errors"
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
				t.Errorf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

// --- test doubles ------------------------------------------------------

var errBroken = errors.New("device on fire")

// errWriter accepts `ok` bytes in total, then fails.
type errWriter struct {
	ok      int
	written int
}

func (w *errWriter) Write(p []byte) (int, error) {
	if w.written >= w.ok {
		return 0, errBroken
	}
	n := len(p)
	if w.written+n > w.ok {
		n = w.ok - w.written
	}
	w.written += n
	if n < len(p) {
		return n, errBroken
	}
	return n, nil
}

// errReader yields its data, then fails instead of reaching EOF.
type errReader struct {
	data string
	pos  int
}

func (r *errReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errBroken
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

// oneByteReader hands over a single byte per Read, which is legal and which
// a Reader implementation must cope with.
type oneByteReader struct {
	data string
	pos  int
}

func (r *oneByteReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	if len(p) == 0 {
		return 0, nil
	}
	p[0] = r.data[r.pos]
	r.pos++
	return 1, nil
}

var (
	_ io.Writer                                            = (*CountingWriter)(nil)
	_ io.Reader                                            = (*UpperReader)(nil)
	_ func(io.Reader) (int, error)                         = CountLines
	_ func(io.Reader) ([]string, error)                    = ReadLines
	_ func(io.Reader, io.Writer, io.Writer) (int64, error) = Duplicate
	_ func(io.Writer, []string) error                      = WriteLines
)

// --- TODO 1: CountingWriter --------------------------------------------

func TestCountingWriter(t *testing.T) {
	var buf bytes.Buffer
	cw := &CountingWriter{W: &buf}

	n, err := cw.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("TODO 1: Write returned (%d, %v), want (5, nil)", n, err)
	}
	if _, err := cw.Write([]byte(" world")); err != nil {
		t.Fatalf("TODO 1: second Write returned %v", err)
	}

	if buf.String() != "hello world" {
		t.Errorf("TODO 1: the wrapped writer holds %q, want %q — CountingWriter "+
			"must pass the bytes through, not just count them", buf.String(), "hello world")
	}
	if cw.N != 11 {
		t.Errorf("TODO 1: N = %d, want 11", cw.N)
	}
}

func TestCountingWriterPropagatesErrors(t *testing.T) {
	cw := &CountingWriter{W: &errWriter{ok: 4}}

	n, err := cw.Write([]byte("abcdefgh"))
	if err == nil {
		t.Fatal("TODO 1: the underlying writer failed, but Write returned nil — " +
			"return the wrapped writer's error unchanged")
	}
	if n != 4 {
		t.Errorf("TODO 1: Write returned n = %d, want 4 (what the underlying "+
			"writer actually accepted)", n)
	}
	if cw.N != 4 {
		t.Errorf("TODO 1: N = %d, want 4 — count the bytes the wrapped writer "+
			"reported, not len(p)", cw.N)
	}
}

func TestCountingWriterWorksWithIoCopy(t *testing.T) {
	var buf bytes.Buffer
	cw := &CountingWriter{W: &buf}

	if _, err := io.Copy(cw, strings.NewReader("copied through")); err != nil {
		t.Fatalf("TODO 1: io.Copy into a CountingWriter failed: %v", err)
	}
	if cw.N != 14 || buf.String() != "copied through" {
		t.Errorf("TODO 1: after io.Copy, N = %d and buffer = %q", cw.N, buf.String())
	}
}

// --- TODOs 2 & 3: CountLines and ReadLines ------------------------------

func TestCountLines(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"one\ntwo\nthree\n", 3},
		{"one\ntwo\nthree", 3}, // no trailing newline
		{"single", 1},
		{"", 0},
		{"\n\n", 2}, // two empty lines
	}
	for _, c := range cases {
		got, err := CountLines(strings.NewReader(c.in))
		if err != nil {
			t.Errorf("TODO 2: CountLines(%q) returned %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("TODO 2: CountLines(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestCountLinesReportsReaderErrors(t *testing.T) {
	_, err := CountLines(&errReader{data: "a\nb\n"})
	if err == nil {
		t.Error("TODO 2: the reader failed, but CountLines returned nil.\n" +
			"  scanner.Scan() returns false for BOTH end-of-input and failure.\n" +
			"  Only scanner.Err() distinguishes them, so you must check it.")
	}
}

func TestReadLines(t *testing.T) {
	got, err := ReadLines(strings.NewReader("alpha\nbeta\ngamma"))
	if err != nil {
		t.Fatalf("TODO 3: ReadLines returned %v", err)
	}
	want := []string{"alpha", "beta", "gamma"}
	if len(got) != len(want) {
		t.Fatalf("TODO 3: ReadLines = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 3: ReadLines = %q, want %q (newlines must be stripped — "+
				"scanner.Text() does that for you)", got, want)
		}
	}

	if got, _ := ReadLines(strings.NewReader("")); len(got) != 0 {
		t.Errorf("TODO 3: ReadLines(\"\") = %q, want empty", got)
	}
}

// --- TODO 4: UpperReader -----------------------------------------------

func TestUpperReader(t *testing.T) {
	got, err := io.ReadAll(&UpperReader{R: strings.NewReader("go is fun")})
	if err != nil {
		t.Fatalf("TODO 4: io.ReadAll returned %v", err)
	}
	if string(got) != "GO IS FUN" {
		t.Errorf("TODO 4: got %q, want %q", got, "GO IS FUN")
	}
}

func TestUpperReaderHandlesTinyReads(t *testing.T) {
	// The underlying reader hands over one byte at a time. An implementation
	// that assumes a full buffer, or uppercases all of p rather than p[:n],
	// breaks here.
	got, err := io.ReadAll(&UpperReader{R: &oneByteReader{data: "abc def"}})
	if err != nil {
		t.Fatalf("TODO 4: io.ReadAll returned %v", err)
	}
	if string(got) != "ABC DEF" {
		t.Errorf("TODO 4: with a one-byte-at-a-time source, got %q, want %q.\n"+
			"  Read may return fewer bytes than len(p) with no error at all — "+
			"only touch p[:n].", got, "ABC DEF")
	}
}

func TestUpperReaderLeavesNonLettersAlone(t *testing.T) {
	got, _ := io.ReadAll(&UpperReader{R: strings.NewReader("go1 -- OK!")})
	if string(got) != "GO1 -- OK!" {
		t.Errorf("TODO 4: got %q, want %q", got, "GO1 -- OK!")
	}
}

// --- TODO 5: Duplicate --------------------------------------------------

func TestDuplicate(t *testing.T) {
	var a, b bytes.Buffer

	n, err := Duplicate(strings.NewReader("stream me"), &a, &b)
	if err != nil {
		t.Fatalf("TODO 5: Duplicate returned %v", err)
	}
	if n != 9 {
		t.Errorf("TODO 5: copied %d bytes, want 9", n)
	}
	if a.String() != "stream me" || b.String() != "stream me" {
		t.Errorf("TODO 5: a = %q, b = %q — both writers must receive everything",
			a.String(), b.String())
	}
}

func TestDuplicateEmpty(t *testing.T) {
	var a, b bytes.Buffer
	n, err := Duplicate(strings.NewReader(""), &a, &b)
	if err != nil || n != 0 {
		t.Errorf("TODO 5: on empty input got (%d, %v), want (0, nil)", n, err)
	}
}

func TestDuplicateReportsWriterErrors(t *testing.T) {
	var good bytes.Buffer
	_, err := Duplicate(strings.NewReader("a fairly long stream of bytes"),
		&good, &errWriter{ok: 2})
	if err == nil {
		t.Error("TODO 5: one of the writers failed, but Duplicate returned nil — " +
			"io.MultiWriter stops and reports the first error, so just pass it on")
	}
}

// --- TODO 6: WriteLines -------------------------------------------------

func TestWriteLines(t *testing.T) {
	var buf bytes.Buffer

	if err := WriteLines(&buf, []string{"alpha", "beta", "gamma"}); err != nil {
		t.Fatalf("TODO 6: WriteLines returned %v", err)
	}

	want := "alpha\nbeta\ngamma\n"
	if buf.String() != want {
		if buf.Len() == 0 {
			t.Fatalf("TODO 6: nothing was written at all.\n" +
				"  That's the bufio.Writer trap: it holds bytes in memory until you\n" +
				"  Flush. Without the Flush, the data dies with the buffer.")
		}
		t.Errorf("TODO 6: got %q, want %q", buf.String(), want)
	}
	if buf.Len() != 17 {
		t.Errorf("TODO 6: wrote %d bytes, want 17", buf.Len())
	}
}

func TestWriteLinesEmpty(t *testing.T) {
	var buf bytes.Buffer
	if err := WriteLines(&buf, nil); err != nil {
		t.Errorf("TODO 6: WriteLines with no lines returned %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("TODO 6: wrote %q for an empty list, want nothing", buf.String())
	}
}

func TestWriteLinesReportsErrors(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = strings.Repeat("x", 100)
	}
	if err := WriteLines(&errWriter{ok: 10}, lines); err == nil {
		t.Error("TODO 6: the writer failed, but WriteLines returned nil — return " +
			"the error from WriteString or from Flush")
	}
}

// --- main()'s narration -------------------------------------------------

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
		{regexp.MustCompile(`(?m)^3$`), "TODO 7", "3 lines counted"},
		{regexp.MustCompile(`(?m)^hello world 11$`), "TODO 8", "\"hello world 11\""},
		{regexp.MustCompile(`(?m)^GO IS FUN$`), "TODO 9", "GO IS FUN"},
		{regexp.MustCompile(`(?m)^stream me stream me 9$`), "TODO 10", "\"stream me stream me 9\""},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s.\n  your output was:\n%s", c.todo, c.want, indent(out))
		}
	}

	// TODO 11's byte count depends on whatever lines you chose to write, so
	// there's no fixed number to match — just check the last line printed is
	// a positive integer (i.e. something actually got written).
	trimmed := strings.Split(strings.TrimRight(out, "\n"), "\n")
	last := trimmed[len(trimmed)-1]
	if n, err := strconv.Atoi(strings.TrimSpace(last)); err != nil || n <= 0 {
		t.Errorf("TODO 11: expected the last printed line to be a positive byte count, got %q.\n"+
			"  your output was:\n%s", last, indent(out))
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
