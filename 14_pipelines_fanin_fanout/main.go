package main

import (
	"fmt"
	"sort"
	"sync"
)

// ============================================================
// CONCEPT: pipelines, fan-out, fan-in — and goroutine leaks
// ============================================================
//
// A PIPELINE is a chain of stages connected by channels. Every stage has the
// same shape: take input channels, return an output channel, do the work in a
// goroutine, close the output when the input runs dry.
//
//   func gen(nums ...int) <-chan int {
//       out := make(chan int)
//       go func() {
//           defer close(out)          // ALWAYS close what you created
//           for _, n := range nums {
//               out <- n
//           }
//       }()
//       return out                    // returns immediately; work happens behind it
//   }
//
//   func sq(in <-chan int) <-chan int {
//       out := make(chan int)
//       go func() {
//           defer close(out)
//           for n := range in {       // ends when `in` is closed
//               out <- n * n
//           }
//       }()
//       return out
//   }
//
// Composing them reads like a shell pipe, and nothing is buffered in memory
// along the way — values flow through one at a time:
//
//   for v := range sq(sq(gen(1, 2, 3))) { fmt.Println(v) }
//
// FAN-OUT — several goroutines reading the SAME input channel, to parallelise
// a slow stage. No special API: just call the stage more than once on one
// input, and the values get shared out between them.
//
//   in := gen(nums...)
//   a, b, c := sq(in), sq(in), sq(in)      // three workers on one source
//
// FAN-IN — merging those back into one channel. This is the one that needs a
// WaitGroup: you can only close the merged output once every contributor is
// finished.
//
//   func merge(cs ...<-chan int) <-chan int {
//       out := make(chan int)
//       var wg sync.WaitGroup
//       for _, c := range cs {
//           wg.Add(1)
//           go func(c <-chan int) {
//               defer wg.Done()
//               for v := range c { out <- v }
//           }(c)
//       }
//       go func() { wg.Wait(); close(out) }()
//       return out
//   }
//
// GOROUTINE LEAKS — the part everyone gets wrong.
//
// A goroutine blocked forever on a channel send is never collected. It is not
// like a leaked object the GC will notice: the runtime has no idea it's
// pointless. It holds its stack and every variable it captured, for the life
// of the process.
//
// The pipelines above leak whenever the CONSUMER stops early:
//
//   for v := range sq(gen(1, 2, 3, 4, 5)) {
//       if v > 4 { break }        // gen and sq are now stuck sending, forever
//   }
//
// The fix is a `done` channel, closed by the consumer, that every stage
// selects on while sending:
//
//   select {
//   case out <- v:
//   case <-done:
//       return                    // consumer left; abandon the send and exit
//   }
//
// `defer close(out)` still runs on that path, so the whole chain unwinds.
// (Exercise 15 replaces `done` with context.Context, which is the same idea
// with cancellation reasons, deadlines and propagation built in — but the
// mechanism underneath is exactly this select.)

// TODO 1: write `func gen(nums ...int) <-chan int` — the generator above.
func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, val := range nums {
			out <- val
		}
	}()
	return out
}

// TODO 2: write `func sq(in <-chan int) <-chan int` — squares every value.
func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

// TODO 3: write `func merge(cs ...<-chan int) <-chan int` — fan-in. It must
// work for zero channels too (return a channel that's simply closed).
func merge(cs ...<-chan int) <-chan int {
	var wg sync.WaitGroup
	out := make(chan int)

	for _, c := range cs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for in := range c {
				out <- in
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}

// TODO 4: write `func pipeline(nums []int, fanOut int) []int` that runs
// gen -> `fanOut` parallel sq stages -> merge, and collects everything into a
// slice. Order will not match the input — that's expected, the checker sorts.
// Treat fanOut <= 0 as 1.
func pipeline(nums []int, fanOut int) []int {
	if fanOut <= 0 {
		fanOut = 1
	}

	in := gen(nums...)
	outChans := []<-chan int{}
	for range fanOut {
		outChans = append(outChans, sq(in))
	}
	res := []int{}
	for out := range merge(outChans...) {
		res = append(res, out)
	}
	return res
}

func pipelineDone(nums []int, fanOut int, done chan struct{}) <-chan int {
	if fanOut <= 0 {
		fanOut = 1
	}

	in := genDone(done, nums...)
	outChans := []<-chan int{}
	for range fanOut {
		outChans = append(outChans, sqDone(done, in))
	}

	return merge(outChans...)
}

// TODO 5: now the leak-free versions. Write
//
//	func genDone(done <-chan struct{}, nums ...int) <-chan int
//	func sqDone(done <-chan struct{}, in <-chan int) <-chan int
//
// Same as TODOs 1 and 2, but every send is a select against `done`.
func genDone(done <-chan struct{}, nums ...int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)

		for _, n := range nums {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}()

	return out
}

func sqDone(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)

	go func() {
		defer close(out)
		for i := range in {
			select {
			case out <- i * i:
			case <-done:
				return
			}
		}
	}()
	return out
}

// TODO 6: write `func firstN(in <-chan int, n int, done chan struct{}) []int`
// that takes n values and then CLOSES done to shut the pipeline behind it.
// Close it with `defer close(done)` so an early return still cleans up.
// If `in` runs dry first, return what you got. For n <= 0 return empty
// without receiving anything at all.
func firstN(in <-chan int, n int, done chan struct{}) []int {
	res := []int{}
	if n == 0 {
		return res
	}
	defer close(done)
	cnt := 0
	for val := range in {
		cnt++
		res = append(res, val)

		if cnt == n {
			return res
		}
	}

	return res
}

func main() {
	// TODO 7: print the sorted output of pipeline over 1..8 with fanOut 3.
	// (sort.Ints is in the "sort" package.)
	input := []int{}
	for i := range 8 {
		input = append(input, i+1)
	}

	output := pipeline(input, 3)
	sort.Ints(output)
	fmt.Println(output)

	// TODO 8: build a done-aware pipeline over 1..1000, take only the first 3
	// values with firstN, and print them. Everything upstream should shut
	// down instead of blocking forever on a send nobody will receive.
	input2 := []int{}
	done := make(chan struct{})
	for i := range 1000 {
		input2 = append(input2, i+1)
	}

	// Fanout needs to be 1 here because the order otherwise is undeterministic
	res := firstN(pipelineDone(input2, 1, done), 3, done)
	fmt.Println(res)
}

// EXPECTED OUTPUT:
// [1 4 9 16 25 36 49 64]
// [1 4 9]
