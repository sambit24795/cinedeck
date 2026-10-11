package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/spf13/cobra"
)

// libraryRow is the --json element of `watchlist ls` and `history`.
type libraryRow struct {
	ID         catalog.ID `json:"id"`
	Title      string     `json:"title"`
	Year       int        `json:"year"`
	Rating     float64    `json:"rating"` // TMDB vote_average
	Genres     []string   `json:"genres"`
	Liked      bool       `json:"liked"`
	YourRating int        `json:"your_rating,omitempty"`
	At         string     `json:"at"` // RFC3339 UTC: when it was added to the watchlist / watched
}

func newWatchlistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watchlist",
		Short: "Manage your watchlist",
	}
	cmd.AddCommand(newWatchlistAddCmd())
	cmd.AddCommand(newWatchlistRMCmd())
	cmd.AddCommand(newWatchlistLSCmd())
	return cmd
}

func newWatchlistAddCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "add <id|title>",
		Short: "Add a movie or TV show to your watchlist",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := resolveMeta(cmd, strings.Join(args, " "), first)
			if err != nil {
				return err
			}
			lib, err := openLibrary()
			if err != nil {
				return err
			}
			defer lib.Close()

			ctx := cmd.Context()
			if err := lib.UpsertItem(ctx, m); err != nil {
				return err
			}
			entry, err := lib.Entry(ctx, m.ID)
			if err != nil {
				return err
			}

			var changed bool
			var text string
			if entry.Watched {
				text = "Already watched, not added: " + itemLabel(m)
			} else {
				changed, err = lib.AddToWatchlist(ctx, m.ID)
				if err != nil {
					return err
				}
				if changed {
					text = "Added to watchlist: " + itemLabel(m)
				} else {
					text = "Already on your watchlist: " + itemLabel(m)
				}
			}
			return printAction(cmd, actionOutput{
				ID:      m.ID,
				Title:   m.Title,
				Year:    m.Year,
				Action:  "watchlist add",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

func newWatchlistRMCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "rm <id|title>",
		Short: "Remove a movie or TV show from your watchlist",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveRef(cmd, strings.Join(args, " "), first)
			if err != nil {
				return err
			}
			lib, err := openLibraryIfExists()
			if err != nil {
				return err
			}
			if lib != nil {
				defer lib.Close()
			}

			ctx := cmd.Context()
			changed := false
			if lib != nil {
				changed, err = lib.RemoveFromWatchlist(ctx, id)
				if err != nil {
					return err
				}
			}
			title, year, label := lookup(ctx, lib, id)

			text := "Removed from watchlist: " + label
			if !changed {
				text = "Not on your watchlist: " + label
			}
			return printAction(cmd, actionOutput{
				ID:      id,
				Title:   title,
				Year:    year,
				Action:  "watchlist rm",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

func newWatchlistLSCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "ls",
		Short: "List your watchlist",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			lib, err := openLibraryIfExists()
			if err != nil {
				return err
			}
			if lib != nil {
				defer lib.Close()
			}

			var entries []library.Entry
			if lib != nil {
				entries, err = lib.Watchlist(cmd.Context())
				if err != nil {
					return err
				}
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				return printJSON(cmd.OutOrStdout(), entryRows(entries, func(e library.Entry) time.Time { return e.AddedAt }))
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "Your watchlist is empty. Add titles with: cinedeck watchlist add <id|title>")
				return nil
			}

			rows := make([][]string, 0, len(entries))
			for _, e := range entries {
				rows = append(rows, []string{
					string(e.Meta.ID.Type),
					e.Meta.Title,
					formatYear(e.Meta.Year),
					formatRating(e.Meta.Rating),
					e.Meta.ID.String(),
					e.AddedAt.UTC().Format("2006-01-02"),
				})
			}
			return printTable(cmd.OutOrStdout(),
				[]string{"type", "title", "year", "rating", "id", "added"},
				rows,
			)
		},
	}
}

// entryRows converts entries to the shared --json shape. at picks the timestamp.
func entryRows(entries []library.Entry, at func(library.Entry) time.Time) []libraryRow {
	rows := make([]libraryRow, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, libraryRow{
			ID:         e.Meta.ID,
			Title:      e.Meta.Title,
			Year:       e.Meta.Year,
			Rating:     e.Meta.Rating,
			Genres:     e.Meta.Genres,
			Liked:      e.Liked,
			YourRating: e.Rating,
			At:         at(e).Format(time.RFC3339),
		})
	}
	return rows
}

// userRatingCell renders a rating, or "-" when there is none.
func userRatingCell(n int) string {
	if n == 0 {
		return "-"
	}
	return strconv.Itoa(n)
}

// likedCell renders a like, or "-" when there is none.
func likedCell(liked bool) string {
	if !liked {
		return "-"
	}
	return "yes"
}
