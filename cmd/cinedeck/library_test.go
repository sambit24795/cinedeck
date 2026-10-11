package main

import (
	"os"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/library"
)

// libraryRoutes serves the M3 detail fixtures and search.
func libraryRoutes() map[string]string {
	return map[string]string{
		"/movie/27205":  "meta_movie_27205.json",
		"/tv/1396":      "meta_tv_1396.json",
		"/search/multi": "search_multi.json",
	}
}

// openTestLibrary opens the library at config.LibraryPath() (CINEDECK_DATA_DIR is a temp dir) and closes it with t.Cleanup.
func openTestLibrary(t *testing.T) *library.Library {
	t.Helper()
	path, err := config.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	l, err := library.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return l
}

func TestDataDirIsSandboxed(t *testing.T) {
	dir := os.Getenv("CINEDECK_DATA_DIR")
	if dir == "" {
		t.Fatal("CINEDECK_DATA_DIR is empty")
	}
	if strings.Contains(dir, ".local/share") {
		t.Errorf("CINEDECK_DATA_DIR = %q, want it not to contain %q", dir, ".local/share")
	}
}

func TestItemLabel(t *testing.T) {
	m := &catalog.Meta{
		ID:    catalog.ID{Type: catalog.Movie, N: 27205},
		Title: "Inception",
		Year:  2010,
	}
	if got, want := itemLabel(m), "Inception (2010) [movie:27205]"; got != want {
		t.Errorf("itemLabel = %q, want %q", got, want)
	}

	m = &catalog.Meta{
		ID:    catalog.ID{Type: catalog.Movie, N: 1},
		Title: "Movie",
	}
	if got, want := itemLabel(m), "Movie [movie:1]"; got != want {
		t.Errorf("itemLabel = %q, want %q", got, want)
	}
}
