package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func TestRateCommand(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	tests := []struct {
		rating string
		want   string
	}{
		{"9", "Rated Inception (2010) [movie:27205] 9/10\n"},
		{"7", "Rated Inception (2010) [movie:27205] 7/10 (was 9)\n"},
		{"3", "Rated Inception (2010) [movie:27205] 3/10 (was 7, counts as disliked)\n"},
		{"3", "Rated Inception (2010) [movie:27205] 3/10 (counts as disliked)\n"},
	}
	for _, tt := range tests {
		out, err := runRootCmd(t, "rate", "movie:27205", tt.rating, "--config", cfg)
		if err != nil {
			t.Fatalf("rate %s Execute: %v", tt.rating, err)
		}
		if out != tt.want {
			t.Errorf("rate %s stdout = %q, want %q", tt.rating, out, tt.want)
		}
	}
}

func TestRateEntry(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "rate", "movie:27205", "9", "--config", cfg); err != nil {
		t.Fatalf("rate Execute: %v", err)
	}
	e, err := openTestLibrary(t).Entry(context.Background(), catalog.ID{Type: catalog.Movie, N: 27205})
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if e.Rating != 9 {
		t.Errorf("Rating = %d, want 9", e.Rating)
	}
	if !e.Watched {
		t.Error("Watched = false, want true")
	}
	if e.Liked {
		t.Error("Liked = true, want false")
	}
}

func TestRateFirst(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "rate", "inception", "9", "--first", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Rated Inception (2010) [movie:27205] 9/10\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestRateJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "rate", "movie:27205", "9", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got actionOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Action != "rate" {
		t.Errorf("Action = %q, want %q", got.Action, "rate")
	}
	if got.Rating != 9 {
		t.Errorf("Rating = %d, want 9", got.Rating)
	}
	if !got.Changed {
		t.Error("Changed = false, want true")
	}

	out, err = runRootCmd(t, "rate", "movie:27205", "9", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("second Unmarshal: %v", err)
	}
	if got.Changed {
		t.Error("second Changed = true, want false")
	}
}

func TestRateInvalid(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	for _, n := range []string{"11", "0", "abc", "7.5"} {
		_, err := runRootCmd(t, "rate", "movie:27205", n, "--config", cfg)
		if err == nil {
			t.Fatalf("rate %s Execute = nil, want error", n)
		}
		if !strings.Contains(err.Error(), "invalid rating") {
			t.Errorf("rate %s error = %q, want it to contain %q", n, err, "invalid rating")
		}
		if strings.Contains(err.Error(), "config init") {
			t.Errorf("rate %s error = %q, want no config error", n, err)
		}
	}

	if _, err := runRootCmd(t, "rate", "movie:27205", "--config", cfg); err == nil {
		t.Fatal("rate with one argument = nil, want error")
	}
}

func TestRateRemovesWatchlist(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()
	id := catalog.ID{Type: catalog.Movie, N: 27205}

	lib := openTestLibrary(t)
	if err := lib.UpsertItem(ctx, &catalog.Meta{ID: id, Title: "Inception", Year: 2010}); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.AddToWatchlist(ctx, id); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	if _, err := runRootCmd(t, "rate", "movie:27205", "8", "--config", cfg); err != nil {
		t.Fatalf("rate Execute: %v", err)
	}

	entries, err := openTestLibrary(t).Watchlist(ctx)
	if err != nil {
		t.Fatalf("Watchlist: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("watchlist length = %d, want 0", len(entries))
	}
}

func TestParseRating(t *testing.T) {
	if n, err := parseRating(" 10 "); err != nil || n != 10 {
		t.Errorf("parseRating( 10 ) = %d, %v; want 10, nil", n, err)
	}
	if n, err := parseRating("1"); err != nil || n != 1 {
		t.Errorf("parseRating(1) = %d, %v; want 1, nil", n, err)
	}
}

func TestUnrateCommand(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "rate", "movie:27205", "9", "--config", cfg); err != nil {
		t.Fatalf("rate Execute: %v", err)
	}

	out, err := runRootCmd(t, "unrate", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("unrate Execute: %v", err)
	}
	if want := "Removed your rating from Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "unrate", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second unrate Execute: %v", err)
	}
	if want := "You have not rated: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	e, err := openTestLibrary(t).Entry(context.Background(), catalog.ID{Type: catalog.Movie, N: 27205})
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !e.Watched {
		t.Error("Watched = false, want true")
	}
}

func TestUnrateNoToken(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	out, err := runRootCmd(t, "unrate", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "You have not rated: movie:27205\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}
