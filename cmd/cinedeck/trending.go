package main

import (
	"github.com/spf13/cobra"
)

func newTrendingCmd() *cobra.Command {
	var typ string
	cmd := &cobra.Command{
		Use:   "trending",
		Short: "Show trending movies or TV shows",
		RunE: func(cmd *cobra.Command, args []string) error {
			mt, err := parseMediaType(typ)
			if err != nil {
				return err
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
