package main

// ============================================================
// CHECKER for 23_http_client — run with:  go test -race
// ============================================================
// Every check runs against an httptest server on localhost. Nothing here
// touches the network, which is exactly why your functions take a URL
// instead of building one from a constant.

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func mustFinish(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v", what, d)
	}
}

// jsonServer replies with a fixed body and status for every request.
func jsonServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

var (
	_ func(context.Context, string) (string, error)     = FetchText
	_ func(context.Context, string) (Todo, error)       = FetchTodo
	_ func(context.Context, string, Todo) (Todo, error) = CreateTodo
	_ func(context.Context, string) (int, error)        = FetchStatus
	_ func(context.Context, []string) ([]string, error) = FetchAllTexts
	_                                                   = Todo{ID: 1, Title: "t", Done: false}
)

// --- TODO 1: the client ---------------------------------------------------

func TestClientHasATimeout(t *testing.T) {
	if client == nil {
		t.Fatal("TODO 1: `client` is nil — declare it at package level")
	}
	if client == http.DefaultClient {
		t.Error("TODO 1: `client` is http.DefaultClient, which has no timeout at all")
	}
	if client.Timeout == 0 {
		t.Error("TODO 1: client.Timeout is 0, meaning no timeout.\n" +
			"  A server that accepts the connection and then goes quiet will block\n" +
			"  that goroutine for the lifetime of the process.")
	}
	if client.Timeout > time.Minute {
		t.Errorf("TODO 1: client.Timeout is %v — that is not really a timeout", client.Timeout)
	}
}

// --- TODO 2: FetchText -----------------------------------------------------

func TestFetchText(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, `hello from the server`)

	var got string
	var err error
	mustFinish(t, 10*time.Second, "FetchText", func() {
		got, err = FetchText(context.Background(), srv.URL)
	})

	if err != nil {
		t.Fatalf("TODO 2: FetchText returned %v", err)
	}
	if got != "hello from the server" {
		t.Errorf("TODO 2: got %q, want %q", got, "hello from the server")
	}
}

func TestFetchTextTreatsBadStatusAsAnError(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError, http.StatusTeapot} {
		srv := jsonServer(t, status, `something went wrong`)

		_, err := FetchText(context.Background(), srv.URL)
		if err == nil {
			t.Errorf("TODO 2: a %d response returned nil error.\n"+
				"  client.Do only reports TRANSPORT failures. A %d means the server\n"+
				"  answered you perfectly and said no — you have to check\n"+
				"  resp.StatusCode yourself.", status, status)
			continue
		}
		if !strings.Contains(err.Error(), "404") &&
			!strings.Contains(err.Error(), "500") &&
			!strings.Contains(err.Error(), "418") {
			t.Errorf("TODO 2: error for status %d is %q — include the status code "+
				"so the caller can tell what happened", status, err)
		}
	}
}

func TestFetchTextHonoursContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
			io.WriteString(w, "too late")
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(srv.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	var err error
	mustFinish(t, 10*time.Second, "FetchText under a deadline", func() {
		_, err = FetchText(ctx, srv.URL)
	})
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("TODO 2: the context expired mid-request, so this should have failed")
	}
	if elapsed > 3*time.Second {
		t.Errorf("TODO 2: took %v for a 100ms context.\n"+
			"  Build the request with http.NewRequestWithContext — a plain\n"+
			"  http.NewRequest ignores the context entirely.", elapsed)
	}
}

func TestFetchTextClosesTheBody(t *testing.T) {
	// Keep-alive reuse only happens when the body is drained and closed. If
	// the server sees a brand new connection for every request, the bodies
	// are being leaked.
	var conns atomic.Int64

	// NewUnstartedServer, so ConnState can be set BEFORE the server begins
	// serving. Assigning to srv.Config on a running server is itself a race.
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "ok")
	}))
	srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			conns.Add(1)
		}
	}
	srv.Start()
	t.Cleanup(srv.Close)

	for i := 0; i < 8; i++ {
		if _, err := FetchText(context.Background(), srv.URL); err != nil {
			t.Fatalf("TODO 2: request %d failed: %v", i, err)
		}
	}

	if n := conns.Load(); n > 4 {
		t.Errorf("TODO 2: 8 sequential requests opened %d connections.\n"+
			"  A connection only returns to the pool once its body is read to the\n"+
			"  end and closed. `defer resp.Body.Close()` right after the error check.", n)
	}
}

// --- TODO 3: FetchTodo ------------------------------------------------------

func TestFetchTodo(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, `{"id":7,"title":"write Go","done":true}`)

	got, err := FetchTodo(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("TODO 3: FetchTodo returned %v", err)
	}
	if got.ID != 7 || got.Title != "write Go" || !got.Done {
		t.Errorf("TODO 3: got %+v, want {ID:7 Title:write Go Done:true} — check "+
			"your json tags are lowercase", got)
	}
}

func TestFetchTodoRejectsBadJSON(t *testing.T) {
	srv := jsonServer(t, http.StatusOK, `{"id": not json}`)
	if _, err := FetchTodo(context.Background(), srv.URL); err == nil {
		t.Error("TODO 3: the body was not valid JSON, so decoding should have failed")
	}
}

func TestFetchTodoChecksStatus(t *testing.T) {
	srv := jsonServer(t, http.StatusInternalServerError, `{"error":"boom"}`)
	if _, err := FetchTodo(context.Background(), srv.URL); err == nil {
		t.Error("TODO 3: a 500 should be an error even when the body parses fine")
	}
}

// --- TODO 4: CreateTodo -------------------------------------------------------

