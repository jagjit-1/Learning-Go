package main

import (
	"errors"
	"fmt"
	"strconv"
)

func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, errors.New("division by zero")
	}
	return a / b, nil
}

type NegativeAgeError struct {
	Value int
}

func (e *NegativeAgeError) Error() string {
	return fmt.Sprintf("negative age not allowed: %d", e.Value)
}

func ParseAge(input string) (int, error) {
	n, err := strconv.Atoi(input)
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, &NegativeAgeError{Value: n}
	}
	if n > 150 {
		return 0, fmt.Errorf("age must be between 0 and 150")
	}
	return n, nil
}

func main() {
	if result, err := Divide(10, 2); err != nil {
		fmt.Println("Error:", err)
	} else {
		fmt.Printf("Result: %.2f\n", result)
	}

	if _, err := Divide(10, 0); err != nil {
		fmt.Println("Error:", err)
	}

	for _, input := range []string{"25", "abc", "-5", "200"} {
		age, err := ParseAge(input)
		if err != nil {
			fmt.Println("Error parsing age:", err)
			continue
		}
		fmt.Println("Parsed age:", age)
	}

	_, err := ParseAge("-10")
	var negErr *NegativeAgeError
	if errors.As(err, &negErr) {
		fmt.Printf("Rejected negative age: %d\n", negErr.Value)
	} else if err != nil {
		fmt.Println("Some other error occurred:", err)
	}
}
