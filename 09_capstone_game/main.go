package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

// ============================================================
// CONCEPT: Reading user input + tying it all together
// ============================================================
//
// bufio.NewScanner(os.Stdin) reads input line by line:
//   scanner := bufio.NewScanner(os.Stdin)
//   scanner.Scan()               // reads one line, returns bool (success)
//   text := scanner.Text()       // the line, as a string
//
// strings.TrimSpace(s) removes leading/trailing whitespace (input from
// a terminal often has a trailing newline you need to strip).
//
// rand.Intn(n) returns a random int in [0, n) — for a 1-100 range you'd
// use rand.Intn(100) + 1.
//
// This exercise is DELIBERATELY less scaffolded than the previous ones —
// you now have all the pieces (variables, control flow, functions, structs,
// methods, interfaces, errors). Design it yourself. Suggested shape below,
// but feel free to deviate if you have a cleaner idea — that's the point.

// ------------------------------------------------------------
// SPEC: Number Guessing Game
// ------------------------------------------------------------
// 1. Generate a random target number between 1 and 100.
// 2. Loop: prompt the player to guess, read their input.
// 3. Parse the input to an int. If parsing fails, print a friendly
//    error (don't crash!) and re-prompt — DO NOT count this as a wrong guess.
// 4. If the guess is too high/low, say so and let them try again.
// 5. Track the number of valid guesses made.
// 6. If they guess correctly, congratulate them and print how many
//    guesses it took, then end the game.
// 7. If they exceed 7 valid guesses without success, end the game with
//    a custom error type (see below) revealing the number.
//
// SUGGESTED STRUCTURE (you can deviate):
//
//   type GameOverError struct {
//       Attempts int
//       Target   int
//   }
//   func (e *GameOverError) Error() string { ... }
//
//   type Game struct {
//       Target   int
//       Attempts int
//       MaxTries int
//   }
//
//   func NewGame(maxTries int) *Game { ... }              // constructor pattern
//   func (g *Game) Guess(n int) (result string, err error) {
//       // result: "correct" / "too high" / "too low"
//       // err: non-nil (*GameOverError) if attempts exceeded
//   }
//
// This constructor + methods pattern (NewX() *X returning a pointer) is
// the standard Go idiom for anything resembling a "class" — get used to
// seeing and writing it, it'll show up constantly on Day 2.

type GameOverError struct {
	Attempts int
	Target   int
}

func (gfe *GameOverError) Error() string {
	return fmt.Sprintf("Exhausted max tries of %d, the answer was %d", gfe.Attempts, gfe.Target)
}

type Game struct {
	MaxTries int
	Target   int
	Attempts int
}

func NewGame(maxTries int) *Game {
	return &Game{MaxTries: maxTries, Target: rand.Intn(100) + 1}
}

func (g *Game) Guess(n int) (result string, err error) {
	if g.Attempts == g.MaxTries {
		err = &GameOverError{Attempts: g.Attempts, Target: g.Target}
		return
	}

	g.Attempts++

	if n == g.Target {
		result = "correct"
	} else if n < g.Target {
		result = "Too low!"
	} else {
		result = "Too high!"
	}
	return
}

func main() {
	// TODO: implement the full game using the spec above.
	// Minimum requirement: it must run in a real terminal loop reading
	// real stdin input via bufio.Scanner — don't hardcode guesses.

	scanner := bufio.NewScanner(os.Stdin)
	game := NewGame(7)
	// _ = scanner // remove this line once you're using scanner for real
	fmt.Println("Guess a number between 1 and 100!")
	for {
		fmt.Print("Your guess: ")
		scanner.Scan()
		guess := scanner.Text()
		guess = strings.TrimSpace(guess)

		intGuess, err := strconv.Atoi(guess)

		if err != nil {
			fmt.Println("Invalid guess, please retry with valid integer")
			continue
		}
		res, err := game.Guess(intGuess)

		if err != nil {
			fmt.Println(err)
			return
		} else if res == "correct" {
			fmt.Printf("Congratulations, you have guessed the correct number in %d attempts\n", game.Attempts)
			return
		}
		fmt.Println(res)
	}
}

// EXPECTED BEHAVIOR (not a fixed output — this is interactive):
// Guess a number between 1 and 100!
// Your guess: 50
// Too low!
// Your guess: 75
// Too high!
// Your guess: abc
// That's not a valid number, try again.
// Your guess: 62
// Correct! You got it in 3 guesses.
