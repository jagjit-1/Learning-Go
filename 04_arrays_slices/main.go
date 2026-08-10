package main

import "fmt"

// ============================================================
// CONCEPT: Arrays vs Slices
// ============================================================
//
// ARRAY — fixed size, baked into the type. [5]int and [3]int are
// DIFFERENT types. You'll rarely use raw arrays directly.
//   var a [3]int = [3]int{1, 2, 3}
//   b := [3]int{4, 5, 6}
//
// SLICE — dynamic-size view over an underlying array. This is what
// you'll use 95% of the time (like List<T> in C#, but exposes more
// of what's happening underneath).
//
//   s := []int{1, 2, 3}          // slice literal (no size in brackets)
//   s := make([]int, 0, 5)       // make(type, len, cap) - empty, room for 5
//   s = append(s, 4)              // append returns a (possibly new) slice —
//                                    you MUST reassign, append does not mutate in place
//                                    if it needs to grow beyond capacity
//
// len(s) -> current number of elements
// cap(s) -> current capacity of underlying array before reallocation
//
// SLICING an existing slice: s[low:high] (high is EXCLUSIVE)
//   s := []int{10, 20, 30, 40, 50}
//   s[1:3]   // [20 30]
//   s[:2]    // [10 20]  (from start)
//   s[2:]    // [30 40 50]  (to end)
//
// IMPORTANT: a slice expression shares the SAME underlying array —
// mutating one can mutate the other, UNTIL an append forces a
// reallocation (see Exercise instructions below — you're going to
// prove this to yourself).
//
// RANGE — iterate over a slice, gives you (index, value):
//   for i, v := range s {
//       fmt.Println(i, v)
//   }
//   for _, v := range s { }   // ignore index with blank identifier `_`

func removeSong(pl []string, index int) []string {
	return append(pl[:index], pl[index+1:]...)
}

func main() {
	// TODO 1: create a slice of strings `playlist` with 3 song names
	// using a slice literal, print it with fmt.Println
	playlist := []string{"Livin on a prayer", "Who u foolin", "headliner"}
	fmt.Println(playlist)
	// TODO 2: append 2 more songs to `playlist` (remember: reassign the result)
	playlist = append(playlist, "Never fold")
	playlist = append(playlist, "Passionfruit")
	// TODO 3: use `range` to print every song with its position, 1-indexed,
	// e.g. "1: Bohemian Rhapsody"
	for i, song := range playlist {
		fmt.Printf("%d: %s\n", i+1, song)
	}
	// TODO 4: THE SHARING EXPERIMENT —
	//   a) create `nums := make([]int, 0, 2)` (len 0, cap 2)
	//   b) append 2 elements one at a time, printing len(nums) and cap(nums) after each
	//   c) create `sub := nums[0:1]`
	//   d) mutate `sub[0] = 999`
	//   e) print `nums` — did it change? Why?
	//   f) now append a 3rd element to `nums` (this EXCEEDS cap 2, forcing
	//      a reallocation to a new underlying array)
	//   g) mutate `sub[0] = 111`
	//   h) print `nums` again — did IT change this time? Explain to yourself why not.
	nums := make([]int, 0, 2)

	nums = append(nums, 10)
	fmt.Printf("len=%d cap=%d\n", len(nums), cap(nums))

	nums = append(nums, 20)
	fmt.Printf("len=%d cap=%d\n", len(nums), cap(nums))

	sub := nums[0:1]
	sub[0] = 999
	fmt.Println(nums)

	// Third elem appending
	nums = append(nums, 30)
	sub[0] = 111
	fmt.Println(nums)

	// TODO 5: write a function `removeSong(pl []string, index int) []string`
	// that removes the song at `index` and returns the new slice
	// (hint: append(pl[:index], pl[index+1:]...) — note the `...` which
	// "unpacks" a slice into individual arguments)
	// Call it on your playlist and print the result.
	fmt.Println(removeSong(playlist, 1))
}

// EXPECTED OUTPUT (roughly — exact values depend on your song names):
// [Song1 Song2 Song3]
// 1: Song1
// 2: Song2
// 3: Song3
// 4: Song4
// 5: Song5
// len=1 cap=2
// len=2 cap=2
// [999 20]          <- sub mutation DID affect nums (still sharing array)
// [999 20 30]
// [999 20 30]        <- sub mutation did NOT affect nums (reallocated after append)
// [Song1 Song3 Song4 Song5]   <- after removing index 1
