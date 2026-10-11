package catalog

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
)

// Genres returns the genre ids for t, keyed by lower-case genre name.
func (c *Client) Genres(ctx context.Context, t MediaType) (map[string]int, error) {
	var resp struct {
		Genres []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"genres"`
	}
	if err := c.get(ctx, "/genre/"+string(t)+"/list", nil, &resp); err != nil {
		return nil, err
	}

	genres := make(map[string]int, len(resp.Genres))
	for _, g := range resp.Genres {
		genres[strings.ToLower(g.Name)] = g.ID
	}
	return genres, nil
}

// genreIDs resolves genre names to ids for t, in the order given, without
// duplicates. Names are matched ignoring letter case and surrounding spaces.
func (c *Client) genreIDs(ctx context.Context, t MediaType, names []string) ([]int, error) {
	if len(names) == 0 {
		return nil, nil
	}

	genres, err := c.Genres(ctx, t)
	if err != nil {
		return nil, err
	}

	ids := make([]int, 0, len(names))
	seen := make(map[int]bool, len(names))
	for _, name := range names {
		id, ok := genres[strings.ToLower(strings.TrimSpace(name))]
		if !ok {
			available := make([]string, 0, len(genres))
			for key := range genres {
				available = append(available, key)
			}
			sort.Strings(available)
			return nil, fmt.Errorf("unknown genre %q for %s; available: %s", name, t, strings.Join(available, ", "))
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// ShortsOptions filters the short-movie list.
type ShortsOptions struct {
	Genre string // genre name, any letter case; empty means all genres
	Year  int    // 0 means any year
}

// Shorts returns short movies matching o.
func (c *Client) Shorts(ctx context.Context, o ShortsOptions) ([]Result, error) {
	q := url.Values{}
	q.Set("with_runtime.lte", "40")
	q.Set("sort_by", "popularity.desc")
	q.Set("include_adult", "false")

	if o.Genre != "" {
		genres, err := c.Genres(ctx, Movie)
		if err != nil {
			return nil, err
		}
		id, ok := genres[strings.ToLower(o.Genre)]
		if !ok {
			return nil, fmt.Errorf("unknown genre %q", o.Genre)
		}
		q.Set("with_genres", strconv.Itoa(id))
	}
	if o.Year > 0 {
		q.Set("primary_release_year", strconv.Itoa(o.Year))
	}

	var resp struct {
		Results []item `json:"results"`
	}
	if err := c.get(ctx, "/discover/movie", q, &resp); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(resp.Results))
	for _, it := range resp.Results {
		results = append(results, it.result(Movie))
	}
	return results, nil
}
