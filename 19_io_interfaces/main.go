package main

import "fmt"

// ============================================================
// CONCEPT: io.Reader and io.Writer — the two interfaces that matter
// ============================================================
//
// Almost everything in Go that moves bytes speaks one of two interfaces:
//
//   type Reader interface { Read(p []byte) (n int, err error) }
//   type Writer interface { Write(p []byte) (n int, err error) }
//
// One method each. That's why a file, a network connection, an HTTP body, a
// gzip stream, an in-memory buffer and a string all plug into the same
// functions. Write your own code against io.Reader rather than *os.File and
// it works with all of them — and becomes trivially testable, because
// strings.NewReader gives you a Reader with no filesystem involved.
//
// READ's contract is the part people get wrong. Read fills as much of p as it
// can and returns how many bytes it managed:
//
//   - it may return FEWER bytes than len(p) without anything being wrong
//   - n > 0 with err == io.EOF is legal; handle the bytes, then the EOF
//   - io.EOF is not a failure, it's the normal end of the stream
//
// So never write `if n < len(p) { something is wrong }`. Loop, or let the
// standard library loop for you:
//
//   io.ReadAll(r)          // everything, into a []byte
//   io.Copy(dst, src)      // stream it across, returns bytes copied
//   io.ReadFull(r, buf)    // exactly len(buf) bytes, or an error
//
// THE USEFUL ADAPTERS, all just Readers and Writers themselves:
//
//   strings.NewReader(s)        a Reader over a string
//   bytes.NewReader(b)          a Reader over a []byte
//   var buf bytes.Buffer        a Writer AND a Reader, grows on demand
//   io.MultiWriter(a, b)        one Writer that fans out to several
//   io.TeeReader(r, w)          a Reader that copies what it reads into w
//   io.LimitReader(r, n)        stop after n bytes
//   io.Discard                  a Writer that throws everything away
//
// BUFFERING. bufio wraps a Reader or Writer to cut down on syscalls:
//
//   scanner := bufio.NewScanner(r)     // you met this in Exercise 9
//   for scanner.Scan() { line := scanner.Text() }
//   if err := scanner.Err(); err != nil { ... }   // Scan() returning false
//                                                 // means EOF *or* error
//
//   w := bufio.NewWriter(dst)
//   w.WriteString("hello")
//   w.Flush()      // WITHOUT THIS, nothing reaches dst. The single most
//                  // common bufio bug. `defer w.Flush()` is the habit —
//                  // though for real error handling you want to check it.
//
// WRAPPING. Because the interfaces are one method, a decorator is a struct
// with an embedded (or named) Reader/Writer field:
//
//   type CountingWriter struct {
//       W io.Writer
//       N int64
//   }
//   func (c *CountingWriter) Write(p []byte) (int, error) {
//       n, err := c.W.Write(p)
//       c.N += int64(n)          // count what actually landed, not len(p)
//       return n, err
//   }
//
// Your Write must return the number of bytes consumed and pass errors back
// unchanged — io.Copy treats "n < len(p) with a nil error" as
// io.ErrShortWrite and gives up.

// TODO 1: write `type CountingWriter struct { W io.Writer; N int64 }` with a
// pointer-receiver `Write` that delegates to W and adds to N. Count the bytes
// W reports, not len(p), and return W's error untouched.

// TODO 2: write `func CountLines(r io.Reader) (int, error)` using a
// bufio.Scanner. Remember to check scanner.Err() — Scan() returning false
// means "end of input OR something broke", and only Err() tells you which.

// TODO 3: write `func ReadLines(r io.Reader) ([]string, error)` returning
// every line, without its newline.

// TODO 4: write `type UpperReader struct { R io.Reader }` implementing
// io.Reader, uppercasing ASCII letters as they stream past. Uppercase only
// the n bytes you actually read, not the whole buffer.

// TODO 5: write `func Duplicate(src io.Reader, a, b io.Writer) (int64, error)`
// that streams src into BOTH writers in one pass, returning the byte count.
// io.MultiWriter turns two writers into one.

// TODO 6: write `func WriteLines(w io.Writer, lines []string) error` using a
// bufio.Writer, one line each with a trailing newline. Flush before returning
// and return the Flush error if there is one — the whole point of this TODO
// is that forgetting Flush silently writes nothing.

func main() {
	// TODO 7: count the lines in a strings.NewReader with three lines, print it.

	// TODO 8: wrap a bytes.Buffer in a CountingWriter, write "hello world"
	// through it, print the buffer contents and the count.

	// TODO 9: io.ReadAll an UpperReader over "go is fun" and print the result.

	// TODO 10: Duplicate "stream me" into two buffers, print both plus the
	// number of bytes copied.

	// TODO 11: WriteLines three lines into a buffer and print how many bytes
	// ended up there.

	fmt.Print()
}

// EXPECTED OUTPUT:
// 3
// hello world 11
// GO IS FUN
// stream me stream me 9
// 17
