package catalog

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

const (
	shortMaxMinutes   = 40 // TMDB has no "short" type; this is the Academy definition
	maxDetailRequests = 5  // detail lookups in flight at once
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

// DiscoverOptions filters a discover query.
type DiscoverOptions struct {
	Type     MediaType // Movie or TV; "" means Movie
	Genres   []string  // genre names, any letter case; all must match (AND); empty means any genre
	Language string    // original language, 2-letter code such as "ko"; "" means any
	Country  string    // origin country, 2-letter code such as "KR"; "" means any
	Year     int       // movies: release year; TV: first air year; 0 means any year
	MinVotes int       // minimum vote count; 0 means no minimum
}

// Discover returns movies or TV shows matching o. Only the first page (20
// results) is returned.
func (c *Client) Discover(ctx context.Context, o DiscoverOptions) ([]Result, error) {
	t := o.Type
	if t == "" {
		t = Movie
	}
	if t != Movie && t != TV {
		return nil, fmt.Errorf("invalid type %q: must be movie or tv", t)
	}

	ids, err := c.genreIDs(ctx, t, o.Genres)
	if err != nil {
		return nil, err
	}

	q := url.Values{}
	q.Set("sort_by", "popularity.desc")
	q.Set("include_adult", "false")
	if len(ids) > 0 {
		parts := make([]string, len(ids))
		for i, id := range ids {
			parts[i] = strconv.Itoa(id)
		}
		q.Set("with_genres", strings.Join(parts, ","))
	}
	if o.Language != "" {
		q.Set("with_original_language", o.Language)
	}
	if o.Country != "" {
		q.Set("with_origin_country", o.Country)
	}
	if o.MinVotes > 0 {
		q.Set("vote_count.gte", strconv.Itoa(o.MinVotes))
	}
	if o.Year > 0 {
		if t == TV {
			q.Set("first_air_date_year", strconv.Itoa(o.Year))
		} else {
			q.Set("primary_release_year", strconv.Itoa(o.Year))
		}
	}

	var resp struct {
		Results []item `json:"results"`
	}
	if err := c.get(ctx, "/discover/"+string(t), q, &resp); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(resp.Results))
	for _, it := range resp.Results {
		results = append(results, it.result(t))
	}
	return results, nil
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
		ids, err := c.genreIDs(ctx, Movie, []string{o.Genre})
		if err != nil {
			return nil, err
		}
		q.Set("with_genres", strconv.Itoa(ids[0]))
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
	return c.onlyShorts(ctx, results)
}

// onlyShorts keeps the results whose runtime is 1 to 40 minutes, in the original order.
func (c *Client) onlyShorts(ctx context.Context, in []Result) ([]Result, error) {
	if len(in) == 0 {
		return []Result{}, nil
	}

	runtimes := make([]int, len(in))
	errs := make([]error, len(in))
	sem := make(chan struct{}, maxDetailRequests)
	var wg sync.WaitGroup
	for i := range in {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			d, err := c.Details(ctx, in[i].ID)
			if err != nil {
				errs[i] = err
				return
			}
			runtimes[i] = d.Runtime
		}(i)
	}
	wg.Wait()

	out := make([]Result, 0, len(in))
	for i, r := range in {
		if err := errs[i]; err != nil {
			// A 404 means the title no longer exists, so it cannot be a
			// short we can verify; drop it.
			var apiErr *APIError
			if errors.As(err, &apiErr) && apiErr.Status == 404 {
				continue
			}
			return nil, fmt.Errorf("shorts: runtime of %s: %w", r.ID.String(), err)
		}
		if runtimes[i] >= 1 && runtimes[i] <= shortMaxMinutes {
			out = append(out, r)
		}
	}
	return out, nil
}
