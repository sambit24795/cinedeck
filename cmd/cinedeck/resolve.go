package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

// searcher is the part of the catalog that resolveID needs.
type searcher interface {
	Search(ctx context.Context, query string) ([]catalog.Result, error)
}

var _ searcher = (*catalog.Client)(nil)

// resolveID turns arg into an ID. It accepts a typed id directly, otherwise it
// searches and either picks the first hit or asks the user to choose.
func resolveID(cmd *cobra.Command, c searcher, arg string, first bool) (catalog.ID, error) {
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

	// With --json, stdout must stay valid JSON, so the picker goes to stderr.
	jsonOut, _ := cmd.Flags().GetBool("json")
	out := cmd.OutOrStdout()
	if jsonOut {
		out = cmd.ErrOrStderr()
	}
	for i, r := range results {
		title := r.Title
		if r.Year > 0 {
			title = fmt.Sprintf("%s (%d)", r.Title, r.Year)
		}
		fmt.Fprintf(out, "%d) %s [%s]\n", i+1, title, r.ID)
	}
	fmt.Fprint(out, "Pick a number: ")

	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if jsonOut {
		// In a terminal the answer shares the prompt line; under --json the
		// notice that follows must start on its own line.
		fmt.Fprint(out, "\n")
	}
	if err != nil && strings.TrimSpace(line) == "" {
		return catalog.ID{}, errors.New("no selection made")
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(results) {
		return catalog.ID{}, fmt.Errorf("invalid selection %q", strings.TrimSpace(line))
	}
	return results[n-1].ID, nil
}
