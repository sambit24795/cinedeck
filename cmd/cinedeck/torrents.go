package main

import (
	"fmt"
	"os"
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
	cmd.AddCommand(newTorrentsInfoCmd())
	cmd.AddCommand(newTorrentsProvidersCmd())
	return cmd
}

// isInfoHash reports whether s is a 40-character hexadecimal info hash.
func isInfoHash(s string) bool {
	if len(s) != 40 {
		return false
	}
	for _, c := range s {
		switch {
		case c >= '0' && c <= '9', c >= 'a' && c <= 'f', c >= 'A' && c <= 'F':
		default:
			return false
		}
	}
	return true
}

// infoOutput is the JSON shape for `torrents info`.
type infoOutput struct {
	Title     string          `json:"title"`
	InfoHash  string          `json:"info_hash"`
	Size      uint64          `json:"size"`
	Files     []indexer.File  `json:"files"`
	Providers []providerCount `json:"providers"`
}

type providerCount struct {
	Provider string `json:"provider"`
	Seeders  int    `json:"seeders"`
	Leechers int    `json:"leechers"`
}

func newTorrentsInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <hash|magnet|file.torrent>",
		Short: "Show details for one torrent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			reg := newTorrentsRegistryFunc()
			selected := reg.Enabled(enabledProviderNames(cfg))
			i := indexer.NewIndexer(indexer.WithTimeout(parseTimeout(cfg.IndexerTimeout)))

			arg := args[0]
			var out infoOutput

			switch {
			case strings.HasPrefix(arg, "magnet:"):
				hash, title, err := indexer.ParseMagnet(arg)
				if err != nil {
					return err
				}
				out.InfoHash = hash
				out.Title = title
				if err := addProviderCounts(cmd, i, selected, hash, &out); err != nil {
					return err
				}
			case isInfoHash(arg):
				out.InfoHash = strings.ToLower(arg)
				if err := addProviderCounts(cmd, i, selected, out.InfoHash, &out); err != nil {
					return err
				}
			case strings.HasSuffix(strings.ToLower(arg), ".torrent"):
				data, err := os.ReadFile(arg)
				if err != nil {
					return err
				}
				t, err := i.ParseTorrent(cmd.Context(), data)
				if err != nil {
					return err
				}
				out.Title = t.Title
				out.InfoHash = t.InfoHash
				out.Size = t.Size
				out.Files = t.Files
			default:
				return fmt.Errorf("not a magnet, info hash or .torrent path: %q", arg)
			}

			if out.Files == nil {
				out.Files = []indexer.File{}
			}
			if out.Providers == nil {
				out.Providers = []providerCount{}
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), out)
			}
			return printTorrentInfo(cmd, out)
		},
	}
}

// addProviderCounts fills out from the providers that know infoHash.
func addProviderCounts(cmd *cobra.Command, i *indexer.Indexer, selected []indexer.Provider, infoHash string, out *infoOutput) error {
	res, err := i.LookupHash(cmd.Context(), infoHash, selected)
	if err != nil {
		return err
	}
	for _, pe := range res.Errors {
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", pe.Error())
	}
	for _, r := range res.Releases {
		if out.Title == "" && r.Title != "" {
			out.Title = r.Title
		}
		if out.Size == 0 {
			out.Size = r.Size
		}
		out.Providers = append(out.Providers, providerCount{
			Provider: r.Provider,
			Seeders:  r.Seeders,
			Leechers: r.Leechers,
		})
	}
	return nil
}

// printTorrentInfo writes the text form of a torrent info result.
func printTorrentInfo(cmd *cobra.Command, out infoOutput) error {
	w := cmd.OutOrStdout()
	if out.Title != "" {
		fmt.Fprintf(w, "Title: %s\n", out.Title)
	}
	fmt.Fprintf(w, "Info hash: %s\n", out.InfoHash)
	if out.Size > 0 {
		fmt.Fprintf(w, "Size: %s\n", formatSize(out.Size))
	}
	if len(out.Files) > 0 {
		fmt.Fprintln(w, "Files:")
		for _, f := range out.Files {
			fmt.Fprintf(w, "  %s (%s)\n", f.Name, formatSize(f.Size))
		}
	}
	if len(out.Providers) > 0 {
		fmt.Fprintln(w, "Providers:")
		for _, p := range out.Providers {
			fmt.Fprintf(w, "  %s seeders=%s leechers=%s\n", p.Provider, formatCount(p.Seeders), formatCount(p.Leechers))
		}
	}
	return nil
}

// providersOutput is one row of `torrents providers`.
type providersOutput struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

func newTorrentsProvidersCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "providers",
		Short: "List indexer providers",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, cfg, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			reg := newTorrentsRegistryFunc()
			enabled := make(map[string]bool)
			for _, n := range enabledProviderNames(cfg) {
				enabled[n] = true
			}

			rows := make([]providersOutput, 0, len(reg.Names()))
			for _, n := range reg.Names() {
				rows = append(rows, providersOutput{Name: n, Enabled: enabled[n]})
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), rows)
			}
			tableRows := make([][]string, 0, len(rows))
			for _, r := range rows {
				tableRows = append(tableRows, []string{r.Name, strconv.FormatBool(r.Enabled)})
			}
			return printTable(cmd.OutOrStdout(), []string{"provider", "enabled"}, tableRows)
		},
	}
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
