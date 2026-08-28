package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
)

// ============================================================
// CONCEPT: encoding/json
// ============================================================
//
// Two functions do most of the work:
//
//   data, err := json.Marshal(v)        // Go value  -> []byte
//   err := json.Unmarshal(data, &v)     // []byte    -> Go value
//
// Unmarshal needs a POINTER. It has to write into your variable, and Go is
// pass-by-value, so `json.Unmarshal(data, v)` compiles and silently does
// nothing useful — it returns an InvalidUnmarshalError.
//
// ONLY EXPORTED FIELDS are marshalled. A lowercase field is invisible to the
// json package (it uses reflection, which cannot read unexported fields).
// This is the number one "why is my JSON empty" cause.
//
// STRUCT TAGS control the mapping:
//
//   type Product struct {
//       Name  string   `json:"name"`
//       Price float64  `json:"price"`
//       SKU   string   `json:"sku,omitempty"`   // dropped when empty
//       Cost  float64  `json:"-"`               // never marshalled at all
//   }
//
// Field ORDER in the output follows declaration order in the struct.
//
// omitempty drops the field when the value is the zero value: "", 0, false,
// nil, or an empty slice/map. Note what that does NOT mean — it can't tell
// "absent" from "explicitly zero". If that distinction matters, use a POINTER:
//
//   Debug *bool `json:"debug"`
//
//   nil        -> the key was absent
//   &false     -> the key was present and false
//
// Unmarshalling is FORGIVING by default. Unknown keys in the JSON are
// ignored, and missing keys leave the field at its zero value. Neither is an
// error. (json.Decoder.DisallowUnknownFields() opts into strictness.)
//
// Field matching is case-insensitive when there's no tag, so "name" fills
// Name. Rely on tags anyway — they're the contract.
//
// DYNAMIC JSON, when you don't know the shape: unmarshal into
// map[string]any or just `any`. Everything arrives as one of:
//
//   float64   for EVERY number — there is no int in JSON. A big int64 can
//             lose precision this way; use json.Number if that matters.
//   string, bool, nil
//   []any            for arrays
//   map[string]any   for objects
//
// You then need type assertions to get anything out, which is why a struct
// is better whenever you know the shape.
//
// CUSTOM ENCODING — implement the interfaces and json will call them:
//
//   func (t Temperature) MarshalJSON() ([]byte, error)
//   func (t *Temperature) UnmarshalJSON(data []byte) error
//
// MarshalJSON returns the COMPLETE JSON value including quotes if it's a
// string: []byte(`"21.5C"`), not []byte(`21.5C`). UnmarshalJSON needs a
// pointer receiver, because it assigns to the receiver.
//
// STREAMING, for large input or many documents: json.NewDecoder(r).Decode(&v)
// reads from an io.Reader instead of holding it all in memory, and
// json.NewEncoder(w).Encode(v) writes straight out. Note Encode appends a
// newline; Marshal does not.

