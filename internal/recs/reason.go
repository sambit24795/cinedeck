package recs

import (
	"fmt"
	"strings"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// Reason explains why c is recommended, for example
//
//	"because you liked Breaking Bad (shared: Vince Gilligan, new mexico, dark comedy)"
func Reason(seeds []Seed, c Candidate) string {
	if len(seeds) == 0 {
		return "matches your taste"
	}

	s, ok := explainingSeed(seeds, c)
	if !ok {
		return "matches your taste"
	}

	shared := sharedNames(s, c)
	text := ""
	if s.Liked {
		text = "because you liked " + s.Meta.Title
	} else {
		text = fmt.Sprintf("because you rated %s %d/10", s.Meta.Title, s.Rating)
	}
	if len(shared) > 0 {
		text += " (shared: " + strings.Join(shared, ", ") + ")"
	}
	return text
}

// explainingSeed picks the seed that explains c best.
func explainingSeed(seeds []Seed, c Candidate) (Seed, bool) {
	if len(c.RecBy) > 0 {
		recBy := map[catalog.ID]bool{}
		for _, id := range c.RecBy {
			recBy[id] = true
		}

		var best Seed
		found := false
		for _, s := range seeds {
			if !recBy[s.Meta.ID] {
				continue
			}
			if !found || s.Weight > best.Weight || (s.Weight == best.Weight && seedLess(s, best)) {
				best = s
				found = true
			}
		}
		if found {
			return best, true
		}
	}

	var best Seed
	bestValue := 0.0
	found := false
	for _, s := range seeds {
		value := s.Weight * float64(sharedCount(s, c))
		if value <= 0 {
			continue
		}
		if !found || value > bestValue || (value == bestValue && seedLess(s, best)) {
			best = s
			bestValue = value
			found = true
		}
	}
	return best, found
}

// seedLess orders seeds by type then id, ascending.
func seedLess(a, b Seed) bool {
	if a.Meta.ID.Type != b.Meta.ID.Type {
		return a.Meta.ID.Type < b.Meta.ID.Type
	}
	return a.Meta.ID.N < b.Meta.ID.N
}

// sharedCount counts the distinct features shared by s and c.
func sharedCount(s Seed, c Candidate) int {
	genres := stringSet(c.Genres)
	kws := intSet(c.Keywords)
	dirs := intSet(c.Directors)
	cast := intSet(c.Cast)

	n := 0
	for _, g := range CanonGenres(s.Meta.ID.Type, s.Meta.GenreIDs) {
		if genres[g] {
			n++
		}
	}
	n += countKeywordIntersect(s.Meta.Keywords, kws)
	n += countPersonIntersect(s.Meta.Directors, dirs)
	n += countPersonIntersect(s.Meta.Cast, cast)
	return n
}

// sharedNames returns at most three shared display names, in a fixed order.
func sharedNames(s Seed, c Candidate) []string {
	dirs := intSet(c.Directors)
	kws := intSet(c.Keywords)
	cast := intSet(c.Cast)
	genres := stringSet(c.Genres)

	names := []string{}
	for _, p := range s.Meta.Directors {
		if len(names) == 3 {
			return names
		}
		if dirs[p.ID] {
			names = append(names, p.Name)
		}
	}
	for _, k := range s.Meta.Keywords {
		if len(names) == 3 {
			return names
		}
		if kws[k.ID] {
			names = append(names, k.Name)
		}
	}
	for _, p := range s.Meta.Cast {
		if len(names) == 3 {
			return names
		}
		if cast[p.ID] {
			names = append(names, p.Name)
		}
	}
	for _, g := range CanonGenres(s.Meta.ID.Type, s.Meta.GenreIDs) {
		if len(names) == 3 {
			return names
		}
		if genres[g] {
			names = append(names, GenreName(g))
		}
	}
	return names
}

func countKeywordIntersect(list []catalog.Keyword, set map[int]bool) int {
	seen := map[int]bool{}
	n := 0
	for _, k := range list {
		if set[k.ID] && !seen[k.ID] {
			seen[k.ID] = true
			n++
		}
	}
	return n
}

func countPersonIntersect(list []catalog.Person, set map[int]bool) int {
	seen := map[int]bool{}
	n := 0
	for _, p := range list {
		if set[p.ID] && !seen[p.ID] {
			seen[p.ID] = true
			n++
		}
	}
	return n
}

func stringSet(v []string) map[string]bool {
	set := make(map[string]bool, len(v))
	for _, s := range v {
		set[s] = true
	}
	return set
}

func intSet(v []int) map[int]bool {
	set := make(map[int]bool, len(v))
	for _, n := range v {
		set[n] = true
	}
	return set
}
