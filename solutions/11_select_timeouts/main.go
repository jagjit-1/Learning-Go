package main

import (
	"errors"
	"fmt"
	"time"
)

func first(a, b <-chan string) string {
	select {
	case v := <-a:
		return v
	case v := <-b:
		return v
	}
}

var ErrTimeout = errors.New("timed out")

func recvWithTimeout(ch <-chan int, d time.Duration) (int, error) {
	select {
	case v := <-ch:
		return v, nil
	case <-time.After(d):
		return 0, ErrTimeout
	}
}

func trySend(ch chan<- int, v int) bool {
	select {
	case ch <- v:
		return true
	default:
		return false
	}
}

func tryRecv(ch <-chan int) (int, bool) {
	select {
	case v, ok := <-ch:
		return v, ok
	default:
		return 0, false
	}
}

func merge2(a, b <-chan int, done <-chan struct{}) []int {
	out := []int{}
	for a != nil || b != nil {
		select {
		case v, ok := <-a:
			if !ok {
				a = nil // drop this case out of the select for good
				continue
			}
			out = append(out, v)
		case v, ok := <-b:
			if !ok {
				b = nil
				continue
			}
			out = append(out, v)
		case <-done:
			return out
		}
	}
	return out
}

func main() {
	fast := make(chan string)
	slow := make(chan string)
	go func() {
		time.Sleep(10 * time.Millisecond)
		fast <- "fast"
	}()
	go func() {
		time.Sleep(200 * time.Millisecond)
		slow <- "slow"
	}()
	fmt.Println(first(fast, slow))

	quick := make(chan int, 1)
	quick <- 42
	v, err := recvWithTimeout(quick, time.Second)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(v)
	}

	never := make(chan int)
	if _, err := recvWithTimeout(never, 50*time.Millisecond); err != nil {
		fmt.Println(err)
	}

	buf := make(chan int, 1)
	fmt.Println(trySend(buf, 1))
	fmt.Println(trySend(buf, 2))

	empty := make(chan int)
	ev, ok := tryRecv(empty)
	fmt.Println(ev, ok)

	a := make(chan int)
	b := make(chan int)
	go func() {
		defer close(a)
		for i := 1; i <= 3; i++ {
			a <- i
		}
	}()
	go func() {
		defer close(b)
		for i := 4; i <= 6; i++ {
			b <- i
		}
	}()
	fmt.Println(len(merge2(a, b, make(chan struct{}))))
}
