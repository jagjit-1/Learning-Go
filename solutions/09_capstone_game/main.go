package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
)

type GameOverError struct {
	Attempts int
	Target   int
}

func (e *GameOverError) Error() string {
	return fmt.Sprintf("game over after %d attempts, the number was %d", e.Attempts, e.Target)
}

type Game struct {
	Target   int
	Attempts int
	MaxTries int
}

func NewGame(maxTries int) *Game {
	return &Game{
		Target:   rand.Intn(100) + 1,
		Attempts: 0,
		MaxTries: maxTries,
	}
}

func (g *Game) Guess(n int) (result string, err error) {
	g.Attempts++
	if g.Attempts > g.MaxTries {
		return "", &GameOverError{Attempts: g.Attempts - 1, Target: g.Target}
	}
	switch {
	case n < g.Target:
		return "too low", nil
	case n > g.Target:
		return "too high", nil
	default:
		return "correct", nil
	}
}

func main() {
	scanner := bufio.NewScanner(os.Stdin)
	game := NewGame(7)

	fmt.Println("Guess a number between 1 and 100! You have 7 tries.")

	for {
		fmt.Print("Your guess: ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		n, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("That's not a valid number, try again.")
			continue
		}

		result, err := game.Guess(n)
		if err != nil {
			fmt.Println(err)
			break
		}

		switch result {
		case "too low":
			fmt.Println("Too low!")
		case "too high":
			fmt.Println("Too high!")
		case "correct":
			fmt.Printf("Correct! You got it in %d guesses.\n", game.Attempts)
			return
		}
	}
}
