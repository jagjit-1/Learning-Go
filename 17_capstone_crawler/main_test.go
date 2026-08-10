package main

// ============================================================
// CHECKER for 17_capstone_crawler — run with:  go test -race
// ============================================================
// The crawler is driven by an in-memory Fetcher defined here, so nothing
// touches the network. That fake also counts calls and measures how many
// fetches were in flight at once, which is how the dedup and concurrency
// checks work.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = old
			w.Close()
			if rec := recover(); rec != nil {
				t.Errorf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

func mustFinish(t *testing.T, d time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(d):
		t.Fatalf("%s did not finish within %v.\n"+
			"  The test graph contains links back to pages already seen. Without a\n"+
			"  visited set — claimed BEFORE fetching, under a lock — the crawl goes\n"+
			"  round the cycle forever.", what, d)
	}
}

// --- the fake web ------------------------------------------------------

type testPage struct {
	title string
	links []string
	err   error
}

type testFetcher struct {
	pages map[string]testPage
	delay time.Duration

	mu    sync.Mutex
	calls map[string]int

	cur atomic.Int64
	max atomic.Int64
}

func newTestFetcher(pages map[string]testPage) *testFetcher {
	return &testFetcher{pages: pages, calls: make(map[string]int)}
}

func (f *testFetcher) Fetch(ctx context.Context, url string) (string, []string, error) {
	// A real fetcher would fail a cancelled request rather than serving it.
	select {
	case <-ctx.Done():
		return "", nil, ctx.Err()
	default:
	}

	f.mu.Lock()
	f.calls[url]++
	f.mu.Unlock()

	c := f.cur.Add(1)
	for {
		old := f.max.Load()
		if c <= old || f.max.CompareAndSwap(old, c) {
			break
		}
	}
	defer f.cur.Add(-1)

	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return "", nil, ctx.Err()
		}
	}

	p, ok := f.pages[url]
	if !ok {
		return "", nil, fmt.Errorf("no such page: %s", url)
	}
	if p.err != nil {
		return "", nil, p.err
	}
	return p.title, p.links, nil
}

func (f *testFetcher) callCount(url string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[url]
}

func (f *testFetcher) totalCalls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, c := range f.calls {
		n += c
	}
	return n
}

// A tree with back-edges to ancestors only, so every page has exactly one
// possible depth and the expected result is fully deterministic.
//
//	a (0) -> b, c
//	b (1) -> d, a
//	c (1) -> a
//	d (2) -> b
func siteGraph() map[string]testPage {
	return map[string]testPage{
		"a": {title: "Page A", links: []string{"b", "c"}},
		"b": {title: "Page B", links: []string{"d", "a"}},
		"c": {title: "Page C", links: []string{"a"}},
		"d": {title: "Page D", links: []string{"b"}},
	}
}

func urlsOf(pages []Page) []string {
	out := make([]string, len(pages))
	for i, p := range pages {
		out[i] = p.URL
	}
	return out
}

var (
	_ Fetcher = (*testFetcher)(nil)
	_         = Page{URL: "u", Title: "t", Depth: 0}
	_         = Crawler{Fetcher: nil, Workers: 1, MaxDepth: 1}
	_         = Result{Pages: nil, Errors: nil}
)

// --- Spec 1: find everything ------------------------------------------

func TestCrawlFindsEveryPage(t *testing.T) {
	f := newTestFetcher(siteGraph())
	c := &Crawler{Fetcher: f, Workers: 3, MaxDepth: 5}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl", func() { res = c.Crawl(context.Background(), "a") })

	if len(res.Pages) != 4 {
		t.Fatalf("found %d pages, want 4 (a, b, c, d).\n  got: %v",
			len(res.Pages), urlsOf(res.Pages))
	}

	want := []Page{
		{URL: "a", Title: "Page A", Depth: 0},
		{URL: "b", Title: "Page B", Depth: 1},
		{URL: "c", Title: "Page C", Depth: 1},
		{URL: "d", Title: "Page D", Depth: 2},
	}
	for i, w := range want {
		got := res.Pages[i]
		if got.URL != w.URL {
			t.Fatalf("spec 7: page %d is %q, want %q — Pages must be sorted by URL "+
				"so the result is deterministic.\n  got: %v", i, got.URL, w.URL,
				urlsOf(res.Pages))
		}
		if got.Title != w.Title {
			t.Errorf("page %q has Title %q, want %q", got.URL, got.Title, w.Title)
		}
		if got.Depth != w.Depth {
			t.Errorf("page %q has Depth %d, want %d (the start page is depth 0)",
				got.URL, got.Depth, w.Depth)
		}
	}
}

