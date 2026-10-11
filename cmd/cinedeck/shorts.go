package main

import (
	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

func newShortsCmd() *cobra.Command {
	var genre string
	var year int
	cmd := &cobra.Command{
		Use:   "shorts",
		Short: "Find short movies",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, _, err := newCatalog(cmd)
			if err != nil {
				return err
			}
			results, err := c.Shorts(cmd.Context(), catalog.ShortsOptions{Genre: genre, Year: year})
			if err != nil {
				return err
			}
			return printResults(cmd, results)
		},
	}
	cmd.Flags().StringVar(&genre, "genre", "", "genre name")
	cmd.Flags().IntVar(&year, "year", 0, "release year")
	return cmd
}
