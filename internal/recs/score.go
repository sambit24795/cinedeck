package recs

import (
	"math"
	"sort"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

const (
	WeightGenre    = 0.35 // weights of the four cosine terms; they sum to 1
	WeightKeyword  = 0.35
	WeightDirector = 0.15
	WeightCast     = 0.15

	BoostSource     = 0.10 // for candidates that TMDB recommends for seeds
	BoostRating     = 0.05
	BoostPopularity = 0.05

	RatingMinVotes       = 50    // vote_average counts only with at least this many votes
	PopularitySaturation = 100.0 // popularity at which the popularity boost is full
	SourceSaturation     = 2.0   // sum of seed weights at which the source boost is full
)

// Candidate is a title that might be recommended. Feature lists need not be sorted or unique.
type Candidate struct {
	ID           catalog.ID
	Title        string
	Year         int
	Genres       []string // canonical genre keys
	Keywords     []int
	Cast         []int
	Directors    []int
	Rating       float64 // TMDB vote_average
	Votes        int
	Popularity   float64
	SourceWeight float64      // sum of the weights of the seeds whose /recommendations list contains it
	RecBy        []catalog.ID // those seeds
	Enriched     bool         // Keywords, Cast and Directors were filled from the title's own details
}

// Scored is a Candidate with its score.
type Scored struct {
	Candidate
	Score float64
}

// Score computes the taste-match score of c against p.
func Score(p Profile, c Candidate) float64 {
	cosGenre := cosineString(p.Genres, c.Genres)
	cosKeyword := cosineInt(p.Keywords, c.Keywords)
	cosDirector := cosineInt(p.Directors, c.Directors)
	cosCast := cosineInt(p.Cast, c.Cast)

	overlap := WeightGenre*cosGenre + WeightKeyword*cosKeyword + WeightDirector*cosDirector + WeightCast*cosCast

	source := BoostSource * math.Min(1, c.SourceWeight/SourceSaturation)

	rating := 0.0
	if c.Votes >= RatingMinVotes {
		rating = BoostRating * clamp(c.Rating/10, 0, 1)
	}

	popularity := BoostPopularity * math.Min(1, math.Log10(1+math.Max(c.Popularity, 0))/math.Log10(1+PopularitySaturation))

	return round6(overlap + source + rating + popularity)
}

// Rank computes Score for each candidate and returns a new slice sorted by
// score, votes, type and id, in that order. The input is not modified.
func Rank(p Profile, cs []Candidate) []Scored {
	out := make([]Scored, len(cs))
	for i, c := range cs {
		out[i] = Scored{Candidate: c, Score: Score(p, c)}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		if out[i].Votes != out[j].Votes {
			return out[i].Votes > out[j].Votes
		}
		if out[i].ID.Type != out[j].ID.Type {
			return out[i].ID.Type < out[j].ID.Type
		}
		return out[i].ID.N < out[j].ID.N
	})

	return out
}

// cosineString is the cosine between the string-keyed profile vector p and the
// distinct values of feats as a 0/1 vector.
func cosineString(p map[string]float64, feats []string) float64 {
	if len(p) == 0 {
		return 0
	}
	keys := distinctSortedStrings(feats)
	if len(keys) == 0 {
		return 0
	}

	norm := 0.0
	for _, k := range sortedStringKeys(p) {
		v := p[k]
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return 0
	}

	sum := 0.0
	for _, k := range keys {
		sum += p[k]
	}
	return sum / (norm * math.Sqrt(float64(len(keys))))
}

// cosineInt is the cosine between the int-keyed profile vector p and the
// distinct values of feats as a 0/1 vector.
func cosineInt(p map[int]float64, feats []int) float64 {
	if len(p) == 0 {
		return 0
	}
	keys := distinctSortedInts(feats)
	if len(keys) == 0 {
		return 0
	}

	norm := 0.0
	for _, k := range sortedIntKeys(p) {
		v := p[k]
		norm += v * v
	}
	norm = math.Sqrt(norm)
	if norm == 0 {
		return 0
	}

	sum := 0.0
	for _, k := range keys {
		sum += p[k]
	}
	return sum / (norm * math.Sqrt(float64(len(keys))))
}

func distinctSortedStrings(v []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, s := range v {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	sort.Strings(out)
	return out
}

func distinctSortedInts(v []int) []int {
	seen := map[int]bool{}
	out := []int{}
	for _, n := range v {
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	sort.Ints(out)
	return out
}

func sortedStringKeys(m map[string]float64) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedIntKeys(m map[int]float64) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func round6(x float64) float64 {
	return math.Round(x*1e6) / 1e6
}
