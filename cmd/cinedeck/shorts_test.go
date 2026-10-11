package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestShortsCommand(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/genre/movie/list": "genre_movie_list.json",
		"/discover/movie":   "discover_shorts.json",
		"/movie/13042":      "movie_13042.json",
		"/movie/399106":     "movie_399106.json",
	})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "shorts", "--genre", "animation", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Presto", "movie:13042"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestShortsCommandYearWithoutGenre(t *testing.T) {
	var mu sync.Mutex
	var gotPaths []string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotPaths = append(gotPaths, r.URL.Path)
		if r.URL.Path == "/discover/movie" {
			gotQuery = r.URL.Query()
		}
		mu.Unlock()

		var name string
		switch r.URL.Path {
		case "/discover/movie":
			name = "discover_shorts.json"
		case "/movie/13042":
			name = "movie_13042.json"
		case "/movie/399106":
			name = "movie_399106.json"
		default:
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("../../testdata/" + name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "shorts", "--year", "2008", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	mu.Lock()
	paths := append([]string(nil), gotPaths...)
	query := gotQuery
	mu.Unlock()

	if got := query.Get("primary_release_year"); got != "2008" {
		t.Errorf("primary_release_year = %q, want %q", got, "2008")
	}
	sawDiscover := false
	for _, p := range paths {
		if p == "/genre/movie/list" {
			t.Errorf("paths = %v, want no genre lookup", paths)
		}
		if p == "/discover/movie" {
			sawDiscover = true
		}
	}
	if !sawDiscover {
		t.Errorf("paths = %v, want it to contain /discover/movie", paths)
	}
	if !strings.Contains(out, "Presto") {
		t.Errorf("output = %q, want it to contain %q", out, "Presto")
	}
}

func TestShortsCommandJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/discover/movie": "discover_shorts.json",
		"/movie/13042":    "movie_13042.json",
		"/movie/399106":   "movie_399106.json",
	})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "shorts", "--json", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != "movie:13042" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "movie:13042")
	}
}

func TestShortsCommandDropsFeatureFilms(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/discover/movie": "discover_shorts_live.json",
		"/movie/35":       "movie_35.json",
		"/movie/1076364":  "movie_1076364.json",
		"/movie/14813":    "movie_14813.json",
		"/movie/421892":   "movie_421892.json",
	})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "shorts", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Carl's Date", "Mickey's Christmas Carol"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	for _, bad := range []string{"Simpsons", "Shrek 5"} {
		if strings.Contains(out, bad) {
			t.Errorf("output = %q, want it not to contain %q", out, bad)
		}
	}
}
