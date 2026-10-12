package main

import (
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/spf13/cobra"
)

// loadConfigOnly loads the configuration without requiring a TMDB token.
// Commands that do not need TMDB (torrents, free, providers) use this.
func loadConfigOnly(cmd *cobra.Command) (config.Config, error) {
	path, err := configPath(cmd)
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(path)
}

// newCatalog loads the configuration and builds a TMDB client for cmd.
func newCatalog(cmd *cobra.Command) (*catalog.Client, config.Config, error) {
	path, err := configPath(cmd)
	if err != nil {
		return nil, config.Config{}, err
	}

	cfg, err := config.Load(path)
	if err != nil {
		return nil, config.Config{}, err
	}
	if cfg.TMDBToken == "" {
		return nil, cfg, errors.New(`no TMDB token: run "cinedeck config init"`)
	}

	dir := os.Getenv("CINEDECK_CACHE_DIR")
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			return nil, cfg, err
		}
		dir = filepath.Join(base, "cinedeck", "tmdb")
	}

	c := catalog.New(cfg.TMDBToken,
		catalog.WithBaseURL(cfg.TMDBURL),
		catalog.WithCache(dir, 24*time.Hour),
	)
	return c, cfg, nil
}
