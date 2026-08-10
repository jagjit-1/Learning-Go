package main

import (
	"fmt"
	"strings"
)

// ============================================================
// CONCEPT: Maps and Structs
// ============================================================
//
// MAP — key-value store, like Dictionary<K,V> in C#:
//   m := make(map[string]int)
//   m["apple"] = 3
//   m["apple"]++             // maps of numbers support this directly
//   delete(m, "apple")
//
//   m := map[string]int{"apple": 3, "banana": 5}  // literal form
//
// THE "COMMA OK" IDIOM — checking if a key exists (there's no null/undefined,
// so a missing key returns the zero value, which is ambiguous with a real
// zero — this idiom disambiguates):
//   value, ok := m["apple"]
//   if ok {
//       fmt.Println("found:", value)
//   } else {
//       fmt.Println("not present")
//   }
//
// Iterating a map with range gives (key, value) — NOTE: map iteration
// order is intentionally RANDOMIZED by Go at runtime (unlike slices).
// This is deliberate, to stop people relying on insertion order.
//   for k, v := range m { }
//
// STRUCT — you've seen these already, formalizing here:
//   type Contact struct {
//       Name  string
//       Phone string
//   }
//   c := Contact{Name: "Jagjit", Phone: "555-1234"}  // field-named literal (preferred)
//   c2 := Contact{"Jagjit", "555-1234"}                // positional (fragile, avoid)
//
// strings.Fields(s) splits a string on whitespace into []string — useful
// for the word-count exercise below.
// strings.ToLower(s) lowercases a string.

type Contact struct {
	Name  string
	Phone string
	Email string
}

func findContact(book []Contact, name string) (Contact, bool) {
	for _, val := range book {
		if val.Name == name {
			return val, true
		}
	}
	return Contact{}, false
}

func main() {
	// TODO 1: word frequency counter —
	// given: sentence := "the quick brown fox jumps over the lazy dog the fox runs"
	// split it with strings.Fields, build a map[string]int counting each word,
	// then print each word and its count (order doesn't matter)
	sentence := "the quick brown fox jumps over the lazy dog the fox runs"
	parts := strings.Fields(sentence)
	wordCount := make(map[string]int)

	for _, word := range parts {
		wordCount[word]++
	}
	for word, cnt := range wordCount {
		fmt.Printf("%s: %d\n", word, cnt)
	}
	// TODO 2: using the comma-ok idiom, check whether "fox" and "cat" are
	// in your frequency map — print "fox appears N times" or "cat not found"
	// for each
	foxCount, foxFound := wordCount["fox"]
	_, catFound := wordCount["cat"]

	if foxFound {
		fmt.Println("fox appears", foxCount, "times")
	}

	if !catFound {
		fmt.Println("cat not found")
	}
	// TODO 3: define a `Contact` struct with Name, Phone, Email (all strings)

	// TODO 4: create a slice `addressBook := []Contact{...}` with 3 contacts
	// using field-named struct literals
	addressBook := []Contact{{Name: "A", Phone: "1", Email: "A"}, {Name: "B", Phone: "2", Email: "B"}, {Name: "C", Phone: "3", Email: "C"}}
	// TODO 5: range over addressBook and print each contact as
	// "Name: <name>, Phone: <phone>"
	for _, val := range addressBook {
		fmt.Printf("Name: %s, Phone: %s\n", val.Name, val.Phone)
	}
	// TODO 6: write a function `findContact(book []Contact, name string) (Contact, bool)`
	// that searches the slice for a matching Name and returns (contact, true)
	// or (Contact{}, false) if not found — this mirrors the comma-ok idiom
	// pattern for your OWN function, not just built-in maps
	_, found := findContact(addressBook, "A")
	fmt.Println("Address book of A, is present", found)
}

// EXPECTED OUTPUT (word counts will print in random order - that's correct!):
// the: 3
// quick: 1
// fox: 2
// ... (rest of the words)
// fox appears 2 times
// cat not found
// Name: Alice, Phone: 111-1111
// Name: Bob, Phone: 222-2222
// Name: Carol, Phone: 333-3333
// found / not found results from findContact
