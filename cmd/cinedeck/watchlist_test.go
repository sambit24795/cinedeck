package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

var (
	movieID = catalog.ID{Type: catalog.Movie, N: 27205}
	tvID    = catalog.ID{Type: catalog.TV, N: 1396}
)

// libraryDateClock returns 2026-10-01, then 2026-10-05, then 2026-10-10, one
// value per call.
func libraryDateClock() func() time.Time {
	times := []time.Time{
		time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 10, 10, 10, 0, 0, 0, time.UTC),
	}
	i := 0
	return func() time.Time {
		if i >= len(times) {
			return times[len(times)-1]
		}
		t := times[i]
		i++
		return t
	}
}

// seedLibrary stores Inception and Breaking Bad directly.
func seedLibrary(t *testing.T, lib *library.Library) {
	t.Helper()
	ctx := context.Background()
	items := []*catalog.Meta{
		{ID: movieID, Title: "Inception", Year: 2010, Rating: 8.4, Genres: []string{"Action", "Science Fiction"}},
		{ID: tvID, Title: "Breaking Bad", Year: 2008, Rating: 8.9, Genres: []string{"Drama", "Crime"}},
	}
	for _, m := range items {
		if err := lib.UpsertItem(ctx, m); err != nil {
			t.Fatalf("UpsertItem %s: %v", m.ID, err)
		}
	}
}

func TestWatchlistAdd(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watchlist", "add", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Added to watchlist: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "watchlist", "add", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if want := "Already on your watchlist: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestWatchlistAddFirst(t *testing.T) {
	cfg, paths := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watchlist", "add", "inception", "--first", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Added to watchlist: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if !containsPath(*paths, "/search/multi") {
		t.Errorf("paths = %v, want %q", *paths, "/search/multi")
	}
}

func TestWatchlistAddWatched(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "watched", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("watched Execute: %v", err)
	}
	out, err := runRootCmd(t, "watchlist", "add", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("watchlist add Execute: %v", err)
	}
	if want := "Already watched, not added: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	entries, err := openTestLibrary(t).Watchlist(context.Background())
	if err != nil {
		t.Fatalf("Watchlist: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("watchlist length = %d, want 0", len(entries))
	}
}

func TestWatchlistRM(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "watchlist", "add", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("watchlist add Execute: %v", err)
	}

	out, err := runRootCmd(t, "watchlist", "rm", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("watchlist rm Execute: %v", err)
	}
	if want := "Removed from watchlist: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "watchlist", "rm", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second watchlist rm Execute: %v", err)
	}
	if want := "Not on your watchlist: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "watchlist", "rm", "movie:99", "--config", cfg)
	if err != nil {
		t.Fatalf("unknown watchlist rm Execute: %v", err)
	}
	if want := "Not on your watchlist: movie:99\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestWatchlistLS(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.AddToWatchlist(ctx, movieID); err != nil {
		t.Fatalf("AddToWatchlist movie: %v", err)
	}
	if _, err := lib.AddToWatchlist(ctx, tvID); err != nil {
		t.Fatalf("AddToWatchlist tv: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out, err := runRootCmd(t, "watchlist", "ls", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3:\n%s", len(lines), out)
	}
	for _, want := range []string{"TYPE", "TITLE", "YEAR", "RATING", "ID", "ADDED"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header = %q, want it to contain %q", lines[0], want)
		}
	}
	for _, want := range []string{"Breaking Bad", "tv:1396", "8.9", "2026-10-05"} {
		if !strings.Contains(lines[1], want) {
			t.Errorf("first row = %q, want it to contain %q", lines[1], want)
		}
	}
	for _, want := range []string{"Inception", "movie:27205", "2026-10-01"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("second row = %q, want it to contain %q", lines[2], want)
		}
	}
}

func TestWatchlistLSEmpty(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watchlist", "ls", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Your watchlist is empty. Add titles with: cinedeck watchlist add <id|title>\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "watchlist", "ls", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("json Execute: %v", err)
	}
	var rows []libraryRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rows == nil {
		t.Error("rows = nil, want non-nil empty slice")
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}

func TestWatchlistLSJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.AddToWatchlist(ctx, movieID); err != nil {
		t.Fatalf("AddToWatchlist movie: %v", err)
	}
	if _, err := lib.AddToWatchlist(ctx, tvID); err != nil {
		t.Fatalf("AddToWatchlist tv: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out, err := runRootCmd(t, "watchlist", "ls", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var rows []libraryRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].ID != tvID {
		t.Errorf("rows[0].ID = %v, want %v", rows[0].ID, tvID)
	}
	if rows[0].At != "2026-10-05T10:00:00Z" {
		t.Errorf("rows[0].At = %q, want %q", rows[0].At, "2026-10-05T10:00:00Z")
	}
	if rows[0].Genres == nil {
		t.Error("rows[0].Genres = nil, want non-nil")
	}
}

func TestLibraryReadsOffline(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.AddToWatchlist(ctx, movieID); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if _, err := lib.MarkWatched(ctx, tvID); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// The server stays up and counts requests, so a command that did call
	// TMDB would be seen. Run once without a token and once with one.
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)

	for _, token := range []string{"", "tok"} {
		t.Setenv("CINEDECK_TMDB_TOKEN", token)
		out, err := runRootCmd(t, "watchlist", "ls", "--config", cfg)
		if err != nil {
			t.Fatalf("watchlist ls (token %q) Execute: %v", token, err)
		}
		if !strings.Contains(out, "Inception") {
			t.Errorf("watchlist ls output = %q, want the stored title", out)
		}
		out, err = runRootCmd(t, "history", "--config", cfg)
		if err != nil {
			t.Fatalf("history (token %q) Execute: %v", token, err)
		}
		if !strings.Contains(out, "Breaking Bad") {
			t.Errorf("history output = %q, want the stored title", out)
		}
	}
	if n := requests.Load(); n != 0 {
		t.Errorf("server saw %d requests, want 0", n)
	}
}

func TestBareWatchlist(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "watchlist", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"add", "rm", "ls"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestWatchlistHistoryInHelp(t *testing.T) {
	cmd := newRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"watchlist", "history"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help output = %q, want it to contain %q", out.String(), want)
		}
	}
}
