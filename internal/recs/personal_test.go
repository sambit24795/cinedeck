package recs

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

// relG builds a Related with genre ids.
func relG(id catalog.ID, title string, votes int, genreIDs ...int) catalog.Related {
	r := rel(id, title, votes)
	r.GenreIDs = genreIDs
	return r
}

// likedEntry builds a liked library entry.
func likedEntry(id catalog.ID, title string, genreIDs []int, kw ...catalog.Keyword) library.Entry {
	return library.Entry{
		Meta:    catalog.Meta{ID: id, Title: title, GenreIDs: genreIDs, Keywords: kw},
		Liked:   true,
		LikedAt: at(1),
	}
}

func TestPersonalNoTaste(t *testing.T) {
	ctx := context.Background()
	src := &fakeSource{}
	entries := []library.Entry{
		{Meta: catalog.Meta{ID: movieID(1)}, Rating: 3, RatedAt: at(1)},
		{Meta: catalog.Meta{ID: movieID(2)}, OnWatchlist: true},
	}

	res, err := Personal(ctx, src, entries, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if res.Seeds != 0 {
		t.Errorf("Seeds = %d, want 0", res.Seeds)
	}
	if res.Picks == nil || len(res.Picks) != 0 {
		t.Errorf("Picks = %#v, want a non-nil empty slice", res.Picks)
	}
	if src.calls != 0 || res.Requests != 0 {
		t.Errorf("calls/Requests = %d/%d, want 0/0", src.calls, res.Requests)
	}

	res, err = Personal(ctx, src, nil, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal(nil): %v", err)
	}
	if res.Seeds != 0 || len(res.Picks) != 0 || src.calls != 0 {
		t.Errorf("Personal(nil) = %+v, calls %d; want empty", res, src.calls)
	}
}

func TestPersonalBasicFlow(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	entries := []library.Entry{
		seed,
		{Meta: catalog.Meta{ID: movieID(11)}, Watched: true},
		{Meta: catalog.Meta{ID: movieID(13)}, Rating: 3, RatedAt: at(1)},
	}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			movieID(10): {ID: movieID(10), Title: "R1", GenreIDs: []int{878}, Keywords: []catalog.Keyword{{ID: 5, Name: "k5"}}, Votes: 900},
		},
		recs: map[catalog.ID][]catalog.Related{
			movieID(1): {
				relG(movieID(10), "R1", 900, 878),
				relG(movieID(11), "R2", 800, 18),
				relG(movieID(13), "R3", 700, 878),
				relG(movieID(1), "Seed", 500, 878),
			},
		},
		discover: func(catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{relG(movieID(12), "D1", 600, 878)}
		},
	}

	res, err := Personal(ctx, src, entries, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if res.Seeds != 1 {
		t.Errorf("Seeds = %d, want 1", res.Seeds)
	}
	wantIDs := []catalog.ID{movieID(10), movieID(12)}
	if !reflect.DeepEqual(pickIDs(res.Picks), wantIDs) {
		t.Fatalf("ids = %v, want %v", pickIDs(res.Picks), wantIDs)
	}
	if got, want := res.Picks[0].Reason, "because you liked Seed (shared: k5, Sci-Fi)"; got != want {
		t.Errorf("Picks[0].Reason = %q, want %q", got, want)
	}
	if got, want := res.Picks[1].Reason, "because you liked Seed (shared: Sci-Fi)"; got != want {
		t.Errorf("Picks[1].Reason = %q, want %q", got, want)
	}
	if !(res.Picks[0].Score > res.Picks[1].Score && res.Picks[1].Score > 0) {
		t.Errorf("scores = %v, %v; want descending and positive", res.Picks[0].Score, res.Picks[1].Score)
	}
	if res.Requests != src.calls {
		t.Errorf("Requests = %d, src.calls = %d; want equal", res.Requests, src.calls)
	}
}

type metaErrorSource struct {
	*fakeSource
	err error
}

