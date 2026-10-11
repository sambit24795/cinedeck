package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/spf13/cobra"
)

// parseRating accepts a whole number from 1 to 10.
func parseRating(s string) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || n < 1 || n > 10 {
		return 0, fmt.Errorf("invalid rating %q: want a whole number from 1 to 10", s)
	}
	return n, nil
}

func newRateCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "rate <id|title> <n>",
		Short: "Rate a movie or TV show from 1 to 10",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := parseRating(args[len(args)-1])
			if err != nil {
				return err
			}
			m, err := resolveMeta(cmd, strings.Join(args[:len(args)-1], " "), first)
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
			previous, err := lib.Rate(ctx, m.ID, n)
			if err != nil {
				return err
			}

			text := fmt.Sprintf("Rated %s %d/10", itemLabel(m), n)
			var notes []string
			if previous > 0 && previous != n {
				notes = append(notes, fmt.Sprintf("was %d", previous))
			}
			if n <= library.DislikeMax {
				notes = append(notes, "counts as disliked")
			}
			if len(notes) > 0 {
				text += " (" + strings.Join(notes, ", ") + ")"
			}
			return printAction(cmd, actionOutput{
				ID:      m.ID,
				Title:   m.Title,
				Year:    m.Year,
				Action:  "rate",
				Changed: previous != n,
				Rating:  n,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

func newUnrateCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "unrate <id|title>",
		Short: "Remove your rating",
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
				changed, err = lib.Unrate(ctx, id)
				if err != nil {
					return err
				}
			}
			title, year, label := lookup(ctx, lib, id)

			text := "Removed your rating from " + label
			if !changed {
				text = "You have not rated: " + label
			}
			return printAction(cmd, actionOutput{
				ID:      id,
				Title:   title,
				Year:    year,
				Action:  "unrate",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}
