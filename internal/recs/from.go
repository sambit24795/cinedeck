package recs

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

const (
	MinSimilarVotes = 100 // /similar entries with fewer votes are dropped
	DefaultLimit    = 10
	MaxLimit        = 50

	MaxQueryKeywords = 5 // keywords sent to discover
	MaxQueryGenres   = 3 // genre ids sent to a "any of" discover

	// CrossEnrichTop is how many cross-type candidates are looked up in
	// detail to check their shared keywords before filtering and ranking.
	CrossEnrichTop = 12
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
// Request bound: the same type needs 1 (meta) + 2 (recommendations, similar);
// the other type needs 2 (discover) + at most CrossEnrichTop (12) detail
// lookups, so --type any uses at most 1 + 2 + 2 + 12 = 17 requests.
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

// poolEntry is one candidate of the cross-type pool: its list entry and which
// of the two discover queries produced it.
type poolEntry struct {
	rel              catalog.Related
	fromKeywordQuery bool
}

// pickFrom copies a list entry into a Pick with the given reason.
func pickFrom(r catalog.Related, reason string) Pick {
	return Pick{ID: r.ID, Title: r.Title, Year: r.Year, Rating: r.Rating, Reason: reason}
}

// sharedGenreCount returns how many of the source's canonical genres appear in
// the list entry's genre ids for target.
func sharedGenreCount(sourceKeys []string, target catalog.MediaType, genreIDs []int) int {
	if len(sourceKeys) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, g := range CanonGenres(target, genreIDs) {
		set[g] = true
	}
	n := 0
	for _, k := range sourceKeys {
		if set[k] {
			n++
		}
	}
	return n
}

// sharedKeywords returns the number of distinct candidate keywords that the
// source also has, and the names of up to two of them in the source's order.
func sharedKeywords(source, cm *catalog.Meta) (count int, names []string) {
	inSource := map[int]bool{}
	for _, k := range source.Keywords {
		inSource[k.ID] = true
	}
	seen := map[int]bool{}
	for _, k := range cm.Keywords {
		if inSource[k.ID] {
			seen[k.ID] = true
		}
	}
	count = len(seen)
	for _, k := range source.Keywords {
		if len(names) == 2 {
			break
		}
		if seen[k.ID] {
			names = append(names, k.Name)
		}
	}
	return count, names
}

// crossPicks finds titles of type target that share keywords and genres with m
// (target differs from m.ID.Type). Shared genres come from the discover list
// entries; shared keywords are checked on the candidate details. A candidate
// is kept when it shares at least two keywords, or one keyword and at least two
// genres. When m has no keywords only the genre query runs.
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

	pool := []poolEntry{}
	seen := map[catalog.ID]bool{}
	addPool := func(results []catalog.Related, fromKeywordQuery bool) {
		for _, r := range results {
			if seen[r.ID] {
				continue
			}
			seen[r.ID] = true
			pool = append(pool, poolEntry{rel: r, fromKeywordQuery: fromKeywordQuery})
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
		addPool(results, true)
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
		addPool(results, false)
	}

	// Without source keywords the keyword query does not run: keep the
	// genre-only behaviour, with no detail requests.
	if len(m.Keywords) == 0 {
		picks := []Pick{}
		for _, pe := range pool {
			if len(picks) >= limit {
				break
			}
			picks = append(picks, pickFrom(pe.rel, "shares genres with "+m.Title))
		}
		return picks, nil
	}

	// Shared genres are known from the list entries alone.
	sg := make([]int, len(pool))
	for i, pe := range pool {
		sg[i] = sharedGenreCount(keys, target, pe.rel.GenreIDs)
	}

	// Stable sort by shared genres descending; ties keep the pool order
	// (query 1 first, then query 2; both are popularity-sorted by TMDB).
	order := make([]int, len(pool))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool { return sg[order[a]] > sg[order[b]] })

	type candidate struct {
		pick Pick
		sk   int
	}
	cands := make([]candidate, 0, len(pool))
	enriched := 0
	for _, idx := range order {
		if enriched >= CrossEnrichTop {
			break
		}
		enriched++
		pe := pool[idx]
		cm, err := src.Meta(ctx, pe.rel.ID)
		if err != nil {
			if ctx.Err() != nil {
				return nil, err
			}
			continue
		}
		sk, names := sharedKeywords(m, cm)
		if sk >= 2 || (sk >= 1 && sg[idx] >= 2) {
			reason := "shares keywords with " + m.Title + " (" + strings.Join(names, ", ") + ")"
			cands = append(cands, candidate{pick: pickFrom(pe.rel, reason), sk: sk})
		}
	}

	// Rank by shared keywords, then shared genres, then pool order: the input
	// is already sorted by shared genres (stable), so a stable sort on the
	// shared keyword count gives the remaining order.
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].sk > cands[j].sk })

	picks := []Pick{}
	for _, c := range cands {
		if len(picks) >= limit {
			break
		}
		picks = append(picks, c.pick)
	}
	return picks, nil
}
