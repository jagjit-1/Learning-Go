package main

// ============================================================
// CHECKER for 20_json — run with:  go test
// ============================================================
// The exact-JSON comparisons are fair game here: the TODO fixes both the tag
// names and the field order, and Marshal emits fields in declaration order.

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("could not create pipe: %v", err)
	}
	os.Stdout = w

	done := make(chan string, 1)
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r)
		done <- buf.String()
	}()

	func() {
		defer func() {
			os.Stdout = old
			w.Close()
			if rec := recover(); rec != nil {
				t.Errorf("main() panicked: %v", rec)
			}
		}()
		fn()
	}()

	return <-done
}

var (
	_ func(any) (string, error)        = ToJSON
	_ func(string, any) error          = FromJSON
	_ func(string) ([]string, error)   = FieldNames
	_ func(string) (float64, error)    = SumPrices
	_ func(string) (bool, bool, error) = DebugSetting
	_ json.Marshaler                   = Temperature(0)
	_ json.Unmarshaler                 = (*Temperature)(nil)
	_                                  = Product{Name: "n", Price: 1, SKU: "s", Tags: nil, Cost: 0}
)

// --- TODOs 1 & 2: the struct and its tags -------------------------------

func TestProductMarshalsAllFields(t *testing.T) {
	got, err := ToJSON(Product{
		Name: "Widget", Price: 9.99, SKU: "W-1",
		Tags: []string{"new", "sale"}, Cost: 4,
	})
	if err != nil {
		t.Fatalf("TODO 2: ToJSON returned %v", err)
	}

	want := `{"name":"Widget","price":9.99,"sku":"W-1","tags":["new","sale"]}`
	if got != want {
		t.Errorf("TODO 1: got\n  %s\nwant\n  %s\n"+
			"  Check: lowercase tag names, fields in the declared order, and Cost\n"+
			"  tagged `json:\"-\"` so it never appears.", got, want)
	}
}

func TestProductOmitsEmptyFields(t *testing.T) {
	got, err := ToJSON(Product{Name: "Plain", Price: 1.5})
	if err != nil {
		t.Fatalf("TODO 2: ToJSON returned %v", err)
	}

	want := `{"name":"Plain","price":1.5}`
	if got != want {
		t.Errorf("TODO 1: got\n  %s\nwant\n  %s\n"+
			"  SKU and Tags need `,omitempty` so an empty string and a nil slice "+
			"drop out entirely.", got, want)
	}
}

func TestProductNeverLeaksCost(t *testing.T) {
	got, _ := ToJSON(Product{Name: "x", Price: 1, Cost: 999})
	if strings.Contains(got, "999") || strings.Contains(strings.ToLower(got), "cost") {
		t.Errorf("TODO 1: Cost showed up in %s — tag it `json:\"-\"`", got)
	}
}

func TestZeroPriceIsNotOmitted(t *testing.T) {
	got, _ := ToJSON(Product{Name: "Free", Price: 0})
	if !strings.Contains(got, `"price":0`) {
		t.Errorf("TODO 1: got %s — Price has no omitempty, so a legitimate 0 must "+
			"still be emitted", got)
	}
}

// --- TODO 3: FromJSON ---------------------------------------------------

func TestFromJSONRoundTrip(t *testing.T) {
	original := Product{Name: "Widget", Price: 9.99, SKU: "W-1", Tags: []string{"new"}}
	data, _ := ToJSON(original)

	var back Product
	if err := FromJSON(data, &back); err != nil {
		t.Fatalf("TODO 3: FromJSON returned %v", err)
	}

	if back.Name != "Widget" || back.Price != 9.99 || back.SKU != "W-1" {
		t.Errorf("TODO 3: round trip gave %+v, want the original back", back)
	}
	if len(back.Tags) != 1 || back.Tags[0] != "new" {
		t.Errorf("TODO 3: Tags came back as %v, want [new]", back.Tags)
	}
}

