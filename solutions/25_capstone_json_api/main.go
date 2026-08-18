package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
)

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidationError struct {
	Field string
	Msg   string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Msg)
}

var ErrNotFound = errors.New("task not found")

const maxTitle = 200

func validateTitle(title string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", &ValidationError{Field: "title", Msg: "must not be empty"}
	}
	if len(title) > maxTitle {
		return "", &ValidationError{Field: "title", Msg: "must be 200 characters or fewer"}
	}
	return title, nil
}

type Store struct {
	mu     sync.RWMutex
	tasks  map[int]Task
	nextID int
}

func NewStore() *Store {
	return &Store{tasks: make(map[int]Task), nextID: 1}
}

func (s *Store) Create(title string) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{ID: s.nextID, Title: title}
	s.tasks[t.ID] = t
	s.nextID++
	return t, nil
}

func (s *Store) List() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Task, 0, len(s.tasks)) // non-nil, so it marshals as []
	for _, t := range s.tasks {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func (s *Store) Get(id int) (Task, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.tasks[id]
	return t, ok
}

func (s *Store) Update(id int, title string, done bool) (Task, error) {
	title, err := validateTitle(title)
	if err != nil {
		return Task{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return Task{}, ErrNotFound
	}
	t := Task{ID: id, Title: title, Done: done}
	s.tasks[id] = t
	return t, nil
}

func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}

// --- HTTP layer --------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

// writeStoreError turns a store error into the right status code, using the
// error's TYPE rather than its message.
func writeStoreError(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, ve.Error())
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "task not found")
	default:
		writeError(w, http.StatusInternalServerError, "internal error")
	}
}

func pathID(r *http.Request) (int, error) {
	return strconv.Atoi(r.PathValue("id"))
}

type taskBody struct {
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func NewAPI(store *Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, store.List())
	})

	mux.HandleFunc("POST /tasks", func(w http.ResponseWriter, r *http.Request) {
		var body taskBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		t, err := store.Create(body.Title)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, t)
	})

	mux.HandleFunc("GET /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id must be a number")
			return
		}
		t, ok := store.Get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeJSON(w, http.StatusOK, t)
	})

	mux.HandleFunc("PUT /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id must be a number")
			return
		}
		var body taskBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		t, err := store.Update(id, body.Title, body.Done)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, t)
	})

	mux.HandleFunc("DELETE /tasks/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, "id must be a number")
			return
		}
		if !store.Delete(id) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		w.WriteHeader(http.StatusNoContent) // 204: no body at all
	})

	return mux
}

func main() {
	srv := httptest.NewServer(NewAPI(NewStore()))
	defer srv.Close()

	post := func(title string) *http.Response {
		resp, err := http.Post(srv.URL+"/tasks", "application/json",
			strings.NewReader(fmt.Sprintf(`{"title":%q}`, title)))
		if err != nil {
			panic(err)
		}
		return resp
	}

	created := post("write Go")
	fmt.Println(created.StatusCode)
	created.Body.Close()
	post("write more Go").Body.Close()

	list, _ := http.Get(srv.URL + "/tasks")
	var tasks []Task
	json.NewDecoder(list.Body).Decode(&tasks)
	list.Body.Close()
	fmt.Println(len(tasks))

	one, _ := http.Get(srv.URL + "/tasks/1")
	var task Task
	json.NewDecoder(one.Body).Decode(&task)
	one.Body.Close()
	fmt.Println(task.Title)

	req, _ := http.NewRequest(http.MethodPut, srv.URL+"/tasks/1",
		strings.NewReader(`{"title":"write Go","done":true}`))
	req.Header.Set("Content-Type", "application/json")
	updated, _ := http.DefaultClient.Do(req)
	io.Copy(io.Discard, updated.Body)
	updated.Body.Close()
	fmt.Println(updated.StatusCode)

	del, _ := http.NewRequest(http.MethodDelete, srv.URL+"/tasks/1", nil)
	deleted, _ := http.DefaultClient.Do(del)
	deleted.Body.Close()
	fmt.Println(deleted.StatusCode)

	missing, _ := http.Get(srv.URL + "/tasks/999")
	missing.Body.Close()
	fmt.Println(missing.StatusCode)
}
