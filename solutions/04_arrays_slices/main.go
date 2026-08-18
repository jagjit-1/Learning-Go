package main

import "fmt"

func removeSong(pl []string, index int) []string {
	return append(pl[:index], pl[index+1:]...)
}

func main() {
	playlist := []string{"Song1", "Song2", "Song3"}
	fmt.Println(playlist)

	playlist = append(playlist, "Song4", "Song5")

	for i, song := range playlist {
		fmt.Printf("%d: %s\n", i+1, song)
	}

	// Sharing experiment
	nums := make([]int, 0, 2)
	nums = append(nums, 10)
	fmt.Println("len:", len(nums), "cap:", cap(nums))
	nums = append(nums, 20)
	fmt.Println("len:", len(nums), "cap:", cap(nums))

	sub := nums[0:1]
	sub[0] = 999
	fmt.Println(nums) // [999 20] - still sharing the array

	nums = append(nums, 30) // exceeds cap 2 -> reallocates
	sub[0] = 111            // mutates the OLD array, which nums no longer points to
	fmt.Println(nums)       // [999 20 30] - unaffected this time

	result := removeSong(playlist, 1)
	fmt.Println(result)
}
