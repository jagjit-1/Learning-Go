package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

func parallelSum(nums []int, workers int) int {
	if len(nums) == 0 {
		return 0
	}
	if workers <= 0 {
		workers = 1
	}
	if workers > len(nums) {
		workers = len(nums)
	}

	chunk := (len(nums) + workers - 1) / workers

	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		total int
	)
	for start := 0; start < len(nums); start += chunk {
		end := start + chunk
		if end > len(nums) {
			end = len(nums)
		}

		wg.Add(1) // before the go statement, not inside it
		go func(part []int) {
			defer wg.Done()

			sum := 0
			for _, n := range part {
				sum += n
			}

			mu.Lock()
			total += sum
			mu.Unlock()
		}(nums[start:end])
	}
	wg.Wait()

	return total
}

type Counter struct {
	mu sync.Mutex
	n  int
}

func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
}

func (c *Counter) Value() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

type SafeMap struct {
	mu   sync.RWMutex
	data map[string]int
}

func NewSafeMap() *SafeMap {
	return &SafeMap{data: make(map[string]int)}
}

func (m *SafeMap) Set(k string, v int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[k] = v
}

func (m *SafeMap) Get(k string) (int, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.data[k]
	return v, ok
}

func (m *SafeMap) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

var (
	configOnce  sync.Once
	config      string
	configLoads atomic.Int64
)

func LoadConfig() string {
	configOnce.Do(func() {
		configLoads.Add(1)
		config = "loaded"
	})
	return config
}

func ConfigLoadCount() int {
	return int(configLoads.Load())
}

type AtomicCounter struct {
	n atomic.Int64
}

func (c *AtomicCounter) Inc() {
	c.n.Add(1)
}

func (c *AtomicCounter) Value() int {
	return int(c.n.Load())
}

func main() {
	nums := make([]int, 100)
	for i := range nums {
		nums[i] = i + 1
	}
	fmt.Println(parallelSum(nums, 4))

	var wg sync.WaitGroup
	counter := &Counter{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				counter.Inc()
			}
		}()
	}
	wg.Wait()
	fmt.Println(counter.Value())

	sm := NewSafeMap()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sm.Set(fmt.Sprintf("key-%d", i), i)
		}()
	}
	wg.Wait()
	fmt.Println(sm.Len())

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			LoadConfig()
		}()
	}
	wg.Wait()
	fmt.Println(ConfigLoadCount())

	atomicCounter := &AtomicCounter{}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				atomicCounter.Inc()
			}
		}()
	}
	wg.Wait()
	fmt.Println(atomicCounter.Value())
}
