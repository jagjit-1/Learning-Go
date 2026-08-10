package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

func doWork(ctx context.Context, d time.Duration) error {
	select {
	case <-time.After(d):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func sumWithContext(ctx context.Context, nums []int, per time.Duration) (int, error) {
	total := 0
	for _, n := range nums {
		select {
		case <-ctx.Done():
			return total, ctx.Err()
		case <-time.After(per):
			total += n
		}
	}
	return total, nil
}

type requestIDKey struct{}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func RequestID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(requestIDKey{}).(string)
	return id, ok
}

func fetchAll(ctx context.Context, ids []int, fetch func(context.Context, int) (string, error)) ([]string, error) {
	// A child context so one failure can cancel the siblings without
	// touching the caller's context.
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([]string, len(ids))

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	for i, id := range ids {
		wg.Add(1)
		go func() {
			defer wg.Done()

			v, err := fetch(ctx, id)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				cancel() // stop the siblings
				return
			}
			results[i] = v // distinct index per goroutine: no lock needed
		}()
	}
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if firstErr != nil {
		return nil, firstErr
	}
	return results, nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	fmt.Println(doWork(ctx, 20*time.Millisecond))

	ctx2, cancel2 := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel2()
	fmt.Println(doWork(ctx2, time.Second))

	ctx3, cancel3 := context.WithCancel(context.Background())
	cancel3()
	fmt.Println(doWork(ctx3, time.Second))

	nums := make([]int, 100)
	for i := range nums {
		nums[i] = i + 1
	}
	ctx4, cancel4 := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel4()
	sum, err := sumWithContext(ctx4, nums, time.Millisecond)
	fmt.Println(sum, err)

	withID := WithRequestID(context.Background(), "req-42")
	id, ok := RequestID(withID)
	fmt.Println(id, ok)
	missing, ok := RequestID(context.Background())
	fmt.Println(missing, ok)

	ids := []int{1, 2, 3, 4, 5}
	good, err := fetchAll(context.Background(), ids, func(ctx context.Context, id int) (string, error) {
		return fmt.Sprintf("item-%d", id), nil
	})
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(good)
	}

	_, err = fetchAll(context.Background(), ids, func(ctx context.Context, id int) (string, error) {
		if id == 3 {
			return "", fmt.Errorf("fetch %d failed", id)
		}
		select {
		case <-time.After(500 * time.Millisecond):
			return fmt.Sprintf("item-%d", id), nil
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})
	fmt.Println(err)
}
