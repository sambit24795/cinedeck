package main

import (
	"net/http"
	"net/http/httptest"
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

func TestTrendingCommandTV(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Write([]byte(`{"results":[{"id":1399,"name":"Game of Thrones","first_air_date":"2011-04-17","vote_average":8.4}]}`))
	}))
	t.Cleanup(srv.Close)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "trending", "--type", "tv", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if gotPath != "/trending/tv/week" {
		t.Errorf("path = %q, want %q", gotPath, "/trending/tv/week")
	}
	for _, want := range []string{"Game of Thrones", "tv:1399"} {
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
