# FAround — Go Basics, Hands-On

9 progressive exercises. Each folder = one `main.go` with:
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
Or from inside an exercise folder, using plain Go:
```bash
go test
```
Add `-v` to see the individual check names as they run:
```bash
go test -v
```

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

## Workflow (this is the actual learning part, not the typing)
1. Read the CONCEPT block.
2. Try to write the TODO **before** looking at `solutions/`.
3. Predict the output in your head, *then* run it.
4. If it doesn't match, figure out why before peeking at the solution.
5. Only check `solutions/<same-folder>/main.go` after a real attempt — peeking early defeats the point.

## When you're stuck
Don't ask "what's the syntax for X" — try it, let the compiler error teach you. Go's compiler errors are unusually specific and readable; learning to read them *is* part of learning Go.
