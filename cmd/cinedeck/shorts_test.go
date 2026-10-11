package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShortsCommand(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/genre/movie/list": "genre_movie_list.json",
		"/discover/movie":   "discover_shorts.json",
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
	var gotPaths []string
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPaths = append(gotPaths, r.URL.Path)
		gotQuery = r.URL.Query()
		body, err := os.ReadFile("../../testdata/discover_shorts.json")
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
	if got := gotQuery.Get("primary_release_year"); got != "2008" {
		t.Errorf("primary_release_year = %q, want %q", got, "2008")
	}
	if len(gotPaths) != 1 || gotPaths[0] != "/discover/movie" {
		t.Errorf("paths = %v, want [/discover/movie] only (no genre lookup)", gotPaths)
	}
	if !strings.Contains(out, "Presto") {
		t.Errorf("output = %q, want it to contain %q", out, "Presto")
	}
}

func TestShortsCommandJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/discover/movie": "discover_shorts.json"})
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
