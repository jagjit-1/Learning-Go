package main

import "fmt"

// ============================================================
// CONCEPT: Control flow — if/else, for, switch
// ============================================================
//
// IF/ELSE: no parentheses around the condition, braces are MANDATORY
// (even for one-line bodies — unlike C#/JS where you can skip braces).
//
//   if x > 5 {
//       fmt.Println("big")
//   } else if x == 5 {
//       fmt.Println("equal")
//   } else {
//       fmt.Println("small")
//   }
//
// You can also scope a variable to the if statement itself:
//   if val := compute(); val > 5 {
//       fmt.Println(val)
//   }
//   // val does not exist out here
//
// FOR: Go has exactly ONE looping keyword — `for`. No `while`, no `do-while`.
// Three shapes:
//
//   for i := 0; i < 10; i++ { }      // classic C-style
//   for condition {  }                 // this IS your "while" loop
//   for {  }                            // infinite loop, use `break` to exit
//
// SWITCH: no `break` needed (Go does NOT fall through by default — this
// is the opposite of C#/JS, which DO fall through unless you break).
// If you WANT fallthrough, you say so explicitly with the `fallthrough` keyword.
//
//   switch day {
//   case "Mon", "Tue", "Wed", "Thu", "Fri":
//       fmt.Println("Weekday")
//   case "Sat", "Sun":
//       fmt.Println("Weekend")
//   default:
//       fmt.Println("not a day")
//   }
//
// switch can also have NO expression at all, acting like an if/else chain:
//   switch {
//   case x < 0:
//       fmt.Println("negative")
//   case x == 0:
//       fmt.Println("zero")
//   default:
//       fmt.Println("positive")
//   }

func main() {
	// TODO 1: FizzBuzz from 1 to 20 using a classic for loop and if/else:
	// multiples of 3 -> "Fizz", multiples of 5 -> "Buzz",
	// multiples of both -> "FizzBuzz", else -> the number itself
	for i := 1; i <= 20; i++ {
		if i%3 == 0 && i%5 == 0 {
			fmt.Println("FizzBuzz")
		} else if i%3 == 0 {
			fmt.Println("Fizz")
		} else if i%5 == 0 {
			fmt.Println("Buzz")
		} else {
			fmt.Println(i)
		}
	}
	// TODO 2: write a "while-style" loop (for condition {}) that starts
	// a counter at 10 and counts DOWN to 1, printing each number
	i := 10
	for i > 0 {
		fmt.Println(i)
		i--
	}
	// TODO 3: write an infinite loop (for {}) that sums numbers 1..100
	// and breaks once the running sum exceeds 1000. Print the final sum
	// and how many numbers it took.
	curr := 0
	sum := 0
	for {
		sum += curr + 1
		curr++
		if sum > 1000 {
			fmt.Println(sum, curr)
			break
		}
	}
	// TODO 4: grade calculator using switch with NO expression (the if/else-chain form):
	// given `score := 78`, print "A" (90+), "B" (80-89), "C" (70-79), "D" (60-69), "F" (below 60)
	score := 78
	switch {
	case score >= 90:
		fmt.Println("A")
	case score <= 89 && score >= 80:
		fmt.Println("B")
	case score <= 79 && score >= 70:
		fmt.Println("C")
	case score <= 69 && score >= 60:
		fmt.Println("D")
	default:
		fmt.Println("F")
	}
	// TODO 5: switch WITH an expression: given `day := "Wed"`, print
	// "Weekday" or "Weekend" per the grouped-case pattern shown in the concept block above
	day := "Wed"
	switch day {
	case "Mon", "Tue", "Wed", "Thu", "Fri":
		fmt.Println("Weekday")
	case "Sun", "Sat":
		fmt.Println("Weekend")
	default:
		fmt.Println("Not a valid day")
	}
}

// EXPECTED OUTPUT (partial):
// 1
// 2
// Fizz
// 4
// Buzz
// Fizz
// 7
// ...
// 10
// 9
// 8
// ...
// 1
// Sum exceeded 1000 after N numbers, total = ...
// C
// Weekday
