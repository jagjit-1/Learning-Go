package main

import (
	"fmt"
	"sync"
)

type Result struct {
	Job   int
	Value int
	Err   error
}

func RunPool(jobs []int, workers int, fn func(int) (int, error)) []Result {
	if workers <= 0 {
		workers = 1
	}

	in := make(chan int)
	out := make(chan Result)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range in {
				v, err := fn(j)
				out <- Result{Job: j, Value: v, Err: err}
			}
		}()
	}

	go func() {
		defer close(in)
		for _, j := range jobs {
			in <- j
		}
	}()

	go func() {
		wg.Wait()
		close(out) // every worker has returned, so nobody can send any more
	}()

	results := make([]Result, 0, len(jobs))
	for r := range out {
		results = append(results, r)
	}
	return results
}

func RunPoolOrdered(jobs []int, workers int, fn func(int) int) []int {
	if workers <= 0 {
		workers = 1
	}

	out := make([]int, len(jobs))
	indices := make(chan int)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range indices {
				out[i] = fn(jobs[i]) // distinct index per job: no lock needed
			}
		}()
	}

	for i := range jobs {
		indices <- i
	}
	close(indices)
	wg.Wait()

	return out
}

func FirstError(jobs []int, workers int, fn func(int) error) error {
	if workers <= 0 {
		workers = 1
	}

	in := make(chan int)
	done := make(chan struct{})
	var closeOnce sync.Once
	stop := func() { closeOnce.Do(func() { close(done) }) }

	var (
		mu       sync.Mutex
		firstErr error
	)

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range in {
				if err := fn(j); err != nil {
					mu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					mu.Unlock()
					stop()
					return
				}
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}

feed:
	for _, j := range jobs {
		select {
		case in <- j:
		case <-done:
			break feed
		}
	}
	close(in)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	return firstErr
}

func main() {
	jobs := make([]int, 10)
	for i := range jobs {
		jobs[i] = i + 1
	}

	total := 0
	for _, r := range RunPool(jobs, 3, func(n int) (int, error) { return n * n, nil }) {
		total += r.Value
	}
	fmt.Println(total)

	fmt.Println(RunPoolOrdered(jobs, 3, func(n int) int { return n * 2 }))

	err := FirstError(jobs, 3, func(n int) error {
		if n == 7 {
			return fmt.Errorf("job %d failed", n)
		}
		return nil
	})
	fmt.Println(err)

	fmt.Println(FirstError(jobs, 3, func(n int) error { return nil }))
}
