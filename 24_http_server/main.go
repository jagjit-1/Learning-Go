package main

import "fmt"

// ============================================================
// CONCEPT: net/http as a server
// ============================================================
//
// The whole server side rests on one interface:
//
//   type Handler interface {
//       ServeHTTP(w http.ResponseWriter, r *http.Request)
//   }
//
// http.HandlerFunc adapts a plain function to it, so you rarely declare a
// type:
//
//   func Health(w http.ResponseWriter, r *http.Request) {
//       w.Write([]byte("ok"))
//   }
//   http.HandlerFunc(Health)     // now it's a Handler
//
// WRITING A RESPONSE, in the order the protocol requires:
//
//   w.Header().Set("Content-Type", "application/json")   // headers FIRST
//   w.WriteHeader(http.StatusCreated)                     // then the status
//   w.Write(body)                                          // then the body
//
// That order is not a style preference. The first Write sends the status line
// and headers, so a Header().Set after it is silently ignored, and a second
// WriteHeader logs "superfluous response.WriteHeader call". If you never call
// WriteHeader, the first Write sends 200 for you.
//
// For errors there's a shortcut that does all three:
//
//   http.Error(w, "bad request", http.StatusBadRequest)
//
// And note what you must do after it: RETURN. http.Error doesn't stop your
// handler, and code that keeps going will write a body after the error.
//
// ROUTING. http.ServeMux got real routing in Go 1.22 — method and wildcards
// in the pattern:
//
//   mux := http.NewServeMux()
//   mux.HandleFunc("GET /health", Health)
//   mux.HandleFunc("GET /greet/{name}", Greet)
//   mux.HandleFunc("POST /echo", Echo)
//
//   name := r.PathValue("name")    // pull the wildcard out
//
// The mux answers a wrong method with 405 Method Not Allowed by itself. Older
// tutorials show `mux.HandleFunc("/greet/", ...)` plus manual string slicing
// and a switch on r.Method — you don't need any of that any more.
//
// MIDDLEWARE is a function that takes a Handler and returns a Handler:
//
//   func Logging(next http.Handler) http.Handler {
//       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//           start := time.Now()
//           next.ServeHTTP(w, r)              // call through
//           log.Println(r.Method, r.URL.Path, time.Since(start))
//       })
//   }
//
//   srv := Logging(Auth(mux))    // wraps outside-in
//
// CAPTURING THE STATUS in middleware needs a trick, because ResponseWriter
// has no getter for what was written. Wrap it — embedding, from Exercise 21:
//
//   type StatusRecorder struct {
//       http.ResponseWriter          // embedded: Header and Write are promoted
//       Status int
//   }
//   func (s *StatusRecorder) WriteHeader(code int) {
//       s.Status = code
//       s.ResponseWriter.WriteHeader(code)
//   }
//
// Only WriteHeader is overridden; everything else passes through untouched.
// Initialise Status to 200, because a handler that just calls Write never
// calls WriteHeader at all and that still means 200.
//
// TESTING, with no port and no network:
//
//   req := httptest.NewRequest("GET", "/health", nil)
//   rec := httptest.NewRecorder()
//   Health(rec, req)
//   rec.Code, rec.Body.String(), rec.Header()
//
// httptest.NewRecorder is a ResponseWriter that just remembers everything.
// For testing a whole router — including the mux's own 404 and 405 — use
// httptest.NewServer as in Exercise 23.

// TODO 1: write `func HealthHandler(w http.ResponseWriter, r *http.Request)`
// answering 200 with the body "ok".

// TODO 2: define `type Info struct { Service string `json:"service"`;
// Version string `json:"version"` }` and write `func InfoHandler` returning
// Info{"learning-go", "1.0"} as JSON with a Content-Type of
// "application/json". Set the header BEFORE writing anything.

// TODO 3: write `func GreetHandler` for the route "GET /greet/{name}",
// answering 200 with "Hello, <name>!". Read the wildcard with r.PathValue.
// If the name is empty, answer 400.

// TODO 4: define
//   type EchoRequest struct { Message string `json:"message"` }
//   type EchoResponse struct { Echo string `json:"echo"` }
// and write `func EchoHandler` for "POST /echo": decode the body, answer 200
// with the message echoed back as JSON. Malformed JSON or an empty message
// is a 400. Remember to return after writing an error.

// TODO 5: define `type StatusRecorder struct` embedding http.ResponseWriter
// with a `Status int` field and a pointer-receiver `WriteHeader` that records
// the code and passes it on.

// TODO 6: write
//   func RecordingMiddleware(next http.Handler, record func(method, path string, status int)) http.Handler
// that wraps w in a StatusRecorder (defaulting Status to 200), calls next,
// then invokes record with the method, the path and the final status.

// TODO 7: write
//   func NewRouter(record func(method, path string, status int)) http.Handler
// building a ServeMux with:
//   GET  /health      -> HealthHandler
//   GET  /info        -> InfoHandler
//   GET  /greet/{name} -> GreetHandler
//   POST /echo        -> EchoHandler
// and returning it wrapped in RecordingMiddleware.

func main() {
	// TODO 8: build a router with a record func that appends to a slice, put
	// it behind httptest.NewServer, and defer srv.Close().

	// TODO 9: GET /health and print the body.

	// TODO 10: GET /greet/Jagjit and print the body.

	// TODO 11: POST /echo with `{"message":"hi"}` and print the response body.

	// TODO 12: GET /nope and print the status code, then print how many
	// requests your middleware recorded.

	fmt.Print()
}

// EXPECTED OUTPUT:
// ok
// Hello, Jagjit!
// {"echo":"hi"}
// 404
// 4
