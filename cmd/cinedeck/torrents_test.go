package main

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// fakeTorrentProvider is a provider stub for command tests.
type fakeTorrentProvider struct {
	name     string
	releases []indexer.Release
	err      error
}

func (f *fakeTorrentProvider) Name() string { return f.name }

func (f *fakeTorrentProvider) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.releases, nil
}

// useFakeProviders replaces the provider registry for one test.
func useFakeProviders(t *testing.T, provs ...indexer.Provider) {
	t.Helper()
	old := newTorrentsRegistryFunc
	newTorrentsRegistryFunc = func() *indexer.Registry {
		r := indexer.NewRegistry()
		for _, p := range provs {
			r.Register(p)
		}
		return r
	}
	t.Cleanup(func() { newTorrentsRegistryFunc = old })
}

// runTorrentsCmd runs a torrents command with a throwaway config and token set.
func runTorrentsCmd(t *testing.T, wantErr bool, args ...string) (stdout, stderr string) {
	t.Helper()
	t.Setenv("CINEDECK_TMDB_TOKEN", "tok")
	t.Setenv("CINEDECK_TMDB_URL", "http://127.0.0.1:1")
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())
	t.Setenv("CINEDECK_INDEXER_PROVIDERS", "fake")

	cmd := newRootCmd()
	var out, errBuf strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs(append([]string{"torrents", "--config", filepath.Join(t.TempDir(), "none.toml")}, args...))
	err := cmd.Execute()
	if wantErr && err == nil {
		t.Fatalf("expected an error, got nil (stdout=%s stderr=%s)", out.String(), errBuf.String())
	}
	if !wantErr && err != nil {
		t.Fatalf("Execute: %v (stderr=%s)", err, errBuf.String())
	}
	return out.String(), errBuf.String()
}

func TestTorrentsSearch(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{
		name: "fake",
		releases: []indexer.Release{
			{Provider: "fake", Title: "Inception 2010 1080p", InfoHash: "aabbccddeeff00112233445566778899aabbccdd", Seeders: 450, Leechers: 120, Size: 2147483648},
			{Provider: "fake", Title: "Inception 2010 720p", InfoHash: "bbccddeeff00112233445566778899aabbccddee", Seeders: 88, Leechers: 30, Size: 1073741824},
		},
	})

	out, _ := runTorrentsCmd(t, false, "search", "inception")
	for _, want := range []string{"Inception", "fake", "450", "aabbccddeeff00112233445566778899aabbccdd", "2.1 GB"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTorrentsSearchJSON(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{
		name: "fake",
		releases: []indexer.Release{
			{Provider: "fake", Title: "Inception", InfoHash: "aabbccddeeff00112233445566778899aabbccdd", Seeders: 10},
		},
	})

	out, _ := runTorrentsCmd(t, false, "search", "inception", "--json")
	var got []indexer.Release
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].InfoHash == "" {
		t.Fatalf("got %+v", got)
	}
}

func TestTorrentsSearchNoResults(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{name: "fake", releases: []indexer.Release{}})
	out, _ := runTorrentsCmd(t, false, "search", "zzz")
	if !strings.Contains(out, "No results.") {
		t.Errorf("output = %q, want the no-results message", out)
	}
}

func TestTorrentsSearchMinSeeders(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{
		name: "fake",
		releases: []indexer.Release{
			{Provider: "fake", Title: "Low", InfoHash: "h1", Seeders: 1},
		},
	})
	out, _ := runTorrentsCmd(t, false, "search", "x", "--min-seeders", "9999")
	if !strings.Contains(out, "No results.") {
		t.Errorf("output = %q, want the no-results message", out)
	}
}

func TestTorrentsSearchInvalidType(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{name: "fake"})
	_, _ = runTorrentsCmd(t, true, "search", "x", "--type", "shows")
}

func TestTorrentsSearchUnknownProvider(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{name: "fake"})
	_, _ = runTorrentsCmd(t, true, "search", "x", "--provider", "nope")
}

func TestTorrentsSearchProviderFailureStillSucceeds(t *testing.T) {
	useFakeProviders(t,
		&fakeTorrentProvider{name: "bad", err: errors.New("timeout")},
		&fakeTorrentProvider{name: "good", releases: []indexer.Release{
			{Provider: "good", Title: "Inception", InfoHash: "h1", Seeders: 5},
		}},
	)
	t.Setenv("CINEDECK_INDEXER_PROVIDERS", "bad,good")
	t.Setenv("CINEDECK_TMDB_TOKEN", "tok")
	t.Setenv("CINEDECK_TMDB_URL", "http://127.0.0.1:1")
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())

	cmd := newRootCmd()
	var out, errBuf strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&errBuf)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"torrents", "search", "inception", "--config", filepath.Join(t.TempDir(), "none.toml")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "Inception") {
		t.Errorf("good provider results missing:\n%s", out.String())
	}
	if !strings.Contains(errBuf.String(), "bad:") {
		t.Errorf("provider failure not reported on stderr: %q", errBuf.String())
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		in   uint64
		want string
	}{
		{0, "-"},
		{500, "500 B"},
		{2147483648, "2.1 GB"},
		{1073741824, "1.1 GB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.in); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
