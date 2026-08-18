package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

type CountingWriter struct {
	W io.Writer
	N int64
}

func (c *CountingWriter) Write(p []byte) (int, error) {
	n, err := c.W.Write(p)
	c.N += int64(n) // what actually landed, not len(p)
	return n, err
}

func CountLines(r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		count++
	}
	if err := scanner.Err(); err != nil {
		return count, err
	}
	return count, nil
}

func ReadLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return lines, err
	}
	return lines, nil
}

type UpperReader struct {
	R io.Reader
}

func (u *UpperReader) Read(p []byte) (int, error) {
	n, err := u.R.Read(p)
	for i := 0; i < n; i++ { // only the n bytes actually read
		if p[i] >= 'a' && p[i] <= 'z' {
			p[i] -= 32
		}
	}
	return n, err
}

func Duplicate(src io.Reader, a, b io.Writer) (int64, error) {
	return io.Copy(io.MultiWriter(a, b), src)
}

func WriteLines(w io.Writer, lines []string) error {
	bw := bufio.NewWriter(w)
	for _, line := range lines {
		if _, err := bw.WriteString(line + "\n"); err != nil {
			return err
		}
	}
	return bw.Flush() // without this the buffer never reaches w
}

func main() {
	n, _ := CountLines(strings.NewReader("one\ntwo\nthree\n"))
	fmt.Println(n)

	var buf bytes.Buffer
	cw := &CountingWriter{W: &buf}
	fmt.Fprint(cw, "hello world")
	fmt.Println(buf.String(), cw.N)

	upper, _ := io.ReadAll(&UpperReader{R: strings.NewReader("go is fun")})
	fmt.Println(string(upper))

	var a, b bytes.Buffer
	copied, _ := Duplicate(strings.NewReader("stream me"), &a, &b)
	fmt.Println(a.String(), b.String(), copied)

	var out bytes.Buffer
	WriteLines(&out, []string{"alpha", "beta", "gamma"})
	fmt.Println(out.Len())
}
