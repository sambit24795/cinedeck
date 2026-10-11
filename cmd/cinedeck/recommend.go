package main

import (
	"errors"
	"fmt"

	"github.com/sambit24795/cinedeck/internal/recs"
	"github.com/spf13/cobra"
)

func newRecommendCmd() *cobra.Command {
	var from, typ string
	var limit int
	var first bool
	cmd := &cobra.Command{
		Use:   "recommend",
		Short: "Recommend movies and TV shows",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			tf, err := recs.ParseTypeFilter(typ)
			if err != nil {
				return err
			}
			if err := recs.CheckLimit(limit); err != nil {
				return err
			}

			if from == "" {
				return errors.New("recommend: personal recommendations are not implemented yet")
			}

			c, _, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			id, err := resolveID(cmd, c, from, first)
			if err != nil {
				return err
			}
			res, err := recs.From(cmd.Context(), c, id, recs.FromOptions{Type: tf, Limit: limit})
			if err != nil {
				return err
			}
			return printFromResult(cmd, res)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "title or id to find similar titles for")
	cmd.Flags().StringVar(&typ, "type", "any", "movie, tv or any")
	cmd.Flags().IntVar(&limit, "limit", recs.DefaultLimit, "maximum number of results per type")
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

// printFromResult writes res as JSON when --json is set, else as a table.
func printFromResult(cmd *cobra.Command, res *recs.FromResult) error {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		return printJSON(cmd.OutOrStdout(), res.Picks)
	}

	out := cmd.OutOrStdout()
	if len(res.Picks) == 0 {
		fmt.Fprintf(out, "No recommendations found for %s.\n", res.Source.Title)
		return nil
	}

	title := res.Source.Title
	if res.Source.Year > 0 {
		title = fmt.Sprintf("%s (%d)", title, res.Source.Year)
	}
	fmt.Fprintf(out, "Titles like %s:\n\n", title)

	rows := make([][]string, 0, len(res.Picks))
	for _, p := range res.Picks {
		rows = append(rows, []string{
			string(p.ID.Type),
			p.Title,
			formatYear(p.Year),
			formatRating(p.Rating),
			p.ID.String(),
			p.Reason,
		})
	}
	return printTable(out, []string{"type", "title", "year", "rating", "id", "reason"}, rows)
}
