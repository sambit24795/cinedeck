package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search movies and TV shows",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, err := newCatalog(cmd)
			if err != nil {
				return err
			}

			results, err := c.Search(cmd.Context(), strings.Join(args, " "))
			if err != nil {
				return err
			}
			return printResults(cmd, results)
		},
	}
}

// printResults writes results as a table, or as JSON when --json is set.
func printResults(cmd *cobra.Command, results []catalog.Result) error {
	if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
		if results == nil {
			results = []catalog.Result{}
		}
		return printJSON(cmd.OutOrStdout(), results)
	}
	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No results.")
		return nil
	}
	return printTable(cmd.OutOrStdout(),
		[]string{"type", "title", "year", "rating", "id"},
		resultRows(results),
	)
}

// resultRows renders results as table rows.
func resultRows(results []catalog.Result) [][]string {
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		rows = append(rows, []string{
			string(r.ID.Type),
			r.Title,
			formatYear(r.Year),
			strconv.FormatFloat(r.Rating, 'f', 1, 64),
			r.ID.String(),
		})
	}
	return rows
}

// formatYear prints 0 as "-".
func formatYear(y int) string {
	if y == 0 {
		return "-"
	}
	return strconv.Itoa(y)
}
