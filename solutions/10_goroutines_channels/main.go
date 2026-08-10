package main

import "fmt"

func produce(n int, out chan<- int) {
	for i := 1; i <= n; i++ {
		out <- i
	}
	close(out)
}

func collect(in <-chan int) []int {
	var out []int
	for v := range in {
		out = append(out, v)
	}
	return out
}

func square(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for v := range in {
			out <- v * v
		}
	}()
	return out
}

func sumConcurrent(nums []int) int {
	if len(nums) == 0 {
		return 0
	}
	mid := len(nums) / 2
	partials := make(chan int, 2)

	half := func(part []int) {
		s := 0
		for _, n := range part {
			s += n
		}
		partials <- s
	}
	go half(nums[:mid])
	go half(nums[mid:])

	return <-partials + <-partials
}

func recvClosed() (int, bool) {
	ch := make(chan int)
	close(ch)
	v, ok := <-ch
	return v, ok
}

func main() {
	// produce blocks on every send until someone receives, so it is the one
	// that goes into a goroutine. collect drives the loop on this goroutine.
	ch := make(chan int)
	go produce(5, ch)
	fmt.Println(collect(ch))

	ch2 := make(chan int)
	go produce(5, ch2)
	fmt.Println(collect(square(ch2)))

	fmt.Println(sumConcurrent([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))

	v, ok := recvClosed()
	fmt.Println(v, ok)

	buf := make(chan int, 3)
	buf <- 1
	buf <- 2
	fmt.Printf("len=%d cap=%d\n", len(buf), cap(buf))
	<-buf
	<-buf
	fmt.Printf("len=%d cap=%d\n", len(buf), cap(buf))
}
