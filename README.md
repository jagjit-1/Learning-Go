# FAround — Go, Hands-On

17 progressive exercises in two sets: **basics** (01–09) and **concurrency**
(10–17). Each folder = one `main.go` with:
1. A short **CONCEPT** comment block teaching the *new* syntax for that exercise (only what you need, nothing ahead).
2. A `// TODO` scaffold — you write the code.
3. An **EXPECTED OUTPUT** comment so you can self-verify without needing solutions.

## Order (do them in sequence — each builds on the last)
1. `01_hello_vars` — package/import/main, var declarations, `:=`, basic types, fmt
2. `02_control_flow` — if/else, the three shapes of `for`, switch
3. `03_functions` — multiple returns, named returns, variadic args, closures
4. `04_arrays_slices` — arrays vs slices, `make`, `append`, `range`
5. `05_maps_structs` — maps, the "comma ok" idiom, structs
6. `06_methods_pointers` — methods, value vs pointer receivers
7. `07_interfaces` — implicit interface satisfaction, type switches
8. `08_error_handling` — `error` as a value, custom error types
9. `09_capstone_game` — number guessing game, combines everything above

### Set B — concurrency (10–17)
10. `10_goroutines_channels` — `go`, unbuffered vs buffered, close, range, directional types
11. `11_select_timeouts` — select, `time.After`, non-blocking send/recv, the nil-channel trick
12. `12_sync_primitives` — WaitGroup, Mutex, RWMutex, Once, sync/atomic
13. `13_worker_pools` — bounded concurrency, ordered results, stopping early
14. `14_pipelines_fanin_fanout` — stages, fan-out, fan-in, and goroutine leaks
15. `15_context` — cancellation, deadlines, propagation, request-scoped values
16. `16_race_detector` — what a data race is, and the three ways out
17. `17_capstone_crawler` — bounded-concurrency crawler over an injected Fetcher

## How to run each one
Once your Go toolchain is set up:
```bash
cd 01_hello_vars
go run main.go
```

## How to check your work
Every exercise folder has a `main_test.go` — a **checker**, not a solution. It
verifies your code without you having to eyeball the EXPECTED OUTPUT block.

Check everything at once, from this folder:
```bash
./check.sh
```
Or one exercise (`3`, `03`, and `03_functions` all work):
```bash
./check.sh 3
```
Or a whole set:
```bash
./check.sh concurrency
```
Or from inside an exercise folder, using plain Go:
```bash
go test -race
```
Add `-v` to see the individual check names as they run:
```bash
go test -race -v
```

**Use `-race` from exercise 10 onward.** `check.sh` always does. A missing
lock usually still produces the right answer on most runs — that's exactly
what makes concurrency bugs dangerous, and the race detector is the only
thing that reliably disagrees. Several checkers in set B would pass broken
code without it.

### Reading the results
- **PASS** — done, move on.
- **FAIL** — it compiled but something's wrong. The message names the TODO and
  says what it expected vs. what it got.
- **TODO (does not compile yet)** — normal before you've written anything.
  `not written yet: add divmod sum` just means the checker calls functions
  that don't exist yet. Note that `go run main.go` still works while this is
  the case: test files are excluded from normal builds.

### What the checkers do and don't pin down
Your own choices stay yours — your name and age in 01, your song titles in 04,
your contacts in 05, your shape dimensions in 07. Those are checked for
*shape*, not exact values. What gets checked exactly is what the exercise
actually specifies: the arithmetic, the function signatures, the slice
aliasing behaviour, the receiver kinds, the error types.

Two checkers are worth knowing about in advance:

- **06** verifies receiver kinds by reflection, not just by behaviour. If
  `Deposit` has a value receiver you'll be told so directly.
- **09** builds your program and *actually plays it* over stdin: it feeds one
  invalid input, then binary-searches 1..100. Binary search needs all 7
  guesses in the worst case, so if your program counts a bad input as a used
  guess, the test runs out of tries and fails. That's deliberate — it's how it
  checks spec step 3 without reading your source.

The unit tests in 09 assume the suggested `Game`/`NewGame`/`Guess` structure.
If you designed something different, delete them and keep `TestEndToEnd`,
which only cares about observable behaviour.

### Extra notes for the concurrency set

- **Deadlocks fail fast.** Every check in 10–17 runs under a deadline, so a
  blocked send or a channel you forgot to close gives you a named failure in
  a few seconds rather than a hung terminal. The message names the usual
  cause.
- **Some checks measure real concurrency.** 13, 14 and 17 count how many of
  your goroutines were inside the work function at the same moment, so
  "bounded to N workers" is verified, not assumed.
- **14 and 17 count goroutines.** They run the same scenario many times over
  and compare live goroutine counts before and after. A stage parked forever
  on a send shows up as a steady climb. A goroutine blocked on a channel is
  never collected — the runtime cannot tell it apart from a busy one.
- **16 ships a deliberately broken function.** `racyCount` is racy on
  purpose and no test calls it. To watch the detector fire:
  ```bash
  SHOW_RACE=1 go run -race .
  ```
  Read both stack traces in the report — they're the two goroutines that
  collided.

## Workflow (this is the actual learning part, not the typing)
1. Read the CONCEPT block.
2. Try to write the TODO **before** looking at `solutions/`.
3. Predict the output in your head, *then* run it.
4. If it doesn't match, figure out why before peeking at the solution.
5. Only check `solutions/<same-folder>/main.go` after a real attempt — peeking early defeats the point.

## When you're stuck
Don't ask "what's the syntax for X" — try it, let the compiler error teach you. Go's compiler errors are unusually specific and readable; learning to read them *is* part of learning Go.
