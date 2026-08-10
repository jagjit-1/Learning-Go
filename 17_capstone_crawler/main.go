package main

import (
	"context"
	"fmt"
)

// ============================================================
// CONCEPT: nothing new — this is the whole set at once
// ============================================================
//
// You have all the pieces: goroutines and channels (10), select and timeouts
// (11), mutexes and WaitGroups (12), bounded worker pools (13), fan-out and
// leak-free shutdown (14), context (15), and the discipline to keep shared
// state guarded (16). This exercise makes you use them together on a problem
// where the work is DISCOVERED AS YOU GO — you don't know the job list up
// front, because each page you fetch reveals more pages.
//
// That last part is what makes it harder than Exercise 13. A fixed job list
// lets you close the jobs channel when the loop ends. Here, "are we done?"
// means "is there no work in flight AND nothing queued", and you have to
// track it. A WaitGroup that you Add to as new links are found handles this
// naturally — as long as you Add BEFORE starting the goroutine.
//
// THE INJECTED DEPENDENCY. Crawler doesn't know where pages come from; it
// takes a Fetcher interface. Real code would hand it an HTTP client; the
// checker hands it an in-memory fake with a fixed link graph. This is the
// standard reason Go code takes interfaces rather than concrete types, and
// it's why the checker can test a crawler without touching the network.

// ------------------------------------------------------------
// SPEC: bounded-concurrency crawler
// ------------------------------------------------------------
//
// Fill in these declarations — the checker uses them by name:
//
//   type Fetcher interface {
//       Fetch(ctx context.Context, url string) (title string, links []string, err error)
//   }
//
//   type Page struct {
//       URL   string
//       Title string
//       Depth int
//   }
//
//   type Crawler struct {
//       Fetcher  Fetcher
//       Workers  int    // max Fetch calls in flight at once
//       MaxDepth int    // start is depth 0; do not fetch beyond this
//   }
//
//   type Result struct {
//       Pages  []Page
//       Errors map[string]error   // keyed by URL
//   }
//
//   func (c *Crawler) Crawl(ctx context.Context, start string) Result
//
// Behaviour required:
//
//  1. Start at `start`, depth 0. For every page fetched, follow its links at
//     depth+1, and so on.
//  2. NEVER fetch the same URL twice, even when pages link back to each other.
//     The test graph contains cycles and will hang forever if you skip this.
//  3. Do not fetch anything deeper than MaxDepth. Depth 0 alone means just
//     the start page.
//  4. At most `Workers` Fetch calls running at any moment. Treat Workers <= 0
//     as 1. (Spawning a goroutine per link is fine — it's the concurrent
//     FETCHES that must be capped. A buffered channel used as a semaphore is
//     the simplest way: send into it before fetching, receive after.)
//  5. A failed Fetch goes into Errors[url] and the crawl carries on. Its
//     links are simply unknown, so there's nothing to follow.
//  6. When ctx is cancelled or times out, stop starting new work and return
//     what you already have. Crawl must not outlive the context by much.
//     Check ctx explicitly before each fetch — a `select` between "slot is
//     free" and "ctx is done" picks at random when both are ready, so it is
//     not on its own a reliable stop.
//  7. Return Pages sorted by URL, so the result is deterministic even though
//     the crawl wasn't.
//  8. No goroutine leaks: every goroutine you start must be able to exit.
//
// Errors must be non-nil (an empty map) even when nothing failed, so callers
// can range over it without a nil check.

func main() {
	// TODO: build a small in-memory Fetcher of your own — a
	// map[string]struct{Title string; Links []string} is plenty — and crawl
	// it. Print the number of pages found, then each page as
	// "<depth> <url> <title>".
	//
	// Then try it again with MaxDepth 0 and confirm you get exactly one page,
	// and with a context that times out mid-crawl.

	_ = context.Background()
	fmt.Print()
}

// EXPECTED BEHAVIOUR (depends on the graph you invent):
// 4
// 0 https://example.com Home
// 1 https://example.com/about About
// 1 https://example.com/blog Blog
// 2 https://example.com/blog/post-1 First post
