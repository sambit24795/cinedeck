package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
)

// setupLibraryCmd points a test at a fresh data directory and the fixture server.
func setupLibraryCmd(t *testing.T) (string, *[]string) {
	t.Helper()
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	paths := &[]string{}
	srv := fixtureServer(t, libraryRoutes(), paths)
	setCatalogEnv(t, srv)
	return filepath.Join(t.TempDir(), "none.toml"), paths
}

func containsPath(paths []string, want string) bool {
	for _, p := range paths {
		if p == want {
			return true
		}
	}
	return false
}

func TestLikeCommand(t *testing.T) {
	cfg, paths := setupLibraryCmd(t)

	out, err := runRootCmd(t, "like", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Liked Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	e, err := openTestLibrary(t).Entry(context.Background(), catalog.ID{Type: catalog.Movie, N: 27205})
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !e.Liked {
		t.Error("Liked = false, want true")
	}
	if e.Meta.Title != "Inception" {
		t.Errorf("Title = %q, want %q", e.Meta.Title, "Inception")
	}
	if len(e.Meta.Keywords) != 14 {
		t.Errorf("len(Keywords) = %d, want 14", len(e.Meta.Keywords))
	}
	if !containsPath(*paths, "/movie/27205") {
		t.Errorf("paths = %v, want %q", *paths, "/movie/27205")
	}
}

func TestLikeCommandAgain(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	out, err := runRootCmd(t, "like", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if want := "Already liked Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestLikeCommandFirst(t *testing.T) {
	cfg, paths := setupLibraryCmd(t)

	out, err := runRootCmd(t, "like", "inception", "--first", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "Liked Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
	if !containsPath(*paths, "/search/multi") {
		t.Errorf("paths = %v, want %q", *paths, "/search/multi")
	}
}

func TestLikeJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "like", "movie:27205", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got actionOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.ID != (catalog.ID{Type: catalog.Movie, N: 27205}) {
		t.Errorf("ID = %v, want movie:27205", got.ID)
	}
	if got.Title != "Inception" {
		t.Errorf("Title = %q, want %q", got.Title, "Inception")
	}
	if got.Year != 2010 {
		t.Errorf("Year = %d, want 2010", got.Year)
	}
	if got.Action != "like" {
		t.Errorf("Action = %q, want %q", got.Action, "like")
	}
	if !got.Changed {
		t.Error("Changed = false, want true")
	}

	out, err = runRootCmd(t, "like", "movie:27205", "--json", "--config", cfg)
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

func TestUnlikeCommand(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like Execute: %v", err)
	}

	out, err := runRootCmd(t, "unlike", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("unlike Execute: %v", err)
	}
	if want := "Unliked Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "unlike", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("second unlike Execute: %v", err)
	}
	if want := "Not in your likes: Inception (2010) [movie:27205]\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "unlike", "movie:99", "--config", cfg)
	if err != nil {
		t.Fatalf("unknown unlike Execute: %v", err)
	}
	if want := "Not in your likes: movie:99\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}
}

func TestUnlikeJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like Execute: %v", err)
	}
	out, err := runRootCmd(t, "unlike", "movie:27205", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("unlike Execute: %v", err)
	}
	var got actionOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Action != "unlike" {
		t.Errorf("Action = %q, want %q", got.Action, "unlike")
	}
}

func TestTypedIDNeedsNoToken(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	out, err := runRootCmd(t, "unlike", "movie:99", "--config", cfg)
	if err != nil {
		t.Fatalf("unlike typed Execute: %v", err)
	}
	if want := "Not in your likes: movie:99\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	_, err = runRootCmd(t, "unlike", "inception", "--config", cfg)
	if err == nil {
		t.Fatal("unlike title Execute = nil, want error")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("error = %q, want it to contain %q", err, "config init")
	}
}

func TestLikeNoTokenCreatesNothing(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	_, err := runRootCmd(t, "like", "movie:27205", "--config", cfg)
	if err == nil {
		t.Fatal("Execute = nil, want error")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("error = %q, want it to contain %q", err, "config init")
	}

	path, err := config.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("Stat = %v, want os.ErrNotExist", statErr)
	}
}

func TestLikeNotFoundCreatesNothing(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	_, err := runRootCmd(t, "like", "movie:1", "--config", cfg)
	if err == nil {
		t.Fatal("Execute = nil, want error")
	}
	if !strings.Contains(err.Error(), "tmdb: 404") {
		t.Errorf("error = %q, want it to contain %q", err, "tmdb: 404")
	}

	path, err := config.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("Stat = %v, want os.ErrNotExist", statErr)
	}
}

func TestLikeNoArgs(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	if _, err := runRootCmd(t, "like", "--config", cfg); err == nil {
		t.Fatal("Execute = nil, want error")
	}
}

func TestLibraryCommandsInHelp(t *testing.T) {
	cmd := newRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"like", "unlike"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("help output = %q, want it to contain %q", out.String(), want)
		}
	}
}
