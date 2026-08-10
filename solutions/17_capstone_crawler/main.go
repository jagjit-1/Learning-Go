package main

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

type Fetcher interface {
	Fetch(ctx context.Context, url string) (title string, links []string, err error)
}

type Page struct {
	URL   string
	Title string
	Depth int
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

func (c *Crawler) Crawl(ctx context.Context, start string) Result {
	workers := c.Workers
	if workers <= 0 {
		workers = 1
	}

	// Buffered channel as a semaphore: at most `workers` tokens exist, so at
	// most `workers` goroutines can be inside Fetch at once.
	sem := make(chan struct{}, workers)

	var (
		mu      sync.Mutex
		visited = make(map[string]bool)
		pages   []Page
		errs    = make(map[string]error)
		wg      sync.WaitGroup
	)

	var crawl func(url string, depth int)
	crawl = func(url string, depth int) {
		defer wg.Done()

		if depth > c.MaxDepth {
			return
		}

		// Check the context explicitly. Relying on the select below is not
		// enough: when a semaphore slot is free AND the context is done, both
		// cases are ready and select picks one at random.
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Claim the URL before doing any work, so two goroutines that reach
		// the same link at once don't both fetch it.
		mu.Lock()
		if visited[url] {
			mu.Unlock()
			return
		}
		visited[url] = true
		mu.Unlock()

		// Acquire a slot, unless the context ended while we waited.
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		title, links, err := c.Fetcher.Fetch(ctx, url)
		<-sem

		if err != nil {
			mu.Lock()
			errs[url] = err
			mu.Unlock()
			return
		}

		mu.Lock()
		pages = append(pages, Page{URL: url, Title: title, Depth: depth})
		mu.Unlock()

		for _, link := range links {
			wg.Add(1) // before the go statement
			go crawl(link, depth+1)
		}
	}

	wg.Add(1)
	go crawl(start, 0)
	wg.Wait()

	sort.Slice(pages, func(i, j int) bool { return pages[i].URL < pages[j].URL })

	return Result{Pages: pages, Errors: errs}
}

// --- a small in-memory Fetcher for main() ---------------------------

type fakePage struct {
	title string
	links []string
}

type mapFetcher map[string]fakePage

func (m mapFetcher) Fetch(ctx context.Context, url string) (string, []string, error) {
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	p, ok := m[url]
	if !ok {
		return "", nil, fmt.Errorf("not found: %s", url)
	}
	return p.title, p.links, nil
}

func main() {
	web := mapFetcher{
		"https://example.com": {"Home", []string{
			"https://example.com/about",
			"https://example.com/blog",
		}},
		"https://example.com/about": {"About", []string{"https://example.com"}},
		"https://example.com/blog": {"Blog", []string{
			"https://example.com/blog/post-1",
			"https://example.com",
		}},
		"https://example.com/blog/post-1": {"First post", []string{"https://example.com/blog"}},
	}

	c := &Crawler{Fetcher: web, Workers: 4, MaxDepth: 3}
	res := c.Crawl(context.Background(), "https://example.com")

	fmt.Println(len(res.Pages))
	for _, p := range res.Pages {
		fmt.Printf("%d %s %s\n", p.Depth, p.URL, p.Title)
	}

	shallow := &Crawler{Fetcher: web, Workers: 4, MaxDepth: 0}
	fmt.Println(len(shallow.Crawl(context.Background(), "https://example.com").Pages))
}
