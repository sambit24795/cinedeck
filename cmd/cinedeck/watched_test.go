package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func TestWatchedCommand(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watched", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Marked as watched: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "watched", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if want := "Already marked as watched: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestWatchedJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watched", "tv:1396", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got actionOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Action != "watched" {
		t.Errorf("Action = %q, want %q", got.Action, "watched")
	}
}

func TestWatchedRemovesWatchlist(t *testing.T) {
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

	if _, err := runRootCmd(t, "watched", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("watched Execute: %v", err)
	}

	entries, err := openTestLibrary(t).Watchlist(ctx)
	if err != nil {
		t.Fatalf("Watchlist: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("watchlist length = %d, want 0", len(entries))
	}
}

func TestUnwatchCommand(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "watched", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("watched Execute: %v", err)
	}

	out, err := runRootCmd(t, "unwatch", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("unwatch Execute: %v", err)
	}
	if want := "Removed from watched: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "unwatch", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second unwatch Execute: %v", err)
	}
	if want := "Not marked as watched: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "unwatch", "movie:99", "--config", cfg)
	if err != nil {
		t.Fatalf("unknown unwatch Execute: %v", err)
	}
	if want := "Not marked as watched: movie:99\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestUnwatchKeepsRating(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "rate", "movie:27205", "9", "--config", cfg); err != nil {
		t.Fatalf("rate Execute: %v", err)
	}
	if _, err := runRootCmd(t, "unwatch", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("unwatch Execute: %v", err)
	}

	e, err := openTestLibrary(t).Entry(context.Background(), catalog.ID{Type: catalog.Movie, N: 27205})
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if e.Rating != 9 {
		t.Errorf("Rating = %d, want 9", e.Rating)
	}
	if e.Watched {
		t.Error("Watched = true, want false")
	}
}

func TestStateCommandsInHelp(t *testing.T) {
	cmd := newRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"rate", "unrate", "watched", "unwatch"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help output = %q, want it to contain %q", out.String(), want)
		}
	}
}
