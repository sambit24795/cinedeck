package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/config"
)

// TestMain makes the tests hermetic: the caller may have the CINEDECK_*
// overrides exported, and config.Load would apply them to every test. It also
// points the data directory at a throw-away directory so library commands can
// never touch the real one.
func TestMain(m *testing.M) {
	for _, key := range []string{"CINEDECK_TMDB_TOKEN", "CINEDECK_TMDB_URL", "CINEDECK_REGION", "CINEDECK_DOWNLOAD_DIR"} {
		os.Unsetenv(key)
	}
	dir, err := os.MkdirTemp("", "cinedeck-test-data-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Setenv("CINEDECK_DATA_DIR", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// authServer starts a server that answers GET /authentication with
// testdata/authentication_ok.json, points CINEDECK_TMDB_URL at it, and
// returns the server and a pointer to the number of requests it received.
func authServer(t *testing.T) (*httptest.Server, *int) {
	t.Helper()
	body, err := os.ReadFile("../../testdata/authentication_ok.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	count := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)
	return srv, &count
}

// runConfigInit runs "config init" with the given stdin and config path and
// returns stdout, stderr and the error.
func runConfigInit(t *testing.T, path, stdin string) (string, string, error) {
	t.Helper()
	cmd := newRootCmd()
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	err := cmd.Execute()
	return out.String(), errOut.String(), err
}

func TestConfigPath(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"config", "path", "--config", "/x/y.toml"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got := out.String(); got != "/x/y.toml\n" {
		t.Errorf("output = %q, want %q", got, "/x/y.toml\n")
	}
}

func TestConfigInit(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("abc\nUS\n\n"))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "abc" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "abc")
	}
	if cfg.Region != "US" {
		t.Errorf("Region = %q, want %q", cfg.Region, "US")
	}
	if want := config.Default().DownloadDir; cfg.DownloadDir != want {
		t.Errorf("DownloadDir = %q, want %q", cfg.DownloadDir, want)
	}
}

func TestConfigInitRejectsShortAPIKey(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	first := newRootCmd()
	first.SetOut(&bytes.Buffer{})
	first.SetIn(strings.NewReader("abc\nUS\n\n"))
	first.SetArgs([]string{"config", "init", "--config", path})
	if err := first.Execute(); err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	const key = "0123456789abcdef0123456789abcdef"
	second := newRootCmd()
	var secondOut bytes.Buffer
	second.SetOut(&secondOut)
	second.SetIn(strings.NewReader(key + "\n"))
	second.SetArgs([]string{"config", "init", "--config", path})
	err := second.Execute()
	if err == nil {
		t.Fatal("Execute = nil, want error")
	}
	want := `that looks like the TMDB "API Key"; paste the longer "API Read Access Token" instead`
	if err.Error() != want {
		t.Errorf("err = %q, want %q", err, want)
	}
	if strings.Contains(secondOut.String(), key) {
		t.Errorf("output echoed the pasted value: %q", secondOut.String())
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "abc" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "abc")
	}
}

func TestConfigInitAcceptsAccessToken(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")
	const token = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.signature"

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader(token + "\n\n\n"))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != token {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, token)
	}
}

func TestConfigInitKeepsExistingValues(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	first := newRootCmd()
	first.SetOut(&bytes.Buffer{})
	first.SetIn(strings.NewReader("abc\nUS\n\n"))
	first.SetArgs([]string{"config", "init", "--config", path})
	if err := first.Execute(); err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	second := newRootCmd()
	var secondOut bytes.Buffer
	second.SetOut(&secondOut)
	second.SetIn(strings.NewReader("\n\n\n"))
	second.SetArgs([]string{"config", "init", "--config", path})
	if err := second.Execute(); err != nil {
		t.Fatalf("second Execute: %v", err)
	}

	if strings.Contains(secondOut.String(), "abc") {
		t.Errorf("prompt echoed the stored token: %q", secondOut.String())
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "abc" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "abc")
	}
	if cfg.Region != "US" {
		t.Errorf("Region = %q, want %q", cfg.Region, "US")
	}
}

