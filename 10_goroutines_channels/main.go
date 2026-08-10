package main

import "fmt"

// ============================================================
// CONCEPT: goroutines and channels
// ============================================================
//
// A goroutine is a function running concurrently, managed by Go's runtime
// rather than the OS. Starting one is a keyword, not a library call:
//
//   go doSomething()          // returns IMMEDIATELY, doSomething runs alongside
//   go func() { ... }()       // anonymous, note the trailing () to call it
//
// They're cheap — a few KB of stack that grows on demand. Tens of thousands
// is normal. This is NOT a thread per goroutine; the runtime multiplexes many
// goroutines onto a few OS threads.
//
// THE CATCH: when main() returns, the program exits and every goroutine dies
// mid-flight, silently. So you need a way to wait. That's what channels do.
//
// CHANNEL — a typed pipe. Send on one side, receive on the other:
//
//   ch := make(chan int)      // unbuffered
//   ch <- 42                  // send (arrow points INTO the channel)
//   v := <-ch                 // receive (arrow points OUT of it)
//
// UNBUFFERED channels are SYNCHRONOUS. `ch <- 42` blocks until another
// goroutine is ready to receive, and vice versa. The send and the receive
// happen at the same moment — this is a rendezvous, not a queue. That
// blocking IS your synchronisation; you rarely need locks on top of it.
//
//   ch := make(chan int)
//   ch <- 42                  // DEADLOCK: nobody is receiving, and there is
//                             // no other goroutine to do it
//
// BUFFERED channels hold values without a waiting receiver:
//
//   ch := make(chan int, 3)   // capacity 3
//   ch <- 1; ch <- 2; ch <- 3 // none of these block
//   ch <- 4                   // blocks: buffer is full
//   len(ch)                   // values currently buffered
//   cap(ch)                   // buffer capacity
//
// CLOSING — the SENDER closes, to say "no more values are coming":
//
//   close(ch)
//
// Receiving from a closed channel returns immediately with the zero value.
// The two-value receive tells you which happened — same comma-ok shape you
// used for maps in Exercise 5:
//
//   v, ok := <-ch             // ok == false means "channel closed and drained"
//
// RANGE over a channel receives until it's closed. This is the normal way to
// consume one:
//
//   for v := range ch {       // loop ends when the channel is closed
//       fmt.Println(v)
//   }
//
// Sending on a closed channel PANICS, and closing twice PANICS. Never let a
// receiver close a channel. Rule: whoever created and sends on it, closes it.
//
// DIRECTIONAL CHANNEL TYPES — a channel parameter can be restricted, and the
// compiler enforces it:
//
//   func produce(out chan<- int)   // send-only: can send and close, cannot receive
//   func consume(in <-chan int)    // receive-only: cannot send or close
//
// A bidirectional `chan int` converts to either automatically, so callers are
// unaffected. Use these on every channel parameter — they document the
// direction of flow and turn a whole class of mistake into a compile error.

// TODO 1: write `func produce(n int, out chan<- int)` that sends 1, 2, ... n
// into `out` and then CLOSES it. Note the send-only type — try adding a
// receive from `out` inside it and read the compile error.

// TODO 2: write `func collect(in <-chan int) []int` that ranges over `in`
// until it's closed and returns everything it received.

// TODO 3: write `func square(in <-chan int) <-chan int`. It must create its
// own output channel, start a goroutine that reads every value from `in`,
// sends v*v to the output, closes the output when `in` is exhausted, and
// RETURN THE CHANNEL IMMEDIATELY — before any work is done.
// This shape (make a channel, go func, return it) is the backbone of every
// pipeline you'll write in Exercise 14.

// TODO 4: write `func sumConcurrent(nums []int) int` that splits nums in half,
// sums each half in its own goroutine, and combines the two partial sums
// through a channel. Must return 0 for an empty slice.

// TODO 5: write `func recvClosed() (int, bool)` that makes a channel, closes
// it immediately, then does a two-value receive and returns both. This proves
// to you what a receive on a closed channel actually gives back.

func main() {
	// TODO 6: use produce + collect together. produce blocks on an unbuffered
	// channel, so ONE of them has to run in a goroutine — work out which, and
	// try it the other way round to see the deadlock. Print the collected slice.

	// TODO 7: chain produce into square into collect. Print the squares.

	// TODO 8: print sumConcurrent for 1..10.

	// TODO 9: print the two values recvClosed returns.

	// TODO 10: make a buffered channel of capacity 3, send two values, and
	// print len and cap. Then drain it and print len and cap again.

	fmt.Print()
}

// EXPECTED OUTPUT:
// [1 2 3 4 5]
// [1 4 9 16 25]
// 55
// 0 false
// len=2 cap=3
// len=0 cap=3
