package main

import (
	"fmt"
	"strings"
)

type Contact struct {
	Name  string
	Phone string
	Email string
}

func findContact(book []Contact, name string) (Contact, bool) {
	for _, c := range book {
		if c.Name == name {
			return c, true
		}
	}
	return Contact{}, false
}

func main() {
	sentence := "the quick brown fox jumps over the lazy dog the fox runs"
	words := strings.Fields(sentence)

	freq := make(map[string]int)
	for _, w := range words {
		freq[w]++
	}
	for word, count := range freq {
		fmt.Printf("%s: %d\n", word, count)
	}

	if count, ok := freq["fox"]; ok {
		fmt.Printf("fox appears %d times\n", count)
	} else {
		fmt.Println("fox not found")
	}
	if count, ok := freq["cat"]; ok {
		fmt.Printf("cat appears %d times\n", count)
	} else {
		fmt.Println("cat not found")
	}

	addressBook := []Contact{
		{Name: "Alice", Phone: "111-1111", Email: "alice@example.com"},
		{Name: "Bob", Phone: "222-2222", Email: "bob@example.com"},
		{Name: "Carol", Phone: "333-3333", Email: "carol@example.com"},
	}

	for _, c := range addressBook {
		fmt.Printf("Name: %s, Phone: %s\n", c.Name, c.Phone)
	}

	if c, found := findContact(addressBook, "Bob"); found {
		fmt.Printf("Found Bob: %+v\n", c)
	}
	if _, found := findContact(addressBook, "Dave"); !found {
		fmt.Println("Dave not in address book")
	}
}
