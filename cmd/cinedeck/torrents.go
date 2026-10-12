package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/indexer"
	"github.com/sambit24795/cinedeck/internal/indexer/providers"
	"github.com/spf13/cobra"
)

// newTorrentsRegistryFunc is a seam for tests to inject fake providers.
var newTorrentsRegistryFunc = defaultTorrentsRegistry

// defaultTorrentsRegistry builds the registry of built-in providers.
func defaultTorrentsRegistry() *indexer.Registry {
	r := indexer.NewRegistry()
	r.Register(providers.NewApibay())
	r.Register(providers.NewYTS())
	r.Register(providers.NewEZTV())
	r.Register(providers.NewNyaa())
	r.Register(providers.NewArchive())
	return r
}

// enabledProviderNames splits the config list into names, dropping blanks.
func enabledProviderNames(cfg config.Config) []string {
	if strings.TrimSpace(cfg.IndexerProviders) == "" {
		return nil
	}
	var out []string
	for _, n := range strings.Split(cfg.IndexerProviders, ",") {
		if n = strings.TrimSpace(n); n != "" {
			out = append(out, n)
		}
	}
	return out
}

func newTorrentsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "torrents",
		Short: "Search public torrent indexers",
	}
	cmd.AddCommand(newTorrentsSearchCmd())
	return cmd
}

func newTorrentsSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search enabled indexers for a title",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := newCatalog(cmd)
			if err != nil {
				return err
			}

			typeStr, _ := cmd.Flags().GetString("type")
			mt, err := parseTorrentType(typeStr)
			if err != nil {
				return err
			}
			sortBy, _ := cmd.Flags().GetString("sort")
			year, _ := cmd.Flags().GetInt("year")
			minSeeders, _ := cmd.Flags().GetInt("min-seeders")
			limit, _ := cmd.Flags().GetInt("limit")
			providerFlag, _ := cmd.Flags().GetStringSlice("provider")

			reg := newTorrentsRegistryFunc()
			names := providerFlag
			if len(names) == 0 {
				names = enabledProviderNames(cfg)
			}
			selected := reg.Enabled(names)

			if len(selected) == 0 {
				return fmt.Errorf("no indexer providers enabled: set indexer_providers in the config (available: %s)", strings.Join(reg.Names(), ", "))
			}
			if len(providerFlag) > 0 {
				if err := checkProviderNames(reg, providerFlag); err != nil {
					return err
				}
			}

			q := indexer.Query{
				Text:       strings.Join(args, " "),
				Type:       mt,
				Year:       year,
				MinSeeders: minSeeders,
				Limit:      limit,
				Sort:       sortBy,
			}
			if err := q.Validate(); err != nil {
				return err
			}

			i := indexer.NewIndexer(indexer.WithTimeout(parseTimeout(cfg.IndexerTimeout)))
			res, err := i.Search(cmd.Context(), q, selected)
			if err != nil {
				return err
			}

			// Provider failures are reported but do not fail the command.
			for _, pe := range res.Errors {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", pe.Error())
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), res.Releases)
			}
			if len(res.Releases) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No results.")
				return nil
			}
			return printTable(cmd.OutOrStdout(),
				[]string{"provider", "title", "seeders", "leechers", "size", "info_hash"},
				torrentRows(res.Releases),
			)
		},
	}

	cmd.Flags().String("type", "any", "movie, tv or any")
	cmd.Flags().Int("year", 0, "filter by year")
	cmd.Flags().Int("min-seeders", 0, "minimum seeder count")
	cmd.Flags().StringSlice("provider", nil, "provider name(s) to use (repeatable)")
	cmd.Flags().String("sort", "seeders", "seeders, size or date")
	cmd.Flags().Int("limit", 20, "maximum results")
	return cmd
}

// parseTorrentType maps the --type flag to a media type.
func parseTorrentType(s string) (indexer.MediaType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "any":
		return indexer.Any, nil
	case "movie":
		return indexer.Movie, nil
	case "tv":
		return indexer.TV, nil
	default:
		return "", fmt.Errorf("invalid type %q: must be movie, tv or any", s)
	}
}

// checkProviderNames returns an error naming the unknown providers.
func checkProviderNames(reg *indexer.Registry, names []string) error {
	known := make(map[string]bool)
	for _, n := range reg.Names() {
		known[n] = true
	}
	for _, n := range names {
		if !known[n] {
			return fmt.Errorf("unknown provider %q: available are %s", n, strings.Join(reg.Names(), ", "))
		}
	}
	return nil
}

// parseTimeout parses a duration string, defaulting to 10s on any error.
func parseTimeout(s string) time.Duration {
	d, err := time.ParseDuration(strings.TrimSpace(s))
	if err != nil || d <= 0 {
		return 10 * time.Second
	}
	return d
}

// torrentRows renders releases as table rows.
func torrentRows(releases []indexer.Release) [][]string {
	rows := make([][]string, 0, len(releases))
	for _, r := range releases {
		rows = append(rows, []string{
			r.Provider,
			r.Title,
			formatCount(r.Seeders),
			formatCount(r.Leechers),
			formatSize(r.Size),
			r.InfoHash,
		})
	}
	return rows
}

// formatCount prints 0 as "-".
func formatCount(n int) string {
	if n == 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

// formatSize renders a byte count in human-readable units.
func formatSize(n uint64) string {
	if n == 0 {
		return "-"
	}
	const unit = 1000
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for q := n / unit; q >= unit; q /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "kMGTPE"[exp])
}
