package main

import "fmt"

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

	fmt.Print()
}

// EXPECTED OUTPUT:
// 201
// 2
// write Go
// 200
// 204
// 404
