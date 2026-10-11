package main

import (
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
