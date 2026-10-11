package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestPrintTable(t *testing.T) {
	var out bytes.Buffer
	if err := printTable(&out, []string{"type", "title"}, [][]string{{"movie", "Inception"}}); err != nil {
		t.Fatalf("printTable: %v", err)
	}

	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2: %q", len(lines), out.String())
	}
	for _, want := range []string{"TYPE", "TITLE"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header line = %q, want it to contain %q", lines[0], want)
		}
	}
	for _, want := range []string{"movie", "Inception"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("row line = %q, want it to contain %q", lines[1], want)
		}
	}
}

func TestPrintJSON(t *testing.T) {
	var out bytes.Buffer
	want := map[string]int{"a": 1}
	if err := printJSON(&out, want); err != nil {
		t.Fatalf("printJSON: %v", err)
	}

	var got map[string]int
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("decoded = %+v, want %+v", got, want)
	}
}
