package recs

import (
	"sort"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// genre is one canonical genre: its lower-case key, its display name and the
// TMDB genre ids for movies and TV. A zero id means the genre does not exist
// for that type.
type genre struct {
	key   string
	name  string
	movie int
	tv    int
}

// genreTable is the canonical genre mapping. The movie and TV columns hold the
// TMDB genre ids, which differ for many genres.
var genreTable = []genre{
	{"action", "Action", 28, 10759},
	{"adventure", "Adventure", 12, 10759},
	{"animation", "Animation", 16, 16},
	{"comedy", "Comedy", 35, 35},
	{"crime", "Crime", 80, 80},
	{"documentary", "Documentary", 99, 99},
	{"drama", "Drama", 18, 18},
	{"family", "Family", 10751, 10751},
	{"fantasy", "Fantasy", 14, 10765},
	{"history", "History", 36, 0},
	{"horror", "Horror", 27, 0},
	{"music", "Music", 10402, 0},
	{"mystery", "Mystery", 9648, 9648},
	{"romance", "Romance", 10749, 0},
	{"scifi", "Sci-Fi", 878, 10765},
	{"thriller", "Thriller", 53, 0},
	{"war", "War", 10752, 10768},
	{"western", "Western", 37, 37},
	{"kids", "Kids", 0, 10762},
	{"news", "News", 0, 10763},
	{"reality", "Reality", 0, 10764},
	{"soap", "Soap", 0, 10766},
	{"talk", "Talk", 0, 10767},
}

// genreID returns the TMDB genre id of g for t, or 0 when it does not exist.
func genreID(t catalog.MediaType, g genre) int {
	switch t {
	case catalog.Movie:
		return g.movie
	case catalog.TV:
		return g.tv
	default:
		return 0
	}
}

// CanonGenres maps TMDB genre ids of type t to canonical keys: sorted, without
// duplicates. Ids with no canonical key (for example movie "TV Movie" 10770)
// are skipped.
func CanonGenres(t catalog.MediaType, ids []int) []string {
	if t != catalog.Movie && t != catalog.TV {
		return nil
	}

	seen := map[string]bool{}
	for _, id := range ids {
		for _, g := range genreTable {
			if genreID(t, g) == id {
				seen[g.key] = true
			}
		}
	}

	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GenreIDsFor maps canonical keys to TMDB genre ids of type t, in the order of
// keys, without duplicates. Keys that do not exist for t are skipped.
func GenreIDsFor(t catalog.MediaType, keys []string) []int {
	if t != catalog.Movie && t != catalog.TV {
		return nil
	}

	ids := []int{}
	seen := map[int]bool{}
	for _, key := range keys {
		for _, g := range genreTable {
			if g.key != key {
				continue
			}
			id := genreID(t, g)
			if id != 0 && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	return ids
}

// GenreName returns the display name of a canonical key ("scifi" -> "Sci-Fi").
// An unknown key is returned unchanged.
func GenreName(key string) string {
	for _, g := range genreTable {
		if g.key == key {
			return g.name
		}
	}
	return key
}
