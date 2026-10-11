package recs

import (
	"sort"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

// MaxSeeds is the number of newest positive titles used to build the profile.
const MaxSeeds = 8

// Seed is a title the user liked or rated highly.
type Seed struct {
	Meta   catalog.Meta
	Weight float64 // rating/10 when rated, else 1
	Liked  bool
	Rating int       // 0 = not rated
	At     time.Time // the later of LikedAt and RatedAt
}

// Classify splits the library into positive seeds and titles that must not be
// recommended.
//
//	seeds:    entries with Positive(), newest At first, ties by ID.Type then ID.N ascending. ALL of them (not cut to MaxSeeds).
//	excluded: every entry that is Liked, rated (Rating > 0) or Watched (this
//	          includes disliked titles). Watchlist-only titles are NOT excluded.
func Classify(entries []library.Entry) (seeds []Seed, excluded map[catalog.ID]bool) {
	seeds = []Seed{}
	excluded = map[catalog.ID]bool{}

	for _, e := range entries {
		if e.Liked || e.Rating > 0 || e.Watched {
			excluded[e.Meta.ID] = true
		}
		if !e.Positive() {
			continue
		}

		weight := 1.0
		if e.Rating > 0 {
			weight = float64(e.Rating) / 10
		}

		at := e.LikedAt
		if e.RatedAt.After(at) {
			at = e.RatedAt
		}

		seeds = append(seeds, Seed{
			Meta:   e.Meta,
			Weight: weight,
			Liked:  e.Liked,
			Rating: e.Rating,
			At:     at,
		})
	}

	sort.SliceStable(seeds, func(i, j int) bool {
		a, b := seeds[i], seeds[j]
		if !a.At.Equal(b.At) {
			return a.At.After(b.At)
		}
		if a.Meta.ID.Type != b.Meta.ID.Type {
			return a.Meta.ID.Type < b.Meta.ID.Type
		}
		return a.Meta.ID.N < b.Meta.ID.N
	})

	return seeds, excluded
}

// Profile is the weighted taste vector. Keys: canonical genre keys, keyword ids, person ids.
type Profile struct {
	Genres    map[string]float64
	Keywords  map[int]float64
	Cast      map[int]float64
	Directors map[int]float64
}

// BuildProfile adds each seed's Weight once to every distinct feature of the
// seed: its canonical genres, keyword ids, cast ids and director ids. All four
// maps are non-nil.
func BuildProfile(seeds []Seed) Profile {
	p := Profile{
		Genres:    map[string]float64{},
		Keywords:  map[int]float64{},
		Cast:      map[int]float64{},
		Directors: map[int]float64{},
	}

	for _, s := range seeds {
		for _, key := range CanonGenres(s.Meta.ID.Type, s.Meta.GenreIDs) {
			p.Genres[key] += s.Weight
		}
		for _, id := range distinctKeywordIDs(s.Meta.Keywords) {
			p.Keywords[id] += s.Weight
		}
		for _, id := range distinctPersonIDs(s.Meta.Cast) {
			p.Cast[id] += s.Weight
		}
		for _, id := range distinctPersonIDs(s.Meta.Directors) {
			p.Directors[id] += s.Weight
		}
	}

	return p
}

// distinctKeywordIDs returns the keyword ids without duplicates, in first-seen order.
func distinctKeywordIDs(ks []catalog.Keyword) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, k := range ks {
		if !seen[k.ID] {
			seen[k.ID] = true
			out = append(out, k.ID)
		}
	}
	return out
}

// distinctPersonIDs returns the person ids without duplicates, in first-seen order.
func distinctPersonIDs(ps []catalog.Person) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, p := range ps {
		if !seen[p.ID] {
			seen[p.ID] = true
			out = append(out, p.ID)
		}
	}
	return out
}

// topGenres returns the n genre keys with the largest weight (ties: key ascending).
func topGenres(p Profile, n int) []string {
	keys := make([]string, 0, len(p.Genres))
	for k := range p.Genres {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if p.Genres[keys[i]] != p.Genres[keys[j]] {
			return p.Genres[keys[i]] > p.Genres[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > n {
		keys = keys[:n]
	}
	return keys
}

// topKeywords returns the n keyword ids with the largest weight (ties: id ascending).
func topKeywords(p Profile, n int) []int {
	ids := make([]int, 0, len(p.Keywords))
	for id := range p.Keywords {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if p.Keywords[ids[i]] != p.Keywords[ids[j]] {
			return p.Keywords[ids[i]] > p.Keywords[ids[j]]
		}
		return ids[i] < ids[j]
	})
	if len(ids) > n {
		ids = ids[:n]
	}
	return ids
}