func TestConfigInitPrintsTokenInstructions(t *testing.T) {
	authServer(t)
	out, _, err := runConfigInit(t, filepath.Join(t.TempDir(), "c.toml"), "abc\nUS\n\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{
		"https://www.themoviedb.org/settings/api",
		"cinedeck (personal use)",
		"https://github.com/sambit24795/cinedeck",
		"Desktop Application",
		"API Read Access Token",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if i, j := strings.Index(out, "API Read Access Token"), strings.Index(out, "TMDB token: "); i < 0 || j < 0 || i >= j {
		t.Errorf("output = %q, want API Read Access Token before the token prompt", out)
	}
}

func TestConfigInitChecksValidToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	const token = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJ4In0.signature"
	body, err := os.ReadFile("../../testdata/authentication_ok.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var paths []string
	var auths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		auths = append(auths, r.Header.Get("Authorization"))
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)

	out, _, err := runConfigInit(t, path, token+"\nUS\n\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(paths) != 1 || paths[0] != "/authentication" {
		t.Errorf("paths = %v, want [/authentication]", paths)
	}
	if len(auths) != 1 || auths[0] != "Bearer "+token {
		t.Errorf("auths = %v, want one Bearer token", auths)
	}
	if !strings.Contains(out, "Checking token with TMDB... ok") {
		t.Errorf("output = %q, want the ok line", out)
	}
	if strings.Contains(out, token) {
		t.Errorf("output echoed the token")
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != token {
		t.Errorf("TMDBToken = %q, want the typed token", cfg.TMDBToken)
	}
}

func TestConfigInitRejectsToken(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	body, err := os.ReadFile("../../testdata/error_401.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)

	out, _, err := runConfigInit(t, path, "badtoken\nUS\n\n")
	if err == nil || !strings.Contains(err.Error(), "TMDB rejected that token") {
		t.Fatalf("err = %v, want it to contain TMDB rejected that token", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("stat err = %v, want the config file not to exist", statErr)
	}
	if strings.Contains(out, "badtoken") {
		t.Errorf("output echoed the token")
	}
}

func TestConfigInitRejectedTokenKeepsOld(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	authServer(t)
	if _, _, err := runConfigInit(t, path, "old\nUS\n\n"); err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	body, err := os.ReadFile("../../testdata/error_401.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	reject := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(body)
	}))
	t.Cleanup(reject.Close)
	t.Setenv("CINEDECK_TMDB_URL", reject.URL)

	_, _, err = runConfigInit(t, path, "new\n\n\n")
	if err == nil || !strings.Contains(err.Error(), "TMDB rejected") {
		t.Fatalf("err = %v, want it to contain TMDB rejected", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "old" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "old")
	}
	if cfg.Region != "US" {
		t.Errorf("Region = %q, want %q", cfg.Region, "US")
	}
}

func TestConfigInitSavesWhenCheckFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)

	out, errOut, err := runConfigInit(t, path, "abc\nUS\n\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "abc" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "abc")
	}
	if !strings.Contains(errOut, "warning: could not check the token with TMDB") {
		t.Errorf("stderr = %q, want the warning", errOut)
	}
	if !strings.Contains(errOut, "saving it anyway") {
		t.Errorf("stderr = %q, want %q", errOut, "saving it anyway")
	}
	if !strings.Contains(out, "could not check") {
		t.Errorf("stdout = %q, want %q", out, "could not check")
	}
}

func TestConfigInitSavesOnServerError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CINEDECK_TMDB_URL", srv.URL)

	_, errOut, err := runConfigInit(t, path, "abc\nUS\n\n")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TMDBToken != "abc" {
		t.Errorf("TMDBToken = %q, want %q", cfg.TMDBToken, "abc")
	}
	if !strings.Contains(errOut, "warning: could not check the token with TMDB") || !strings.Contains(errOut, "saving it anyway") {
		t.Errorf("stderr = %q, want the warning", errOut)
	}
}

func TestConfigInitNoRequestWhenTokenKept(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	_, count := authServer(t)
	if _, _, err := runConfigInit(t, path, "abc\nUS\n\n"); err != nil {
		t.Fatalf("first Execute: %v", err)
	}
	afterFirst := *count
	if _, _, err := runConfigInit(t, path, "\n\n\n"); err != nil {
		t.Fatalf("second Execute: %v", err)
	}
	if *count != afterFirst {
		t.Errorf("requests = %d, want unchanged from %d", *count, afterFirst)
	}
}

func TestConfigInitNoRequestForShortKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.toml")
	_, count := authServer(t)
	const key = "0123456789abcdef0123456789abcdef"
	_, _, err := runConfigInit(t, path, key+"\n")
	want := `that looks like the TMDB "API Key"; paste the longer "API Read Access Token" instead`
	if err == nil || err.Error() != want {
		t.Fatalf("err = %v, want %q", err, want)
	}
	if *count != 0 {
		t.Errorf("requests = %d, want 0", *count)
	}
}

func TestConfigInitSavesIndexerProviders(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader("abc\nUS\n\napibay,yts\n"))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IndexerProviders != "apibay,yts" {
		t.Errorf("IndexerProviders = %q, want apibay,yts", cfg.IndexerProviders)
	}
}

func TestConfigInitEmptyIndexerProviders(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	cmd := newRootCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetIn(strings.NewReader("abc\nUS\n\n\n"))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.IndexerProviders != "" {
		t.Errorf("IndexerProviders = %q, want empty", cfg.IndexerProviders)
	}
}

func TestConfigInitDoesNotEchoStoredTokenInIndexerPrompt(t *testing.T) {
	authServer(t)
	path := filepath.Join(t.TempDir(), "c.toml")

	if err := config.Save(path, config.Config{
		TMDBToken:        "secret-token",
		TMDBURL:          "https://api.themoviedb.org/3",
		Region:           "IN",
		IndexerProviders: "apibay",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetIn(strings.NewReader("\n\n\n\n"))
	cmd.SetArgs([]string{"config", "init", "--config", path})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out.String(), "secret-token") {
		t.Error("output echoed the stored token")
	}
}