func (s *metaErrorSource) Meta(context.Context, catalog.ID) (*catalog.Meta, error) {
	s.fakeSource.calls++
	return nil, s.err
}

func TestPersonalMetaFailureNonCancellation(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	src := &metaErrorSource{
		fakeSource: &fakeSource{
			recs: map[catalog.ID][]catalog.Related{movieID(1): {relG(movieID(10), "R1", 900, 878)}},
		},
		err: errors.New("boom"),
	}

	res, err := Personal(ctx, src, []library.Entry{seed}, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if len(res.Picks) != 1 || res.Picks[0].ID != movieID(10) {
		t.Errorf("Picks = %+v, want movie:10", res.Picks)
	}
}

type cancelSource struct {
	*fakeSource
	cancel func()
}

func (s *cancelSource) Meta(ctx context.Context, _ catalog.ID) (*catalog.Meta, error) {
	s.fakeSource.calls++
	s.cancel()
	return nil, ctx.Err()
}

func TestPersonalCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	src := &cancelSource{
		fakeSource: &fakeSource{
			recs: map[catalog.ID][]catalog.Related{movieID(1): {relG(movieID(10), "R1", 900, 878)}},
		},
		cancel: cancel,
	}

	_, err := Personal(ctx, src, []library.Entry{seed}, PersonalOptions{Limit: 10})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

type recErrorSource struct {
	*fakeSource
	errs map[catalog.ID]error
}

func (s *recErrorSource) Recommendations(ctx context.Context, id catalog.ID) ([]catalog.Related, error) {
	if err := s.errs[id]; err != nil {
		return nil, err
	}
	return s.fakeSource.Recommendations(ctx, id)
}

type discoverErrorSource struct {
	*fakeSource
	err error
}

func (s *discoverErrorSource) DiscoverBy(context.Context, catalog.DiscoverQuery) ([]catalog.Related, error) {
	s.fakeSource.calls++
	return nil, s.err
}

func TestPersonalSkippedAndFailingLists(t *testing.T) {
	ctx := context.Background()
	seed1 := likedEntry(movieID(1), "S1", []int{878}, catalog.Keyword{ID: 1})
	seed2 := likedEntry(movieID(2), "S2", []int{878}, catalog.Keyword{ID: 2})
	fake := &fakeSource{
		recs: map[catalog.ID][]catalog.Related{
			movieID(2): {relG(movieID(3), "R3", 500, 878)},
		},
	}

	skip := &recErrorSource{
		fakeSource: fake,
		errs:       map[catalog.ID]error{movieID(1): &catalog.APIError{Status: 404, Message: "not found"}},
	}
	res, err := Personal(ctx, skip, []library.Entry{seed1, seed2}, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal (404 skip): %v", err)
	}
	if ids := pickIDs(res.Picks); !reflect.DeepEqual(ids, []catalog.ID{movieID(3)}) {
		t.Errorf("ids = %v, want [movie:3]", ids)
	}

	fail := &recErrorSource{
		fakeSource: fake,
		errs:       map[catalog.ID]error{movieID(1): errors.New("boom")},
	}
	if _, err := Personal(ctx, fail, []library.Entry{seed1, seed2}, PersonalOptions{Limit: 10}); err == nil {
		t.Error("Personal (recs error) = nil, want error")
	}

	disc := &discoverErrorSource{
		fakeSource: fake,
		err:        errors.New("disc boom"),
	}
	if _, err := Personal(ctx, disc, []library.Entry{seed1}, PersonalOptions{Limit: 10}); err == nil {
		t.Error("Personal (discover error) = nil, want error")
	}
}

// recCountSource counts Recommendations calls on top of fakeSource.
type recCountSource struct {
	*fakeSource
	recCalls []catalog.ID
}

func (s *recCountSource) Recommendations(ctx context.Context, id catalog.ID) ([]catalog.Related, error) {
	s.recCalls = append(s.recCalls, id)
	return s.fakeSource.Recommendations(ctx, id)
}

func TestPersonalTypeFilter(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	fake := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			tvID(50): {ID: tvID(50), Title: "T50", GenreIDs: []int{10765}, Votes: 900},
		},
		recs: map[catalog.ID][]catalog.Related{
			movieID(1): {relG(movieID(10), "R1", 900, 878)},
		},
		discover: func(catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{
				relG(tvID(50), "T50", 900, 10765),
				relG(movieID(12), "D1", 600, 878),
			}
		},
	}
	src := &recCountSource{fakeSource: fake}

	res, err := Personal(ctx, src, []library.Entry{seed}, PersonalOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if len(src.recCalls) != 0 {
		t.Errorf("Recommendations calls = %v, want none for a movie seed with --type tv", src.recCalls)
	}
	if len(src.queries) != 2 {
		t.Fatalf("queries = %v, want 2", src.queries)
	}
	for _, q := range src.queries {
		if q.Type != catalog.TV {
			t.Errorf("query.Type = %q, want tv", q.Type)
		}
	}
	for _, p := range res.Picks {
		if p.ID.Type != catalog.TV {
			t.Errorf("pick = %v, want a tv id", p.ID)
		}
	}
	if res.Requests != src.calls {
		t.Errorf("Requests = %d, src.calls = %d", res.Requests, src.calls)
	}

	src2 := &fakeSource{
		recs: map[catalog.ID][]catalog.Related{movieID(1): {relG(movieID(10), "R1", 900, 878)}},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return nil
		},
	}
	if _, err := Personal(ctx, src2, []library.Entry{seed}, PersonalOptions{Limit: 10}); err != nil {
		t.Fatalf("Personal both: %v", err)
	}
	seenMovie, seenTV := false, false
	for _, q := range src2.queries {
		if q.Type == catalog.Movie {
			seenMovie = true
		}
		if q.Type == catalog.TV {
			seenTV = true
		}
	}
	if !seenMovie || !seenTV {
		t.Errorf("queries = %v, want both movie and tv", src2.queries)
	}
}

