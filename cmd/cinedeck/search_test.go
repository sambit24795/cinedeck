package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// fixtureServer starts a server that serves the named testdata files for the
// given request paths. When a recorder is passed, every requested path is
// appended to it.
func fixtureServer(t *testing.T, routes map[string]string, paths ...*[]string) *httptest.Server {
	t.Helper()
	var recorder *[]string
	if len(paths) > 0 {
		recorder = paths[0]
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if recorder != nil {
			*recorder = append(*recorder, r.URL.Path)
		}
		name, ok := routes[r.URL.Path]
		if !ok {
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
	return srv
}

// setCatalogEnv points the config environment at a test server.
func setCatalogEnv(t *testing.T, srv *httptest.Server) {
	t.Helper()
	t.Setenv("CINEDECK_TMDB_TOKEN", "tok")
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())
}

// runRootCmd executes the root command with args and returns its output.
func runRootCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSearchCommand(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/search/multi": "search_multi.json"})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "search", "inception", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Inception", "2010", "8.4", "movie:27205"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if strings.Contains(out, "Christopher Nolan") {
		t.Errorf("output = %q, want it not to contain %q", out, "Christopher Nolan")
	}
}

func TestSearchCommandJSON(t *testing.T) {
	srv := fixtureServer(t, map[string]string{"/search/multi": "search_multi.json"})
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "search", "inception", "--json", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].ID != "movie:27205" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "movie:27205")
	}
}

func TestSearchCommandNoToken(t *testing.T) {
	t.Setenv("CINEDECK_TMDB_TOKEN", "")
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())

	out, err := runRootCmd(t, "search", "inception", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil {
		t.Fatalf("Execute = nil, output %q, want error", out)
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("err = %v, want it to contain %q", err, "config init")
	}
}

func TestSearchNoResults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"results":[]}`))
	}))
	t.Cleanup(srv.Close)
	setCatalogEnv(t, srv)
	cfg := filepath.Join(t.TempDir(), "none.toml")

	out, err := runRootCmd(t, "search", "zzz", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.TrimSpace(out) != "No results." {
		t.Errorf("text output = %q, want %q", out, "No results.")
	}

	out, err = runRootCmd(t, "search", "zzz", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("--json output %q is not JSON: %v", out, err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("--json output = %q, want an empty array", out)
	}
}

func TestResultRowsUnratedShowsDash(t *testing.T) {
	rows := resultRows([]catalog.Result{
		{ID: catalog.ID{Type: catalog.Movie, N: 1}, Title: "Unrated", Year: 2020},
		{ID: catalog.ID{Type: catalog.Movie, N: 2}, Title: "Rated", Year: 2021, Rating: 7.4},
	})
	if got := rows[0][3]; got != "-" {
		t.Errorf("unrated rating = %q, want %q", got, "-")
	}
	if got := rows[1][3]; got != "7.4" {
		t.Errorf("rated rating = %q, want %q", got, "7.4")
	}
}
