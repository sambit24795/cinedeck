package main

import (
	"fmt"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

func newTrendingCmd() *cobra.Command {
	var typ string
	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Show trending movies or TV shows",
		RunE: func(cmd *cobra.Command, args []string) error {
			var mt catalog.MediaType
			switch typ {
			case "movie":
				mt = catalog.Movie
			case "tv":
				mt = catalog.TV
			default:
				return fmt.Errorf("invalid type %q: must be movie or tv", typ)
			}

			c, _, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			results, err := c.Trending(cmd.Context(), mt)
			if err != nil {
				return err
			}
			return printResults(cmd, results)
		},
	}
	cmd.Flags().StringVar(&typ, "type", "movie", "movie or tv")
	return cmd
}
