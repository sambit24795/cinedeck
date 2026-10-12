package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/spf13/cobra"
)

// tokenHelp is printed before the token prompt by config init.
const tokenHelp = `You need a free TMDB API token. Each user brings their own.
1. Make a free account at https://www.themoviedb.org, then open https://www.themoviedb.org/settings/api
2. Request an API key. In TMDB's application form enter:
     Application Name:    cinedeck (personal use)
     Application URL:     https://github.com/sambit24795/cinedeck
     Type of Use:         Desktop Application
     Application Summary: A command-line tool for personal use to search movies and TV shows and see where to watch them.
3. On the same page copy the long "API Read Access Token" (it starts with eyJ).
   Do not use the short "API Key".

`

// configPath returns the --config flag value, or the default config path when
// the flag is empty.
func configPath(cmd *cobra.Command) (string, error) {
	if path, _ := cmd.Root().PersistentFlags().GetString("config"); path != "" {
		return path, nil
	}
	return config.Path()
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}
	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath(cmd)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create a config file",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath(cmd)
			if err != nil {
				return err
			}

			cfg, err := loadConfigFile(path)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			in := bufio.NewReader(cmd.InOrStdin())

			fmt.Fprint(out, tokenHelp)

			// The token is a secret, so the prompt never echoes it.
			if cfg.TMDBToken == "" {
				fmt.Fprint(out, "TMDB token: ")
			} else {
				fmt.Fprint(out, "TMDB token [keep current]: ")
			}
			if v, err := readAnswer(in); err == nil && v != "" {
				if isShortAPIKey(v) {
					return errors.New(`that looks like the TMDB "API Key"; paste the longer "API Read Access Token" instead`)
				}
				fmt.Fprint(out, "Checking token with TMDB... ")
				tokenClient := catalog.New(v, catalog.WithBaseURL(tmdbBaseURL(cfg)))
				if err := tokenClient.CheckToken(cmd.Context()); err != nil {
					var apiErr *catalog.APIError
					if errors.As(err, &apiErr) && apiErr.Status == 401 {
						fmt.Fprintln(out, "rejected")
						return errors.New(`TMDB rejected that token (401). Nothing was saved. Use the long "API Read Access Token" from https://www.themoviedb.org/settings/api`)
					}
					fmt.Fprintln(out, "could not check")
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: could not check the token with TMDB (%v); saving it anyway\n", err)
				} else {
					fmt.Fprintln(out, "ok")
				}
				cfg.TMDBToken = v
			}

			fmt.Fprintf(out, "Region [%s]: ", cfg.Region)
			if v, err := readAnswer(in); err == nil && v != "" {
				cfg.Region = v
			}

			fmt.Fprintf(out, "Download dir [%s]: ", cfg.DownloadDir)
			if v, err := readAnswer(in); err == nil && v != "" {
				cfg.DownloadDir = v
			}

			if cfg.IndexerProviders == "" {
				fmt.Fprint(out, "Indexer providers (comma list, empty to skip): ")
			} else {
				fmt.Fprintf(out, "Indexer providers [%s]: ", cfg.IndexerProviders)
			}
			if v, err := readAnswer(in); err == nil && v != "" {
				cfg.IndexerProviders = v
			}

			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(out, "Saved %s\n", path)
			return nil
		},
	}
}

// tmdbBaseURL returns the TMDB API root used for the config init token check.
// The CINEDECK_TMDB_URL environment variable wins over cfg so tests can point
// the check at an httptest server. cfg comes from loadConfigFile, which does
// not apply environment overrides.
func tmdbBaseURL(cfg config.Config) string {
	if u := os.Getenv("CINEDECK_TMDB_URL"); u != "" {
		return u
	}
	return cfg.TMDBURL
}

// loadConfigFile decodes the config file at path over the built-in defaults,
// without applying environment overrides. Unlike config.Load, env values are
// never written back to disk by config init. A missing file is not an error.
func loadConfigFile(path string) (config.Config, error) {
	cfg := config.Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// isShortAPIKey reports whether v is a 32-character hexadecimal value, the
// shape of TMDB's short "API Key". cinedeck needs the longer JWT
// "API Read Access Token" instead.
func isShortAPIKey(v string) bool {
	if len(v) != 32 {
		return false
	}
	for _, c := range v {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// readAnswer reads one line and trims surrounding whitespace.
func readAnswer(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}