func TestFromJSONIgnoresUnknownKeysAndMissingOnes(t *testing.T) {
	var p Product
	if err := FromJSON(`{"name":"OnlyName","surprise":42}`, &p); err != nil {
		t.Fatalf("TODO 3: an unknown key should be ignored, not an error — got %v", err)
	}
	if p.Name != "OnlyName" {
		t.Errorf("TODO 3: Name = %q, want %q", p.Name, "OnlyName")
	}
	if p.Price != 0 {
		t.Errorf("TODO 3: a missing key should leave the zero value, got Price = %v", p.Price)
	}
}

func TestFromJSONReportsBadInput(t *testing.T) {
	var p Product
	if err := FromJSON(`{"name": }`, &p); err == nil {
		t.Error("TODO 3: malformed JSON should return an error")
	}
	if err := FromJSON(`{"price":"not a number"}`, &p); err == nil {
		t.Error("TODO 3: a type mismatch should return an error")
	}
}

// --- TODO 4: Temperature -------------------------------------------------

func TestTemperatureMarshals(t *testing.T) {
	cases := []struct {
		in   Temperature
		want string
	}{
		{21.5, `"21.5C"`},
		{-4, `"-4C"`},
		{0, `"0C"`},
	}
	for _, c := range cases {
		got, err := ToJSON(c.in)
		if err != nil {
			t.Errorf("TODO 4: marshalling %v returned %v", float64(c.in), err)
			continue
		}
		if got != c.want {
			t.Errorf("TODO 4: Temperature(%v) marshalled to %s, want %s\n"+
				"  The result must be a complete JSON value, quotes included, and\n"+
				"  strconv.FormatFloat(f, 'f', -1, 64) avoids trailing zeros.",
				float64(c.in), got, c.want)
		}
	}
}

func TestTemperatureUnmarshals(t *testing.T) {
	var got Temperature
	if err := FromJSON(`"-4C"`, &got); err != nil {
		t.Fatalf("TODO 4: unmarshalling \"-4C\" returned %v", err)
	}
	if got != Temperature(-4) {
		t.Errorf("TODO 4: got %v, want -4", float64(got))
	}

	var warm Temperature
	FromJSON(`"21.5C"`, &warm)
	if warm != Temperature(21.5) {
		t.Errorf("TODO 4: got %v, want 21.5", float64(warm))
	}
}

func TestTemperatureRejectsBadInput(t *testing.T) {
	var got Temperature
	if err := FromJSON(`"21.5F"`, &got); err == nil {
		t.Error("TODO 4: a value not ending in C should be an error")
	}
	if err := FromJSON(`"abcC"`, &got); err == nil {
		t.Error("TODO 4: a non-numeric temperature should be an error")
	}
}

func TestTemperatureInsideAStruct(t *testing.T) {
	// Custom marshallers have to work when nested, not just standalone.
	type reading struct {
		At   string      `json:"at"`
		Temp Temperature `json:"temp"`
	}
	got, err := ToJSON(reading{At: "noon", Temp: 18})
	if err != nil {
		t.Fatalf("TODO 4: %v", err)
	}
	if got != `{"at":"noon","temp":"18C"}` {
		t.Errorf("TODO 4: nested marshalling gave %s, want %s",
			got, `{"at":"noon","temp":"18C"}`)
	}
}

// --- TODO 5: FieldNames --------------------------------------------------

