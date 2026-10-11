package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
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
			return runFrom(cmd, from, first, tf, limit)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "title or id to find similar titles for")
	cmd.Flags().StringVar(&typ, "type", "any", "movie, tv or any")
	cmd.Flags().IntVar(&limit, "limit", recs.DefaultLimit, "maximum number of results per type")
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

// fromStatus renders the user's state for an entry, in a fixed order.
func fromStatus(e library.Entry) string {
	parts := []string{}
	if e.Liked {
		parts = append(parts, "liked")
	}
	if e.Rating > 0 {
		parts = append(parts, fmt.Sprintf("rated %d/10", e.Rating))
	}
	if e.Watched {
		parts = append(parts, "watched")
	}
	if e.OnWatchlist {
		parts = append(parts, "on watchlist")
	}
	return strings.Join(parts, ", ")
}

// runFrom prints --from recommendations, hiding disliked titles and marking
// titles the user already liked, rated, watched or put on the watchlist.
func runFrom(cmd *cobra.Command, from string, first bool, tf catalog.MediaType, limit int) error {
	exclude := map[catalog.ID]bool{}
	statuses := map[catalog.ID]string{}

	lib, err := openLibraryIfExists()
	if err != nil {
		return err
	}
	if lib != nil {
		defer lib.Close()
		entries, err := lib.Entries(cmd.Context())
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.Disliked() {
				exclude[e.Meta.ID] = true
			}
			if s := fromStatus(e); s != "" {
				statuses[e.Meta.ID] = s
			}
		}
	}

	c, _, err := newCatalog(cmd)
	if err != nil {
		return err
	}
	id, err := resolveID(cmd, c, from, first)
	if err != nil {
		return err
	}
	res, err := recs.From(cmd.Context(), c, id, recs.FromOptions{Type: tf, Limit: limit, Exclude: exclude})
	if err != nil {
		return err
	}
	return printFromResult(cmd, res, statuses)
}

// runPersonal prints personal recommendations from the library.
func runPersonal(cmd *cobra.Command, tf catalog.MediaType, limit int) error {
	jsonOut, _ := cmd.Flags().GetBool("json")

	lib, err := openLibraryIfExists()
	if err != nil {
		return err
	}
	var entries []library.Entry
	if lib != nil {
		defer lib.Close()
		entries, err = lib.Entries(cmd.Context())
		if err != nil {
			return err
		}
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

// fromRow is one --from result: a Pick plus the user's status for that title.
type fromRow struct {
	recs.Pick
	Status string `json:"status,omitempty"`
}

// printFromResult writes res as JSON when --json is set, else as a table. The
// status column is printed only when at least one row has a status.
func printFromResult(cmd *cobra.Command, res *recs.FromResult, statuses map[catalog.ID]string) error {
	rows := make([]fromRow, 0, len(res.Picks))
	hasStatus := false
	for _, p := range res.Picks {
		s := statuses[p.ID]
		if s != "" {
			hasStatus = true
		}
		rows = append(rows, fromRow{Pick: p, Status: s})
	}

	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		return printJSON(cmd.OutOrStdout(), rows)
	}

	out := cmd.OutOrStdout()
	if len(rows) == 0 {
		fmt.Fprintf(out, "No recommendations found for %s.\n", res.Source.Title)
		return nil
	}

	title := res.Source.Title
	if res.Source.Year > 0 {
		title = fmt.Sprintf("%s (%d)", title, res.Source.Year)
	}
	fmt.Fprintf(out, "Titles like %s:\n\n", title)

	headers := []string{"type", "title", "year", "rating", "id", "reason"}
	if hasStatus {
		headers = append(headers, "status")
	}
	table := make([][]string, 0, len(rows))
	for _, p := range rows {
		cells := []string{
			string(p.ID.Type),
			p.Title,
			formatYear(p.Year),
			formatRating(p.Rating),
			p.ID.String(),
			p.Reason,
		}
		if hasStatus {
			cells = append(cells, p.Status)
		}
		table = append(table, cells)
	}
	return printTable(out, headers, table)
}
