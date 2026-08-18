package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Product struct {
	Name  string   `json:"name"`
	Price float64  `json:"price"`
	SKU   string   `json:"sku,omitempty"`
	Tags  []string `json:"tags,omitempty"`
	Cost  float64  `json:"-"`
}

func ToJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func FromJSON(data string, v any) error {
	return json.Unmarshal([]byte(data), v)
}

type Temperature float64

func (t Temperature) MarshalJSON() ([]byte, error) {
	s := strconv.FormatFloat(float64(t), 'f', -1, 64) + "C"
	return json.Marshal(s) // let json handle the quoting and escaping
}

func (t *Temperature) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if !strings.HasSuffix(s, "C") {
		return fmt.Errorf("temperature %q does not end in C", s)
	}
	f, err := strconv.ParseFloat(strings.TrimSuffix(s, "C"), 64)
	if err != nil {
		return fmt.Errorf("temperature %q: %w", s, err)
	}
	*t = Temperature(f)
	return nil
}

func FieldNames(data string) ([]string, error) {
	var obj map[string]any
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(obj))
	for k := range obj {
		names = append(names, k)
	}
	sort.Strings(names)
	return names, nil
}

func SumPrices(data string) (float64, error) {
	var products []Product
	if err := json.Unmarshal([]byte(data), &products); err != nil {
		return 0, err
	}
	total := 0.0
	for _, p := range products {
		total += p.Price
	}
	return total, nil
}

func DebugSetting(data string) (bool, bool, error) {
	var cfg struct {
		Debug *bool `json:"debug"`
	}
	if err := json.Unmarshal([]byte(data), &cfg); err != nil {
		return false, false, err
	}
	if cfg.Debug == nil {
		return false, false, nil // key absent
	}
	return true, *cfg.Debug, nil
}

func main() {
	full := Product{Name: "Widget", Price: 9.99, SKU: "W-1", Tags: []string{"new", "sale"}, Cost: 4}
	s, _ := ToJSON(full)
	fmt.Println(s)

	plain, _ := ToJSON(Product{Name: "Plain", Price: 1.5})
	fmt.Println(plain)

	var back Product
	FromJSON(s, &back) // note the &
	fmt.Println(back.Name, back.Price)

	warm, _ := ToJSON(Temperature(21.5))
	fmt.Println(warm)

	var cold Temperature
	FromJSON(`"-4C"`, &cold)
	fmt.Println(float64(cold))

	names, _ := FieldNames(`{"b":1,"a":2,"c":3}`)
	fmt.Println(names)

	total, _ := SumPrices(`[{"name":"a","price":10},{"name":"b","price":2.5}]`)
	fmt.Println(total)

	set, value, _ := DebugSetting(`{}`)
	fmt.Println(set, value)
	set, value, _ = DebugSetting(`{"debug":false}`)
	fmt.Println(set, value)
}
