package main

import "fmt"

func add(a, b int) int {
	return a + b
}

func divmod(a, b int) (int, int) {
	return a / b, a % b
}

func divmodNamed(a, b int) (q, r int) {
	q = a / b
	r = a % b
	return
}

func sum(nums ...int) int {
	total := 0
	for _, n := range nums {
		total += n
	}
	return total
}

func makeCounter() func() int {
	count := 0
	return func() int {
		count++
		return count
	}
}

func main() {
	fmt.Println(add(5, 3))

	q, r := divmod(17, 5)
	fmt.Println(q, r)

	q2, r2 := divmodNamed(17, 5)
	fmt.Println(q2, r2)

	fmt.Println(sum())
	fmt.Println(sum(1, 2, 3))
	fmt.Println(sum(1, 2, 3, 4, 5))

	counterA := makeCounter()
	counterB := makeCounter()
	fmt.Println(counterA(), counterA(), counterA())
	fmt.Println(counterB(), counterB(), counterB())
}
