package recs

import "github.com/sambit24795/cinedeck/internal/catalog"

// withSeedQuota picks at most n entries of ranked (which is in rank order) so
// that every seed gets its best candidate before the rest is filled by rank.
//
//  1. Reserve: go through seeds in the given order. For each seed, take the
//     first entry of ranked that is not taken yet and whose RecBy contains
//     that seed's Meta.ID. A seed with no such entry reserves nothing.
//     Stop reserving when n entries are taken.
//  2. Fill: take the remaining entries of ranked in order until n are taken.
//  3. Return the taken entries in their original rank order.
//
// It never returns more than len(ranked) entries and does not modify ranked.
func withSeedQuota(ranked []Scored, seeds []Seed, n int) []Scored {
	if n <= 0 {
		return []Scored{}
	}

	taken := make([]bool, len(ranked))
	count := 0

	// 1. Reserve one slot per seed, in seed order.
	for _, s := range seeds {
		if count >= n {
			break
		}
		for i := range ranked {
			if taken[i] {
				continue
			}
			if hasRecBy(ranked[i].RecBy, s.Meta.ID) {
				taken[i] = true
				count++
				break
			}
		}
	}

	// 2. Fill the rest by rank.
	for i := range ranked {
		if count >= n {
			break
		}
		if !taken[i] {
			taken[i] = true
			count++
		}
	}

	// 3. Return in original rank order.
	out := make([]Scored, 0, count)
	for i := range ranked {
		if taken[i] {
			out = append(out, ranked[i])
		}
	}
	return out
}

// hasRecBy reports whether recBy contains id.
func hasRecBy(recBy []catalog.ID, id catalog.ID) bool {
	for _, x := range recBy {
		if x == id {
			return true
		}
	}
	return false
}
