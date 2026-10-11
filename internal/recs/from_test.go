package recs

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func rel(id catalog.ID, title string, votes int) catalog.Related {
	return catalog.Related{
		Result: catalog.Result{ID: id, Title: title, Year: 2000, Rating: 7.5, Overview: "overview"},
		Votes:  votes,
	}
}

func pickIDs(picks []Pick) []catalog.ID {
	ids := make([]catalog.ID, len(picks))
	for i, p := range picks {
		ids[i] = p.ID
	}
	return ids
}

func TestCheckLimit(t *testing.T) {
	if err := CheckLimit(1); err != nil {
		t.Errorf("CheckLimit(1) = %v, want nil", err)
	}
	if err := CheckLimit(50); err != nil {
		t.Errorf("CheckLimit(50) = %v, want nil", err)
	}
	for _, n := range []int{0, 51, -1} {
		err := CheckLimit(n)
		if err == nil || !strings.Contains(err.Error(), "invalid --limit") || !strings.Contains(err.Error(), "want 1 to 50") {
			t.Errorf("CheckLimit(%d) = %v, want invalid --limit ... want 1 to 50", n, err)
		}
	}
}

func TestParseTypeFilter(t *testing.T) {
	cases := []struct {
		in   string
		want catalog.MediaType
	}{
		{"", ""},
		{"any", ""},
		{" ANY ", ""},
		{"movie", catalog.Movie},
		{" TV ", catalog.TV},
	}
	for _, tc := range cases {
		got, err := ParseTypeFilter(tc.in)
		if err != nil {
			t.Errorf("ParseTypeFilter(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseTypeFilter(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	_, err := ParseTypeFilter("show")
	if err == nil || !strings.Contains(err.Error(), `invalid --type "show"`) {
		t.Errorf("ParseTypeFilter(show) = %v, want invalid --type \"show\"", err)
	}
}

func TestMergeSame(t *testing.T) {
	self := catalog.ID{Type: catalog.Movie, N: 1}
	recs := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 2}, "A", 5000),
		rel(catalog.ID{Type: catalog.Movie, N: 1}, "self", 9000),
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "B", 0),
	}
	sims := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "B", 800),
		rel(catalog.ID{Type: catalog.Movie, N: 4}, "C", 99),
		rel(catalog.ID{Type: catalog.Movie, N: 5}, "D", 100),
		rel(catalog.ID{Type: catalog.Movie, N: 6}, "E", 5000),
	}

	got := mergeSame(self, "X", recs, sims, 10)
	wantIDs := []catalog.ID{
		{Type: catalog.Movie, N: 2},
		{Type: catalog.Movie, N: 3},
		{Type: catalog.Movie, N: 5},
		{Type: catalog.Movie, N: 6},
	}
	if !reflect.DeepEqual(pickIDs(got), wantIDs) {
		t.Fatalf("ids = %v, want %v", pickIDs(got), wantIDs)
	}
	wantReasons := []string{
		"TMDB recommends it for X",
		"TMDB recommends it for X",
		"similar to X",
		"similar to X",
	}
	gotReasons := make([]string, len(got))
	for i, p := range got {
		gotReasons[i] = p.Reason
	}
	if !reflect.DeepEqual(gotReasons, wantReasons) {
		t.Errorf("reasons = %v, want %v", gotReasons, wantReasons)
	}

	if ids := pickIDs(mergeSame(self, "X", recs, sims, 3)); !reflect.DeepEqual(ids, wantIDs[:3]) {
		t.Errorf("limit 3 ids = %v, want %v", ids, wantIDs[:3])
	}
	if ids := pickIDs(mergeSame(self, "X", recs, sims, 1)); !reflect.DeepEqual(ids, wantIDs[:1]) {
		t.Errorf("limit 1 ids = %v, want %v", ids, wantIDs[:1])
	}
}

func TestMergeSameCopiesFields(t *testing.T) {
	self := catalog.ID{Type: catalog.Movie, N: 1}
	recs := []catalog.Related{rel(catalog.ID{Type: catalog.Movie, N: 2}, "A", 10)}
	got := mergeSame(self, "X", recs, nil, 10)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Title != "A" || got[0].Year != 2000 || got[0].Rating != 7.5 {
		t.Errorf("pick = %+v, want title A year 2000 rating 7.5", got[0])
	}
}

func TestMergeSameEmpty(t *testing.T) {
	got := mergeSame(catalog.ID{Type: catalog.Movie, N: 1}, "X", nil, nil, 10)
	if got == nil {
		t.Fatal("mergeSame = nil, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}