// --- Spec 2: dedup, and therefore termination -------------------------

func TestCrawlFetchesEachURLOnce(t *testing.T) {
	f := newTestFetcher(siteGraph())
	c := &Crawler{Fetcher: f, Workers: 4, MaxDepth: 5}

	mustFinish(t, 15*time.Second, "Crawl", func() { c.Crawl(context.Background(), "a") })

	for _, url := range []string{"a", "b", "c", "d"} {
		if n := f.callCount(url); n != 1 {
			t.Errorf("spec 2: %q was fetched %d times, want exactly 1.\n"+
				"  Mark a URL visited BEFORE fetching it, not after — otherwise two\n"+
				"  goroutines both check, both find it unvisited, and both fetch.", url, n)
		}
	}
	if n := f.totalCalls(); n != 4 {
		t.Errorf("spec 2: %d fetches in total, want 4", n)
	}
}

func TestCrawlSurvivesATightCycle(t *testing.T) {
	f := newTestFetcher(map[string]testPage{
		"x": {title: "X", links: []string{"y"}},
		"y": {title: "Y", links: []string{"x"}},
	})
	// A modest MaxDepth on purpose: without a visited set this graph branches
	// forever, and a low ceiling means you get a clean failure instead of the
	// test process eating all the memory on the machine.
	c := &Crawler{Fetcher: f, Workers: 2, MaxDepth: 12}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl over a cycle", func() {
		res = c.Crawl(context.Background(), "x")
	})

	if len(res.Pages) != 2 {
		t.Errorf("found %d pages, want 2", len(res.Pages))
	}
}

// --- Spec 3: depth limit -----------------------------------------------

func TestCrawlRespectsMaxDepth(t *testing.T) {
	cases := []struct {
		maxDepth int
		want     []string
	}{
		{0, []string{"a"}},
		{1, []string{"a", "b", "c"}},
		{2, []string{"a", "b", "c", "d"}},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("MaxDepth=%d", tc.maxDepth), func(t *testing.T) {
			f := newTestFetcher(siteGraph())
			c := &Crawler{Fetcher: f, Workers: 3, MaxDepth: tc.maxDepth}

			var res Result
			mustFinish(t, 15*time.Second, "Crawl", func() {
				res = c.Crawl(context.Background(), "a")
			})

			got := urlsOf(res.Pages)
			if len(got) != len(tc.want) {
				t.Fatalf("spec 3: MaxDepth %d found %v, want %v (depth 0 is the start "+
					"page alone)", tc.maxDepth, got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Fatalf("spec 3: MaxDepth %d found %v, want %v", tc.maxDepth, got, tc.want)
				}
			}
		})
	}
}

// --- Spec 4: bounded concurrency ---------------------------------------

func TestCrawlBoundsConcurrentFetches(t *testing.T) {
	// One root linking to 24 leaves, so there is plenty available to run
	// at once if the crawler doesn't cap itself.
	pages := map[string]testPage{}
	links := make([]string, 0, 24)
	for i := 0; i < 24; i++ {
		u := fmt.Sprintf("leaf-%02d", i)
		links = append(links, u)
		pages[u] = testPage{title: u}
	}
	pages["root"] = testPage{title: "Root", links: links}

	for _, workers := range []int{1, 2, 4} {
		f := newTestFetcher(pages)
		f.delay = 10 * time.Millisecond
		c := &Crawler{Fetcher: f, Workers: workers, MaxDepth: 3}

		mustFinish(t, 30*time.Second, "Crawl", func() { c.Crawl(context.Background(), "root") })

		if max := f.max.Load(); max > int64(workers) {
			t.Errorf("spec 4: with Workers=%d, %d fetches were in flight at once.\n"+
				"  Cap them with a buffered channel used as a semaphore: send a token\n"+
				"  before Fetch, receive it after. Its capacity is your limit.",
				workers, max)
		} else if workers > 1 && max < 2 {
			t.Errorf("spec 4: with Workers=%d, never more than %d fetch ran at a time "+
				"— the crawl is effectively sequential", workers, max)
		}
	}
}