func TestCreateTodo(t *testing.T) {
	var (
		gotMethod      string
		gotContentType string
		gotBody        Todo
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&gotBody)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(Todo{ID: 99, Title: gotBody.Title, Done: gotBody.Done})
	}))
	t.Cleanup(srv.Close)

	created, err := CreateTodo(context.Background(), srv.URL, Todo{Title: "new task"})
	if err != nil {
		t.Fatalf("TODO 4: CreateTodo returned %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Errorf("TODO 4: the server saw a %s, want POST", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Errorf("TODO 4: Content-Type was %q, want %q — set it with "+
			"req.Header.Set", gotContentType, "application/json")
	}
	if gotBody.Title != "new task" {
		t.Errorf("TODO 4: the server received Title %q, want %q — the todo must be "+
			"marshalled into the request body", gotBody.Title, "new task")
	}
	if created.ID != 99 {
		t.Errorf("TODO 4: got back %+v, want the server's version with ID 99", created)
	}
}

func TestCreateTodoAcceptsBoth200And201(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusCreated} {
		srv := jsonServer(t, status, `{"id":1,"title":"t","done":false}`)
		if _, err := CreateTodo(context.Background(), srv.URL, Todo{Title: "t"}); err != nil {
			t.Errorf("TODO 4: status %d should be accepted, got %v", status, err)
		}
	}

	srv := jsonServer(t, http.StatusBadRequest, `{}`)
	if _, err := CreateTodo(context.Background(), srv.URL, Todo{Title: "t"}); err == nil {
		t.Error("TODO 4: a 400 should be an error")
	}
}

// --- TODO 5: FetchStatus --------------------------------------------------------

func TestFetchStatus(t *testing.T) {
	for _, status := range []int{200, 201, 404, 418, 500} {
		srv := jsonServer(t, status, `body`)

		got, err := FetchStatus(context.Background(), srv.URL)
		if err != nil {
			t.Errorf("TODO 5: FetchStatus for a %d returned an error (%v).\n"+
				"  Only a transport failure is an error here — that's the whole "+
				"point of this function next to FetchText.", status, err)
			continue
		}
		if got != status {
			t.Errorf("TODO 5: got status %d, want %d", got, status)
		}
	}
}

func TestFetchStatusReportsTransportFailures(t *testing.T) {
	// A server that is closed before we call it: nothing to connect to.
	srv := jsonServer(t, http.StatusOK, "ok")
	url := srv.URL
	srv.Close()

	if _, err := FetchStatus(context.Background(), url); err == nil {
		t.Error("TODO 5: connecting to a dead server should return an error")
	}
}

// --- TODO 6: FetchAllTexts --------------------------------------------------------

func TestFetchAllTextsPreservesOrder(t *testing.T) {
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(60 * time.Millisecond)
		io.WriteString(w, "slow")
	}))
	t.Cleanup(slow.Close)

	fast := jsonServer(t, http.StatusOK, "fast")

	urls := []string{slow.URL, fast.URL, slow.URL}

	var got []string
	var err error
	mustFinish(t, 15*time.Second, "FetchAllTexts", func() {
		got, err = FetchAllTexts(context.Background(), urls)
	})

	if err != nil {
		t.Fatalf("TODO 6: FetchAllTexts returned %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("TODO 6: got %d bodies for 3 urls", len(got))
	}
	if got[0] != "slow" || got[1] != "fast" || got[2] != "slow" {
		t.Errorf("TODO 6: got %q, want [slow fast slow] — results must be in the "+
			"order of `urls`, not the order they arrived", got)
	}
}

func TestFetchAllTextsIsConcurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(80 * time.Millisecond)
		io.WriteString(w, "ok")
	}))
	t.Cleanup(srv.Close)

	urls := make([]string, 8)
	for i := range urls {
		urls[i] = srv.URL
	}

	start := time.Now()
	mustFinish(t, 15*time.Second, "FetchAllTexts", func() {
		FetchAllTexts(context.Background(), urls)
	})
	elapsed := time.Since(start)

	if elapsed > 400*time.Millisecond {
		t.Errorf("TODO 6: 8 requests of 80ms took %v. Sequentially that's 640ms, "+
			"concurrently about 80ms — these aren't overlapping.", elapsed)
	}
}

func TestFetchAllTextsReportsFailures(t *testing.T) {
	good := jsonServer(t, http.StatusOK, "fine")
	bad := jsonServer(t, http.StatusInternalServerError, "boom")

	if _, err := FetchAllTexts(context.Background(), []string{good.URL, bad.URL}); err == nil {
		t.Error("TODO 6: one of the fetches failed, so FetchAllTexts should return " +
			"an error")
	}
}

func TestFetchAllTextsEmpty(t *testing.T) {
	got, err := FetchAllTexts(context.Background(), nil)
	if err != nil || len(got) != 0 {
		t.Errorf("TODO 6: with no urls got (%v, %v), want (empty, nil)", got, err)
	}
}

// --- main()'s narration -------------------------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 30*time.Second, "main()", func() { out = captureStdout(t, main) })
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 7")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`\{"id":1,"title":"write Go","done":false\}`), "TODO 8", "the raw JSON body"},
		{regexp.MustCompile(`(?m)^write Go$`), "TODO 9", "\"write Go\" from the decoded Todo"},
		{regexp.MustCompile(`(?m)^404$`), "TODO 10", "404 from FetchStatus"},
		{regexp.MustCompile(`(?m)^true$`), "TODO 10", "true — FetchText errors on a 404"},
		{regexp.MustCompile(`(?m)^3$`), "TODO 11", "3 bodies from FetchAllTexts"},
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
