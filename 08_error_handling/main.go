package main

import (
	"errors"
	"fmt"
	"strconv"
)

// ============================================================
// CONCEPT: error as a value (no exceptions)
// ============================================================
//
// `error` is just an interface:
//   type error interface { Error() string }
//
// The idiomatic pattern — a function that can fail returns (result, error)
// as its LAST return value:
//   func Divide(a, b float64) (float64, error) {
//       if b == 0 {
//           return 0, errors.New("division by zero")
//       }
//       return a / b, nil
//   }
//
// Caller checks explicitly, every time:
//   result, err := Divide(10, 0)
//   if err != nil {
//       fmt.Println("Error:", err)
//       return   // or handle however makes sense
//   }
//
// fmt.Errorf works like Printf but produces an error, and %w WRAPS an
// existing error so you can unwrap it later (preserves the original error
// for inspection while adding context):
//   return fmt.Errorf("processing failed: %w", err)
//
// CUSTOM ERROR TYPES — for when the caller needs to distinguish error
// KINDS, not just read a message. Define a type that satisfies the
// error interface (has an Error() string method):
//   type ValidationError struct {
//       Field string
//       Msg   string
//   }
//   func (e *ValidationError) Error() string {
//       return fmt.Sprintf("%s: %s", e.Field, e.Msg)
//   }
//
// errors.As lets you check "is this error (or one it wraps) a
// *ValidationError specifically?" and extract it:
//   var ve *ValidationError
//   if errors.As(err, &ve) {
//       fmt.Println("bad field:", ve.Field)
//   }
//
// strconv.Atoi(s) converts string -> int, returns (int, error) —
// you'll use this for the input parsing exercise below.

type NegativeAgeError struct {
	Value int
}

func (ne *NegativeAgeError) Error() string {
	return fmt.Sprintf("Rejected negative age: %d", ne.Value)
}

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}

	return a / b, nil
}

func ParseAge(s string) (int, error) {
	val, err := strconv.Atoi(s)

	if err != nil {
		return 0, fmt.Errorf("Error parsing age: %w", err)
	}

	if val < 0 {
		return 0, &NegativeAgeError{Value: val}
	}

	if val > 150 {
		return 0, errors.New("Error parsing age: age must be between 0 and 150")
	}

	return val, nil
}

func main() {
	// TODO 1: write `func Divide(a, b float64) (float64, error)` per the
	// concept block. Call it with (10, 2) and (10, 0), print result or error
	// for each, using the `if err != nil` pattern
	res1, err1 := Divide(10, 2)
	if err1 != nil {
		fmt.Println("Error:", err1)
	} else {
		fmt.Printf("Result: %.2f\n", res1)
	}

	res2, err2 := Divide(10, 0)
	if err2 != nil {
		fmt.Println("Error:", err2)
	} else {
		fmt.Printf("Result: %.2f\n", res2)
	}
	// TODO 2: write `func ParseAge(input string) (int, error)` that uses
	// strconv.Atoi to parse, and additionally returns an error if the
	// parsed number is negative or over 150 (use fmt.Errorf for this case,
	// and just pass through the strconv error for a bad parse)
	// Test it with "25", "abc", "-5", "200"
	vals := []string{"25", "abc", "-5", "200"}
	for _, stringVal := range vals {
		intVal, intError := ParseAge(stringVal)

		if intError != nil {
			fmt.Println(intError)
		} else {
			fmt.Println("Parsed Age:", intVal)
		}
	}
	// TODO 3: define a custom error type `type NegativeAgeError struct { Value int }`
	// with an Error() string method. Modify ParseAge itself to return
	// &NegativeAgeError{Value: n} instead of a plain fmt.Errorf when age < 0
	// (modify it in place rather than writing a second function — the
	// checker in main_test.go tests ParseAge)

	// TODO 4: in main, call ParseAge with "-10", then use errors.As
	// to check specifically for *NegativeAgeError and print
	// "Rejected negative age: -10" only if that specific type matched
	// (vs a generic "some error occurred" for other error kinds)
	_, err := ParseAge("-10")
	var ve *NegativeAgeError

	if errors.As(err, &ve) {
		fmt.Println(err)
	}
}

// EXPECTED OUTPUT (roughly):
// Result: 5.00
// Error: division by zero
// Parsed age: 25
// Error parsing age: strconv.Atoi: parsing "abc": invalid syntax
// Error parsing age: age must be between 0 and 150
// Error parsing age: age must be between 0 and 150
// Rejected negative age: -10
