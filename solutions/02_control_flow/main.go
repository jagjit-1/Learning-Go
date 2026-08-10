package main

import "fmt"

func main() {
	// TODO 1: FizzBuzz
	for i := 1; i <= 20; i++ {
		if i%15 == 0 {
			fmt.Println("FizzBuzz")
		} else if i%3 == 0 {
			fmt.Println("Fizz")
		} else if i%5 == 0 {
			fmt.Println("Buzz")
		} else {
			fmt.Println(i)
		}
	}

	// TODO 2: while-style countdown
	counter := 10
	for counter >= 1 {
		fmt.Println(counter)
		counter--
	}

	// TODO 3: infinite loop with break
	sum := 0
	count := 0
	n := 1
	for {
		sum += n
		count++
		n++
		if sum > 1000 {
			break
		}
	}
	fmt.Printf("Sum exceeded 1000 after %d numbers, total = %d\n", count, sum)

	// TODO 4: grade calculator, expression-less switch
	score := 78
	switch {
	case score >= 90:
		fmt.Println("A")
	case score >= 80:
		fmt.Println("B")
	case score >= 70:
		fmt.Println("C")
	case score >= 60:
		fmt.Println("D")
	default:
		fmt.Println("F")
	}

	// TODO 5: switch with expression
	day := "Wed"
	switch day {
	case "Mon", "Tue", "Wed", "Thu", "Fri":
		fmt.Println("Weekday")
	case "Sat", "Sun":
		fmt.Println("Weekend")
	default:
		fmt.Println("not a day")
	}
}
