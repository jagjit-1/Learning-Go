package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
)

// ============================================================
// CONCEPT: nothing new — Set B all at once
// ============================================================
//
// Generics (18) if you want them, io (19), JSON with tags (20), embedding
// for the middleware wrapper (21), a table-driven habit (22), and the server
// side of net/http (24). Plus a mutex from Set A, because a store behind an
// HTTP handler is touched by many goroutines at once: the server runs every
// request in its own goroutine, and that is not optional.
//
// Two JSON traps that this spec deliberately walks you into:
//
//   var tasks []Task
//   json.Marshal(tasks)     // -> null      A nil slice is not an empty one.
//
//   tasks := []Task{}
//   json.Marshal(tasks)     // -> []        This is what a client expects.
//
// Every front end that does `data.forEach(...)` breaks on null. Initialise
// the slice.
//
// And: decide your error SHAPE once and use it everywhere. A client that has
// to parse a plain-text 400 from one route and JSON from another has no way
// to write one error handler.

// ------------------------------------------------------------
// SPEC: an in-memory task API
// ------------------------------------------------------------
//
// Declare these by name — the checker uses them directly:
//
//   type Task struct {
//       ID    int    `json:"id"`
//       Title string `json:"title"`
//       Done  bool   `json:"done"`
//   }
//
//   type ErrorResponse struct {
//       Error string `json:"error"`
//   }
//
//   type ValidationError struct {
//       Field string
//       Msg   string
//   }
//   func (e *ValidationError) Error() string      // "<Field>: <Msg>"
//
//   type Store struct{ ... }
//   func NewStore() *Store
//   func (s *Store) Create(title string) (Task, error)
//   func (s *Store) List() []Task
//   func (s *Store) Get(id int) (Task, bool)
//   func (s *Store) Update(id int, title string, done bool) (Task, error)
//   func (s *Store) Delete(id int) bool
//
//   func NewAPI(store *Store) http.Handler
//
// STORE RULES
//  1. IDs start at 1 and increase by one per successful Create.
//  2. Create trims whitespace from the title. An empty result, or a title
//     over 200 characters, returns a *ValidationError and creates nothing.
//  3. List returns tasks sorted by ID ascending, and returns an EMPTY,
//     NON-NIL slice when there is nothing — see the trap above. It must also
//     return a copy: a caller mutating the result must not change the store.
//  4. Update returns a *ValidationError for a bad title, and an error for an
//     unknown id. Get and Delete report a miss with their bool.
//  5. Every method is safe to call from many goroutines at once. The checker
//     runs it under -race with hundreds of them.
//
// ROUTES — all bodies JSON, all errors as ErrorResponse
//
//   GET    /tasks         200, a JSON array (never null)
//   POST   /tasks         201 with the created task
//                         400 if the body is bad JSON or the title invalid
//   GET    /tasks/{id}    200, or 404 if unknown, or 400 if id isn't a number
//   PUT    /tasks/{id}    200 with the updated task
//                         400 bad body / bad id, 404 unknown
//   DELETE /tasks/{id}    204 with NO body, or 404 if unknown
//
//   anything else         404, and a wrong method on a known path 405
//                         (register patterns with methods and the mux does it)
//
// Use *ValidationError with errors.As to decide between 400 and 500 rather
// than matching on message text — that's what Exercise 8 was for.

type Task struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"Done"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type ValidationError struct {
	Field string
	Msg   string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Msg)
}

type Store struct {
	GlobalID int
	Items    map[int]Task
	Mut      sync.RWMutex
}

func NewStore() *Store {
	return &Store{Items: map[int]Task{}, GlobalID: 1}
}

func (s *Store) Delete(id int) bool {
	s.Mut.Lock()
	defer s.Mut.Unlock()

	_, ok := s.Items[id]

	if !ok {
		return false
	}

	delete(s.Items, id)
	return true
}

var TaskNotFoundError error = errors.New("Task not found")

func (s *Store) Update(id int, title string, done bool) (Task, error) {
	title = strings.TrimSpace(title)
	if len(title) == 0 || len(title) > 200 {
		return Task{}, &ValidationError{Field: "Title", Msg: "Invalid length"}
	}

	s.Mut.Lock()
	defer s.Mut.Unlock()
	_, ok := s.Items[id]

	if !ok {
		return Task{}, TaskNotFoundError
	}
	updatedTask := Task{ID: id, Title: title, Done: done}
	s.Items[id] = updatedTask
	return updatedTask, nil

}

func (s *Store) Get(id int) (Task, bool) {
	s.Mut.RLock()
	defer s.Mut.RUnlock()
	i, ok := s.Items[id]
	return i, ok
}

func (s *Store) Create(title string) (Task, error) {
	title = strings.TrimSpace(title)
	if len(title) == 0 || len(title) > 200 {
		return Task{}, &ValidationError{Field: "Title", Msg: "Invalid length"}
	}

	s.Mut.Lock()
	defer s.Mut.Unlock()
	newTask := Task{ID: s.GlobalID, Title: title}
	s.Items[s.GlobalID] = newTask
	s.GlobalID++
	return newTask, nil
}

