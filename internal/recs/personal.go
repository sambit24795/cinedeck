package recs

import (
	"context"
	"errors"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

const (
	EnrichTop   = 12 // candidates that get a detail lookup (keywords, cast, directors)
	MaxRequests = 32 // upper bound of TMDB requests per run; the design uses at most MaxSeeds + 4 + EnrichTop = 24
)

// PersonalOptions configures Personal.
type PersonalOptions struct {
	Type  catalog.MediaType // "" = both; Movie or TV
	Limit int               // validated by the caller with CheckLimit
}

// PersonalResult is the result of Personal.
type PersonalResult struct {
	Picks    []Pick
	Seeds    int // number of positive titles in the library (all of them, not cut to MaxSeeds)
	Requests int // number of Source calls made (for tests and for the live bound check)
}

// Personal recommends titles from the user's library. With no positive seed it
// returns an empty result (Seeds == 0) and no error.
func Personal(ctx context.Context, src Source, entries []library.Entry, o PersonalOptions) (*PersonalResult, error) {
	seeds, excluded := Classify(entries)
	res := &PersonalResult{Picks: []Pick{}, Seeds: len(seeds)}
	if len(seeds) == 0 {
		return res, nil
	}

	used := seeds
	if len(used) > MaxSeeds {
		used = used[:MaxSeeds]
	}
	p := BuildProfile(used)

	counted := &countingSource{src: src}
	types := filterTypes(o.Type)
	inTypes := func(t catalog.MediaType) bool {
		for _, x := range types {
			if x == t {
				return true
			}
		}
		return false
	}

	pool := map[catalog.ID]*Candidate{}
	order := []catalog.ID{}
	get := func(r catalog.Related) *Candidate {
		c, ok := pool[r.ID]
		if !ok {
			c = &Candidate{
				ID:         r.ID,
				Title:      r.Title,
				Year:       r.Year,
				Poster:     r.Poster,
				Rating:     r.Rating,
				Votes:      r.Votes,
				Popularity: r.Popularity,
				Genres:     CanonGenres(r.ID.Type, r.GenreIDs),
			}
			pool[r.ID] = c
			order = append(order, r.ID)
		} else if c.Poster == "" {
			c.Poster = r.Poster
		}
		return c
	}
	keep := func(r catalog.Related) bool {
		return !excluded[r.ID] && inTypes(r.ID.Type)
	}

	for _, s := range used {
		if !inTypes(s.Meta.ID.Type) {
			continue
		}
		list, err := counted.Recommendations(ctx, s.Meta.ID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return nil, err
		}
		seen := map[catalog.ID]bool{}
		for _, r := range list {
			if !keep(r) {
				continue
			}
			c := get(r)
			if !seen[r.ID] {
				seen[r.ID] = true
				c.RecBy = append(c.RecBy, s.Meta.ID)
				c.SourceWeight += s.Weight
			}
		}
	}

	for _, t := range types {
		ids := GenreIDsFor(t, topGenres(p, 3))
		kw := topKeywords(p, MaxQueryKeywords)

		if len(kw) > 0 {
			list, err := counted.DiscoverBy(ctx, catalog.DiscoverQuery{Type: t, GenreIDs: ids, KeywordIDs: kw, MinVotes: 100})
			if err != nil {
				if !isNotFound(err) {
					return nil, err
				}
			} else {
				for _, r := range list {
					if keep(r) {
						get(r)
					}
				}
			}
		}

		if len(ids) > 0 {
			list, err := counted.DiscoverBy(ctx, catalog.DiscoverQuery{Type: t, GenreIDs: firstInts(ids, 2), GenresAll: true, MinVotes: 500})
			if err != nil {
				if !isNotFound(err) {
					return nil, err
				}
			} else {
				for _, r := range list {
					if keep(r) {
						get(r)
					}
				}
			}
		}
	}

	candidates := make([]Candidate, 0, len(order))
	for _, id := range order {
		candidates = append(candidates, *pool[id])
	}

	ranked := Rank(p, candidates)
	if len(ranked) > EnrichTop {
		ranked = ranked[:EnrichTop]
	}
	top := make([]Candidate, len(ranked))
	for i, s := range ranked {
		top[i] = s.Candidate
	}

	for i := range top {
		id := top[i].ID
		m, err := counted.Meta(ctx, id)
		if err != nil {
			if ctx.Err() != nil {
				return nil, err
			}
			continue
		}
		top[i].Genres = CanonGenres(id.Type, m.GenreIDs)
		top[i].Keywords = keywordIDs(m.Keywords)
		top[i].Cast = personIDs(m.Cast)
		top[i].Directors = personIDs(m.Directors)
		top[i].Rating = m.Rating
		top[i].Votes = m.Votes
		top[i].Popularity = m.Popularity
		top[i].Enriched = true
	}

	final := Rank(p, top)
	if len(final) > o.Limit {
		final = final[:o.Limit]
	}
	for _, s := range final {
		res.Picks = append(res.Picks, Pick{
			ID:     s.ID,
			Title:  s.Title,
			Year:   s.Year,
			Rating: s.Rating,
			Reason: Reason(used, s.Candidate),
			Score:  s.Score,
			Poster: s.Poster,
		})
	}

	res.Requests = counted.count
	return res, nil
}

// filterTypes returns [Movie, TV] filtered by t (both when t is "").
func filterTypes(t catalog.MediaType) []catalog.MediaType {
	types := []catalog.MediaType{}
	for _, x := range []catalog.MediaType{catalog.Movie, catalog.TV} {
		if t == "" || t == x {
			types = append(types, x)
		}
	}
	return types
}

// isNotFound reports whether err is a 404 APIError.
func isNotFound(err error) bool {
	var apiErr *catalog.APIError
	return errors.As(err, &apiErr) && apiErr.Status == 404
}

func keywordIDs(ks []catalog.Keyword) []int {
	ids := make([]int, 0, len(ks))
	for _, k := range ks {
		ids = append(ids, k.ID)
	}
	return ids
}

func personIDs(ps []catalog.Person) []int {
	ids := make([]int, 0, len(ps))
	for _, p := range ps {
		ids = append(ids, p.ID)
	}
	return ids
}

// countingSource counts the calls to the four Source methods.
type countingSource struct {
	src   Source
	count int
}

func (c *countingSource) Meta(ctx context.Context, id catalog.ID) (*catalog.Meta, error) {
	c.count++
	return c.src.Meta(ctx, id)
}

func (c *countingSource) Recommendations(ctx context.Context, id catalog.ID) ([]catalog.Related, error) {
	c.count++
	return c.src.Recommendations(ctx, id)
}

func (c *countingSource) Similar(ctx context.Context, id catalog.ID) ([]catalog.Related, error) {
	c.count++
	return c.src.Similar(ctx, id)
}

func (c *countingSource) DiscoverBy(ctx context.Context, q catalog.DiscoverQuery) ([]catalog.Related, error) {
	c.count++
	return c.src.DiscoverBy(ctx, q)
}
