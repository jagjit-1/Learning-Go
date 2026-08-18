package main

import "fmt"

// ============================================================
// CONCEPT: net/http as a client
// ============================================================
//
// The one-liners exist and you should almost never use them:
//
//   resp, err := http.Get("https://example.com/api")
//
// They use http.DefaultClient, which has NO TIMEOUT. A server that accepts
// your connection and then says nothing will hang that goroutine forever.
// In production that is how you leak every goroutine you have.
//
// Make your own client instead, once, and reuse it:
//
//   var client = &http.Client{Timeout: 10 * time.Second}
//
// Reuse matters: a Client holds a connection pool, and building a fresh one
// per request throws away keep-alives.
//
// THE THREE THINGS EVERY RESPONSE NEEDS:
//
//   resp, err := client.Do(req)
//   if err != nil { return err }          // 1. transport failure only
//   defer resp.Body.Close()               // 2. ALWAYS, or you leak the conn
//   if resp.StatusCode != http.StatusOK { // 3. err is nil for a 404 or 500!
//       return fmt.Errorf("unexpected status %s", resp.Status)
//   }
//
// Point 3 is the one that bites. `err` reports "I could not talk to the
// server". A 500 means you talked to it perfectly and it told you it was
// broken — that is a successful request as far as err is concerned.
//
// Not closing the body is a real resource leak: the connection can't go back
// to the pool. `defer resp.Body.Close()` goes immediately after the error
// check, before anything that can return early. (You only close it when err
// is nil; on error resp is nil.)
//
// WITH A CONTEXT — always, for anything that isn't a toy. This is how a
// caller's deadline reaches the network layer:
//
//   req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
//   resp, err := client.Do(req)
//
// The Client.Timeout is a blunt overall cap; the context is what composes
// with the rest of your program.
//
// SENDING JSON:
//
//   body, _ := json.Marshal(payload)
//   req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url,
//                                        bytes.NewReader(body))
//   req.Header.Set("Content-Type", "application/json")
//
// READING JSON BACK — decode straight from the body, no need to buffer it:
//
//   var out Thing
//   err := json.NewDecoder(resp.Body).Decode(&out)
//
// TESTING. You do NOT hit the real internet in tests. httptest.NewServer
// starts a real HTTP server on a random localhost port, and gives you its
// URL:
//
//   srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       w.Write([]byte(`{"ok":true}`))
//   }))
//   defer srv.Close()
//   FetchThing(ctx, srv.URL)
//
// That's why every function below takes a full URL rather than building one
// from a hardcoded host: it makes them testable. The checker spins up such a
// server, so none of this exercise touches the network.

// TODO 1: declare a package-level `var client = &http.Client{Timeout: ...}`
// with a sensible timeout (a few seconds). Use it everywhere below.

// TODO 2: write `func FetchText(ctx context.Context, url string) (string, error)`
// that GETs the url and returns the body as a string. Close the body, and
// treat any status outside 200-299 as an error whose message contains the
// status code.

// TODO 3: define
//   type Todo struct { ID int `json:"id"`; Title string `json:"title"`; Done bool `json:"done"` }
// and write `func FetchTodo(ctx context.Context, url string) (Todo, error)`
// that decodes the JSON body straight from resp.Body with a json.Decoder.

// TODO 4: write
//   func CreateTodo(ctx context.Context, url string, t Todo) (Todo, error)
// that POSTs t as JSON with a Content-Type header of "application/json" and
// decodes the created Todo out of the response. Accept 200 or 201; anything
// else is an error.

// TODO 5: write `func FetchStatus(ctx context.Context, url string) (int, error)`
// returning just the status code, with no error for a 404 or a 500 — only
// for a transport failure. This makes the difference from TODO 2 explicit.

// TODO 6: write
//   func FetchAllTexts(ctx context.Context, urls []string) ([]string, error)
// fetching every url CONCURRENTLY, returning the bodies in the order of
// `urls`. If any fetch fails, return the error. You built this shape in
// Exercise 15 — reuse it.

func main() {
	// TODO 7: start an httptest.NewServer whose handler returns
	// `{"id":1,"title":"write Go","done":false}` for any request, remembering
	// to defer srv.Close().

	// TODO 8: print FetchText against it.

	// TODO 9: print the Title from FetchTodo against it.

	// TODO 10: start a second server that always replies 404 and print
	// FetchStatus for it, then print whether FetchText returns an error for it.

	// TODO 11: print the number of bodies FetchAllTexts returns for three
	// copies of the first server's URL.

	fmt.Print()
}

// EXPECTED OUTPUT:
// {"id":1,"title":"write Go","done":false}
// write Go
// 404
// true
// 3
