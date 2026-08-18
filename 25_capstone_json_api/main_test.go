package main

// ============================================================
// CHECKER for 25_capstone_json_api — run with:  go test -race
// ============================================================
// Store is checked directly; the API is driven over a real httptest server,
// so the mux's own 404/405 behaviour counts. -race matters here: the server
// runs every request in its own goroutine, so an unguarded map is a crash
// waiting for production rather than a test failure.

import (
	"bytes"
	"encoding/json"
	"errors"
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

var (
	_ func() *Store             = NewStore
	_ func(*Store) http.Handler = NewAPI
	_ error                     = (*ValidationError)(nil)
	_                           = Task{ID: 1, Title: "t", Done: false}
	_                           = ErrorResponse{Error: "e"}
	_                           = ValidationError{Field: "f", Msg: "m"}
)

// --- Store: creation and validation -------------------------------------

func TestStoreCreateAssignsSequentialIDs(t *testing.T) {
	s := NewStore()
	for want := 1; want <= 3; want++ {
		task, err := s.Create("task")
		if err != nil {
			t.Fatalf("rule 1: Create returned %v", err)
		}
		if task.ID != want {
			t.Errorf("rule 1: got ID %d, want %d — IDs start at 1 and increase by one",
				task.ID, want)
		}
		if task.Done {
			t.Error("rule 1: a new task should not be Done")
		}
	}
}

func TestStoreCreateTrimsAndValidates(t *testing.T) {
	s := NewStore()

	task, err := s.Create("  padded  ")
	if err != nil {
		t.Fatalf("rule 2: Create returned %v", err)
	}
	if task.Title != "padded" {
		t.Errorf("rule 2: Title = %q, want %q — trim the title", task.Title, "padded")
	}

	for _, bad := range []string{"", "   ", "\t\n", strings.Repeat("x", 201)} {
		_, err := s.Create(bad)
		if err == nil {
			t.Errorf("rule 2: Create(%.20q...) should have failed", bad)
			continue
		}
		var ve *ValidationError
		if !errors.As(err, &ve) {
			t.Errorf("rule 2: Create(%.20q...) returned %T, want a *ValidationError "+
				"so the HTTP layer can pick 400 with errors.As rather than by "+
				"matching the message", bad, err)
		}
	}

	if _, err := s.Create(strings.Repeat("x", 200)); err != nil {
		t.Errorf("rule 2: exactly 200 characters is allowed (\"over 200\" fails), got %v", err)
	}
}

func TestStoreCreateDoesNotStoreInvalidTasks(t *testing.T) {
	s := NewStore()
	s.Create("")
	s.Create("   ")
	if got := s.List(); len(got) != 0 {
		t.Errorf("rule 2: a failed Create still added %d tasks", len(got))
	}
}

func TestStoreCreateDoesNotBurnIDsOnFailure(t *testing.T) {
	s := NewStore()
	s.Create("")
	task, err := s.Create("real")
	if err != nil {
		t.Fatal(err)
	}
	if task.ID != 1 {
		t.Errorf("rule 1/2: after a failed Create the first real task got ID %d, "+
			"want 1 — only a SUCCESSFUL create consumes an id", task.ID)
	}
}

// --- Store: List ----------------------------------------------------------

func TestStoreListIsEmptyNotNil(t *testing.T) {
	got := NewStore().List()
	if got == nil {
		t.Fatal("rule 3: List returned nil on an empty store.\n" +
			"  json.Marshal turns a nil slice into `null`, not `[]`, and every\n" +
			"  client that iterates the response breaks on it. Build the slice\n" +
			"  with make([]Task, 0, n).")
	}
	if len(got) != 0 {
		t.Errorf("rule 3: List on an empty store = %v", got)
	}

	data, _ := json.Marshal(got)
	if string(data) != "[]" {
		t.Errorf("rule 3: an empty List marshals to %s, want []", data)
	}
}

func TestStoreListIsSortedByID(t *testing.T) {
	s := NewStore()
	for i := 0; i < 20; i++ {
		s.Create("task")
	}
	got := s.List()
	if len(got) != 20 {
		t.Fatalf("rule 3: List returned %d tasks, want 20", len(got))
	}
	for i, task := range got {
		if task.ID != i+1 {
			t.Fatalf("rule 3: position %d holds ID %d, want %d — sort by ID "+
				"(map iteration order is randomised, so without a sort this is "+
				"different every run)", i, task.ID, i+1)
		}
	}
}

func TestStoreListReturnsACopy(t *testing.T) {
	s := NewStore()
	s.Create("original")

	got := s.List()
	got[0].Title = "tampered"

	if again := s.List(); again[0].Title != "original" {
		t.Error("rule 3: mutating the slice List returned changed the store.\n" +
			"  The lock is long gone by the time the caller touches it — hand back " +
			"a copy.")
	}
}

// --- Store: Get, Update, Delete --------------------------------------------

func TestStoreGet(t *testing.T) {
	s := NewStore()
	created, _ := s.Create("findme")

	got, ok := s.Get(created.ID)
	if !ok {
		t.Fatal("rule 4: Get reported not-found for a task that exists")
	}
	if got.Title != "findme" {
		t.Errorf("rule 4: Get returned %+v", got)
	}

	if _, ok := s.Get(999); ok {
		t.Error("rule 4: Get(999) reported found on an empty-ish store")
	}
}

func TestStoreUpdate(t *testing.T) {
	s := NewStore()
	created, _ := s.Create("before")

	updated, err := s.Update(created.ID, "after", true)
	if err != nil {
		t.Fatalf("rule 4: Update returned %v", err)
	}
	if updated.ID != created.ID || updated.Title != "after" || !updated.Done {
		t.Errorf("rule 4: Update returned %+v, want the same ID with the new "+
			"title and done flag", updated)
	}

	got, _ := s.Get(created.ID)
	if got.Title != "after" || !got.Done {
		t.Errorf("rule 4: the stored task is still %+v — Update must persist", got)
	}
}

func TestStoreUpdateErrors(t *testing.T) {
	s := NewStore()
	created, _ := s.Create("exists")

	if _, err := s.Update(999, "whatever", false); err == nil {
		t.Error("rule 4: updating a task that doesn't exist should return an error")
	}

	_, err := s.Update(created.ID, "  ", false)
	var ve *ValidationError
	if err == nil || !errors.As(err, &ve) {
		t.Errorf("rule 4: Update with a blank title returned %v, want a "+
			"*ValidationError", err)
	}
	if got, _ := s.Get(created.ID); got.Title != "exists" {
		t.Errorf("rule 4: a failed Update changed the stored task to %+v", got)
	}
}

func TestStoreDelete(t *testing.T) {
	s := NewStore()
	created, _ := s.Create("doomed")

	if !s.Delete(created.ID) {
		t.Fatal("rule 4: Delete reported false for a task that exists")
	}
	if _, ok := s.Get(created.ID); ok {
		t.Error("rule 4: the task is still there after Delete")
	}
	if s.Delete(created.ID) {
		t.Error("rule 4: deleting the same task twice should report false the " +
			"second time")
	}
	if s.Delete(999) {
		t.Error("rule 4: Delete(999) reported true")
	}
}

// --- Store: concurrency ------------------------------------------------------

func TestStoreIsConcurrencySafe(t *testing.T) {
	s := NewStore()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Create("concurrent")
		}()
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.List()
			s.Get(1)
			s.Update(1, "updated", true)
			s.Delete(500)
		}()
	}

	mustFinish(t, 30*time.Second, "concurrent Store access", wg.Wait)

	if got := len(s.List()); got != 100 {
		t.Errorf("rule 5: after 100 concurrent Creates the store holds %d tasks, "+
			"want 100 — an unguarded nextID hands the same id to two goroutines", got)
	}

	seen := map[int]bool{}
	for _, task := range s.List() {
		if seen[task.ID] {
			t.Fatalf("rule 5: duplicate ID %d — the read-increment of nextID must "+
				"happen under the same lock as the insert", task.ID)
		}
		seen[task.ID] = true
	}
}

