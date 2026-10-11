package main

import (
	"strings"

	"github.com/spf13/cobra"
)

func newLikeCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "like <id|title>",
		Short: "Like a movie or TV show",
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
			changed, err := lib.Like(ctx, m.ID)
			if err != nil {
				return err
			}

			text := "Liked " + itemLabel(m)
			if !changed {
				text = "Already liked " + itemLabel(m)
			}
			return printAction(cmd, actionOutput{
				ID:      m.ID,
				Title:   m.Title,
				Year:    m.Year,
				Action:  "like",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

func newUnlikeCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "unlike <id|title>",
		Short: "Remove a like",
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
				changed, err = lib.Unlike(ctx, id)
				if err != nil {
					return err
				}
			}
			title, year, label := lookup(ctx, lib, id)

			text := "Unliked " + label
			if !changed {
				text = "Not in your likes: " + label
			}
			return printAction(cmd, actionOutput{
				ID:      id,
				Title:   title,
				Year:    year,
				Action:  "unlike",
				Changed: changed,
			}, text)
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}
