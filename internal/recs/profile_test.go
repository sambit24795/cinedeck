package recs

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

func at(n int) time.Time { return time.Unix(int64(n), 0).UTC() }

func movieID(n int64) catalog.ID { return catalog.ID{Type: catalog.Movie, N: n} }
func tvID(n int64) catalog.ID    { return catalog.ID{Type: catalog.TV, N: n} }

func TestClassify(t *testing.T) {
	entries := []library.Entry{
		{Meta: catalog.Meta{ID: movieID(1)}, Liked: true, LikedAt: at(5)},
		{Meta: catalog.Meta{ID: tvID(2)}, Rating: 9, RatedAt: at(7)},
		{Meta: catalog.Meta{ID: movieID(3)}, Liked: true, Rating: 4},
		{Meta: catalog.Meta{ID: movieID(4)}, Rating: 7},
		{Meta: catalog.Meta{ID: movieID(5)}, Watched: true},
		{Meta: catalog.Meta{ID: movieID(6)}, OnWatchlist: true},
		{Meta: catalog.Meta{ID: movieID(7)}, Liked: true, Rating: 8, LikedAt: at(1), RatedAt: at(9)},
	}

	seeds, excluded := Classify(entries)

	wantIDs := []catalog.ID{movieID(7), tvID(2), movieID(1)}
	if len(seeds) != len(wantIDs) {
		t.Fatalf("len(seeds) = %d, want %d", len(seeds), len(wantIDs))
	}
	for i, s := range seeds {
		if s.Meta.ID != wantIDs[i] {
			t.Errorf("seeds[%d].ID = %v, want %v", i, s.Meta.ID, wantIDs[i])
		}
	}

	wantWeights := []float64{0.8, 0.9, 1}
	for i, s := range seeds {
		if math.Abs(s.Weight-wantWeights[i]) > 1e-9 {
			t.Errorf("seeds[%d].Weight = %v, want %v", i, s.Weight, wantWeights[i])
		}
	}
	if !seeds[0].Liked || seeds[0].Rating != 8 || !seeds[0].At.Equal(at(9)) {
		t.Errorf("seeds[0] = %+v, want liked rated 8 at t9", seeds[0])
	}
	if seeds[1].Liked || seeds[1].Rating != 9 {
		t.Errorf("seeds[1] = %+v, want not liked rated 9", seeds[1])
	}

	for _, id := range []catalog.ID{movieID(1), tvID(2), movieID(3), movieID(4), movieID(5), movieID(7)} {
		if !excluded[id] {
			t.Errorf("excluded[%v] = false, want true", id)
		}
	}
	if excluded[movieID(6)] {
		t.Errorf("excluded[movie:6] = true, want false (watchlist only)")
	}
}

func TestClassifySameAtOrder(t *testing.T) {
	entries := []library.Entry{
		{Meta: catalog.Meta{ID: tvID(1)}, Liked: true, LikedAt: at(1)},
		{Meta: catalog.Meta{ID: movieID(2)}, Liked: true, LikedAt: at(1)},
		{Meta: catalog.Meta{ID: movieID(1)}, Liked: true, LikedAt: at(1)},
	}
	seeds, _ := Classify(entries)
	want := []catalog.ID{movieID(1), movieID(2), tvID(1)}
	for i, s := range seeds {
		if s.Meta.ID != want[i] {
			t.Errorf("seeds[%d].ID = %v, want %v", i, s.Meta.ID, want[i])
		}
	}
}

func TestClassifyNil(t *testing.T) {
	seeds, excluded := Classify(nil)
	if len(seeds) != 0 {
		t.Errorf("len(seeds) = %d, want 0", len(seeds))
	}
	if excluded == nil {
		t.Fatal("excluded = nil, want a non-nil map")
	}
	if len(excluded) != 0 {
		t.Errorf("len(excluded) = %d, want 0", len(excluded))
	}
}

func TestBuildProfile(t *testing.T) {
	a := Seed{
		Meta: catalog.Meta{
			ID:        movieID(1),
			GenreIDs:  []int{878, 28},
			Keywords:  []catalog.Keyword{{ID: 1}, {ID: 2}},
			Cast:      []catalog.Person{{ID: 100}, {ID: 101}},
			Directors: []catalog.Person{{ID: 10}},
		},
		Weight: 1,
	}
	b := Seed{
		Meta: catalog.Meta{
			ID:        tvID(2),
			GenreIDs:  []int{10759, 18},
			Keywords:  []catalog.Keyword{{ID: 2}, {ID: 3}},
			Cast:      []catalog.Person{{ID: 101}},
			Directors: []catalog.Person{{ID: 11}},
		},
		Weight: 0.8,
	}

	p := BuildProfile([]Seed{a, b})

	wantGenres := map[string]float64{"action": 1.8, "scifi": 1, "adventure": 0.8, "drama": 0.8}
	wantKeywords := map[int]float64{1: 1, 2: 1.8, 3: 0.8}
	wantCast := map[int]float64{100: 1, 101: 1.8}
	wantDirectors := map[int]float64{10: 1, 11: 0.8}

	if !floatMapEqual(p.Genres, wantGenres) {
		t.Errorf("Genres = %v, want %v", p.Genres, wantGenres)
	}
	if !intFloatMapEqual(p.Keywords, wantKeywords) {
		t.Errorf("Keywords = %v, want %v", p.Keywords, wantKeywords)
	}
	if !intFloatMapEqual(p.Cast, wantCast) {
		t.Errorf("Cast = %v, want %v", p.Cast, wantCast)
	}
	if !intFloatMapEqual(p.Directors, wantDirectors) {
		t.Errorf("Directors = %v, want %v", p.Directors, wantDirectors)
	}
}