// --- API helpers ---------------------------------------------------------------

func newAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(NewAPI(NewStore()))
	t.Cleanup(srv.Close)
	return srv
}

func do(t *testing.T, method, url, body string) (*http.Response, []byte) {
	t.Helper()
	var rdr io.Reader
	if body != "" {
		rdr = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		t.Fatalf("building request: %v", err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return resp, data
}

// --- API: the happy path -------------------------------------------------------

func TestAPIFullLifecycle(t *testing.T) {
	srv := newAPIServer(t)

	resp, body := do(t, http.MethodPost, srv.URL+"/tasks", `{"title":"write Go"}`)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("POST /tasks gave %d, want 201 (body: %s)", resp.StatusCode, body)
	}
	var created Task
	if err := json.Unmarshal(body, &created); err != nil {
		t.Fatalf("POST /tasks body %s is not a Task: %v", body, err)
	}
	if created.ID == 0 || created.Title != "write Go" {
		t.Fatalf("POST /tasks returned %+v", created)
	}

	resp, body = do(t, http.MethodGet, srv.URL+"/tasks", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /tasks gave %d", resp.StatusCode)
	}
	var list []Task
	if err := json.Unmarshal(body, &list); err != nil {
		t.Fatalf("GET /tasks body %s: %v", body, err)
	}
	if len(list) != 1 {
		t.Fatalf("GET /tasks returned %d tasks, want 1", len(list))
	}

	resp, body = do(t, http.MethodGet, srv.URL+"/tasks/1", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /tasks/1 gave %d (body: %s)", resp.StatusCode, body)
	}

	resp, body = do(t, http.MethodPut, srv.URL+"/tasks/1", `{"title":"write more Go","done":true}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /tasks/1 gave %d (body: %s)", resp.StatusCode, body)
	}
	var updated Task
	json.Unmarshal(body, &updated)
	if updated.Title != "write more Go" || !updated.Done {
		t.Errorf("PUT /tasks/1 returned %+v", updated)
	}

	resp, body = do(t, http.MethodDelete, srv.URL+"/tasks/1", "")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE /tasks/1 gave %d, want 204", resp.StatusCode)
	}
	if len(bytes.TrimSpace(body)) != 0 {
		t.Errorf("DELETE returned a body (%s) — 204 means No Content, so write "+
			"nothing after w.WriteHeader(http.StatusNoContent)", body)
	}

	resp, _ = do(t, http.MethodGet, srv.URL+"/tasks/1", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("after deleting, GET /tasks/1 gave %d, want 404", resp.StatusCode)
	}
}

func TestAPIEmptyListIsAnArray(t *testing.T) {
	srv := newAPIServer(t)

	resp, body := do(t, http.MethodGet, srv.URL+"/tasks", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /tasks gave %d", resp.StatusCode)
	}
	if got := strings.TrimSpace(string(body)); got != "[]" {
		t.Errorf("GET /tasks on an empty store returned %s, want [].\n"+
			"  `null` here breaks every client that iterates the response.", got)
	}
}

func TestAPIListIsSorted(t *testing.T) {
	srv := newAPIServer(t)
	for i := 0; i < 10; i++ {
		do(t, http.MethodPost, srv.URL+"/tasks", `{"title":"task"}`)
	}

	_, body := do(t, http.MethodGet, srv.URL+"/tasks", "")
	var list []Task
	json.Unmarshal(body, &list)
	for i, task := range list {
		if task.ID != i+1 {
			t.Fatalf("GET /tasks returned IDs out of order at position %d: %+v", i, list)
		}
	}
}

func TestAPIContentTypeIsJSON(t *testing.T) {
	srv := newAPIServer(t)
	resp, _ := do(t, http.MethodGet, srv.URL+"/tasks", "")
	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
		t.Errorf("GET /tasks Content-Type = %q, want application/json", ct)
	}
}

// --- API: the error paths --------------------------------------------------------

func TestAPIErrorResponses(t *testing.T) {
	srv := newAPIServer(t)
	do(t, http.MethodPost, srv.URL+"/tasks", `{"title":"exists"}`)

	cases := []struct {
		name   string
		method string
		path   string
		body   string
		want   int
	}{
		{"create with empty title", http.MethodPost, "/tasks", `{"title":""}`, 400},
		{"create with blank title", http.MethodPost, "/tasks", `{"title":"   "}`, 400},
		{"create with no title key", http.MethodPost, "/tasks", `{}`, 400},
		{"create with bad JSON", http.MethodPost, "/tasks", `{"title":`, 400},
		{"create with an over-long title", http.MethodPost, "/tasks",
			`{"title":"` + strings.Repeat("x", 201) + `"}`, 400},
		{"get an unknown id", http.MethodGet, "/tasks/999", "", 404},
		{"get a non-numeric id", http.MethodGet, "/tasks/abc", "", 400},
		{"update an unknown id", http.MethodPut, "/tasks/999", `{"title":"x"}`, 404},
		{"update with a blank title", http.MethodPut, "/tasks/1", `{"title":" "}`, 400},
		{"update with bad JSON", http.MethodPut, "/tasks/1", `nonsense`, 400},
		{"delete an unknown id", http.MethodDelete, "/tasks/999", "", 404},
		{"delete a non-numeric id", http.MethodDelete, "/tasks/abc", "", 400},
		{"unknown path", http.MethodGet, "/nope", "", 404},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			resp, body := do(t, c.method, srv.URL+c.path, c.body)
			if resp.StatusCode != c.want {
				t.Fatalf("%s %s gave %d, want %d (body: %s)",
					c.method, c.path, resp.StatusCode, c.want, body)
			}

			// The mux writes its own plain-text 404 for an unknown path; every
			// error we generate ourselves must use the agreed JSON shape.
			if c.path == "/nope" {
				return
			}
			var er ErrorResponse
			if err := json.Unmarshal(body, &er); err != nil || er.Error == "" {
				t.Errorf("%s %s returned %q — errors should come back as "+
					"ErrorResponse so a client can parse one shape everywhere",
					c.method, c.path, body)
			}
		})
	}
}

func TestAPIRejectsWrongMethods(t *testing.T) {
	srv := newAPIServer(t)

	for _, c := range []struct{ method, path string }{
		{http.MethodDelete, "/tasks"},
		{http.MethodPost, "/tasks/1"},
		{http.MethodPatch, "/tasks"},
	} {
		resp, _ := do(t, c.method, srv.URL+c.path, "")
		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s %s gave %d, want 405 — register patterns with their "+
				"method (\"GET /tasks\") and the mux handles this",
				c.method, c.path, resp.StatusCode)
		}
	}
}

func TestAPIValidationFailureDoesNotCreate(t *testing.T) {
	srv := newAPIServer(t)
	do(t, http.MethodPost, srv.URL+"/tasks", `{"title":""}`)

	_, body := do(t, http.MethodGet, srv.URL+"/tasks", "")
	if got := strings.TrimSpace(string(body)); got != "[]" {
		t.Errorf("after a rejected create, GET /tasks returned %s, want []", got)
	}
}

// --- API: concurrency ----------------------------------------------------------

func TestAPIHandlesConcurrentRequests(t *testing.T) {
	srv := newAPIServer(t)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Post(srv.URL+"/tasks", "application/json",
				strings.NewReader(`{"title":"concurrent"}`))
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			resp, err := http.Get(srv.URL + "/tasks")
			if err != nil {
				return
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}()
	}

	mustFinish(t, 40*time.Second, "100 concurrent requests", wg.Wait)

	_, body := do(t, http.MethodGet, srv.URL+"/tasks", "")
	var list []Task
	json.Unmarshal(body, &list)
	if len(list) != 50 {
		t.Errorf("after 50 concurrent creates the API holds %d tasks, want 50.\n"+
			"  Every request runs in its own goroutine — the store behind them is "+
			"shared state and needs a lock.", len(list))
	}
}

// --- main()'s narration -----------------------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 40*time.Second, "main()", func() { out = captureStdout(t, main) })

	lines := []string{}
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		t.Fatal("main() printed nothing — drive your own API and print the results")
	}

	want := []struct {
		re   *regexp.Regexp
		what string
	}{
		{regexp.MustCompile(`^201$`), "201 from creating a task"},
		{regexp.MustCompile(`^2$`), "2 tasks in the list"},
		{regexp.MustCompile(`^\S.*$`), "the title of the task you fetched"},
		{regexp.MustCompile(`^200$`), "200 from the update"},
		{regexp.MustCompile(`^204$`), "204 from the delete"},
		{regexp.MustCompile(`^404$`), "404 for the missing task"},
	}
	if len(lines) < len(want) {
		t.Fatalf("expected %d lines of output, got %d:\n%s", len(want), len(lines), indent(out))
	}
	for i, w := range want {
		if !w.re.MatchString(lines[i]) {
			t.Errorf("line %d is %q, expected %s.\n  your output was:\n%s",
				i+1, lines[i], w.what, indent(out))
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
