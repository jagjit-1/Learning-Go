package main

import (
	"fmt"
)

// ============================================================
// CONCEPT: select — waiting on several channels at once
// ============================================================
//
// `select` is a switch whose cases are channel operations. It blocks until
// ONE of them can proceed, runs that case, and moves on:
//
//   select {
//   case v := <-a:
//       fmt.Println("from a:", v)
//   case v := <-b:
//       fmt.Println("from b:", v)
//   case c <- 1:
//       fmt.Println("sent to c")
//   }
//
// If several cases are ready at the same moment, select picks one AT RANDOM.
// That's deliberate — it stops the first case starving the others. Never
// write code that depends on case order.
//
// TIMEOUTS — time.After(d) returns a channel that delivers one value after d.
// Put it in a select and you have a deadline:
//
//   select {
//   case v := <-ch:
//       use(v)
//   case <-time.After(2 * time.Second):
//       return errors.New("timed out")
//   }
//
// NON-BLOCKING — a `default` case runs immediately if nothing else is ready,
// so select never blocks at all:
//
//   select {
//   case v := <-ch:
//       use(v)          // there was something waiting
//   default:
//       // nothing available right now, carry on
//   }
//
// Same trick for sends — this is how you drop work instead of blocking when
// a buffer is full:
//
//   select {
//   case ch <- v:
//       sent = true
//   default:
//       sent = false    // buffer full, don't wait
//   }
//
// FOR-SELECT — the standard shape of a long-running goroutine. A `done`
// channel that gets CLOSED is the classic stop signal: closing makes every
// receiver fire at once, which no single send can do.
//
//   for {
//       select {
//       case v := <-work:
//           handle(v)
//       case <-done:
//           return
//       }
//   }
//
// By convention that's a `chan struct{}` — struct{} occupies zero bytes, and
// it makes it obvious the signal carries no data, only timing.
//
// THE NIL CHANNEL TRICK — this looks like a bug and is actually the idiom.
// A receive on a nil channel blocks FOREVER, so a nil case can never be
// chosen. Meanwhile a CLOSED channel is always ready, so a closed one in a
// select fires over and over in a hot loop.
//
// Put those together: when a channel closes, set your local copy to nil and
// it drops out of the select for good.
//
//   for a != nil || b != nil {
//       select {
//       case v, ok := <-a:
//           if !ok { a = nil; continue }   // disable this case
//           use(v)
//       case v, ok := <-b:
//           if !ok { b = nil; continue }
//           use(v)
//       }
//   }
//
// (For that to work the local variables must be plain <-chan int, so you can
// reassign them — which is why the parameters below are directional types.)

// TODO 1: write `func first(a, b <-chan string) string` that returns the first
// value to arrive on either channel, and ignores the other.

// TODO 2: declare a sentinel `var ErrTimeout = errors.New("timed out")` and
// write `func recvWithTimeout(ch <-chan int, d time.Duration) (int, error)`.
// It returns (value, nil) if something arrives within d, or (0, ErrTimeout)
// if not. A sentinel like this is what lets callers use errors.Is.

// TODO 3: write `func trySend(ch chan<- int, v int) bool` — a non-blocking
// send. Returns true if the value went in, false if it would have blocked.

// TODO 4: write `func tryRecv(ch <-chan int) (int, bool)` — a non-blocking
// receive. CAREFUL: a closed channel is always ready, so a plain `<-ch` would
// report success and hand you a zero. Use the two-value form INSIDE the case
// (`case v, ok := <-ch:`) so a closed channel returns (0, false), same as an
// empty one.

// TODO 5: write `func merge2(a, b <-chan int, done <-chan struct{}) []int`.
// Collect every value from both channels until BOTH are closed, then return
// what you gathered. If `done` is closed first, return early with whatever
// you have. Use the nil-channel trick above — if you skip it, the first
// channel to close will spin the loop.

func main() {
	// TODO 6: start two goroutines that sleep for different durations and then
	// send their name. Print the winner from first().

	// TODO 7: call recvWithTimeout twice — once against a channel that gets a
	// value quickly (print the value), once against one that never does (print
	// the error).

	// TODO 8: make a buffered channel of capacity 1. trySend twice and print
	// both results — the second should be false.

	// TODO 9: tryRecv from an empty channel and print both return values.

	// TODO 10: feed two channels from two goroutines, close both, and print
	// the merged length from merge2 (with a `done` you never close).

	fmt.Print()
}

// EXPECTED OUTPUT:
// fast
// 42
// timed out
// true
// false
// 0 false
// 6