func TestCrawlWorkersZero(t *testing.T) {
	f := newTestFetcher(siteGraph())
	c := &Crawler{Fetcher: f, Workers: 0, MaxDepth: 5}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl with Workers=0", func() {
		res = c.Crawl(context.Background(), "a")
	})

	if len(res.Pages) != 4 {
		t.Errorf("spec 4: Workers=0 should mean one worker, not zero — got %d pages",
			len(res.Pages))
	}
	if max := f.max.Load(); max > 1 {
		t.Errorf("spec 4: Workers=0 ran %d fetches at once, want at most 1", max)
	}
}

// --- Spec 5: errors ----------------------------------------------------

func TestCrawlRecordsErrorsAndCarriesOn(t *testing.T) {
	boom := errors.New("503 from origin")
	graph := siteGraph()
	graph["c"] = testPage{err: boom}

	f := newTestFetcher(graph)
	c := &Crawler{Fetcher: f, Workers: 3, MaxDepth: 5}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl with a failing page", func() {
		res = c.Crawl(context.Background(), "a")
	})

	got := urlsOf(res.Pages)
	if len(got) != 3 {
		t.Fatalf("spec 5: found %v, want the 3 pages that worked (a, b, d) — a "+
			"failed fetch must not stop the rest of the crawl", got)
	}
	for _, u := range got {
		if u == "c" {
			t.Error("spec 5: \"c\" failed to fetch, so it should not appear in Pages")
		}
	}

	err, ok := res.Errors["c"]
	if !ok {
		t.Fatalf("spec 5: Errors has no entry for \"c\". Got keys: %v", keysOf(res.Errors))
	}
	if !errors.Is(err, boom) {
		t.Errorf("spec 5: Errors[\"c\"] = %v, want the error the Fetcher returned", err)
	}
}

func TestCrawlErrorsIsNeverNil(t *testing.T) {
	f := newTestFetcher(siteGraph())
	c := &Crawler{Fetcher: f, Workers: 2, MaxDepth: 5}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl", func() { res = c.Crawl(context.Background(), "a") })

	if res.Errors == nil {
		t.Error("spec 5: Errors must be an initialised empty map when nothing failed, " +
			"not nil — callers should be able to range over it without checking")
	}
	if len(res.Errors) != 0 {
		t.Errorf("spec 5: nothing failed, but Errors = %v", res.Errors)
	}
}

func TestCrawlHandlesABrokenStartPage(t *testing.T) {
	f := newTestFetcher(map[string]testPage{})
	c := &Crawler{Fetcher: f, Workers: 2, MaxDepth: 5}

	var res Result
	mustFinish(t, 15*time.Second, "Crawl of a missing start page", func() {
		res = c.Crawl(context.Background(), "gone")
	})

	if len(res.Pages) != 0 {
		t.Errorf("the start page failed, so Pages should be empty — got %v",
			urlsOf(res.Pages))
	}
	if _, ok := res.Errors["gone"]; !ok {
		t.Error("the start page's failure should be recorded in Errors")
	}
}

// --- Spec 6: context ---------------------------------------------------

func TestCrawlStopsWhenTheContextExpires(t *testing.T) {
	// 40 pages in a chain at 30ms each is well over a second of work.
	pages := map[string]testPage{}
	for i := 0; i < 40; i++ {
		next := fmt.Sprintf("p-%02d", i+1)
		pages[fmt.Sprintf("p-%02d", i)] = testPage{
			title: fmt.Sprintf("Page %d", i),
			links: []string{next},
		}
	}
	pages["p-40"] = testPage{title: "Last"}

	f := newTestFetcher(pages)
	f.delay = 30 * time.Millisecond

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	c := &Crawler{Fetcher: f, Workers: 1, MaxDepth: 100}

	start := time.Now()
	var res Result
	mustFinish(t, 15*time.Second, "Crawl under a deadline", func() {
		res = c.Crawl(ctx, "p-00")
	})
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Errorf("spec 6: the context expired after 100ms but Crawl took %v.\n"+
			"  Select on ctx.Done() before starting each fetch, so the crawl winds "+
			"down instead of working through everything it has queued.", elapsed)
	}
	if len(res.Pages) >= 41 {
		t.Errorf("spec 6: the whole site (%d pages) was crawled despite the deadline",
			len(res.Pages))
	}
}

