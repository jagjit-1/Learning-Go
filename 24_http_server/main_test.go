package main

// ============================================================
// CHECKER for 24_http_server — run with:  go test -race
// ============================================================
// Individual handlers are exercised with httptest.NewRecorder; the router
// goes behind a real httptest.NewServer so the mux's own 404 and 405
// behaviour gets checked too.

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"sync"
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

// recorder collects what the middleware reports, safely across goroutines.
type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) record(method, path string, status int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, method+" "+path+" "+http.StatusText(status))
}

func (r *recorder) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

func (r *recorder) all() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.calls...)
}

var (
	_ http.HandlerFunc                                           = HealthHandler
	_ http.HandlerFunc                                           = InfoHandler
	_ http.HandlerFunc                                           = GreetHandler
	_ http.HandlerFunc                                           = EchoHandler
	_ http.ResponseWriter                                        = (*StatusRecorder)(nil)
	_ func(http.Handler, func(string, string, int)) http.Handler = RecordingMiddleware
	_ func(func(string, string, int)) http.Handler               = NewRouter
	_                                                            = Info{Service: "s", Version: "v"}
	_                                                            = EchoRequest{Message: "m"}
	_                                                            = EchoResponse{Echo: "e"}
)

// --- TODO 1: HealthHandler ------------------------------------------------

func TestHealthHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	HealthHandler(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("TODO 1: status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Errorf("TODO 1: body = %q, want %q", got, "ok")
	}
}

// --- TODO 2: InfoHandler --------------------------------------------------

func TestInfoHandler(t *testing.T) {
	rec := httptest.NewRecorder()
	InfoHandler(rec, httptest.NewRequest(http.MethodGet, "/info", nil))

	if rec.Code != http.StatusOK {
		t.Errorf("TODO 2: status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("TODO 2: Content-Type = %q, want application/json.\n"+
			"  If you set the header AFTER writing the body it is silently dropped — "+
			"the first Write flushes the header block.", ct)
	}

	var got Info
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("TODO 2: body %q is not valid JSON: %v", rec.Body.String(), err)
	}
	if got.Service != "learning-go" || got.Version != "1.0" {
		t.Errorf("TODO 2: got %+v, want {learning-go 1.0}", got)
	}
	if !strings.Contains(rec.Body.String(), `"service"`) {
		t.Errorf("TODO 2: body %q — check your json tags are lowercase", rec.Body.String())
	}
}

// --- TODO 3: GreetHandler -------------------------------------------------

func TestGreetHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/greet/Jagjit", nil)
	req.SetPathValue("name", "Jagjit")

	rec := httptest.NewRecorder()
	GreetHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("TODO 3: status = %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "Hello, Jagjit!" {
		t.Errorf("TODO 3: body = %q, want %q.\n"+
			"  Read the wildcard with r.PathValue(\"name\") — no string slicing needed.",
			got, "Hello, Jagjit!")
	}
}

func TestGreetHandlerRejectsAnEmptyName(t *testing.T) {
	rec := httptest.NewRecorder()
	GreetHandler(rec, httptest.NewRequest(http.MethodGet, "/greet/", nil))

	if rec.Code != http.StatusBadRequest {
		t.Errorf("TODO 3: with no name, status = %d, want 400", rec.Code)
	}
}

// --- TODO 4: EchoHandler --------------------------------------------------

func TestEchoHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(`{"message":"hi"}`))
	rec := httptest.NewRecorder()
	EchoHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("TODO 4: status = %d, want 200 (body: %s)", rec.Code, rec.Body.String())
	}

	var got EchoResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("TODO 4: body %q is not valid JSON: %v", rec.Body.String(), err)
	}
	if got.Echo != "hi" {
		t.Errorf("TODO 4: got %+v, want Echo == %q", got, "hi")
	}
}

func TestEchoHandlerRejectsBadInput(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"malformed JSON", `{"message":`},
		{"not JSON at all", `hello`},
		{"empty message", `{"message":""}`},
		{"missing message", `{}`},
		{"empty body", ``},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/echo", strings.NewReader(c.body))
			rec := httptest.NewRecorder()
			EchoHandler(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf("TODO 4: body %q gave status %d, want 400", c.body, rec.Code)
			}
			if strings.Contains(rec.Body.String(), `"echo"`) {
				t.Errorf("TODO 4: an echo response was written after the error.\n" +
					"  http.Error does not stop the handler — you must `return` " +
					"straight after calling it.")
			}
		})
	}
}

// --- TODOs 5 & 6: StatusRecorder and the middleware -------------------------

func TestStatusRecorderDefaultsTo200(t *testing.T) {
	rec := &recorder{}

	// A handler that only calls Write never calls WriteHeader at all.
	h := RecordingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("no explicit status"))
	}), rec.record)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))

	if got := rec.all(); len(got) != 1 || !strings.Contains(got[0], "OK") {
		t.Errorf("TODO 6: recorded %v, want a single 200.\n"+
			"  Initialise StatusRecorder.Status to http.StatusOK — a handler that\n"+
			"  writes a body without calling WriteHeader still produces a 200, and\n"+
			"  a zero Status is not a real status code.", got)
	}
	if w.Code != http.StatusOK || w.Body.String() != "no explicit status" {
		t.Errorf("TODO 5: the wrapped writer must still deliver the response: "+
			"status %d, body %q", w.Code, w.Body.String())
	}
}

