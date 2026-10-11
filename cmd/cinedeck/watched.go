package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newWatchedCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "watched <id|title>",
		Short: "Mark a movie or TV show as watched",
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
			changed, err := lib.MarkWatched(ctx, m.ID)
			if err != nil {
				return err
			}

			text := "Marked as watched: " + itemLabel(m)
			if !changed {
				text = "Already marked as watched: " + itemLabel(m)
			}
			return printAction(cmd, actionOutput{
				ID:      m.ID,
				Title:   m.Title,
				Year:    m.Year,
				Action:  "watched",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

func newUnwatchCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "unwatch <id|title>",
		Short: "Remove the watched mark",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := resolveRef(cmd, strings.Join(args, " "), first)
			if err != nil {
				return err
			}
			lib, err := openLibrary()
			if err != nil {
				return err
			}
			defer lib.Close()

			ctx := cmd.Context()
			changed, err := lib.Unwatch(ctx, id)
			if err != nil {
				return err
			}
			title, year, label := lookup(ctx, lib, id)

			text := "Removed from watched: " + label
			if !changed {
				text = "Not marked as watched: " + label
			}
			return printAction(cmd, actionOutput{
				ID:      id,
				Title:   title,
				Year:    year,
				Action:  "unwatch",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}
