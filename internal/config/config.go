package config

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config holds the user-configurable settings for cinedeck.
type Config struct {
	TMDBToken   string `toml:"tmdb_token"`
	TMDBURL     string `toml:"tmdb_url"`
	Region      string `toml:"region"`
	DownloadDir string `toml:"download_dir"`
}

// Default returns a Config with the built-in defaults.
func Default() Config {
	home, _ := os.UserHomeDir()
	return Config{
		TMDBURL:     "https://api.themoviedb.org/3",
		Region:      "IN",
		DownloadDir: filepath.Join(home, "Downloads", "cinedeck"),
	}
}

// Path returns the default config file path.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cinedeck", "config.toml"), nil
}

// Load reads the config file at path over the defaults, then applies
// environment overrides. A missing file is not an error.
func Load(path string) (Config, error) {
	c := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return c, err
		}
	} else {
		if err := toml.Unmarshal(data, &c); err != nil {
			return c, err
		}
	}

	if v := os.Getenv("CINEDECK_TMDB_TOKEN"); v != "" {
		c.TMDBToken = v
	}
	if v := os.Getenv("CINEDECK_TMDB_URL"); v != "" {
		c.TMDBURL = v
	}
	if v := os.Getenv("CINEDECK_REGION"); v != "" {
		c.Region = v
	}
	if v := os.Getenv("CINEDECK_DOWNLOAD_DIR"); v != "" {
		c.DownloadDir = v
	}

	return c, nil
}

// Save writes c to path, creating the parent directory with mode 0o700 and
// the file with mode 0o600.
func Save(path string, c Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(c); err != nil {
		return err
	}

	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		return err
	}
	// os.WriteFile only applies the mode when it creates the file, so tighten
	// the permissions in case the file already existed with a looser mode.
	return os.Chmod(path, 0o600)
}
