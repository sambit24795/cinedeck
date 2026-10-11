package config

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

// TestMain makes the tests hermetic: the caller may have the CINEDECK_*
// overrides exported, and Load would apply them to every test.
func TestMain(m *testing.M) {
	for _, key := range []string{"CINEDECK_TMDB_TOKEN", "CINEDECK_TMDB_URL", "CINEDECK_REGION", "CINEDECK_DOWNLOAD_DIR"} {
		os.Unsetenv(key)
	}
	os.Exit(m.Run())
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.toml")
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if want := Default(); !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	want := Config{
		TMDBToken:   "tok",
		TMDBURL:     "https://example.com/3",
		Region:      "US",
		DownloadDir: "/tmp/dl",
	}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestLoadEnvOverride(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("region = \"US\"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	t.Setenv("CINEDECK_REGION", "GB")
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Region != "GB" {
		t.Errorf("Region = %q, want %q", got.Region, "GB")
	}
}

func TestLoadInvalidTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("region = \n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("Load: got nil error, want an error")
	}
}

func TestSaveTightensExistingFilePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not meaningful on Windows")
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("region = \"US\"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := Save(path, Default()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want %o", got, 0o600)
	}
}

func TestSaveFilePermission(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("file permissions are not meaningful on Windows")
	}
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := Save(path, Default()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("file mode = %o, want %o", got, 0o600)
	}
}
