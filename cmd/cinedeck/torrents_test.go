package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
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

func TestTorrentsInfoHash(t *testing.T) {
	hash := "aabbccddeeff00112233445566778899aabbccdd"
	useFakeProviders(t, &fakeTorrentProvider{
		name: "fake",
		releases: []indexer.Release{
			{Provider: "fake", Title: "Inception", InfoHash: hash, Seeders: 42, Leechers: 7},
		},
	})
	out, _ := runTorrentsCmd(t, false, "info", hash)
	for _, want := range []string{hash, "Inception", "fake", "42"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTorrentsInfoTorrentFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.torrent")
	writeTestTorrent(t, path, "Sample Movie 1999", 2048)

	useFakeProviders(t, &fakeTorrentProvider{name: "fake"})
	out, _ := runTorrentsCmd(t, false, "info", path)
	for _, want := range []string{"Sample Movie 1999", "2.0 kB", "Files:"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
}

func TestTorrentsInfoBadArg(t *testing.T) {
	useFakeProviders(t, &fakeTorrentProvider{name: "fake"})
	_, _ = runTorrentsCmd(t, true, "info", "notahash")
}

func TestTorrentsInfoJSON(t *testing.T) {
	hash := "aabbccddeeff00112233445566778899aabbccdd"
	useFakeProviders(t, &fakeTorrentProvider{
		name:     "fake",
		releases: []indexer.Release{{Provider: "fake", Title: "X", InfoHash: hash, Seeders: 1}},
	})
	out, _ := runTorrentsCmd(t, false, "info", hash, "--json")
	var got infoOutput
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not valid JSON: %v\n%s", err, out)
	}
	if got.InfoHash != hash {
		t.Errorf("InfoHash = %q, want %q", got.InfoHash, hash)
	}
	if len(got.Providers) != 1 || got.Providers[0].Provider != "fake" {
		t.Errorf("Providers = %+v", got.Providers)
	}
}

func TestTorrentsProviders(t *testing.T) {
	useFakeProviders(t,
		&fakeTorrentProvider{name: "alpha"},
		&fakeTorrentProvider{name: "beta"},
	)
	t.Setenv("CINEDECK_INDEXER_PROVIDERS", "beta")
	t.Setenv("CINEDECK_TMDB_TOKEN", "tok")
	t.Setenv("CINEDECK_TMDB_URL", "http://127.0.0.1:1")
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())

	cmd := newRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"torrents", "providers", "--config", filepath.Join(t.TempDir(), "none.toml")})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "beta") {
		t.Errorf("providers output missing a name:\n%s", got)
	}
	if !strings.Contains(got, "true") {
		t.Errorf("expected an enabled provider:\n%s", got)
	}
}

// writeTestTorrent writes a minimal single-file .torrent to path.
func writeTestTorrent(t *testing.T, path, name string, length int64) {
	t.Helper()
	info := metainfo.Info{
		Name:        name,
		Length:      length,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	}
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		t.Fatalf("bencode.Marshal(info): %v", err)
	}
	mi := metainfo.MetaInfo{InfoBytes: infoBytes}
	data, err := bencode.Marshal(mi)
	if err != nil {
		t.Fatalf("bencode.Marshal(metainfo): %v", err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
