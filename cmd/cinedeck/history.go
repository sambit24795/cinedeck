package main

import (
	"fmt"
	"time"

	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/spf13/cobra"
)

func newHistoryCmd() *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show what you have watched",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit < 1 {
				return fmt.Errorf("invalid --limit %d: must be at least 1", limit)
			}
			lib, err := openLibrary()
			if err != nil {
				return err
			}
			defer lib.Close()

			entries, err := lib.History(cmd.Context(), limit)
			if err != nil {
				return err
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), entryRows(entries, func(e library.Entry) time.Time { return e.WatchedAt }))
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No watch history yet. Mark titles with: cinedeck watched <id|title>")
				return nil
			}

			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, []string{
					e.WatchedAt.UTC().Format("2006-01-02"),
					string(e.Meta.ID.Type),
					e.Meta.Title,
					formatYear(e.Meta.Year),
					userRatingCell(e.Rating),
					likedCell(e.Liked),
					e.Meta.ID.String(),
				})
			}
			return printTable(cmd.OutOrStdout(),
				[]string{"watched", "type", "title", "year", "rated", "liked", "id"},
				rows,
			)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 50, "maximum number of entries")
	return cmd
}
