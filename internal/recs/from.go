package recs

import (
	"context"
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

const (
	MinSimilarVotes = 100 // /similar entries with fewer votes are dropped
	DefaultLimit    = 10
	MaxLimit        = 50

	MaxQueryKeywords = 5 // keywords sent to discover
	MaxQueryGenres   = 3 // genre ids sent to a "any of" discover
)

// CheckLimit returns nil for 1..MaxLimit, else the error: invalid --limit <n>: want 1 to 50
func CheckLimit(n int) error {
	if n < 1 || n > MaxLimit {
		return fmt.Errorf("invalid --limit %d: want 1 to %d", n, MaxLimit)
	}
	return nil
}

// ParseTypeFilter accepts "", "any", "movie" or "tv" (trimmed, any letter case)
// and returns "" for "" and "any". Anything else returns the error:
// invalid --type "<s as given>": must be movie, tv or any
func ParseTypeFilter(s string) (catalog.MediaType, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "any":
		return "", nil
	case string(catalog.Movie):
		return catalog.Movie, nil
	case string(catalog.TV):
		return catalog.TV, nil
	default:
		return "", fmt.Errorf("invalid --type %q: must be movie, tv or any", s)
	}
}

// mergeSame builds the same-type list: all recs in order, then the sims with
// Votes >= MinSimilarVotes in order; the id `self` and duplicates (by ID) are
// skipped; the first `limit` entries are kept. Reasons: recs get
// "TMDB recommends it for <title>", sims get "similar to <title>".
func mergeSame(self catalog.ID, title string, recs, sims []catalog.Related, limit int) []Pick {
	picks := []Pick{}
	seen := map[catalog.ID]bool{self: true}

	add := func(list []catalog.Related, reason string) {
		for _, r := range list {
			if len(picks) >= limit {
				return
			}
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			picks = append(picks, Pick{
				ID:     r.ID,
				Title:  r.Title,
				Year:   r.Year,
				Rating: r.Rating,
				Reason: reason,
			})
		}
	}

	add(recs, "TMDB recommends it for "+title)

	filtered := make([]catalog.Related, 0, len(sims))
	for _, r := range sims {
		if r.Votes >= MinSimilarVotes {
			filtered = append(filtered, r)
		}
	}
	add(filtered, "similar to "+title)

	return picks
}

// FromOptions configures From.
type FromOptions struct {
	Type  catalog.MediaType // "" = both types; Movie or TV = only that type
	Limit int               // per type; the caller has validated it with CheckLimit
}

// FromResult is the result of From.
type FromResult struct {
	Source *catalog.Meta
	Picks  []Pick
}

// From recommends titles like id. Source titles are never included in the result.
func From(ctx context.Context, src Source, id catalog.ID, o FromOptions) (*FromResult, error) {
	m, err := src.Meta(ctx, id)
	if err != nil {
		return nil, err
	}

	types := []catalog.MediaType{o.Type}
	if o.Type == "" {
		types = []catalog.MediaType{id.Type, otherType(id.Type)}
	}

	picks := []Pick{}
	for _, t := range types {
		if t == id.Type {
			recs, err := src.Recommendations(ctx, id)
			if err != nil {
				return nil, err
			}
			sims, err := src.Similar(ctx, id)
			if err != nil {
				return nil, err
			}
			picks = append(picks, mergeSame(id, m.Title, recs, sims, o.Limit)...)
			continue
		}

		cross, err := crossPicks(ctx, src, m, t, o.Limit)
		if err != nil {
			return nil, err
		}
		picks = append(picks, cross...)
	}

	return &FromResult{Source: m, Picks: picks}, nil
}

// otherType returns the other media type.
func otherType(t catalog.MediaType) catalog.MediaType {
	if t == catalog.Movie {
		return catalog.TV
	}
	return catalog.Movie
}

// firstInts returns the first n entries of v (all of them when len(v) <= n).
func firstInts(v []int, n int) []int {
	if len(v) > n {
		return v[:n]
	}
	return v
}

// crossPicks finds titles of type target that share keywords and genres with m
// (target differs from m.ID.Type).
func crossPicks(ctx context.Context, src Source, m *catalog.Meta, target catalog.MediaType, limit int) ([]Pick, error) {
	keys := CanonGenres(m.ID.Type, m.GenreIDs)
	ids := GenreIDsFor(target, keys)

	kw := make([]int, 0, MaxQueryKeywords)
	for _, k := range m.Keywords {
		if len(kw) == MaxQueryKeywords {
			break
		}
		kw = append(kw, k.ID)
	}

	picks := []Pick{}
	seen := map[catalog.ID]bool{}
	add := func(list []catalog.Related, reason string) {
		for _, r := range list {
			if len(picks) >= limit {
				return
			}
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			picks = append(picks, Pick{
				ID:     r.ID,
				Title:  r.Title,
				Year:   r.Year,
				Rating: r.Rating,
				Reason: reason,
			})
		}
	}

	if len(kw) > 0 {
		results, err := src.DiscoverBy(ctx, catalog.DiscoverQuery{
			Type:       target,
			GenreIDs:   firstInts(ids, MaxQueryGenres),
			KeywordIDs: kw,
			MinVotes:   100,
		})
		if err != nil {
			return nil, err
		}
		add(results, "shares keywords with "+m.Title)
	}

	if len(ids) > 0 {
		results, err := src.DiscoverBy(ctx, catalog.DiscoverQuery{
			Type:      target,
			GenreIDs:  firstInts(ids, 2),
			GenresAll: true,
			MinVotes:  500,
		})
		if err != nil {
			return nil, err
		}
		add(results, "shares genres with "+m.Title)
	}

	return picks, nil
}