func (s *Store) List() []Task {
	s.Mut.RLock()
	result := []Task{}
	for _, task := range s.Items {
		result = append(result, task)
	}
	s.Mut.RUnlock()

	slices.SortFunc(result, func(a Task, b Task) int { return a.ID - b.ID })
	return result
}

func WriteJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func WriteError(w http.ResponseWriter, status int, msg string) {
	WriteJSON(w, status, ErrorResponse{Error: msg})
}

func WriteStoreError(w http.ResponseWriter, err error) {
	var ve *ValidationError
	switch {
	case errors.As(err, &ve):
		WriteError(w, http.StatusBadRequest, ve.Error())
	case errors.Is(err, TaskNotFoundError):
		WriteError(w, http.StatusNotFound, "Task not found")
	default:
		WriteError(w, http.StatusInternalServerError, "Internal server error")
	}
}

func (s *Store) UpdateTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	task := Task{}
	reqBody, _ := io.ReadAll(r.Body)
	if err := json.Unmarshal(reqBody, &task); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	updatedTask, err := s.Update(id, task.Title, task.Done)
	if err != nil {
		WriteStoreError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, updatedTask)
}

func (s *Store) DeleteTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	found := s.Delete(id)
	if !found {
		WriteStoreError(w, TaskNotFoundError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Store) GetTaskByIDHandler(w http.ResponseWriter, r *http.Request) {
	rawID := r.PathValue("id")
	id, err := strconv.Atoi(rawID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid ID")
		return
	}

	task, found := s.Get(id)

	if !found {
		WriteStoreError(w, TaskNotFoundError)
		return
	}

	WriteJSON(w, http.StatusOK, task)
}

func (s *Store) CreateTaskHandler(w http.ResponseWriter, r *http.Request) {
	byBody, _ := io.ReadAll(r.Body)
	taskBody := Task{}

	if err := json.Unmarshal(byBody, &taskBody); err != nil {
		WriteError(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	createdTask, err := s.Create(taskBody.Title)

	if err != nil {
		WriteStoreError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, createdTask)
}

func (s *Store) GetAllTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := s.List()
	WriteJSON(w, http.StatusOK, tasks)
}

func NewAPI(s *Store) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /tasks", s.GetAllTasksHandler)
	mux.HandleFunc("POST /tasks", s.CreateTaskHandler)
	mux.HandleFunc("PUT /tasks/{id}", s.UpdateTaskByIDHandler)
	mux.HandleFunc("GET /tasks/{id}", s.GetTaskByIDHandler)
	mux.HandleFunc("DELETE /tasks/{id}", s.DeleteTaskByIDHandler)

	return mux
}

func main() {
	// TODO: build a Store, wrap it in NewAPI, serve it with httptest.NewServer,
	// then drive it: create two tasks, list them, fetch one, update it, delete
	// it, and request something that doesn't exist.
	//
	// Print, one per line:
	//   the status code from creating a task        (201)
	//   the number of tasks in the list             (2)
	//   the title of the task you fetched
	//   the status code from the update             (200)
	//   the status code from the delete             (204)
	//   the status code for a task that isn't there (404)
	store := NewStore()
	handler := NewAPI(store)
	server := httptest.NewServer(handler)

	client := &http.Client{}

	// CREATE 2 TASKS
	reqBody, _ := json.Marshal(Task{Title: "write Go", Done: false})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/tasks", bytes.NewReader(reqBody))
	res, _ := client.Do(req)

	fmt.Println(res.StatusCode)

	reqBody, _ = json.Marshal(Task{Title: "write Go", Done: false})
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, server.URL+"/tasks", bytes.NewReader(reqBody))
	res, _ = client.Do(req)

	// GET ALL TASKS
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/tasks", nil)
	res, _ = client.Do(req)

	resBody, _ := io.ReadAll(res.Body)
	allTasks := []Task{}
	json.Unmarshal(resBody, &allTasks)

	fmt.Println(len(allTasks))

	// GET TASK BY ID
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/tasks"+"/1", nil)
	res, _ = client.Do(req)
	task := Task{}

	resBody, _ = io.ReadAll(res.Body)
	json.Unmarshal(resBody, &task)
	fmt.Println(task.Title)

	// UPDATE TASK BY ID
	reqBody, _ = json.Marshal(Task{Title: "write Go again", Done: true})
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodPut, server.URL+"/tasks/1", bytes.NewReader(reqBody))
	res, _ = client.Do(req)

	fmt.Println(res.StatusCode)

	// DELETE TASK BY ID
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodDelete, server.URL+"/tasks/2", nil)
	res, _ = client.Do(req)

	fmt.Println(res.StatusCode)

	// FETCH INVALID ID TASK
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/tasks/2", nil)
	res, _ = client.Do(req)

	fmt.Println(res.StatusCode)
}

// EXPECTED OUTPUT:
// 201
// 2
// write Go
// 200
// 204
// 404