func TestStatusRecorderCapturesExplicitCodes(t *testing.T) {
	rec := &recorder{}
	h := RecordingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusTeapot)
	}), rec.record)

	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/tea", nil))

	got := rec.all()
	if len(got) != 1 || !strings.Contains(got[0], http.StatusText(http.StatusTeapot)) {
		t.Errorf("TODO 5/6: recorded %v, want the 418 to be captured — override "+
			"WriteHeader on the wrapper and pass the code through", got)
	}
	if w.Code != http.StatusTeapot {
		t.Errorf("TODO 5: the real writer got status %d, want 418 — the wrapper "+
			"must still call s.ResponseWriter.WriteHeader(code)", w.Code)
	}
}

func TestMiddlewareReportsMethodAndPath(t *testing.T) {
	rec := &recorder{}
	h := RecordingMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}), rec.record)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodDelete, "/things/4", nil))

	got := rec.all()
	if len(got) != 1 {
		t.Fatalf("TODO 6: recorded %d calls, want 1", len(got))
	}
	if !strings.Contains(got[0], "DELETE") || !strings.Contains(got[0], "/things/4") {
		t.Errorf("TODO 6: recorded %q, want it to mention DELETE and /things/4", got[0])
	}
}

// --- TODO 7: the router ------------------------------------------------------

func newTestServer(t *testing.T) (*httptest.Server, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(NewRouter(rec.record))
	t.Cleanup(srv.Close)
	return srv, rec
}

func TestRouterRoutes(t *testing.T) {
	srv, _ := newTestServer(t)

	t.Run("health", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/health")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 || strings.TrimSpace(string(body)) != "ok" {
			t.Errorf("TODO 7: GET /health gave %d %q", resp.StatusCode, body)
		}
	})

	t.Run("greet with wildcard", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/greet/Jagjit")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			t.Fatalf("TODO 7: GET /greet/Jagjit gave %d — register the route as "+
				"\"GET /greet/{name}\"", resp.StatusCode)
		}
		if strings.TrimSpace(string(body)) != "Hello, Jagjit!" {
			t.Errorf("TODO 7: body = %q, want %q", body, "Hello, Jagjit!")
		}
	})

	t.Run("echo", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/echo", "application/json",
			strings.NewReader(`{"message":"hi"}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode != 200 || !strings.Contains(string(body), `"hi"`) {
			t.Errorf("TODO 7: POST /echo gave %d %q", resp.StatusCode, body)
		}
	})

	t.Run("info", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/info")
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("TODO 7: GET /info gave %d", resp.StatusCode)
		}
	})
}

func TestRouterRejectsWrongMethods(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Post(srv.URL+"/health", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("TODO 7: POST /health gave %d, want 405.\n"+
			"  Register the pattern with its method (\"GET /health\") and the mux\n"+
			"  handles this for you — no switch on r.Method required.", resp.StatusCode)
	}
}

func TestRouter404s(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.Get(srv.URL + "/nope")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("TODO 7: GET /nope gave %d, want 404", resp.StatusCode)
	}
}

func TestMiddlewareSeesEveryRequestIncludingMisses(t *testing.T) {
	srv, rec := newTestServer(t)

	for _, path := range []string{"/health", "/info", "/nope", "/greet/x"} {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}

	if n := rec.len(); n != 4 {
		t.Errorf("TODO 7: the middleware recorded %d requests out of 4.\n"+
			"  Wrap the WHOLE mux, not the individual handlers — that way the 404s\n"+
			"  and 405s the mux generates are recorded too. Recorded: %v",
			n, rec.all())
	}

	found404 := false
	for _, c := range rec.all() {
		if strings.Contains(c, "/nope") && strings.Contains(c, http.StatusText(404)) {
			found404 = true
		}
	}
	if !found404 {
		t.Errorf("TODO 7: the /nope request wasn't recorded as a 404. Recorded: %v",
			rec.all())
	}
}

func TestRouterHandlesConcurrentRequests(t *testing.T) {
	srv, rec := newTestServer(t)

	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/health")
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	mustFinish(t, 20*time.Second, "30 concurrent requests", wg.Wait)

	if n := rec.len(); n != 30 {
		t.Errorf("TODO 7: recorded %d of 30 concurrent requests", n)
	}
}

// --- main()'s narration ---------------------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 30*time.Second, "main()", func() { out = captureStdout(t, main) })
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 8")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`(?m)^ok$`), "TODO 9", "\"ok\" from /health"},
		{regexp.MustCompile(`(?m)^Hello, \S+!$`), "TODO 10", "the greeting"},
		{regexp.MustCompile(`\{"echo":"hi"\}`), "TODO 11", `{"echo":"hi"}`},
		{regexp.MustCompile(`(?m)^404$`), "TODO 12", "404 for the unknown path"},
		{regexp.MustCompile(`(?m)^4$`), "TODO 12", "4 recorded requests"},
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