func TestPersonalQueryShapes(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{28, 878, 12},
		catalog.Keyword{ID: 1}, catalog.Keyword{ID: 2}, catalog.Keyword{ID: 3},
		catalog.Keyword{ID: 4}, catalog.Keyword{ID: 5}, catalog.Keyword{ID: 6},
		catalog.Keyword{ID: 7})
	src := &fakeSource{
		discover: func(catalog.DiscoverQuery) []catalog.Related { return nil },
	}

	if _, err := Personal(ctx, src, []library.Entry{seed}, PersonalOptions{Limit: 10}); err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if len(src.queries) != 4 {
		t.Fatalf("queries = %v, want 4", src.queries)
	}
	want := []catalog.DiscoverQuery{
		{Type: catalog.Movie, GenreIDs: []int{28, 12, 878}, KeywordIDs: []int{1, 2, 3, 4, 5}, MinVotes: 100},
		{Type: catalog.Movie, GenreIDs: []int{28, 12}, GenresAll: true, MinVotes: 500},
		{Type: catalog.TV, GenreIDs: []int{10759, 10765}, KeywordIDs: []int{1, 2, 3, 4, 5}, MinVotes: 100},
		{Type: catalog.TV, GenreIDs: []int{10759, 10765}, GenresAll: true, MinVotes: 500},
	}
	for i := range want {
		if !reflect.DeepEqual(src.queries[i], want[i]) {
			t.Errorf("query %d = %+v, want %+v", i, src.queries[i], want[i])
		}
	}
}

