package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func infoServer(t *testing.T) *httptest.Server {
	t.Helper()
	return fixtureServer(t, map[string]string{
		"/search/multi": "search_multi.json",
		"/movie/27205":  "movie_27205.json",
		"/tv/1399":      "tv_1399.json",
	})
}

func TestInfoMovie(t *testing.T) {
	setCatalogEnv(t, infoServer(t))

	out, err := runRootCmd(t, "info", "movie:27205", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Inception (2010)", "148", "Christopher Nolan"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestInfoTV(t *testing.T) {
	setCatalogEnv(t, infoServer(t))

	out, err := runRootCmd(t, "info", "tv:1399", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Game of Thrones", "8", "David Benioff"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
}

func TestInfoFirst(t *testing.T) {
	setCatalogEnv(t, infoServer(t))

	out, err := runRootCmd(t, "info", "inception", "--first", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Inception (2010)") {
		t.Errorf("output = %q, want it to contain %q", out, "Inception (2010)")
	}
}

func TestInfoInteractive(t *testing.T) {
	setCatalogEnv(t, infoServer(t))

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("2\n"))
	cmd.SetArgs([]string{"info", "inception", "--config", filepath.Join(t.TempDir(), "none.toml")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "Game of Thrones") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "Game of Thrones")
	}
}

func TestInfoJSON(t *testing.T) {
	setCatalogEnv(t, infoServer(t))

	out, err := runRootCmd(t, "info", "movie:27205", "--json", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if got.ID != "movie:27205" {
		t.Errorf("id = %q, want %q", got.ID, "movie:27205")
	}
}

func TestInfoSkipsEmptyFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"title":"Obscure","release_date":"","vote_average":0,"overview":"Plot."}`))
	}))
	t.Cleanup(srv.Close)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "info", "movie:1", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Obscure\n") {
		t.Errorf("output = %q, want first line %q", out, "Obscure")
	}
	for _, bad := range []string{"(0)", "Rating"} {
		if strings.Contains(out, bad) {
			t.Errorf("output = %q, want it not to contain %q", out, bad)
		}
	}
}
