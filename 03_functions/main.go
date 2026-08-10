package main

import "fmt"

// ============================================================
// CONCEPT: Functions
// ============================================================
//
// Basic shape:
//   func add(a int, b int) int {
//       return a + b
//   }
//
// Shorthand when consecutive params share a type:
//   func add(a, b int) int { return a + b }
//
// MULTIPLE RETURN VALUES — this is used EVERYWHERE in Go, especially
// for the (value, error) pattern you'll see in Exercise 8:
//   func divmod(a, b int) (int, int) {
//       return a / b, a % b
//   }
//   q, r := divmod(17, 5)
//
// NAMED RETURNS — you can name the return values; a bare `return`
// sends back whatever they currently hold:
//   func divmod(a, b int) (q int, r int) {
//       q = a / b
//       r = a % b
//       return // "naked" return, sends back q and r
//   }
//
// VARIADIC FUNCTIONS — variable number of arguments, becomes a slice
// inside the function:
//   func sum(nums ...int) int {
//       total := 0
//       for _, n := range nums {
//           total += n
//       }
//       return total
//   }
//   sum(1, 2, 3)       // works
//   sum()               // also works, nums is an empty slice
//
// CLOSURES — functions are values, and can capture variables from
// their enclosing scope:
//   func makeCounter() func() int {
//       count := 0
//       return func() int {
//           count++
//           return count
//       }
//   }
//   counter := makeCounter()
//   counter() // 1
//   counter() // 2

func add(a, b int) int {
	return a + b
}

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

func divmodNamed(a, b int) (q int, r int) {
	q = a / b
	r = a % b
	return
}

func sum(nums ...int) int {
	cnt := 0
	for _, n := range nums {
		cnt += n
	}
	return cnt
}

func makeCounter() func() int {
	counter := 0
	return func() int {
		counter++
		return counter
	}
}

func main() {
	// TODO 1: write func `add(a, b int) int` above main, call it here, print result
	fmt.Println(add(1, 2))
	// TODO 2: write func `divmod(a, b int) (int, int)` (plain multi-return,
	// not named), call it with divmod(17, 5), print both values
	a, b := divmod(17, 5)
	fmt.Println(a, b)
	// TODO 3: rewrite divmod as `divmodNamed(a, b int) (q, r int)` using
	// named returns and a naked `return`. Call it, print both values.
	q, r := divmodNamed(17, 5)
	fmt.Println(q, r)
	// TODO 4: write a variadic `func sum(nums ...int) int`, call it with
	// 3 different argument counts (0, 3, 5 numbers) and print each result
	fmt.Println(sum())
	fmt.Println(sum(1, 2, 3))
	fmt.Println(sum(1, 2, 3, 4, 5))
	// TODO 5: write `makeCounter() func() int` as shown in the concept block.
	// Create TWO separate counters, call each 3 times, print all six results,
	// and confirm they count independently (this proves each closure has
	// its own captured `count` variable)
	counterA := makeCounter()
	counterB := makeCounter()

	fmt.Println(counterA())
	fmt.Println(counterA())
	fmt.Println(counterA())

	fmt.Println(counterB())
	fmt.Println(counterB())
	fmt.Println(counterB())

}

// EXPECTED OUTPUT (yours will vary slightly):
// 8
// 3 2
// 3 2
// 0
// 6
// 15
// 1 2 3   <- counter A called 3x
// 1 2 3   <- counter B called 3x, independent of A
