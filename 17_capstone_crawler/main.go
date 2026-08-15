package main

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
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

type ConcurrentMap struct {
	Mut   sync.Mutex
	Value map[string]struct{}
}

func (cm *ConcurrentMap) Store(key string) bool {
	cm.Mut.Lock()
	defer cm.Mut.Unlock()
	_, ok := cm.Value[key]

	if ok {
		return false
	}

	cm.Value[key] = struct{}{}
	return true
}

type Fetcher interface {
	Fetch(ctx context.Context, url string) (title string, links []string, err error)
}

type InMemoryFetcher struct {
	UrlTitleMapping map[string]string
	UrlLinksMapping map[string][]string
}

func (f *InMemoryFetcher) Fetch(ctx context.Context, url string) (title string, links []string, err error) {
	title, ok := f.UrlTitleMapping[url]

	if !ok {
		return "", []string{}, errors.New("Invalid url")
	}

	links, linksFound := f.UrlLinksMapping[url]

	if !linksFound {
		return title, []string{}, nil
	}

	return title, links, nil
}

type Page struct {
	URL   string
	Title string
	Depth int
}

type PageResult struct {
	Page Page
	Err  error
}

type Crawler struct {
	Fetcher  Fetcher
	Workers  int
	MaxDepth int
}

type Result struct {
	Pages  []Page
	Errors map[string]error
}

func ProduceJobs(depth int, links []string, out chan<- Job, seen *ConcurrentMap, wg *sync.WaitGroup) {
	for _, link := range links {
		ok := seen.Store(link)

		if ok {
			wg.Add(1)
			out <- Job{depth: depth, url: link}
		}
	}
}

func ProduceResult(title string, url string, depth int, pages chan<- PageResult, err error) {
	page := Page{Title: title, URL: url, Depth: depth}
	res := PageResult{Page: page, Err: err}
	pages <- res
}

// Fetch Job Executer
func FetchJobExecuter(ctx context.Context, crawler *Crawler, pages chan PageResult, jobs chan Job, workers int, wg *sync.WaitGroup, seen *ConcurrentMap) {
	gate := make(chan struct{}, workers)

	for job := range jobs {
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
				title, links, err, ok := FetchJob(ctx, crawler, job.url, pages, gate)

				if ok {
					ProduceResult(title, job.url, job.depth, pages, err)

					if job.depth < crawler.MaxDepth && err == nil {
						ProduceJobs(job.depth+1, links, jobs, seen, wg)
					}
				}
			}
		}()
	}

}

func FetchJob(ctx context.Context, crawler *Crawler, url string, pages chan PageResult, gate chan struct{}) (string, []string, error, bool) {
	// Fetch page
	select {
	case gate <- struct{}{}:
		title, links, err := crawler.Fetcher.Fetch(ctx, url)
		<-gate
		return title, links, err, true
	case <-ctx.Done():
		return "", []string{}, nil, false
	}
}

func (crawler *Crawler) Crawl(ctx context.Context, start string) Result {
	workers := 1
	if crawler.Workers > 0 {
		workers = crawler.Workers
	}

	jobs := make(chan Job)
	pages := make(chan PageResult)
	seen := ConcurrentMap{Value: make(map[string]struct{})}

	var wg sync.WaitGroup

	go FetchJobExecuter(ctx, crawler, pages, jobs, workers, &wg, &seen)

	wg.Add(1)
	seen.Store(start)
	jobs <- Job{url: start}

	go func() {
		wg.Wait()
		defer close(jobs)
		defer close(pages)
	}()

	results := Result{Errors: make(map[string]error)}

	for page := range pages {
		if page.Err != nil {
			results.Errors[page.Page.URL] = page.Err
		} else {
			results.Pages = append(results.Pages, page.Page)
		}
	}
	slices.SortFunc(results.Pages, func(a Page, b Page) int {
		return strings.Compare(a.URL, b.URL)
	})
	return results
}

type Job struct {
	url   string
	depth int
}

func newGraphFetcher() *InMemoryFetcher {
	return &InMemoryFetcher{
		UrlTitleMapping: map[string]string{
			"A": "Home",
			"B": "About",
			"C": "Blog",
			"D": "Post1",
		},
		UrlLinksMapping: map[string][]string{
			"A": {"B", "C", "missing"}, // "missing" -> error, cycle via C->A
			"B": {"A"},                 // cycle back to start
			"C": {"D", "A"},            // cycle back to start
			"D": {},
		},
	}
}

func main() {
	// TODO: build a small in-memory Fetcher of your own — a
	// map[string]struct{Title string; Links []string} is plenty — and crawl
	// it. Print the number of pages found, then each page as
	// "<depth> <url> <title>".
	crawler := &Crawler{MaxDepth: 3, Fetcher: newGraphFetcher(), Workers: 3}
	res := crawler.Crawl(context.Background(), "A")
	fmt.Println(len(res.Pages))
	for _, page := range res.Pages {
		if res.Errors[page.URL] == nil {
			fmt.Println(page.Depth, page.URL, page.Title)
		}
	}
	// Then try it again with MaxDepth 0 and confirm you get exactly one page,
	// and with a context that times out mid-crawl.
	crawler1 := &Crawler{MaxDepth: 0, Fetcher: newGraphFetcher(), Workers: 10}
	res1 := crawler1.Crawl(context.Background(), "A")

	if len(res1.Pages) != 1 {
		fmt.Println("Validation for depth 0 failed")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 0)
	cancel()
	res2 := crawler.Crawl(ctx, "A")

	if len(res2.Pages) != 0 {
		fmt.Println("Validation for ctx cancel failed")
	}
	fmt.Print()
}

// EXPECTED BEHAVIOUR (depends on the graph you invent):
// 4
// 0 https://example.com Home
// 1 https://example.com/about About
// 1 https://example.com/blog Blog
// 2 https://example.com/blog/post-1 First post
