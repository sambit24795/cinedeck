package recs

import (
	"math"
	"reflect"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func testProfile() Profile {
	return Profile{
		Genres:    map[string]float64{"action": 1.8, "scifi": 1, "adventure": 0.8, "drama": 0.8},
		Keywords:  map[int]float64{1: 1, 2: 1.8, 3: 0.8},
		Directors: map[int]float64{10: 1, 11: 0.8},
		Cast:      map[int]float64{100: 1, 101: 1.8},
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) <= 1e-9 }

func TestScoreX(t *testing.T) {
	p := testProfile()
	x := Candidate{
		ID:           movieID(1),
		Genres:       []string{"action", "scifi"},
		Keywords:     []int{2, 3},
		Directors:    []int{10},
		Cast:         []int{101},
		Rating:       8.0,
		Votes:        1000,
		Popularity:   100,
		SourceWeight: 1.0,
	}

	if got := Score(p, x); !approx(got, 0.974483) {
		t.Errorf("Score = %v, want 0.974483", got)
	}

	x.SourceWeight = 0
	if got := Score(p, x); !approx(got, 0.924483) {
		t.Errorf("Score(SourceWeight 0) = %v, want 0.924483", got)
	}

	x.SourceWeight = 2
	if got := Score(p, x); !approx(got, 1.024483) {
		t.Errorf("Score(SourceWeight 2) = %v, want 1.024483", got)
	}

	x.SourceWeight = 5
	if got := Score(p, x); !approx(got, 1.024483) {
		t.Errorf("Score(SourceWeight 5) = %v, want 1.024483", got)
	}
}

func TestScoreDuplicatesAndOrder(t *testing.T) {
	p := testProfile()
	x := Candidate{
		ID:           movieID(1),
		Genres:       []string{"action", "scifi"},
		Keywords:     []int{2, 3},
		Directors:    []int{10},
		Cast:         []int{101},
		Rating:       8.0,
		Votes:        1000,
		Popularity:   100,
		SourceWeight: 1.0,
	}
	if got := Score(p, x); !approx(got, 0.974483) {
		t.Fatalf("base Score = %v, want 0.974483", got)
	}

	x.Keywords = []int{3, 2, 3}
	if got := Score(p, x); !approx(got, 0.974483) {
		t.Errorf("Score(duplicate keywords) = %v, want 0.974483", got)
	}
	x.Genres = []string{"scifi", "action"}
	x.Keywords = []int{3, 2}
	if got := Score(p, x); !approx(got, 0.974483) {
		t.Errorf("Score(reordered features) = %v, want 0.974483", got)
	}
}

func TestScoreY(t *testing.T) {
	y := Candidate{Rating: 9, Votes: 10, Popularity: 0, SourceWeight: 0}
	if got := Score(testProfile(), y); got != 0 {
		t.Errorf("Score(Y) = %v, want 0", got)
	}
}

func TestScoreZ(t *testing.T) {
	p := testProfile()
	z := Candidate{
		Genres:       []string{"drama"},
		Rating:       6.0,
		Votes:        50,
		Popularity:   9,
		SourceWeight: 0,
	}
	if got := Score(p, z); !approx(got, 0.174122) {
		t.Errorf("Score(Z) = %v, want 0.174122", got)
	}

	z.Votes = 49
	if got := Score(p, z); !approx(got, 0.144122) {
		t.Errorf("Score(Z votes 49) = %v, want 0.144122", got)
	}

	z.Votes = 50
	z.Popularity = 0
	if got := Score(p, z); !approx(got, 0.149176) {
		t.Errorf("Score(Z popularity 0) = %v, want 0.149176", got)
	}

	z.Popularity = 100000
	if got := Score(p, z); !approx(got, 0.199176) {
		t.Errorf("Score(Z popularity 100000) = %v, want 0.199176", got)
	}
}

func TestScoreW(t *testing.T) {
	w := Candidate{Genres: []string{"horror"}, Keywords: []int{99}}
	if got := Score(testProfile(), w); got != 0 {
		t.Errorf("Score(W) = %v, want 0", got)
	}
}

func TestScoreEmptyProfile(t *testing.T) {
	x := Candidate{
		ID:           movieID(1),
		Genres:       []string{"action", "scifi"},
		Keywords:     []int{2, 3},
		Directors:    []int{10},
		Cast:         []int{101},
		Rating:       8.0,
		Votes:        1000,
		Popularity:   100,
		SourceWeight: 1.0,
	}

	if got := Score(Profile{}, x); !approx(got, 0.14) {
		t.Errorf("Score(Profile{}) = %v, want 0.14", got)
	}
	empty := Profile{
		Genres:    map[string]float64{},
		Keywords:  map[int]float64{},
		Cast:      map[int]float64{},
		Directors: map[int]float64{},
	}
	if got := Score(empty, x); !approx(got, 0.14) {
		t.Errorf("Score(empty maps) = %v, want 0.14", got)
	}
}

func scoredIDs(scored []Scored) []catalog.ID {
	ids := make([]catalog.ID, len(scored))
	for i, s := range scored {
		ids[i] = s.ID
	}
	return ids
}

func TestRank(t *testing.T) {
	cs := []Candidate{
		{ID: movieID(5), Votes: 10},
		{ID: movieID(3), Votes: 10},
		{ID: tvID(1), Votes: 20},
		{ID: movieID(9), Votes: 20},
	}
	original := append([]Candidate(nil), cs...)

	got := Rank(Profile{}, cs)
	want := []catalog.ID{movieID(9), tvID(1), movieID(3), movieID(5)}
	if !reflect.DeepEqual(scoredIDs(got), want) {
		t.Errorf("order = %v, want %v", scoredIDs(got), want)
	}
	for _, s := range got {
		if s.Score != 0 {
			t.Errorf("Score = %v, want 0", s.Score)
		}
	}
	if !reflect.DeepEqual(cs, original) {
		t.Errorf("input modified: %v, want %v", cs, original)
	}

	rev := make([]Candidate, len(cs))
	for i := range cs {
		rev[i] = cs[len(cs)-1-i]
	}
	if got2 := Rank(Profile{}, rev); !reflect.DeepEqual(scoredIDs(got2), want) {
		t.Errorf("order after reversing input = %v, want %v", scoredIDs(got2), want)
	}
}

func TestRankHigherScoreFirst(t *testing.T) {
	high := Candidate{
		ID:       movieID(1),
		Genres:   []string{"action"},
		Votes:    10,
		Enriched: false,
		Title:    "high",
	}
	low := Candidate{ID: movieID(2), Title: "low"}
	got := Rank(testProfile(), []Candidate{low, high})
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != high.ID {
		t.Errorf("got[0].ID = %v, want %v", got[0].ID, high.ID)
	}
	if !(got[0].Score > got[1].Score) {
		t.Errorf("scores = %v, %v; want first > second", got[0].Score, got[1].Score)
	}
}
