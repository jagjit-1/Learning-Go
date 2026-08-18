package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
)

func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok")) // no WriteHeader call means 200
}

type Info struct {
	Service string `json:"service"`
	Version string `json:"version"`
}

func InfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json") // before any write
	json.NewEncoder(w).Encode(Info{Service: "learning-go", Version: "1.0"})
}

func GreetHandler(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Hello, %s!", name)
}

type EchoRequest struct {
	Message string `json:"message"`
}

type EchoResponse struct {
	Echo string `json:"echo"`
}

func EchoHandler(w http.ResponseWriter, r *http.Request) {
	var req EchoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return // http.Error does not stop the handler
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(EchoResponse{Echo: req.Message})
}

type StatusRecorder struct {
	http.ResponseWriter // Header and Write come along for free
	Status              int
}

func (s *StatusRecorder) WriteHeader(code int) {
	s.Status = code
	s.ResponseWriter.WriteHeader(code)
}

func RecordingMiddleware(next http.Handler, record func(method, path string, status int)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 200 by default: a handler that only calls Write never reaches
		// WriteHeader, and that still means 200 on the wire.
		rec := &StatusRecorder{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(rec, r)
		record(r.Method, r.URL.Path, rec.Status)
	})
}

func NewRouter(record func(method, path string, status int)) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", HealthHandler)
	mux.HandleFunc("GET /info", InfoHandler)
	mux.HandleFunc("GET /greet/{name}", GreetHandler)
	mux.HandleFunc("POST /echo", EchoHandler)
	return RecordingMiddleware(mux, record)
}

func main() {
	var seen []string
	router := NewRouter(func(method, path string, status int) {
		seen = append(seen, fmt.Sprintf("%s %s %d", method, path, status))
	})

	srv := httptest.NewServer(router)
	defer srv.Close()

	get := func(path string) *http.Response {
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			panic(err)
		}
		return resp
	}

	health := get("/health")
	body, _ := io.ReadAll(health.Body)
	health.Body.Close()
	fmt.Println(string(body))

	greet := get("/greet/Jagjit")
	body, _ = io.ReadAll(greet.Body)
	greet.Body.Close()
	fmt.Println(string(body))

	echo, _ := http.Post(srv.URL+"/echo", "application/json",
		strings.NewReader(`{"message":"hi"}`))
	body, _ = io.ReadAll(echo.Body)
	echo.Body.Close()
	fmt.Println(strings.TrimSpace(string(body)))

	missing := get("/nope")
	missing.Body.Close()
	fmt.Println(missing.StatusCode)

	fmt.Println(len(seen))
}
