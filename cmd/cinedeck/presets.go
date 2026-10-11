package main

import (
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

// presetMinVotes is the minimum vote count used by the kdrama and anime
// presets so titles with almost no votes do not appear.
const presetMinVotes = 50

func newKdramaCmd() *cobra.Command {
	var genre string
	var year int
	cmd := &cobra.Command{
		Use:   "kdrama",
		Short: "Korean-language TV dramas",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkYear(year); err != nil {
				return err
			}
			genres := []string{"Drama"}
			if g := strings.TrimSpace(genre); g != "" {
				genres = append(genres, g)
			}
			return runDiscover(cmd, catalog.DiscoverOptions{
				Type:     catalog.TV,
				Genres:   genres,
				Language: "ko",
				Year:     year,
				MinVotes: presetMinVotes,
			})
		},
	}
	cmd.Flags().StringVar(&genre, "genre", "", "extra genre name, added to the preset's own genre")
	cmd.Flags().IntVar(&year, "year", 0, "release year (movies) or first air year (TV)")
	return cmd
}

func newAnimeCmd() *cobra.Command {
	var typ, genre string
	var year int
	cmd := &cobra.Command{
		Use:   "anime",
		Short: "Japanese animation (TV by default)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mt, err := parseMediaType(typ)
			if err != nil {
				return err
			}
			if err := checkYear(year); err != nil {
				return err
			}
			genres := []string{"Animation"}
			if g := strings.TrimSpace(genre); g != "" {
				genres = append(genres, g)
			}
			return runDiscover(cmd, catalog.DiscoverOptions{
				Type:     mt,
				Genres:   genres,
				Language: "ja",
				Year:     year,
				MinVotes: presetMinVotes,
			})
		},
	}
	cmd.Flags().StringVar(&typ, "type", "tv", "tv or movie")
	cmd.Flags().StringVar(&genre, "genre", "", "extra genre name, added to the preset's own genre")
	cmd.Flags().IntVar(&year, "year", 0, "release year (movies) or first air year (TV)")
	return cmd
}
