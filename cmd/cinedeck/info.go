package main

import (
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/indexer"
	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "info <id|title>",
		Short: "Show details for a movie or TV show",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			id, err := resolveID(cmd, c, args[0], first)
			if err != nil {
				return err
			}
			d, err := c.Details(cmd.Context(), id)
			if err != nil {
				return err
			}

			// Saved torrents are best-effort: a missing database just means none.
			saved := []indexer.Release{}
			if lib, err := openLibraryIfExists(); err == nil && lib != nil {
				saved, _ = lib.ReleasesFor(cmd.Context(), id.String())
				lib.Close()
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), detailsOutput{Details: d, SavedTorrents: saved})
			}
			printDetails(cmd, d)
			printSavedTorrents(cmd, saved)
			return nil
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

// detailsOutput is the --json shape for info: the details plus saved torrents.
type detailsOutput struct {
	*catalog.Details
	SavedTorrents []indexer.Release `json:"saved_torrents"`
}

// printSavedTorrents writes an "Available torrents" section when there are saved releases.
func printSavedTorrents(cmd *cobra.Command, saved []indexer.Release) {
	if len(saved) == 0 {
		return
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Available torrents")
	for _, r := range saved {
		fmt.Fprintf(out, "  %s  seeders=%s  %s  %s\n", r.Provider, formatCount(r.Seeders), formatSize(r.Size), r.InfoHash)
	}
}

// printDetails writes d as one field per line, then the overview.
func printDetails(cmd *cobra.Command, d *catalog.Details) {
	out := cmd.OutOrStdout()
	if d.Year > 0 {
		fmt.Fprintf(out, "%s (%d)\n", d.Title, d.Year)
	} else {
		fmt.Fprintln(out, d.Title)
	}
	if d.Rating > 0 {
		fmt.Fprintf(out, "Rating: %.1f\n", d.Rating)
	}
	if len(d.Genres) > 0 {
		fmt.Fprintf(out, "Genres: %s\n", strings.Join(d.Genres, ", "))
	}
	if d.Runtime > 0 {
		fmt.Fprintf(out, "Runtime: %d min\n", d.Runtime)
	}
	if d.Seasons > 0 {
		fmt.Fprintf(out, "Seasons: %d\n", d.Seasons)
	}
	if d.Episodes > 0 {
		fmt.Fprintf(out, "Episodes: %d\n", d.Episodes)
	}
	if len(d.Directors) > 0 {
		fmt.Fprintf(out, "Director: %s\n", strings.Join(d.Directors, ", "))
	}
	if len(d.Creators) > 0 {
		fmt.Fprintf(out, "Created by: %s\n", strings.Join(d.Creators, ", "))
	}
	if len(d.Cast) > 0 {
		fmt.Fprintf(out, "Cast: %s\n", strings.Join(d.Cast, ", "))
	}
	fmt.Fprintln(out)
	if d.Overview != "" {
		fmt.Fprintln(out, d.Overview)
	}
}
