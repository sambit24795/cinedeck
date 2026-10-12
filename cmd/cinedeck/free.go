package main

import (
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/indexer"
	"github.com/sambit24795/cinedeck/internal/indexer/providers"
	"github.com/spf13/cobra"
)

// newArchiveProvider is a seam for tests to point the free command at a
// local fixture server instead of archive.org.
var newFreeArchiveProvider = func() *providers.Archive { return providers.NewArchive() }

// newFreeCmd searches the Internet Archive for public-domain movies and media.
// It is the legal source shortcut: no TMDB token is required.
func newFreeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "free <query>",
		Short: "Search the Internet Archive for free, public-domain media",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			archive := newFreeArchiveProvider()
			releases, err := archive.Search(cmd.Context(), indexer.Query{Text: query})
			if err != nil {
				return err
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), releases)
			}
			if len(releases) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No results.")
				return nil
			}

			rows := freeRows(releases)
			return printTable(cmd.OutOrStdout(),
				[]string{"title", "year", "details"},
				rows,
			)
		},
	}
	return cmd
}

// freeRows renders Archive releases as table rows for the free command.
func freeRows(releases []indexer.Release) [][]string {
	rows := make([][]string, 0, len(releases))
	for _, r := range releases {
		rows = append(rows, []string{
			r.Title,
			formatYear(r.Year),
			r.InfoURL,
		})
	}
	return rows
}
