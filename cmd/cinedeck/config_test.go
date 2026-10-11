package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/config"
)

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