func TestFieldNames(t *testing.T) {
	got, err := FieldNames(`{"b":1,"a":2,"c":3}`)
	if err != nil {
		t.Fatalf("TODO 5: FieldNames returned %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(got) != 3 {
		t.Fatalf("TODO 5: got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("TODO 5: got %v, want %v (sorted)", got, want)
		}
	}
}

func TestFieldNamesHandlesMixedTypes(t *testing.T) {
	got, err := FieldNames(`{"n":1,"s":"x","b":true,"arr":[1,2],"obj":{"k":1},"nul":null}`)
	if err != nil {
		t.Fatalf("TODO 5: FieldNames returned %v — map[string]any accepts any "+
			"value type, so nothing here should fail: %v", err, err)
	}
	if len(got) != 6 {
		t.Errorf("TODO 5: got %v, want 6 keys", got)
	}
}

func TestFieldNamesRejectsNonObjects(t *testing.T) {
	if _, err := FieldNames(`[1,2,3]`); err == nil {
		t.Error("TODO 5: a JSON array cannot unmarshal into map[string]any, so " +
			"this should return an error")
	}
	if _, err := FieldNames(`not json`); err == nil {
		t.Error("TODO 5: malformed JSON should return an error")
	}
}

// --- TODO 6: SumPrices ---------------------------------------------------

func TestSumPrices(t *testing.T) {
	got, err := SumPrices(`[{"name":"a","price":10},{"name":"b","price":2.5}]`)
	if err != nil {
		t.Fatalf("TODO 6: SumPrices returned %v", err)
	}
	if got != 12.5 {
		t.Errorf("TODO 6: got %v, want 12.5", got)
	}

	if got, err := SumPrices(`[]`); err != nil || got != 0 {
		t.Errorf("TODO 6: empty array gave (%v, %v), want (0, nil)", got, err)
	}
	if _, err := SumPrices(`{"name":"a"}`); err == nil {
		t.Error("TODO 6: an object is not an array of products; expected an error")
	}
}

// --- TODO 7: absent vs false ---------------------------------------------

func TestDebugSetting(t *testing.T) {
	cases := []struct {
		in      string
		wantSet bool
		wantVal bool
	}{
		{`{}`, false, false},
		{`{"debug":false}`, true, false},
		{`{"debug":true}`, true, true},
		{`{"other":1}`, false, false},
		{`{"debug":null}`, false, false},
	}
	for _, c := range cases {
		set, val, err := DebugSetting(c.in)
		if err != nil {
			t.Errorf("TODO 7: DebugSetting(%s) returned %v", c.in, err)
			continue
		}
		if set != c.wantSet || val != c.wantVal {
			t.Errorf("TODO 7: DebugSetting(%s) = (set=%v, value=%v), want (set=%v, value=%v)\n"+
				"  A plain `bool` field cannot tell absent from false — both leave it\n"+
				"  at the zero value. A *bool stays nil when the key wasn't there.",
				c.in, set, val, c.wantSet, c.wantVal)
		}
	}

	if _, _, err := DebugSetting(`{`); err == nil {
		t.Error("TODO 7: malformed JSON should return an error")
	}
}

// --- main()'s narration ---------------------------------------------------

func TestOutput(t *testing.T) {
	out := captureStdout(t, main)
	if strings.TrimSpace(out) == "" {
		t.Fatal("main() printed nothing — start with TODO 8")
	}

	checks := []struct {
		re   *regexp.Regexp
		todo string
		want string
	}{
		{regexp.MustCompile(`\{"name":"Widget","price":9\.99,"sku":"W-1","tags":\["new","sale"\]\}`),
			"TODO 8", "the fully populated product JSON"},
		{regexp.MustCompile(`(?m)^\{"name":"[^"]+","price":[\d.]+\}$`),
			"TODO 9", "a product JSON with sku and tags omitted"},
		{regexp.MustCompile(`(?m)^Widget 9\.99$`), "TODO 10", "\"Widget 9.99\" after the round trip"},
		{regexp.MustCompile(`(?m)^"21\.5C"$`), "TODO 11", `"21.5C"`},
		{regexp.MustCompile(`(?m)^-4$`), "TODO 11", "-4 back from the custom unmarshaller"},
		{regexp.MustCompile(`\[a b c\]`), "TODO 12", "[a b c]"},
		{regexp.MustCompile(`(?m)^12\.5$`), "TODO 13", "12.5"},
		{regexp.MustCompile(`(?m)^false false$`), "TODO 14", "\"false false\" for {}"},
		{regexp.MustCompile(`(?m)^true false$`), "TODO 14", "\"true false\" for {\"debug\":false}"},
	}
	for _, c := range checks {
		if !c.re.MatchString(out) {
			t.Errorf("%s: expected %s.\n  your output was:\n%s", c.todo, c.want, indent(out))
		}
	}
}

func indent(s string) string {
	return "    " + strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n    ")
}
