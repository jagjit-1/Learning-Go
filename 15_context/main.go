package main

import "fmt"

// ============================================================
// CONCEPT: context.Context — cancellation that propagates
// ============================================================
//
// Exercise 14's `done` channel works, but it only carries one bit: stop. It
// can't say WHY, can't carry a deadline, and doesn't compose down a call
// chain. context.Context is that pattern, finished.
//
// A Context is immutable. You don't modify one; you DERIVE a child from it,
// and cancelling a parent cancels every descendant.
//
//   ctx := context.Background()              // the root, never cancelled
//   ctx, cancel := context.WithCancel(ctx)   // cancel it by hand
//   ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
//   ctx, cancel := context.WithDeadline(ctx, someTime)
//
//   defer cancel()   // ALWAYS. Even for WithTimeout, even on the happy path.
//
// That defer is not optional bookkeeping. Not calling cancel leaks the timer
// and the child context until the parent is done — `go vet` flags it as
// "the cancel function is not used on all paths".
//
// Two methods matter:
//
//   <-ctx.Done()     // a channel, closed when this context is cancelled
//   ctx.Err()        // nil while live; after that, WHY it ended
//
// ctx.Err() returns one of two sentinels, and you compare with errors.Is:
//
//   context.Canceled           // someone called cancel()
//   context.DeadlineExceeded   // the timeout or deadline ran out
//
// The shape of a cancellable operation is always the same select:
//
//   func doWork(ctx context.Context) error {
//       select {
//       case <-time.After(5 * time.Second):   // the actual work
//           return nil
//       case <-ctx.Done():
//           return ctx.Err()                  // gives the caller the reason
//       }
//   }
//
// For a loop, check between iterations — cancellation is cooperative, nothing
// interrupts a goroutine from outside:
//
//   for _, item := range items {
//       select {
//       case <-ctx.Done():
//           return ctx.Err()
//       default:
//       }
//       process(item)
//   }
//
// CONVENTIONS the whole ecosystem follows:
//   - ctx is the FIRST parameter, always named ctx: func F(ctx context.Context, ...)
//   - never store a Context in a struct field; pass it through calls
//   - never pass a nil Context; use context.TODO() if you don't have one yet
//
// context.WithValue — request-scoped data (a request ID, a trace span). Use
// it sparingly: it's an untyped bag, and values in it are invisible to the
// compiler. NOT for passing optional arguments.
//
// The key must be an unexported custom type, so no other package can collide
// with your key or read it by accident:
//
//   type ctxKey struct{}                                  // unexported
//   ctx = context.WithValue(ctx, ctxKey{}, "req-42")
//   v, ok := ctx.Value(ctxKey{}).(string)                 // type assertion
//
// A plain string key ("requestID") would work and is a real bug: any package
// in the process can set the same key.

// TODO 1: write `func doWork(ctx context.Context, d time.Duration) error`
// that simulates work taking d, and returns nil if it finished, or ctx.Err()
// if the context was cancelled first.

// TODO 2: write
//   func sumWithContext(ctx context.Context, nums []int, per time.Duration) (int, error)
// that adds up nums, taking `per` for each element, and gives up as soon as
// the context is done — returning (partial sum so far, ctx.Err()).

// TODO 3: request IDs. Define an UNEXPORTED key type, then
//   func WithRequestID(ctx context.Context, id string) context.Context
//   func RequestID(ctx context.Context) (string, bool)
// RequestID returns ("", false) when there's no ID on the context.

// TODO 4: write
//   func fetchAll(ctx context.Context, ids []int,
//                 fetch func(context.Context, int) (string, error)) ([]string, error)
// Fetch every id CONCURRENTLY (one goroutine each is fine here). Results come
// back in the order of `ids`, not completion order. If any fetch fails,
// return that error and make sure the others are cancelled — derive a
// cancellable child context and pass THAT to fetch, not the parent.

func main() {
	// TODO 5: doWork with 20ms of work and a 1s timeout — print the error
	// (which should be <nil>).

	// TODO 6: doWork with 1s of work and a 50ms timeout — print the error.

	// TODO 7: WithCancel, cancel it immediately, then doWork — print the error.
	// Note it's a different error from TODO 6.

	// TODO 8: sumWithContext over 1..100 at 1ms each with a 20ms timeout —
	// print the partial sum and the error.

	// TODO 9: put a request ID on a context, read it back, and print it.
	// Then read from a bare context.Background() and print the ok flag.

	// TODO 10: fetchAll over ids 1..5 with a fetch that succeeds — print the
	// results. Then one where id 3 fails — print the error.

	fmt.Print()
}

// EXPECTED OUTPUT:
// <nil>
// context deadline exceeded
// context canceled
// <partial sum> context deadline exceeded
// req-42 true
//  false
// [item-1 item-2 item-3 item-4 item-5]
// fetch 3 failed
