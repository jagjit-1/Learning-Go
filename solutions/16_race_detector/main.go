package main

import (
	"fmt"
	"os"
	"sort"
	"sync"
	"sync/atomic"
)

// racyCount is DELIBERATELY broken. Nothing calls it unless SHOW_RACE is set.
func racyCount(n int) int {
	counter := 0

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter++ // read, add, write — three steps, no ordering
		}()
	}
	wg.Wait()

	return counter
}

func countMutex(n int) int {
	var (
		mu      sync.Mutex
		counter int
		wg      sync.WaitGroup
	)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			counter++
			mu.Unlock()
		}()
	}
	wg.Wait()

	return counter
}

func countAtomic(n int) int {
	var (
		counter atomic.Int64
		wg      sync.WaitGroup
	)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Add(1)
		}()
	}
	wg.Wait()

	return int(counter.Load())
}

func countChannel(n int) int {
	ch := make(chan int)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch <- 1
		}()
	}
	go func() {
		wg.Wait()
		close(ch)
	}()

	// Only this goroutine ever touches `total`, so there is nothing to guard.
	total := 0
	for v := range ch {
		total += v
	}
	return total
}

func appendConcurrent(n int) []int {
	var (
		mu  sync.Mutex
		out []int
		wg  sync.WaitGroup
	)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			mu.Lock()
			out = append(out, i)
			mu.Unlock()
		}()
	}
	wg.Wait()

	return out
}

type Registry struct {
	mu    sync.RWMutex
	names map[string]int
}

func NewRegistry() *Registry {
	return &Registry{names: make(map[string]int)}
}

func (r *Registry) Register(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.names[name]++
}

func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.names)
}

func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]string, 0, len(r.names)) // a copy, not the live keys
	for name := range r.names {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func main() {
	if os.Getenv("SHOW_RACE") != "" {
		fmt.Println(racyCount(1000))
		return
	}

	fmt.Println(countMutex(1000))
	fmt.Println(countAtomic(1000))
	fmt.Println(countChannel(1000))

	fmt.Println(len(appendConcurrent(500)))

	r := NewRegistry()
	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Register(fmt.Sprintf("name-%d", i))
		}()
	}
	wg.Wait()

	fmt.Println(r.Count())
	fmt.Println(r.Names()[0])
}
