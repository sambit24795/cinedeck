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
	second.SetOut(&bytes.Buffer{})
	second.SetIn(strings.NewReader("\n\n\n"))
	second.SetArgs([]string{"config", "init", "--config", path})
	if err := second.Execute(); err != nil {
		t.Fatalf("second Execute: %v", err)
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
