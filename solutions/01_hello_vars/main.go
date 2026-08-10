package main

import "fmt"

func main() {
	name := "Jagjit"
	age := 24
	var heightMeters float64 = 1.75
	isLearningGo := true

	fmt.Printf("%s is %d years old, %.2fm tall. Learning Go: %t\n",
		name, age, heightMeters, isLearningGo)

	ageFloat := float64(age)
	fmt.Println(ageFloat / 2)

	const daysInWeek = 7
	fmt.Printf("There are %d days in a week\n", daysInWeek)
}
