package indexer

import (
	"testing"
)

func TestQueryValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		q       Query
		wantErr bool
	}{
		{"text only", Query{Text: "inception"}, false},
		{"hash only", Query{InfoHash: "abc"}, false},
		{"empty", Query{}, true},
		{"bad sort", Query{Text: "x", Sort: "nope"}, true},
		{"bad type", Query{Text: "x", Type: "documentary"}, true},
		{"valid sort", Query{Text: "x", Sort: "size"}, false},
		{"valid type", Query{Text: "x", Type: TV}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.q.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSortReleases(t *testing.T) {
	t.Parallel()

	releases := []Release{
		{Provider: "b", Title: "low", Seeders: 1, Size: 50},
		{Provider: "a", Title: "high", Seeders: 100, Size: 10},
		{Provider: "a", Title: "mid", Seeders: 50, Size: 30},
	}

	SortReleases(releases, "seeders")
	if releases[0].Seeders != 100 || releases[1].Seeders != 50 || releases[2].Seeders != 1 {
		t.Errorf("seeders order wrong: %+v", releases)
	}

	SortReleases(releases, "size")
	if releases[0].Size != 50 || releases[1].Size != 30 || releases[2].Size != 10 {
		t.Errorf("size order wrong: %+v", releases)
	}
}

func TestSortReleasesStableTieBreak(t *testing.T) {
	t.Parallel()

	releases := []Release{
		{Provider: "b", Title: "z", Seeders: 5},
		{Provider: "a", Title: "m", Seeders: 5},
		{Provider: "a", Title: "b", Seeders: 5},
	}
	SortReleases(releases, "seeders")

	got := []string{releases[0].Provider + releases[0].Title, releases[1].Provider + releases[1].Title, releases[2].Provider + releases[2].Title}
	want := []string{"ab", "am", "bz"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("tie order = %v, want %v", got, want)
		}
	}
}

func TestDedupeReleases(t *testing.T) {
	t.Parallel()

	releases := []Release{
		{Provider: "a", Title: "one", InfoHash: "h1", Seeders: 10},
		{Provider: "b", Title: "two", InfoHash: "h1", Seeders: 99},
		{Provider: "a", Title: "three", InfoHash: "h2", Seeders: 5},
		{Provider: "a", Title: "nohash-a", InfoHash: "", Seeders: 1},
		{Provider: "b", Title: "nohash-b", InfoHash: "", Seeders: 1},
	}

	got := DedupeReleases(releases)
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4: %+v", len(got), got)
	}
	if got[0].InfoHash != "h1" || got[0].Seeders != 99 {
		t.Errorf("h1 should keep the higher-seeded entry, got %+v", got[0])
	}
	if got[1].InfoHash != "h2" {
		t.Errorf("expected h2 second, got %+v", got[1])
	}
	if got[2].InfoHash != "" || got[3].InfoHash != "" {
		t.Errorf("empty-hash entries should both survive, got %+v", got[2:])
	}
}

func TestDedupeReleasesEmptyInput(t *testing.T) {
	t.Parallel()

	if got := DedupeReleases(nil); got == nil {
		t.Error("DedupeReleases(nil) returned nil, want non-nil empty slice")
	}
}