// TODO 1: define `type Product struct` with these fields IN THIS ORDER —
// the checker compares the JSON exactly, and order follows declaration:
//
//	Name  string    -> "name"
//	Price float64   -> "price"
//	SKU   string    -> "sku", omitted when empty
//	Tags  []string  -> "tags", omitted when empty
//	Cost  float64   -> never appears in JSON at all
type Product struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	SKU   string   `json:"sku,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	Cost  float64  `json:"-"`
}

// TODO 2: write `func ToJSON(v any) (string, error)` wrapping json.Marshal.
func ToJSON(v any) (string, error) {
	val, err := json.Marshal(v)
	return string(val), err
}

// TODO 3: write `func FromJSON(data string, v any) error` wrapping
// json.Unmarshal. Callers pass a pointer; you just pass it through.
func FromJSON(data string, v any) error {
	return json.Unmarshal([]byte(data), &v)
}

// TODO 4: define `type Temperature float64` that marshals as a string with a
// C suffix — 21.5 becomes "21.5C" — and unmarshals back. Use
// strconv.FormatFloat(f, 'f', -1, 64) so 21.5 doesn't become 21.500000.
// Reject a value that doesn't end in C with an error.
type Temperature float64

func (t Temperature) MarshalJSON() ([]byte, error) {
	val := strconv.FormatFloat(float64(t), 'f', -1, 64) + "C"
	return json.Marshal(val)
}

func (t *Temperature) UnmarshalJSON(data []byte) error {
	var sval string
	err := json.Unmarshal(data, &sval)

	if err != nil {
		return err
	}

	if sval[len(sval)-1] != 'C' {
		return errors.New("Invalid json value for temperature")
	}

	floatVal, err1 := strconv.ParseFloat(sval[:len(sval)-1], 64)
	if err1 != nil {
		return err1
	}

	*t = Temperature(floatVal)

	return nil
}

// TODO 5: write `func FieldNames(data string) ([]string, error)` that takes
// an arbitrary JSON object and returns its top-level keys, SORTED.
// Unmarshal into a map[string]any.
func FieldNames(data string) ([]string, error) {
	val := map[string]any{}

	err := json.Unmarshal([]byte(data), &val)

	if err != nil {
		return []string{}, err
	}
	res := []string{}
	for key := range val {
		res = append(res, key)
	}
	slices.Sort(res)

	return res, nil
}

// TODO 6: write `func SumPrices(data string) (float64, error)` taking a JSON
// ARRAY of products and returning the total price.
func SumPrices(data string) (float64, error) {
	val := []map[string]any{}

	err := json.Unmarshal([]byte(data), &val)

	if err != nil {
		return 0, err
	}

	var res float64

	for _, v := range val {
		v, ok := v["price"]
		
		if tv, tok := v.(float64); ok && tok {
			res += tv
		}
	}

	return res, nil
}

// TODO 7: write `func DebugSetting(data string) (set bool, value bool, err error)`
// for JSON like {"debug": false}. `set` reports whether the key was present
// at all. Use a struct with a *bool field — this is the case omitempty
// cannot express.
func DebugSetting(data string) (set bool, value bool, err error) {
	val := Dummy{}

	err = json.Unmarshal([]byte(data), &val)
	if err != nil {
		return false, false, err
	}
	if val.Debug != nil {
		set = true
		value = *val.Debug
	}
	return
}

type Dummy struct {
	Debug *bool `json:"debug,omitempty"`
}

func main() {
	// TODO 8: marshal a fully populated Product and print the JSON.
	p := Product{Name: "Widget", Price: 9.99, SKU: "W-1", Tags: []string{"new", "sale"}}
	sp, _ := json.Marshal(p)
	fmt.Println(string(sp))
	// TODO 9: marshal a Product with no SKU and no Tags — print it and note
	// which keys vanished.
	p1 := Product{Name: "Plain", Price: 1.5}
	sp1, _ := json.Marshal(p1)
	fmt.Println(string(sp1))
	// TODO 10: round-trip: unmarshal the JSON from TODO 8 back into a Product
	// and print its Name and Price.
	up := Product{}
	json.Unmarshal(sp, &up)
	fmt.Println(up.Name, up.Price)
	// TODO 11: marshal a Temperature of 21.5 and print it, then unmarshal
	// `"-4C"` and print the float you get back.
	t := Temperature(21.5)
	st, _ := json.Marshal(t)
	fmt.Println(string(st))

	t1 := Temperature(0)
	json.Unmarshal([]byte(`"-4C"`), &t1)
	fmt.Println(t1)
	// TODO 12: print FieldNames of `{"b":1,"a":2,"c":3}`.
	vals, _ := FieldNames(`{"b":1,"a":2,"c":3}`)
	fmt.Println(vals)
	// TODO 13: print SumPrices of a two-product JSON array.
	psum, _ := SumPrices(`[{"price":6}, {"price":6.5}]`)
	fmt.Println(psum)
	// TODO 14: print DebugSetting for `{}` and for `{"debug":false}`.
	set, val, _ := DebugSetting(`{}`)
	fmt.Println(set, val)
	set, val, _ = DebugSetting(`{"debug":false}`)
	fmt.Println(set, val)
}

// EXPECTED OUTPUT:
// {"name":"Widget","price":9.99,"sku":"W-1","tags":["new","sale"]}
// {"name":"Plain","price":1.5}
// Widget 9.99
// "21.5C"
// -4
// [a b c]
// 12.5
// false false
// true false
