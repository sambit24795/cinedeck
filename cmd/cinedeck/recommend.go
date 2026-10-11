package main

import (
	"fmt"
	"strconv"

	"github.com/sambit24795/cinedeck/internal/catalog"
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
				return runPersonal(cmd, tf, limit)
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

// runPersonal prints personal recommendations from the library.
func runPersonal(cmd *cobra.Command, tf catalog.MediaType, limit int) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	lib, err := openLibrary()
	if err != nil {
		return err
	}
	defer lib.Close()

	entries, err := lib.Entries(cmd.Context())
	if err != nil {
		return err
	}

	if seeds, _ := recs.Classify(entries); len(seeds) == 0 {
		lines := []string{
			"Your library has no liked or highly rated titles yet.",
			"Like some first, for example: cinedeck like <id|title>   (or rate one 8 or higher: cinedeck rate <id|title> 9)",
		}
		if !jsonOut {
			for _, line := range lines {
				fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		}
		if err := printJSON(cmd.OutOrStdout(), []recs.Pick{}); err != nil {
			return err
		}
		for _, line := range lines {
			fmt.Fprintln(cmd.ErrOrStderr(), line)
		}
		return nil
	}

	c, _, err := newCatalog(cmd)
	if err != nil {
		return err
	}
	res, err := recs.Personal(cmd.Context(), c, entries, recs.PersonalOptions{Type: tf, Limit: limit})
	if err != nil {
		return err
	}

	if res.Seeds < 3 {
		fmt.Fprintf(cmd.ErrOrStderr(), "Tip: you have %d liked or highly rated title(s); recommendations improve with more (like titles, or rate them 8 or higher).\n", res.Seeds)
	}

	if jsonOut {
		return printJSON(cmd.OutOrStdout(), res.Picks)
	}
	if len(res.Picks) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No recommendations found. Try liking more titles.")
		return nil
	}

	noun := "titles"
	if res.Seeds == 1 {
		noun = "title"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Recommended for you, based on %d liked or highly rated %s:\n\n", res.Seeds, noun)

	rows := make([][]string, 0, len(res.Picks))
	for _, p := range res.Picks {
		rows = append(rows, []string{
			string(p.ID.Type),
			p.Title,
			formatYear(p.Year),
			formatRating(p.Rating),
			p.ID.String(),
			strconv.FormatFloat(p.Score, 'f', 2, 64),
			p.Reason,
		})
	}
	return printTable(cmd.OutOrStdout(), []string{"type", "title", "year", "rating", "id", "score", "reason"}, rows)
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
