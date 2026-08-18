package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"
)

// One client, reused: it carries the connection pool, and it has a timeout,
// unlike http.DefaultClient.
var client = &http.Client{Timeout: 5 * time.Second}

func FetchText(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close() // straight after the error check

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return "", fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

type Todo struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func FetchTodo(ctx context.Context, url string) (Todo, error) {
	var todo Todo

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return todo, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return todo, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return todo, fmt.Errorf("GET %s: unexpected status %d", url, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&todo); err != nil {
		return Todo{}, fmt.Errorf("decoding todo: %w", err)
	}
	return todo, nil
}

func CreateTodo(ctx context.Context, url string, t Todo) (Todo, error) {
	var created Todo

	body, err := json.Marshal(t)
	if err != nil {
		return created, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return created, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return created, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return created, fmt.Errorf("POST %s: unexpected status %d", url, resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return Todo{}, fmt.Errorf("decoding created todo: %w", err)
	}
	return created, nil
}

func FetchStatus(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, err // only a transport failure is an error here
	}
	defer resp.Body.Close()

	io.Copy(io.Discard, resp.Body) // drain so the connection can be reused
	return resp.StatusCode, nil
}

func FetchAllTexts(ctx context.Context, urls []string) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	out := make([]string, len(urls))

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for i, url := range urls {
		wg.Add(1)
		go func() {
			defer wg.Done()

			body, err := FetchText(ctx, url)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				cancel()
				return
			}
			out[i] = body
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func main() {
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":1,"title":"write Go","done":false}`)
	}))
	defer srv.Close()

	text, _ := FetchText(ctx, srv.URL)
	fmt.Println(text)

	todo, _ := FetchTodo(ctx, srv.URL)
	fmt.Println(todo.Title)

	missing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "nope", http.StatusNotFound)
	}))
	defer missing.Close()

	code, _ := FetchStatus(ctx, missing.URL)
	fmt.Println(code)

	_, err := FetchText(ctx, missing.URL)
	fmt.Println(err != nil)

	bodies, _ := FetchAllTexts(ctx, []string{srv.URL, srv.URL, srv.URL})
	fmt.Println(len(bodies))
}
