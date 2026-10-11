package recs

import (
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

const (
	MinSimilarVotes = 100 // /similar entries with fewer votes are dropped
	DefaultLimit    = 10
	MaxLimit        = 50
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