func TestCrawlWithAnAlreadyCancelledContext(t *testing.T) {
	f := newTestFetcher(siteGraph())
	c := &Crawler{Fetcher: f, Workers: 3, MaxDepth: 5}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var res Result
	mustFinish(t, 15*time.Second, "Crawl with a cancelled context", func() {
		res = c.Crawl(ctx, "a")
	})

	if len(res.Pages) > 1 {
		t.Errorf("spec 6: the context was cancelled before Crawl started, but it "+
			"still collected %d pages", len(res.Pages))
	}
}

// --- Spec 8: no leaks --------------------------------------------------

func TestCrawlDoesNotLeakGoroutines(t *testing.T) {
	pages := map[string]testPage{}
	links := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		u := fmt.Sprintf("n-%02d", i)
		links = append(links, u)
		pages[u] = testPage{title: u, links: []string{"hub"}}
	}
	pages["hub"] = testPage{title: "Hub", links: links}

	runtime.GC()
	baseline := goroutinesBelow(0, 500*time.Millisecond)

	const runs = 10
	mustFinish(t, 60*time.Second, "10 crawls, half of them cancelled", func() {
		for i := 0; i < runs; i++ {
			f := newTestFetcher(pages)
			f.delay = 5 * time.Millisecond
			c := &Crawler{Fetcher: f, Workers: 4, MaxDepth: 2}

			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			c.Crawl(ctx, "hub")
			cancel()
		}
	})

	after := goroutinesBelow(baseline+5, 10*time.Second)
	if after > baseline+5 {
		t.Errorf("spec 8: %d goroutines are still alive after %d crawls (started "+
			"from %d).\n"+
			"  Some goroutines are parked on a semaphore slot or a send that will\n"+
			"  never come. Every blocking operation needs a ctx.Done() escape, and\n"+
			"  Crawl must not return until the goroutines it started have exited.",
			after, runs, baseline)
	}
}

func goroutinesBelow(target int, d time.Duration) int {
	deadline := time.Now().Add(d)
	for {
		runtime.Gosched()
		n := runtime.NumGoroutine()
		if n <= target || time.Now().After(deadline) {
			return n
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func keysOf(m map[string]error) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// --- main()'s narration ------------------------------------------------

func TestOutput(t *testing.T) {
	var out string
	mustFinish(t, 30*time.Second, "main()", func() { out = captureStdout(t, main) })

	lines := []string{}
	for _, l := range strings.Split(out, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		t.Fatal("main() printed nothing — build a small Fetcher of your own and crawl it")
	}

	countRe := regexp.MustCompile(`^\d+$`)
	pageRe := regexp.MustCompile(`^(\d+) (\S+) (.+)$`)

	countAt := -1
	for i, l := range lines {
		if countRe.MatchString(l) {
			countAt = i
			break
		}
	}
	if countAt < 0 {
		t.Fatalf("expected a line with just the number of pages found.\n"+
			"  your output was:\n%s", indent(out))
	}

	var urls []string
	for _, l := range lines[countAt+1:] {
		m := pageRe.FindStringSubmatch(l)
		if m == nil {
			break
		}
		urls = append(urls, m[2])
	}

	if len(urls) == 0 {
		t.Fatalf("expected lines shaped like \"<depth> <url> <title>\" after the "+
			"count.\n  your output was:\n%s", indent(out))
	}

	want, _ := strconv.Atoi(lines[countAt])
	if want != len(urls) {
		t.Errorf("you printed %d as the page count but listed %d pages",
			want, len(urls))
	}

	if !sort.StringsAreSorted(urls) {
		t.Errorf("spec 7: the pages you printed are not in URL order: %v.\n"+
			"  Crawl must sort Pages by URL before returning.", urls)
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
