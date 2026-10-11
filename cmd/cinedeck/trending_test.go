package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestTrendingCommand(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/trending/movie/week": "trending_movie.json"})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "trending", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Inception", "Presto"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestTrendingCommandInvalidType(t *testing.T) {
	out, err := runRootCmd(t, "trending", "--type", "person", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil {
		t.Fatalf("Execute = nil, output %q, want error", out)
	}
}
