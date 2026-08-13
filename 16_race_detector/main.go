package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
)

// ============================================================
// CONCEPT: data races, and the three ways out
// ============================================================
//
// A DATA RACE is two goroutines touching the same memory with no ordering
// between them, at least one of them writing. Not "sometimes wrong" — the Go
// memory model says a racy program has NO defined behaviour at all.
//
// The classic:
//
//   counter++
//
// One line, three machine steps: read counter, add 1, write it back. Two
// goroutines can both read 7, both compute 8, both store 8. Two increments,
// one result.
//
// THE RACE DETECTOR is built into the toolchain:
//
//   go run -race .
//   go test -race
//
// It instruments memory access and reports the two conflicting accesses with
// both stack traces. It has no false positives — anything it reports is a
// real race. But it only sees races that ACTUALLY HAPPEN on that run, so
// exercise your code properly under it. This is why the checkers in this set
// hammer things with hundreds of goroutines rather than two.
//
// This file ships with a deliberately broken racyCount for you to look at.
// Nothing calls it unless you ask:
//
//   SHOW_RACE=1 go run -race .
//
// Read the report it prints. The "Previous write" and "Write at" stacks are
// the two goroutines that collided.
//
// THREE WAYS TO FIX A RACE, in the order you should reach for them:
//
// 1. DON'T SHARE. Give each goroutine its own data and combine at the end,
//    or hand ownership through a channel. A value only one goroutine can
//    reach cannot be raced on. This is the one to try first.
//
// 2. sync/atomic. One variable, lock-free, cheap. Only covers a single word.
//
// 3. sync.Mutex. Everything else — especially when two or more fields have to
//    stay consistent with each other.
//
// MAPS ARE SPECIAL. A concurrent map write doesn't just race, it aborts the
// process:
//
//   fatal error: concurrent map writes
//
// That one is NOT a panic. recover() cannot catch it, and it fires with or
// without -race. Any map touched by more than one goroutine needs a lock (or
// sync.Map, for the narrow cases where that's a better fit).
//
// LEAKING THE PROTECTED STATE. This method looks safe and is not:
//
//   func (r *Registry) Names() []string {
//       r.mu.RLock()
//       defer r.mu.RUnlock()
//       return r.names          // the caller now has the live slice
//   }
//
// The lock is released on return, and the caller walks away holding a slice
// that shares its backing array with the one you're still guarding. Return a
// COPY. The same goes for maps and for pointers to internal structs.

// TODO 1: write `func racyCount(n int) int` — n goroutines, each doing
// `counter++` on ONE shared int with nothing protecting it, a WaitGroup to
// wait for them. This one is MEANT to be broken. Run `SHOW_RACE=1 go run -race .`
// and read the report before you go any further.
func racyCount(n int) int {
	count := 0
	var wg sync.WaitGroup

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count++
		}()
	}

	wg.Wait()
	return count
}

// TODO 2: write `func countMutex(n int) int` — same thing, correct, using
// sync.Mutex.
func countMutex(n int) int {
	count := 0
	var wg sync.WaitGroup
	var mut sync.Mutex

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer mut.Unlock()

			mut.Lock()
			count++
		}()
	}

	wg.Wait()
	return count
}

// TODO 3: write `func countAtomic(n int) int` — same thing with an
// atomic.Int64 and no lock at all.
func countAtomic(n int) int {
	var count atomic.Int64
	var wg sync.WaitGroup

	for range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			count.Add(1)
		}()
	}

	wg.Wait()
	return int(count.Load())
}

// TODO 4: write `func countChannel(n int) int` — same thing with no shared
// variable whatsoever: each goroutine sends 1 on a channel and a single
// collector adds them up. This is fix #1, "don't share".
func countChannel(n int) int {
	var wg sync.WaitGroup
	ch := make(chan int)

	for range n {
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
	count := 0
	for range ch {
		count++
	}
	return count
}

// TODO 5: write `func appendConcurrent(n int) []int` — n goroutines each
// appending their own index to one shared slice. append READS the slice
// header, may reallocate, and WRITES it back, so it needs the same protection
// an int does. Order doesn't matter; all n values must survive.
func appendConcurrent(n int) []int {
	nums := []int{}
	var wg sync.WaitGroup
	var mut sync.Mutex

	for i := range n {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer mut.Unlock()
			mut.Lock()
			nums = append(nums, i)
		}()
	}

	wg.Wait()
	return nums
}

// TODO 6: define `type Registry` holding a map[string]int and a sync.RWMutex,
// with `NewRegistry() *Registry`, `Register(name string)` (bumps that name's
// count), `Count() int` (how many distinct names), and `Names() []string`
// returning a SORTED COPY of the keys. Returning the live map or an
// unsorted view of it defeats the lock.
type Registry struct {
	Value map[string]int
	Mut   sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{Value: map[string]int{}}
}

func (reg *Registry) Register(name string) {
	reg.Mut.Lock()
	defer reg.Mut.Unlock()

	reg.Value[name]++
}

func (reg *Registry) Count() int {
	reg.Mut.Lock()
	defer reg.Mut.Unlock()
	count := 0
	for range reg.Value {
		count++
	}

	return count
}

func (reg *Registry) Names() []string {
	reg.Mut.Lock()
	defer reg.Mut.Unlock()
	res := []string{}
	for key := range reg.Value {
		res = append(res, key)
	}
	slices.Sort(res)
	return res
}

func main() {
	// TODO 7: if os.Getenv("SHOW_RACE") != "", call racyCount(1000) and print
	// the result, then return. Leave it out of the normal path so the rest of
	// the program stays race-free.
	if os.Getenv("SHOW_RACE") != "" {
		fmt.Println(racyCount(1000))
	}

	// TODO 8: print countMutex(1000), countAtomic(1000) and countChannel(1000).
	// All three must print 1000, every time.
	fmt.Println(countMutex(1000))
	fmt.Println(countAtomic(1000))
	fmt.Println(countChannel(1000))
	// TODO 9: print len(appendConcurrent(500)).
	fmt.Println(len(appendConcurrent(500)))
	// TODO 10: register 200 distinct names concurrently and print Count(),
	// then print the first entry from Names().
	registry := NewRegistry()
	var wg sync.WaitGroup
	for i := range 200 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			registry.Register("name-" + strconv.Itoa(i))
		}()
	}

	wg.Wait()
	fmt.Println(registry.Count())
	names := registry.Names()
	fmt.Println(names[0])
}

// EXPECTED OUTPUT (with SHOW_RACE unset):
// 1000
// 1000
// 1000
// 500
// 200
// name-0
//
// EXPECTED with SHOW_RACE=1 go run -race . :
// a WARNING: DATA RACE report, and a number that is usually NOT 1000
