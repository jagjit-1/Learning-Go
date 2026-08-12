package main

import (
	"errors"
	"fmt"
	"sync"
)

// ============================================================
// CONCEPT: worker pools — bounded concurrency
// ============================================================
//
// "Spawn a goroutine per item" works fine for ten items and falls over at a
// million: you'd open a million files, or hammer an API with a million
// simultaneous requests. A worker pool fixes the concurrency at N no matter
// how many items there are.
//
// The shape: one jobs channel, N workers reading it, one results channel.
//
//   jobs := make(chan int)
//   results := make(chan int)
//
//   for w := 0; w < workers; w++ {
//       go func() {
//           for j := range jobs {      // every worker ranges the SAME channel
//               results <- process(j)
//           }
//       }()
//   }
//
// Several goroutines receiving from one channel is not a mistake — each value
// goes to exactly one of them. The channel IS the queue, and the runtime does
// the hand-out. That's the whole trick.
//
// FEEDING AND FINISHING. Two things have to happen in order:
//
//   1. Close `jobs` when there's no more work. That ends every worker's
//      `range`, which is how workers know to stop.
//   2. Close `results` only after ALL workers have finished — otherwise a
//      worker sends on a closed channel and panics. A WaitGroup plus a small
//      closer goroutine is the standard way:
//
//   go func() {
//       for _, j := range jobList { jobs <- j }
//       close(jobs)
//   }()
//
//   go func() {
//       wg.Wait()
//       close(results)     // safe: every worker has returned
//   }()
//
//   for r := range results { collect(r) }
//
// Why does the feeder need its own goroutine? If `results` is unbuffered and
// you feed jobs on the main goroutine, the workers block sending results while
// you block sending jobs. Classic deadlock. Feed in a goroutine, drain on the
// main one.
//
// RESULT ORDER is not input order — that's the point of running things in
// parallel. If you need the original order, don't collect through a channel:
// preallocate `out := make([]T, len(jobs))` and have each worker write
// out[i]. Concurrent writes to DIFFERENT indices of a slice need no lock;
// the slice header isn't being modified, only distinct elements.
//
// STOPPING EARLY. When one job fails and the rest are pointless, close a
// `done` channel and have workers select on it. Use sync.Once to make sure
// you only close it once — closing a closed channel panics.
//
//   var once sync.Once
//   stop := func() { once.Do(func() { close(done) }) }

// TODO 1: define `type Result struct { Job int; Value int; Err error }`
type Result struct {
	Job   int
	Value int
	Err   error
}

// TODO 2: write
//
//	func RunPool(jobs []int, workers int, fn func(int) (int, error)) []Result
//
// Run fn over every job with at most `workers` running at a time. Return one
// Result per job, in any order, carrying whatever fn returned. Treat
// workers <= 0 as 1. An empty job list must return promptly, not deadlock.
func RunPool(jobs []int, workers int, fn func(int) (int, error)) []Result {
	if len(jobs) == 0 {
		return []Result{}
	}

	if workers <= 0 {
		workers = 1
	}
	jobQueue := make(chan int)
	go func() {
		for _, job := range jobs {
			jobQueue <- job
		}
		close(jobQueue)
	}()
	var wg sync.WaitGroup
	result := make(chan Result)

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for job := range jobQueue {
				val, ok := fn(job)
				result <- Result{Job: job, Value: val, Err: ok}
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return collect(result)
}

func collect(ch <-chan Result) []Result {
	res := []Result{}

	for r := range ch {
		res = append(res, r)
	}
	return res
}

// TODO 3: write
// func RunPoolOrdered(jobs []int, workers int, fn func(int) int) []int
// Same bounded concurrency, but the output must be in INPUT order. Use the
// preallocated-slice trick rather than a results channel.
func RunPoolOrdered(jobs []int, workers int, fn func(int) int) []int {
	if len(jobs) == 0 {
		return []int{}
	}

	if workers <= 0 {
		workers = 1
	}
	jobQueue := make(chan JobRequest)
	go func() {
		for idx, job := range jobs {
			jobQueue <- JobRequest{Index: idx, Job: job}
		}
		close(jobQueue)
	}()
	result := make([]int, len(jobs))
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for jobRequest := range jobQueue {
				val := fn(jobRequest.Job)
				result[jobRequest.Index] = val
			}
		}()

	}

	wg.Wait()

	return result
}

type JobRequest struct {
	Index int
	Job   int
}

// TODO 4: write
//
//	func FirstError(jobs []int, workers int, fn func(int) error) error
//
// Run the jobs, and as soon as one fn returns an error, stop handing out new
// work and return that error. Return nil if every job succeeded. The
// remaining jobs must be ABANDONED, not merely ignored — the whole point is
// not doing the work.
func FirstError(jobs []int, workers int, fn func(int) error) error {
	if len(jobs) == 0 {
		return nil
	}

	if workers <= 0 {
		workers = 1
	}
	var once sync.Once
	var wg sync.WaitGroup
	ch := make(chan int)
	done := make(chan struct{})

	// Producer
	go func() {
		defer close(ch)
		for _, job := range jobs {
			select {
			case ch <- job:
			case <-done:
				return
			}
		}
	}()

	var resErr error

	// Consumer
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range ch {
				select {
				case <-done:
					return
				default:
					err := fn(job)
					if err != nil {
						once.Do(func() {
							resErr = err
							close(done)
						})
						return
					}
				}
			}
		}()
	}

	wg.Wait()

	return resErr
}

func main() {
	// TODO 5: RunPool over 1..10 with 3 workers, fn = n*n and no errors.
	// Sum the Values and print the total (sum of squares 1..10 = 385).
	fn := func(n int) (int, error) { return n * n, nil }
	nums := []int{}
	for i := range 10 {
		nums = append(nums, i+1)
	}
	res := RunPool(nums, 3, fn)
	total := 0
	for _, r := range res {
		total += r.Value
	}
	fmt.Println(total)
	// TODO 6: RunPoolOrdered over 1..10 with 3 workers, fn = n*2. Print the
	// slice — it must come out in order even though the work didn't.
	fn1 := func(n int) int { return n * 2 }
	val := RunPoolOrdered(nums, 3, fn1)
	fmt.Println(val)
	// TODO 7: FirstError over 1..10 where fn fails on 7. Print the error.
	fn2 := func(n int) error {
		if n == 7 {
			return errors.New("job 7 failed")
		}
		return nil
	}

	err := FirstError(nums, 3, fn2)
	fmt.Println(err)
	// TODO 8: FirstError over 1..10 where nothing fails. Print the nil result.
	fn3 := func(n int) error {
		return nil
	}

	err = FirstError(nums, 3, fn3)
	fmt.Println(err)
}

// EXPECTED OUTPUT:
// 385
// [2 4 6 8 10 12 14 16 18 20]
// job 7 failed
// <nil>
