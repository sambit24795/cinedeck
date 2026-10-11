package main

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

// resolveID turns arg into an ID. It accepts a typed id directly, otherwise it
// searches and either picks the first hit or asks the user to choose.
func resolveID(cmd *cobra.Command, c *catalog.Client, arg string, first bool) (catalog.ID, error) {
	if id, err := catalog.ParseID(arg); err == nil {
		return id, nil
	}

	results, err := c.Search(cmd.Context(), arg)
	if err != nil {
		return catalog.ID{}, err
	}
	if len(results) == 0 {
		return catalog.ID{}, fmt.Errorf("no match for %q", arg)
	}
	if len(results) == 1 || first {
		return results[0].ID, nil
	}

	out := cmd.OutOrStdout()
	for i, r := range results {
		title := r.Title
		if r.Year > 0 {
			title = fmt.Sprintf("%s (%d)", r.Title, r.Year)
		}
		fmt.Fprintf(out, "%d) %s [%s]\n", i+1, title, r.ID)
	}
	fmt.Fprint(out, "Pick a number: ")

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return catalog.ID{}, err
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(results) {
		return catalog.ID{}, fmt.Errorf("invalid selection %q", strings.TrimSpace(line))
	}
	return results[n-1].ID, nil
}
