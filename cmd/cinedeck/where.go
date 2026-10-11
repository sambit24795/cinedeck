package main

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

// justWatchNotice is the required attribution for TMDB watch-provider data.
const justWatchNotice = "Where-to-watch data is provided by JustWatch."

// whereOutput is the --json shape for both the found and the not-found case.
type whereOutput struct {
	ID          catalog.ID         `json:"id"`
	Region      string             `json:"region"`
	Available   bool               `json:"available"`
	Link        string             `json:"link"`
	Stream      []catalog.Provider `json:"stream"`
	Free        []catalog.Provider `json:"free"`
	Ads         []catalog.Provider `json:"ads"`
	Rent        []catalog.Provider `json:"rent"`
	Buy         []catalog.Provider `json:"buy"`
	Attribution string             `json:"attribution"`
}

func newWhereCmd() *cobra.Command {
	var first bool
	cmd := &cobra.Command{
		Use:   "where <id|title>",
		Short: "Show where to watch a movie or TV show",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, cfg, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			id, err := resolveID(cmd, c, strings.Join(args, " "), first)
			if err != nil {
				return err
			}
			region := regionFor(cmd, cfg)

			w, err := c.WhereToWatch(cmd.Context(), id, region)
			if err != nil && !errors.Is(err, catalog.ErrNoProviders) {
				return err
			}

			if jsonOut, _ := cmd.Flags().GetBool("json"); jsonOut {
				out := whereOutput{
					ID:          id,
					Region:      region,
					Stream:      []catalog.Provider{},
					Free:        []catalog.Provider{},
					Ads:         []catalog.Provider{},
					Rent:        []catalog.Provider{},
					Buy:         []catalog.Provider{},
					Attribution: justWatchNotice,
				}
				if err == nil {
					out.Available = true
					out.Link = w.Link
					out.Stream = w.Stream
					out.Free = w.Free
					out.Ads = w.Ads
					out.Rent = w.Rent
					out.Buy = w.Buy
				}
				// Keep stdout valid JSON; the notice goes to stderr too.
				fmt.Fprintln(cmd.ErrOrStderr(), justWatchNotice)
				return printJSON(cmd.OutOrStdout(), out)
			}

			stdout := cmd.OutOrStdout()
			if err != nil {
				// ErrNoProviders: "nothing to stream there" is a valid answer.
				fmt.Fprintf(stdout, "No watch providers for %s in %s.\n\n%s\n", id, region, justWatchNotice)
				return nil
			}
			printWhere(stdout, w)
			return nil
		},
	}
	cmd.Flags().BoolVar(&first, "first", false, "pick the first search result")
	return cmd
}

// printWhere writes the text form of w, grouped in the M2 order.
func printWhere(out io.Writer, w *catalog.WhereToWatch) {
	fmt.Fprintf(out, "Where to watch %s in %s\n", w.ID, w.Region)
	fmt.Fprintln(out)

	groups := []struct {
		heading string
		list    []catalog.Provider
	}{
		{"Stream", w.Stream},
		{"Free", w.Free},
		{"With ads", w.Ads},
		{"Rent", w.Rent},
		{"Buy", w.Buy},
	}
	for _, g := range groups {
		if len(g.list) == 0 {
			continue
		}
		fmt.Fprintln(out, g.heading)
		for _, p := range g.list {
			fmt.Fprintf(out, "  %s\n", p.Name)
		}
	}

	if w.Link != "" {
		fmt.Fprintf(out, "\nTMDB watch page: %s\n", w.Link)
	}
	fmt.Fprintf(out, "\n%s\n", justWatchNotice)
}
