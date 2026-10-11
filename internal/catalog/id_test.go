package catalog

import (
	"encoding/json"
	"testing"
)

func TestParseID(t *testing.T) {
	cases := []struct {
		in   string
		want ID
	}{
		{"movie:27205", ID{Movie, 27205}},
		{"tv:1399", ID{TV, 1399}},
	}
	for _, tc := range cases {
		got, err := ParseID(tc.in)
		if err != nil {
			t.Errorf("ParseID(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseID(%q) = %+v, want %+v", tc.in, got, tc.want)
		}
	}
}

func TestParseIDErrors(t *testing.T) {
	for _, in := range []string{"inception", "movie:", "movie:abc", "person:5", "movie:-1", ""} {
		if got, err := ParseID(in); err == nil {
			t.Errorf("ParseID(%q) = %+v, want error", in, got)
		}
	}
}

func TestIDString(t *testing.T) {
	if got := (ID{TV, 1399}).String(); got != "tv:1399" {
		t.Errorf("String() = %q, want %q", got, "tv:1399")
	}
}

func TestIDMarshalJSON(t *testing.T) {
	b, err := json.Marshal(ID{Movie, 27205})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(b), `"movie:27205"`; got != want {
		t.Errorf("Marshal = %s, want %s", got, want)
	}
}

func TestIDUnmarshalJSON(t *testing.T) {
	for _, want := range []ID{{Movie, 27205}, {TV, 1399}} {
		b, err := json.Marshal(want)
		if err != nil {
			t.Fatalf("Marshal(%+v): %v", want, err)
		}
		var got ID
		if err := json.Unmarshal(b, &got); err != nil {
			t.Fatalf("Unmarshal(%s): %v", b, err)
		}
		if got != want {
			t.Errorf("round trip = %+v, want %+v", got, want)
		}
	}

	var got ID
	if err := json.Unmarshal([]byte(`"person:5"`), &got); err == nil {
		t.Errorf("Unmarshal invalid id = %+v, want error", got)
	}
	if got != (ID{}) {
		t.Errorf("failed Unmarshal changed the value to %+v", got)
	}
}
