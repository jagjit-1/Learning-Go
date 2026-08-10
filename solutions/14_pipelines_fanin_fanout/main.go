package main

import (
	"fmt"
	"sort"
	"sync"
)

func gen(nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			out <- n
		}
	}()
	return out
}

func sq(in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			out <- n * n
		}
	}()
	return out
}

func merge(cs ...<-chan int) <-chan int {
	out := make(chan int)

	var wg sync.WaitGroup
	for _, c := range cs {
		wg.Add(1)
		go func(c <-chan int) {
			defer wg.Done()
			for v := range c {
				out <- v
			}
		}(c)
	}

	go func() {
		wg.Wait()
		close(out) // only safe once every contributor has returned
	}()

	return out
}

func pipeline(nums []int, fanOut int) []int {
	if fanOut <= 0 {
		fanOut = 1
	}

	in := gen(nums...)

	stages := make([]<-chan int, 0, fanOut)
	for i := 0; i < fanOut; i++ {
		stages = append(stages, sq(in)) // all reading the same source
	}

	out := []int{}
	for v := range merge(stages...) {
		out = append(out, v)
	}
	return out
}

func genDone(done <-chan struct{}, nums ...int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for _, n := range nums {
			select {
			case out <- n:
			case <-done:
				return
			}
		}
	}()
	return out
}

func sqDone(done <-chan struct{}, in <-chan int) <-chan int {
	out := make(chan int)
	go func() {
		defer close(out)
		for n := range in {
			select {
			case out <- n * n:
			case <-done:
				return
			}
		}
	}()
	return out
}

func firstN(in <-chan int, n int, done chan struct{}) []int {
	defer close(done) // signals upstream even if we return early

	out := make([]int, 0, n)
	if n <= 0 {
		return out
	}
	for v := range in {
		out = append(out, v)
		if len(out) == n {
			break
		}
	}
	return out
}

func main() {
	nums := []int{1, 2, 3, 4, 5, 6, 7, 8}
	got := pipeline(nums, 3)
	sort.Ints(got)
	fmt.Println(got)

	big := make([]int, 1000)
	for i := range big {
		big[i] = i + 1
	}
	done := make(chan struct{})
	fmt.Println(firstN(sqDone(done, genDone(done, big...)), 3, done))
}
