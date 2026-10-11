package recs

import (
	"context"
	"reflect"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

// mkScored builds a hand-made Scored value for the quota tests.
func mkScored(id catalog.ID, score float64, recBy ...catalog.ID) Scored {
	return Scored{Candidate: Candidate{ID: id, RecBy: recBy}, Score: score}
}

func quotaSeeds(ids ...catalog.ID) []Seed {
	seeds := make([]Seed, len(ids))
	for i, id := range ids {
		seeds[i] = Seed{Meta: catalog.Meta{ID: id}}
	}
	return seeds
}

func TestWithSeedQuotaReserveAndFill(t *testing.T) {
	a, b, c, d, e := movieID(10), movieID(11), movieID(12), movieID(13), movieID(14)
	s1, s2 := movieID(1), movieID(2)
	ranked := []Scored{
		mkScored(a, 0.9, s1),
		mkScored(b, 0.8, s1),
		mkScored(c, 0.7, s1),
		mkScored(d, 0.6),
		mkScored(e, 0.5, s2),
	}
	seeds := quotaSeeds(s1, s2)

	if got, want := scoredIDs(withSeedQuota(ranked, seeds, 3)), []catalog.ID{a, b, e}; !reflect.DeepEqual(got, want) {
		t.Errorf("n=3 ids = %v, want %v", got, want)
	}
	if got, want := scoredIDs(withSeedQuota(ranked, seeds, 5)), []catalog.ID{a, b, c, d, e}; !reflect.DeepEqual(got, want) {
		t.Errorf("n=5 ids = %v, want %v", got, want)
	}
	if got, want := scoredIDs(withSeedQuota(ranked, seeds, 1)), []catalog.ID{a}; !reflect.DeepEqual(got, want) {
		t.Errorf("n=1 ids = %v, want %v", got, want)
	}
	if got := withSeedQuota(ranked, seeds, 0); len(got) != 0 {
		t.Errorf("n=0 = %v, want empty", scoredIDs(got))
	}
}

func TestWithSeedQuotaSeedWithoutCandidate(t *testing.T) {
	a, b := movieID(10), movieID(11)
	s1, s3 := movieID(1), movieID(3)
	ranked := []Scored{mkScored(a, 0.9, s1), mkScored(b, 0.8)}

	got := withSeedQuota(ranked, quotaSeeds(s3, s1), 2)
	want := []catalog.ID{a, b}
	if !reflect.DeepEqual(scoredIDs(got), want) {
		t.Errorf("ids = %v, want %v", scoredIDs(got), want)
	}
}

func TestWithSeedQuotaSharedCandidate(t *testing.T) {
	a, b, c := movieID(10), movieID(11), movieID(12)
	s1, s2 := movieID(1), movieID(2)
	ranked := []Scored{
		mkScored(a, 0.9, s1, s2),
		mkScored(b, 0.8, s2),
		mkScored(c, 0.7),
	}

	got := withSeedQuota(ranked, quotaSeeds(s1, s2), 2)
	want := []catalog.ID{a, b}
	if !reflect.DeepEqual(scoredIDs(got), want) {
		t.Errorf("ids = %v, want %v", scoredIDs(got), want)
	}
}

func TestWithSeedQuotaShortAndNoMutation(t *testing.T) {
	a, b := movieID(10), movieID(11)
	s1 := movieID(1)
	ranked := []Scored{mkScored(a, 0.9, s1), mkScored(b, 0.8)}
	original := append([]Scored(nil), ranked...)

	got := withSeedQuota(ranked, quotaSeeds(s1), 5)
	want := []catalog.ID{a, b}
	if !reflect.DeepEqual(scoredIDs(got), want) {
		t.Errorf("ids = %v, want %v", scoredIDs(got), want)
	}
	if !reflect.DeepEqual(ranked, original) {
		t.Errorf("ranked = %v, want it unchanged %v", ranked, original)
	}
}

// TestPersonalSeedQuota is the end-to-end case: five movie seeds that share a
// genre would fill every slot by rank, so the newest TV seed must get a slot
// through the per-seed quota.
func TestPersonalSeedQuota(t *testing.T) {
	ctx := context.Background()

	tvSeed := tvID(1)
	movieSeeds := []catalog.ID{movieID(1), movieID(2), movieID(3), movieID(4), movieID(5)}
	movieCands := []catalog.ID{
		movieID(101), movieID(102), movieID(103), movieID(104), movieID(105),
		movieID(106), movieID(107), movieID(108), movieID(109), movieID(110),
	}
	tvCands := []catalog.ID{tvID(101), tvID(102)}

	// The TV seed is the newest, so it is first in the seed order.
	entries := []library.Entry{
		{Meta: catalog.Meta{ID: tvSeed, Title: "T", GenreIDs: []int{18}}, Liked: true, LikedAt: at(2)},
	}
	for _, id := range movieSeeds {
		entries = append(entries, likedEntry(id, "M", []int{878}))
	}

	recs := map[catalog.ID][]catalog.Related{}
	for i, sid := range movieSeeds {
		recs[sid] = []catalog.Related{
			relG(movieCands[2*i], "M", 1000, 878),
			relG(movieCands[2*i+1], "M", 1000, 878),
		}
	}
	recs[tvSeed] = []catalog.Related{
		relG(tvCands[0], "T", 1000, 18),
		relG(tvCands[1], "T", 1000, 18),
	}
	newSource := func() *fakeSource {
		return &fakeSource{
			recs:     recs,
			discover: func(catalog.DiscoverQuery) []catalog.Related { return nil },
		}
	}

	res, err := Personal(ctx, newSource(), entries, PersonalOptions{Limit: 5})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if len(res.Picks) != 5 {
		t.Fatalf("len(Picks) = %d, want 5", len(res.Picks))
	}
	hasTV := false
	for _, p := range res.Picks {
		if p.ID.Type == catalog.TV {
			hasTV = true
		}
	}
	if !hasTV {
		t.Errorf("picks = %v, want at least one TV pick from the TV seed", pickIDs(res.Picks))
	}

	// Without the quota the picks are in pure rank order. Build the same
	// candidate list by hand and call Rank: the top five are all movies, which
	// is the behaviour the quota fixes.
	seeds, _ := Classify(entries)
	p := BuildProfile(seeds)
	candidates := make([]Candidate, 0, len(movieCands)+len(tvCands))
	for i, sid := range movieSeeds {
		for j := 0; j < 2; j++ {
			candidates = append(candidates, Candidate{
				ID:           movieCands[2*i+j],
				Genres:       CanonGenres(catalog.Movie, []int{878}),
				Rating:       7.5,
				Votes:        1000,
				SourceWeight: 1,
				RecBy:        []catalog.ID{sid},
			})
		}
	}
	for _, id := range tvCands {
		candidates = append(candidates, Candidate{
			ID:           id,
			Genres:       CanonGenres(catalog.TV, []int{18}),
			Rating:       7.5,
			Votes:        1000,
			SourceWeight: 1,
			RecBy:        []catalog.ID{tvSeed},
		})
	}
	ranked := Rank(p, candidates)
	for i := 0; i < 5; i++ {
		if ranked[i].ID.Type != catalog.Movie {
			t.Errorf("raw rank[%d] = %v, want a movie", i, ranked[i].ID)
		}
	}
}
