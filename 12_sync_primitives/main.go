package main

import "fmt"

// ============================================================
// CONCEPT: sync — waiting, locking, doing-once
// ============================================================
//
// Channels are the default answer in Go, but sometimes you just want to
// guard a variable or wait for a batch of goroutines. That's the sync package.
//
// The rule of thumb from the Go team: "Don't communicate by sharing memory;
// share memory by communicating." Use channels to hand ownership of data
// between goroutines. Use a mutex when several goroutines genuinely need to
// touch the SAME piece of state.
//
// sync.WaitGroup — wait for N goroutines to finish:
//
//   var wg sync.WaitGroup
//   for _, item := range items {
//       wg.Add(1)                 // BEFORE starting the goroutine, never inside it
//       go func() {
//           defer wg.Done()       // defer, so a panic still decrements
//           process(item)
//       }()
//   }
//   wg.Wait()                     // blocks until the counter hits zero
//
// `wg.Add(1)` inside the goroutine is a classic bug: Wait can run before the
// goroutine is scheduled, see a counter of 0, and return early.
//
// (Since Go 1.22 the loop variable is fresh per iteration, so capturing
// `item` above is safe. In older Go you had to write `go func(item T){...}(item)`
// or every goroutine would see the last value. You'll still meet that
// pattern in older code.)
//
// sync.Mutex — mutual exclusion. Lock, touch the state, unlock:
//
//   type Counter struct {
//       mu sync.Mutex
//       n  int
//   }
//   func (c *Counter) Inc() {
//       c.mu.Lock()
//       defer c.mu.Unlock()
//       c.n++                     // n++ is READ, ADD, WRITE — three steps, not one
//   }
//
// The mutex is the FIRST field by convention, and it protects the fields
// below it. Methods must be on POINTER receivers — a value receiver copies
// the mutex, and a copied mutex guards nothing. (`go vet` catches this:
// "passes lock by value".)
//
// sync.RWMutex — when reads vastly outnumber writes. Any number of readers
// can hold RLock at once; Lock is exclusive against everything:
//
//   func (m *SafeMap) Get(k string) int {
//       m.mu.RLock()
//       defer m.mu.RUnlock()
//       return m.data[k]
//   }
//
// This matters more than it looks: a plain Go map PANICS on concurrent
// write ("fatal error: concurrent map writes"), and that one isn't a
// recoverable panic — it kills the process.
//
// sync.Once — run something exactly once, however many goroutines race to it:
//
//   var once sync.Once
//   once.Do(func() { expensiveInit() })
//
// Every caller blocks until the first Do finishes, so nobody sees a half-built
// value. This is how you do lazy initialisation safely.
//
// sync/atomic — lock-free operations on a single value. The typed wrappers
// (Go 1.19+) are the nice way in:
//
//   var n atomic.Int64
//   n.Add(1)
//   n.Load()
//   n.CompareAndSwap(old, new)
//
// Faster than a mutex, but only covers ONE variable. The moment you need two
// fields to stay consistent with each other, go back to a mutex.

// TODO 1: write `func parallelSum(nums []int, workers int) int` that splits
// nums into `workers` chunks, sums each in its own goroutine, and adds the
// partial sums into a shared total guarded by a sync.Mutex. Use a WaitGroup
// to know when they're all done. Treat workers <= 0 as 1, and handle an
// empty slice.

// TODO 2: define `type Counter struct` holding a sync.Mutex and an int, with
// pointer-receiver methods `Inc()` and `Value() int`. Both must lock — reading
// an int while another goroutine writes it is just as much a race as two
// writes.

// TODO 3: define `type SafeMap` wrapping a map[string]int with a
// sync.RWMutex, plus `NewSafeMap() *SafeMap`, `Set(k string, v int)`,
// `Get(k string) (int, bool)` and `Len() int`. Use RLock for the two readers
// and Lock for Set.

// TODO 4: lazy init with sync.Once. Declare package-level `var once sync.Once`,
// a config string, and an atomic.Int64 counting how many times the init body
// actually ran. Write `func LoadConfig() string` that does the init inside
// once.Do and returns the config, and `func ConfigLoadCount() int` returning
// the counter. However many goroutines call LoadConfig, the count stays 1.

// TODO 5: define `type AtomicCounter struct` holding an atomic.Int64, with
// `Inc()` and `Value() int` — the same job as TODO 2 with no lock at all.

func main() {
	// TODO 6: print parallelSum over 1..100 with 4 workers.

	// TODO 7: start 100 goroutines that each call Inc() 100 times on one
	// Counter, wait for them with a WaitGroup, and print Value().

	// TODO 8: fill a SafeMap from several goroutines and print its Len().

	// TODO 9: call LoadConfig from 50 goroutines, then print ConfigLoadCount().

	// TODO 10: do the TODO 7 experiment again with AtomicCounter and print it.

	fmt.Print()
}

// EXPECTED OUTPUT:
// 5050
// 10000
// 100
// 1
// 10000