func TestPersonalRequestBound(t *testing.T) {
	ctx := context.Background()

	entries := make([]library.Entry, 0, 10)
	recs := map[catalog.ID][]catalog.Related{}
	metas := map[catalog.ID]*catalog.Meta{}
	for i := 0; i < 30; i++ {
		id := catalog.ID{Type: catalog.Movie, N: int64(1001 + i)}
		metas[id] = &catalog.Meta{ID: id, Title: "R", GenreIDs: []int{878}, Votes: 900}
	}
	for i := 0; i < 30; i++ {
		id := catalog.ID{Type: catalog.Movie, N: int64(2001 + i)}
		metas[id] = &catalog.Meta{ID: id, Title: "D", GenreIDs: []int{878}, Votes: 900}
	}
	for i := 0; i < 10; i++ {
		id := catalog.ID{Type: catalog.Movie, N: int64(101 + i)}
		entries = append(entries, likedEntry(id, "S", []int{878}, catalog.Keyword{ID: 100 + i}))
		if i < 8 {
			list := make([]catalog.Related, 0, 30)
			for j := 0; j < 30; j++ {
				list = append(list, relG(catalog.ID{Type: catalog.Movie, N: int64(1001 + j)}, "R", 900, 878))
			}
			recs[id] = list
		}
	}

	src := &fakeSource{
		metas: metas,
		recs:  recs,
		discover: func(catalog.DiscoverQuery) []catalog.Related {
			list := make([]catalog.Related, 0, 30)
			for j := 0; j < 30; j++ {
				list = append(list, relG(catalog.ID{Type: catalog.Movie, N: int64(2001 + j)}, "D", 900, 878))
			}
			return list
		},
	}

	res, err := Personal(ctx, src, entries, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if res.Requests != src.calls {
		t.Errorf("Requests = %d, src.calls = %d; want equal", res.Requests, src.calls)
	}
	if src.calls != 24 {
		t.Errorf("src.calls = %d, want 24", src.calls)
	}
	if src.calls > MaxRequests {
		t.Errorf("src.calls = %d, want <= MaxRequests %d", src.calls, MaxRequests)
	}
	if len(res.Picks) != 10 {
		t.Errorf("len(Picks) = %d, want 10", len(res.Picks))
	}
	if res.Seeds != 10 {
		t.Errorf("Seeds = %d, want 10", res.Seeds)
	}
}

func TestPersonalDeterminism(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	newSource := func() *fakeSource {
		return &fakeSource{
			metas: map[catalog.ID]*catalog.Meta{
				movieID(10): {ID: movieID(10), Title: "R1", GenreIDs: []int{878}, Keywords: []catalog.Keyword{{ID: 5, Name: "k5"}}, Votes: 900},
				movieID(12): {ID: movieID(12), Title: "D1", GenreIDs: []int{878}, Votes: 600},
				movieID(13): {ID: movieID(13), Title: "D2", GenreIDs: []int{878}, Votes: 500},
			},
			recs: map[catalog.ID][]catalog.Related{
				movieID(1): {
					relG(movieID(10), "R1", 900, 878),
					relG(movieID(11), "R2", 800, 18),
				},
			},
			discover: func(catalog.DiscoverQuery) []catalog.Related {
				return []catalog.Related{
					relG(movieID(12), "D1", 600, 878),
					relG(movieID(13), "D2", 500, 878),
				}
			},
		}
	}

	res1, err := Personal(ctx, newSource(), []library.Entry{seed}, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal 1: %v", err)
	}
	res2, err := Personal(ctx, newSource(), []library.Entry{seed}, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal 2: %v", err)
	}
	if !reflect.DeepEqual(res1.Picks, res2.Picks) {
		t.Errorf("picks differ:\n%+v\n%+v", res1.Picks, res2.Picks)
	}

	res3, err := Personal(ctx, newSource(), []library.Entry{seed}, PersonalOptions{Limit: 3})
	if err != nil {
		t.Fatalf("Personal 3: %v", err)
	}
	if len(res3.Picks) != 3 {
		t.Fatalf("len(Picks) = %d, want 3", len(res3.Picks))
	}
	for i := 1; i < len(res3.Picks); i++ {
		if res3.Picks[i-1].Score < res3.Picks[i].Score {
			t.Errorf("picks not sorted: %v then %v", res3.Picks[i-1].Score, res3.Picks[i].Score)
		}
	}
}