func TestBuildProfileDedupesFeatures(t *testing.T) {
	s := Seed{
		Meta: catalog.Meta{
			ID:        movieID(1),
			GenreIDs:  []int{878, 878},
			Keywords:  []catalog.Keyword{{ID: 1}, {ID: 1}, {ID: 1}},
			Cast:      []catalog.Person{{ID: 100}, {ID: 100}},
			Directors: []catalog.Person{{ID: 10}, {ID: 10}},
		},
		Weight: 1,
	}
	p := BuildProfile([]Seed{s})
	if !intFloatMapEqual(p.Keywords, map[int]float64{1: 1}) {
		t.Errorf("Keywords = %v, want {1:1}", p.Keywords)
	}
	if !intFloatMapEqual(p.Cast, map[int]float64{100: 1}) {
		t.Errorf("Cast = %v, want {100:1}", p.Cast)
	}
	if !intFloatMapEqual(p.Directors, map[int]float64{10: 1}) {
		t.Errorf("Directors = %v, want {10:1}", p.Directors)
	}
	if !floatMapEqual(p.Genres, map[string]float64{"scifi": 1}) {
		t.Errorf("Genres = %v, want {scifi:1}", p.Genres)
	}
}

func TestBuildProfileNil(t *testing.T) {
	p := BuildProfile(nil)
	if p.Genres == nil || p.Keywords == nil || p.Cast == nil || p.Directors == nil {
		t.Fatalf("profile maps = %+v, want four non-nil maps", p)
	}
	if len(p.Genres)+len(p.Keywords)+len(p.Cast)+len(p.Directors) != 0 {
		t.Errorf("profile = %+v, want empty maps", p)
	}
}

func TestTopGenresAndKeywords(t *testing.T) {
	p := BuildProfile([]Seed{
		{
			Meta: catalog.Meta{
				ID:        movieID(1),
				GenreIDs:  []int{878, 28},
				Keywords:  []catalog.Keyword{{ID: 1}, {ID: 2}},
				Cast:      []catalog.Person{{ID: 100}, {ID: 101}},
				Directors: []catalog.Person{{ID: 10}},
			},
			Weight: 1,
		},
		{
			Meta: catalog.Meta{
				ID:        tvID(2),
				GenreIDs:  []int{10759, 18},
				Keywords:  []catalog.Keyword{{ID: 2}, {ID: 3}},
				Cast:      []catalog.Person{{ID: 101}},
				Directors: []catalog.Person{{ID: 11}},
			},
			Weight: 0.8,
		},
	})

	if got := topGenres(p, 3); !reflect.DeepEqual(got, []string{"action", "scifi", "adventure"}) {
		t.Errorf("topGenres(p, 3) = %v, want [action scifi adventure]", got)
	}
	if got := topGenres(p, 2); !reflect.DeepEqual(got, []string{"action", "scifi"}) {
		t.Errorf("topGenres(p, 2) = %v, want [action scifi]", got)
	}
	if got := topGenres(p, 10); len(got) != 4 {
		t.Errorf("topGenres(p, 10) = %v, want all 4", got)
	}

	if got := topKeywords(p, 2); !reflect.DeepEqual(got, []int{2, 1}) {
		t.Errorf("topKeywords(p, 2) = %v, want [2 1]", got)
	}

	equal := BuildProfile([]Seed{
		{Meta: catalog.Meta{ID: movieID(1), Keywords: []catalog.Keyword{{ID: 5}, {ID: 3}, {ID: 4}}}, Weight: 1},
	})
	if got := topKeywords(equal, 3); !reflect.DeepEqual(got, []int{3, 4, 5}) {
		t.Errorf("topKeywords(equal, 3) = %v, want [3 4 5]", got)
	}
}

func floatMapEqual(got, want map[string]float64) bool {
	if len(got) != len(want) {
		return false
	}
	for k, w := range want {
		if math.Abs(got[k]-w) > 1e-9 {
			return false
		}
	}
	return true
}

func intFloatMapEqual(got, want map[int]float64) bool {
	if len(got) != len(want) {
		return false
	}
	for k, w := range want {
		if math.Abs(got[k]-w) > 1e-9 {
			return false
		}
	}
	return true
}
