package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
)

// runRootCmdSplit is like runRootCmd but keeps stderr in its own buffer.
func runRootCmdSplit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), errBuf.String(), err
}

// newWhereServer serves the M2 fixtures and records the requested paths.
func newWhereServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	paths := &[]string{}
	routes := map[string]string{
		"/search/multi":                "search_multi.json",
		"/movie/27205/watch/providers": "watch_movie_27205.json",
		"/tv/1399/watch/providers":     "watch_tv_1399.json",
	}
	srv := fixtureServer(t, routes, paths)
	return srv, paths
}

// setupWhere points the environment at the fixture server and returns a
// nonexistent config path.
func setupWhere(t *testing.T) (*httptest.Server, *[]string, string) {
	t.Helper()
	t.Setenv("CINEDECK_REGION", "")
	srv, paths := newWhereServer(t)
	setCatalogEnv(t, srv)
	return srv, paths, filepath.Join(t.TempDir(), "none.toml")
}

const wantWhereIN = `Where to watch movie:27205 in IN

Stream
  JioHotstar
  Netflix
  Amazon Prime Video
Rent
  Apple TV
  Google Play Movies
Buy
  Apple TV

TMDB watch page: https://www.themoviedb.org/movie/27205-inception/watch?locale=IN

Where-to-watch data is provided by JustWatch.
`

const wantWhereGB = `No watch providers for movie:27205 in GB.

Where-to-watch data is provided by JustWatch.
`

func TestWhereMovieText(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "in", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != wantWhereIN {
		t.Errorf("stdout = %q, want %q", out, wantWhereIN)
	}
}

func TestWhereTVUS(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "tv:1399", "--region", "US", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Stream", "Max", "Free", "Tubi TV", "Rent", "Buy"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if strings.Contains(out, "With ads") {
		t.Errorf("output = %q, want it not to contain %q", out, "With ads")
	}
	if strings.Index(out, "Max") > strings.Index(out, "Amazon Prime Video") {
		t.Errorf("output = %q, want Max before Amazon Prime Video", out)
	}
}

func TestWhereTVIN(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "tv:1399", "--region", "IN", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"With ads", "MX Player"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if strings.Contains(out, "Rent") {
		t.Errorf("output = %q, want it not to contain %q", out, "Rent")
	}
}

func TestWhereMovieDE(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "DE", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Buy", "locale=DE"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if strings.Contains(out, "Stream") {
		t.Errorf("output = %q, want it not to contain %q", out, "Stream")
	}
}

func TestWhereMovieGBText(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "GB", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != wantWhereGB {
		t.Errorf("stdout = %q, want %q", out, wantWhereGB)
	}
}

func TestWhereMovieGBJSON(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, errOut, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "GB", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got struct {
		Available bool              `json:"available"`
		Region    string            `json:"region"`
		Stream    []json.RawMessage `json:"stream"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if got.Available {
		t.Errorf("available = %v, want false", got.Available)
	}
	if got.Region != "GB" {
		t.Errorf("region = %q, want %q", got.Region, "GB")
	}
	if got.Stream == nil || len(got.Stream) != 0 {
		t.Errorf("stream = %#v, want a non-nil empty array", got.Stream)
	}
	if want := justWatchNotice + "\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

func TestWhereMovieINJSON(t *testing.T) {
	_, _, cfg := setupWhere(t)

	out, errOut, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "IN", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got whereOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if got.ID != (catalog.ID{Type: catalog.Movie, N: 27205}) {
		t.Errorf("ID = %v, want %v", got.ID, catalog.ID{Type: catalog.Movie, N: 27205})
	}
	if !got.Available {
		t.Error("Available = false, want true")
	}
	if len(got.Stream) == 0 || got.Stream[0].Name != "JioHotstar" {
		t.Errorf("Stream = %v, want first provider JioHotstar", got.Stream)
	}
	if got.Attribution != justWatchNotice {
		t.Errorf("Attribution = %q, want %q", got.Attribution, justWatchNotice)
	}
	if want := justWatchNotice + "\n"; errOut != want {
		t.Errorf("stderr = %q, want %q", errOut, want)
	}
}

func TestWhereRegionFromConfig(t *testing.T) {
	_, _, _ = setupWhere(t)
	cfg := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Save(cfg, config.Config{Region: "US"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, _, err := runRootCmdSplit(t, "where", "tv:1399", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Where to watch tv:1399 in US\n") {
		t.Errorf("output = %q, want it to start with %q", out, "Where to watch tv:1399 in US\n")
	}
}

func TestWhereFlagBeatsConfig(t *testing.T) {
	_, _, _ = setupWhere(t)
	cfg := filepath.Join(t.TempDir(), "config.toml")
	if err := config.Save(cfg, config.Config{Region: "US"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	out, _, err := runRootCmdSplit(t, "where", "tv:1399", "--region", "IN", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Where to watch tv:1399 in IN\n") {
		t.Errorf("output = %q, want it to start with %q", out, "Where to watch tv:1399 in IN\n")
	}
}

func TestWhereTitleFirst(t *testing.T) {
	_, paths, cfg := setupWhere(t)

	out, _, err := runRootCmdSplit(t, "where", "inception", "--first", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Where to watch movie:27205 in") {
		t.Errorf("output = %q, want it to contain %q", out, "Where to watch movie:27205 in")
	}
	found := false
	for _, p := range *paths {
		if p == "/movie/27205/watch/providers" {
			found = true
		}
	}
	if !found {
		t.Errorf("paths = %v, want /movie/27205/watch/providers", *paths)
	}
}

func TestWhereTitlePicker(t *testing.T) {
	_, _, cfg := setupWhere(t)

	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader("2\n"))
	cmd.SetArgs([]string{"where", "inception", "--region", "US", "--config", cfg})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "Where to watch tv:1399 in US") {
		t.Errorf("output = %q, want it to contain %q", out.String(), "Where to watch tv:1399 in US")
	}
	if !strings.Contains(out.String(), "Pick a number: ") {
		t.Errorf("output = %q, want the picker prompt", out.String())
	}
}

func TestWhereTitlePickerJSON(t *testing.T) {
	_, _, cfg := setupWhere(t)

	cmd := newRootCmd()
	var out, errBuf bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader("2\n"))
	cmd.SetArgs([]string{"where", "inception", "--region", "US", "--json", "--config", cfg})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got whereOutput
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("stdout %q is not JSON: %v", out.String(), err)
	}
	if want := "\n" + justWatchNotice + "\n"; !strings.HasSuffix(errBuf.String(), want) {
		t.Errorf("stderr = %q, want it to end with %q", errBuf.String(), want)
	}
}

func TestWhereInvalidRegion(t *testing.T) {
	_, _, cfg := setupWhere(t)

	_, _, err := runRootCmdSplit(t, "where", "movie:27205", "--region", "XYZ", "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "invalid region") {
		t.Fatalf("err = %v, want it to contain %q", err, "invalid region")
	}
}

func TestWhereNoToken(t *testing.T) {
	_, _, cfg := setupWhere(t)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	_, _, err := runRootCmdSplit(t, "where", "movie:27205", "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Fatalf("err = %v, want it to contain %q", err, "config init")
	}
}
