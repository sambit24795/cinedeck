package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/spf13/cobra"
)

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

			// The token is a secret, so the prompt never echoes it.
			if cfg.TMDBToken == "" {
				fmt.Fprint(out, "TMDB token: ")
			} else {
				fmt.Fprint(out, "TMDB token [keep current]: ")
			}
			if v, err := readAnswer(in); err == nil && v != "" {
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

			if err := config.Save(path, cfg); err != nil {
				return err
			}
			fmt.Fprintf(out, "Saved %s\n", path)
			return nil
		},
	}
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

// readAnswer reads one line and trims surrounding whitespace.
func readAnswer(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	return strings.TrimSpace(line), err
}
