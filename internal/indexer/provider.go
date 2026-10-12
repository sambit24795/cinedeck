package indexer

import (
	"context"
	"fmt"
	"sort"
)

// MediaType filters search results by kind.
type MediaType string

const (
	Any   MediaType = "any"
	Movie MediaType = "movie"
	TV    MediaType = "tv"
)

// Query describes a torrent search or hash lookup.
type Query struct {
	Text       string    // required unless InfoHash is set
	InfoHash   string    // when set, look up this exact hash instead of searching
	Type       MediaType // Any means no type filter
	Year       int       // 0 means any year
	MinSeeders int       // 0 means no minimum
	Limit      int       // 0 means the provider's default
	Sort       string    // "seeders" (default), "size" or "date"
}

// Release is one torrent returned by a provider.
type Release struct {
	Provider string `json:"provider"`
	Title    string `json:"title"`
	InfoHash string `json:"info_hash"`
	Magnet   string `json:"magnet,omitempty"`
	Size     uint64 `json:"size"`
	Seeders  int    `json:"seeders"`
	Leechers int    `json:"leechers"`
	Year     int    `json:"year,omitempty"`
	Uploader string `json:"uploader,omitempty"`
	Category string `json:"category,omitempty"`
	InfoURL  string `json:"info_url,omitempty"`
}

// Provider is a torrent indexer source.
type Provider interface {
	Name() string
	Search(ctx context.Context, q Query) ([]Release, error)
}

// Validate checks the query fields and returns a descriptive error.
func (q Query) Validate() error {
	if q.Text == "" && q.InfoHash == "" {
		return fmt.Errorf("query needs text or an info hash")
	}
	switch q.Type {
	case "", Any, Movie, TV:
	default:
		return fmt.Errorf("invalid type %q: must be movie, tv or any", q.Type)
	}
	switch q.Sort {
	case "", "seeders", "size", "date":
	default:
		return fmt.Errorf("invalid sort %q: must be seeders, size or date", q.Sort)
	}
	return nil
}

// SortReleases sorts releases in place by the given key, descending, stably.
// An empty or unknown sort key means seeders. Ties break by provider then title.
func SortReleases(releases []Release, sortBy string) {
	less := func(a, b Release) bool {
		switch sortBy {
		case "size":
			if a.Size != b.Size {
				return a.Size > b.Size
			}
		case "date":
			// Providers order by date internally; preserve that order on ties.
		default: // "seeders"
			if a.Seeders != b.Seeders {
				return a.Seeders > b.Seeders
			}
		}
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		return a.Title < b.Title
	}
	sort.SliceStable(releases, func(i, j int) bool { return less(releases[i], releases[j]) })
}

// DedupeReleases keeps the highest-seeded entry for each info hash. Entries with
// an empty hash are always kept. The input order of distinct hashes is preserved.
func DedupeReleases(releases []Release) []Release {
	out := make([]Release, 0, len(releases))
	index := make(map[string]int, len(releases))
	for _, r := range releases {
		if r.InfoHash == "" {
			out = append(out, r)
			continue
		}
		if i, ok := index[r.InfoHash]; ok {
			if r.Seeders > out[i].Seeders {
				out[i] = r
			}
			continue
		}
		index[r.InfoHash] = len(out)
		out = append(out, r)
	}
	return out
}
