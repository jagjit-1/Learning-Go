package main

import (
	"fmt"
)

// ============================================================
// CONCEPT: The absolute basics
// ============================================================
//
// Every Go program has a `package main` and a `func main()` — this is
// the entry point, like `static void Main()` in C#.
//
// `import "fmt"` brings in the formatting/printing package (like
// `using System;` in C#, but Go imports are per-package, not global).
//
// Variable declaration has THREE forms:
//
//   var x int = 5        // explicit type
//   var y = 5             // type inferred
//   z := 5                 // short form, ONLY inside functions, type inferred
//
// `:=` is the one you'll use 90% of the time. It declares AND assigns.
// You cannot use `:=` for a variable that already exists in the same scope
// (that's a compile error — Go is strict about this).
//
// Basic types: int, float64, string, bool
// (there's also int8/16/32/64, uint variants, float32 — ignore these for now)
//
// fmt.Println(a, b, c)         -> prints with spaces, newline at end
// fmt.Printf("%s is %d\n", a, b) -> formatted, like C's printf
//   Common verbs: %s (string) %d (int) %f (float) %v (any value, "default format") %T (type)
//
// Go is STRICT about unused variables and unused imports — both are
// compile errors, not warnings. This is deliberate (forces clean code).

func main() {
	// TODO 1: declare a string variable `name` using `:=` with your name
	name := "jagjit singh"
	// TODO 2: declare an int variable `age` using `:=`
	age := 23
	// TODO 3: declare a float64 variable `heightMeters` using explicit `var` syntax
	var heightMeters = 6.1
	// TODO 4: declare a bool variable `isLearningGo` and set it to true
	var isLearningGo bool = true
	// TODO 5: print all four using a single fmt.Printf call with appropriate verbs,
	// in this exact sentence shape:
	// "Jagjit is 24 years old, 1.75m tall. Learning Go: true"
	fmt.Printf("%s is %d years old, %fm tall. Learning Go: %t\n", name, age, heightMeters, isLearningGo)
	// TODO 6: type conversion — Go does NOT auto-convert between numeric types
	// (unlike C#/JS). Declare `ageFloat := float64(age)` and print it divided by 2.
	// Try removing the float64() conversion and see what error you get.
	ageFloat := float64(age)
	fmt.Printf("%f\n", ageFloat/2)
	// TODO 7: constants — declare `const daysInWeek = 7` and print
	// "There are 7 days in a week"
	const daysInWeek = 7
	fmt.Printf("There are %d days in a week\n", daysInWeek)
}

// EXPECTED OUTPUT (yours will vary by name/age):
// Jagjit is 24 years old, 1.75m tall. Learning Go: true
// 12
// There are 7 days in a week
