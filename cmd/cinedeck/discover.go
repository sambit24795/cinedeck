package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	var typ, genre, lang, country string
	var year int
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Browse movies or TV shows by genre, language, country and year",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mt, err := parseMediaType(typ)
			if err != nil {
				return err
			}
			language, err := normalizeLang(lang)
			if err != nil {
				return err
			}
			origin, err := normalizeCountry(country)
			if err != nil {
				return err
			}
			if err := checkYear(year); err != nil {
				return err
			}

			genre = strings.TrimSpace(genre)
			if genre == "" && language == "" && origin == "" && year == 0 {
				return errors.New("discover needs at least one filter: --genre, --lang, --country or --year")
			}
			var genres []string
			if genre != "" {
				genres = []string{genre}
			}
			return runDiscover(cmd, catalog.DiscoverOptions{
				Type:     mt,
				Genres:   genres,
				Language: language,
				Country:  origin,
				Year:     year,
			})
		},
	}
	cmd.Flags().StringVar(&typ, "type", "movie", "movie or tv")
	cmd.Flags().StringVar(&genre, "genre", "", "genre name (TV and movie genre names differ)")
	cmd.Flags().StringVar(&lang, "lang", "", "original language, 2-letter code such as ko")
	cmd.Flags().StringVar(&country, "country", "", "origin country, 2-letter code such as KR")
	cmd.Flags().IntVar(&year, "year", 0, "release year (movies) or first air year (TV)")
	return cmd
}

// parseMediaType trims and lower-cases s, accepting only "movie" or "tv".
func parseMediaType(s string) (catalog.MediaType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case string(catalog.Movie):
		return catalog.Movie, nil
	case string(catalog.TV):
		return catalog.TV, nil
	default:
		return "", fmt.Errorf("invalid type %q: must be movie or tv", s)
	}
}

// normalizeLang trims and lower-cases s. An empty value is allowed; otherwise
// it must be exactly two letters a-z.
func normalizeLang(s string) (string, error) {
	v := strings.ToLower(strings.TrimSpace(s))
	if v == "" {
		return "", nil
	}
	if len(v) != 2 || v[0] < 'a' || v[0] > 'z' || v[1] < 'a' || v[1] > 'z' {
		return "", fmt.Errorf("invalid --lang %q: want a 2-letter language code like ko", s)
	}
	return v, nil
}

// normalizeCountry trims and upper-cases s. An empty value is allowed;
// otherwise it must be exactly two letters A-Z.
func normalizeCountry(s string) (string, error) {
	v := strings.ToUpper(strings.TrimSpace(s))
	if v == "" {
		return "", nil
	}
	if len(v) != 2 || v[0] < 'A' || v[0] > 'Z' || v[1] < 'A' || v[1] > 'Z' {
		return "", fmt.Errorf("invalid --country %q: want a 2-letter country code like KR", s)
	}
	return v, nil
}

// checkYear reports whether y is 0 (any year) or between 1870 and 2100.
func checkYear(y int) error {
	if y == 0 {
		return nil
	}
	if y < 1870 || y > 2100 {
		return fmt.Errorf("invalid --year %d: want a year between 1870 and 2100", y)
	}
	return nil
}

// runDiscover builds a catalog client and prints the results of a discover query.
func runDiscover(cmd *cobra.Command, o catalog.DiscoverOptions) error {
	c, _, err := newCatalog(cmd)
	if err != nil {
		return err
	}
	results, err := c.Discover(cmd.Context(), o)
	if err != nil {
		return err
	}
	return printResults(cmd, results)
}
