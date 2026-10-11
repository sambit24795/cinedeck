package recs

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func meta(id catalog.ID, title string, genreIDs []int, keywordIDs ...int) *catalog.Meta {
	m := &catalog.Meta{ID: id, Title: title, GenreIDs: genreIDs}
	for _, k := range keywordIDs {
		m.Keywords = append(m.Keywords, catalog.Keyword{ID: k})
	}
	return m
}

func rel(id catalog.ID, title string, votes int) catalog.Related {
	return catalog.Related{
		Result: catalog.Result{ID: id, Title: title, Year: 2000, Rating: 7.5, Overview: "overview"},
		Votes:  votes,
	}
}

// namedMeta builds a Meta whose keywords carry display names, for reason tests.
func namedMeta(id catalog.ID, title string, genreIDs []int, kws ...catalog.Keyword) *catalog.Meta {
	m := &catalog.Meta{ID: id, Title: title, GenreIDs: genreIDs}
	m.Keywords = append(m.Keywords, kws...)
	return m
}

func kw(id int, name string) catalog.Keyword { return catalog.Keyword{ID: id, Name: name} }

func pickIDs(picks []Pick) []catalog.ID {
	ids := make([]catalog.ID, len(picks))
	for i, p := range picks {
		ids[i] = p.ID
	}
	return ids
}

func TestCheckLimit(t *testing.T) {
	if err := CheckLimit(1); err != nil {
		t.Errorf("CheckLimit(1) = %v, want nil", err)
	}
	if err := CheckLimit(50); err != nil {
		t.Errorf("CheckLimit(50) = %v, want nil", err)
	}
	for _, n := range []int{0, 51, -1} {
		err := CheckLimit(n)
		if err == nil || !strings.Contains(err.Error(), "invalid --limit") || !strings.Contains(err.Error(), "want 1 to 50") {
			t.Errorf("CheckLimit(%d) = %v, want invalid --limit ... want 1 to 50", n, err)
		}
	}
}

func TestParseTypeFilter(t *testing.T) {
	cases := []struct {
		in   string
		want catalog.MediaType
	}{
		{"", ""},
		{"any", ""},
		{" ANY ", ""},
		{"movie", catalog.Movie},
		{" TV ", catalog.TV},
	}
	for _, tc := range cases {
		got, err := ParseTypeFilter(tc.in)
		if err != nil {
			t.Errorf("ParseTypeFilter(%q) error: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("ParseTypeFilter(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}

	_, err := ParseTypeFilter("show")
	if err == nil || !strings.Contains(err.Error(), `invalid --type "show"`) {
		t.Errorf("ParseTypeFilter(show) = %v, want invalid --type \"show\"", err)
	}
}

func TestMergeSame(t *testing.T) {
	self := catalog.ID{Type: catalog.Movie, N: 1}
	recs := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 2}, "A", 5000),
		rel(catalog.ID{Type: catalog.Movie, N: 1}, "self", 9000),
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "B", 0),
	}
	sims := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "B", 800),
		rel(catalog.ID{Type: catalog.Movie, N: 4}, "C", 99),
		rel(catalog.ID{Type: catalog.Movie, N: 5}, "D", 100),
		rel(catalog.ID{Type: catalog.Movie, N: 6}, "E", 5000),
	}

	got := mergeSame(self, "X", recs, sims, 10)
	wantIDs := []catalog.ID{
		{Type: catalog.Movie, N: 2},
		{Type: catalog.Movie, N: 3},
		{Type: catalog.Movie, N: 5},
		{Type: catalog.Movie, N: 6},
	}
	if !reflect.DeepEqual(pickIDs(got), wantIDs) {
		t.Fatalf("ids = %v, want %v", pickIDs(got), wantIDs)
	}
	wantReasons := []string{
		"TMDB recommends it for X",
		"TMDB recommends it for X",
		"similar to X",
		"similar to X",
	}
	gotReasons := make([]string, len(got))
	for i, p := range got {
		gotReasons[i] = p.Reason
	}
	if !reflect.DeepEqual(gotReasons, wantReasons) {
		t.Errorf("reasons = %v, want %v", gotReasons, wantReasons)
	}

	if ids := pickIDs(mergeSame(self, "X", recs, sims, 3)); !reflect.DeepEqual(ids, wantIDs[:3]) {
		t.Errorf("limit 3 ids = %v, want %v", ids, wantIDs[:3])
	}
	if ids := pickIDs(mergeSame(self, "X", recs, sims, 1)); !reflect.DeepEqual(ids, wantIDs[:1]) {
		t.Errorf("limit 1 ids = %v, want %v", ids, wantIDs[:1])
	}
}

func TestMergeSameCopiesFields(t *testing.T) {
	self := catalog.ID{Type: catalog.Movie, N: 1}
	recs := []catalog.Related{rel(catalog.ID{Type: catalog.Movie, N: 2}, "A", 10)}
	got := mergeSame(self, "X", recs, nil, 10)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if got[0].Title != "A" || got[0].Year != 2000 || got[0].Rating != 7.5 {
		t.Errorf("pick = %+v, want title A year 2000 rating 7.5", got[0])
	}
}

func TestMergeSameEmpty(t *testing.T) {
	got := mergeSame(catalog.ID{Type: catalog.Movie, N: 1}, "X", nil, nil, 10)
	if got == nil {
		t.Fatal("mergeSame = nil, want a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestFromSameTypeOnly(t *testing.T) {
	ctx := context.Background()
	m := meta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, 11, 12, 13, 14, 15, 16, 17)
	recs := []catalog.Related{rel(catalog.ID{Type: catalog.Movie, N: 2}, "R2", 5000)}
	sims := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 4}, "S4", 50),
		rel(catalog.ID{Type: catalog.Movie, N: 5}, "S5", 150),
	}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{m.ID: m},
		recs:  map[catalog.ID][]catalog.Related{m.ID: recs},
		sims:  map[catalog.ID][]catalog.Related{m.ID: sims},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.Movie, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if src.calls != 3 {
		t.Errorf("calls = %d, want 3", src.calls)
	}
	if len(src.queries) != 0 {
		t.Errorf("queries = %v, want none", src.queries)
	}
	if res.Source != m {
		t.Errorf("Source = %p, want %p", res.Source, m)
	}
	want := mergeSame(m.ID, m.Title, recs, sims, 10)
	if !reflect.DeepEqual(res.Picks, want) {
		t.Errorf("Picks = %+v, want %+v", res.Picks, want)
	}
}

func TestFromCrossOnly(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12},
		kw(11, "one"), kw(12, "two"), kw(13, "three"), kw(14, "four"), kw(15, "five"), kw(16, "six"), kw(17, "seven"))
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			m.ID:     m,
			tvID(50): namedMeta(tvID(50), "T1", []int{10765}, kw(11, "one"), kw(12, "two")),
			tvID(51): namedMeta(tvID(51), "T2", []int{10759, 10765}, kw(11, "one")),
			tvID(52): namedMeta(tvID(52), "T3", []int{18}, kw(13, "three"), kw(14, "four"), kw(15, "five")),
		},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			if len(q.KeywordIDs) > 0 {
				return []catalog.Related{
					relG(tvID(50), "T1", 900, 10765),
					relG(tvID(51), "T2", 0, 10759, 10765),
				}
			}
			return []catalog.Related{
				relG(tvID(51), "T2", 0, 10759, 10765),
				relG(tvID(52), "T3", 0, 18),
			}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if len(src.queries) != 2 {
		t.Fatalf("queries = %v, want 2", src.queries)
	}
	want0 := catalog.DiscoverQuery{Type: catalog.TV, GenreIDs: []int{10759, 10765}, KeywordIDs: []int{11, 12, 13, 14, 15}, MinVotes: 100}
	if !reflect.DeepEqual(src.queries[0], want0) {
		t.Errorf("query 0 = %+v, want %+v", src.queries[0], want0)
	}
	want1 := catalog.DiscoverQuery{Type: catalog.TV, GenreIDs: []int{10759, 10765}, GenresAll: true, MinVotes: 500}
	if !reflect.DeepEqual(src.queries[1], want1) {
		t.Errorf("query 1 = %+v, want %+v", src.queries[1], want1)
	}
	// tv:52 comes from query 2 and has the most shared keywords, so it ranks
	// first; tv:50 shares two keywords; tv:51 shares one keyword and three genes.
	wantIDs := []catalog.ID{tvID(52), tvID(50), tvID(51)}
	if !reflect.DeepEqual(pickIDs(res.Picks), wantIDs) {
		t.Errorf("ids = %v, want %v", pickIDs(res.Picks), wantIDs)
	}
	wantReasons := []string{
		"shares keywords with X (three, four)",
		"shares keywords with X (one, two)",
		"shares keywords with X (one)",
	}
	for i, p := range res.Picks {
		if p.Reason != wantReasons[i] {
			t.Errorf("reason %d = %q, want %q", i, p.Reason, wantReasons[i])
		}
	}
}

func TestFromBoth(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12},
		kw(11, "one"), kw(12, "two"), kw(13, "three"), kw(14, "four"), kw(15, "five"), kw(16, "six"), kw(17, "seven"))
	recs := []catalog.Related{rel(catalog.ID{Type: catalog.Movie, N: 2}, "R2", 5000)}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			m.ID:     m,
			tvID(50): namedMeta(tvID(50), "T1", []int{10759}, kw(11, "one"), kw(12, "two")),
		},
		recs: map[catalog.ID][]catalog.Related{m.ID: recs},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{relG(tvID(50), "T1", 900, 10759)}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// 1 (source meta) + 2 (recommendations, similar) + 2 (discover) + 1 (candidate meta).
	if src.calls != 6 {
		t.Errorf("calls = %d, want 6", src.calls)
	}
	wantIDs := []catalog.ID{{Type: catalog.Movie, N: 2}, tvID(50)}
	if !reflect.DeepEqual(pickIDs(res.Picks), wantIDs) {
		t.Errorf("ids = %v, want %v", pickIDs(res.Picks), wantIDs)
	}
}

func TestFromCrossTVToMovies(t *testing.T) {
	ctx := context.Background()
	m := meta(catalog.ID{Type: catalog.TV, N: 2}, "Y", []int{10759, 18})
	src := &fakeSource{metas: map[catalog.ID]*catalog.Meta{m.ID: m}}

	_, err := From(ctx, src, m.ID, FromOptions{Type: catalog.Movie, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if len(src.queries) != 1 {
		t.Fatalf("queries = %v, want 1", src.queries)
	}
	want := catalog.DiscoverQuery{Type: catalog.Movie, GenreIDs: []int{28, 12}, GenresAll: true, MinVotes: 500}
	if !reflect.DeepEqual(src.queries[0], want) {
		t.Errorf("query = %+v, want %+v", src.queries[0], want)
	}
}

func TestFromCrossNoQuery(t *testing.T) {
	ctx := context.Background()
	m := meta(catalog.ID{Type: catalog.Movie, N: 3}, "Z", []int{53})
	src := &fakeSource{metas: map[catalog.ID]*catalog.Meta{m.ID: m}}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if len(src.queries) != 0 {
		t.Errorf("queries = %v, want none", src.queries)
	}
	if res.Picks == nil || len(res.Picks) != 0 {
		t.Errorf("Picks = %#v, want a non-nil empty slice", res.Picks)
	}
}

func TestFromLimitPerType(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, kw(11, "one"), kw(12, "two"))
	recs := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 2}, "R2", 5000),
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "R3", 5000),
	}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			m.ID:     m,
			tvID(50): namedMeta(tvID(50), "T1", []int{10765}, kw(11, "one"), kw(12, "two")),
			tvID(51): namedMeta(tvID(51), "T2", []int{10765}, kw(11, "one")),
		},
		recs: map[catalog.ID][]catalog.Related{m.ID: recs},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{
				relG(tvID(50), "T1", 900, 10765),
				relG(tvID(51), "T2", 900, 10765),
			}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Limit: 1})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	wantIDs := []catalog.ID{{Type: catalog.Movie, N: 2}, tvID(50)}
	if !reflect.DeepEqual(pickIDs(res.Picks), wantIDs) {
		t.Errorf("ids = %v, want %v", pickIDs(res.Picks), wantIDs)
	}
}

func TestFromMetaError(t *testing.T) {
	ctx := context.Background()
	wantErr := errors.New("boom")
	src := &fakeSource{err: wantErr}
	_, err := From(ctx, src, catalog.ID{Type: catalog.Movie, N: 1}, FromOptions{Limit: 10})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}

	missing := &fakeSource{}
	_, err = From(ctx, missing, catalog.ID{Type: catalog.Movie, N: 999}, FromOptions{Limit: 10})
	var apiErr *catalog.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 404 {
		t.Errorf("err = %v, want a 404 APIError", err)
	}
}

// recsErrorSource overrides Recommendations to return a fixed error.
type recsErrorSource struct {
	*fakeSource
	err error
}

func (s *recsErrorSource) Recommendations(context.Context, catalog.ID) ([]catalog.Related, error) {
	return nil, s.err
}

func TestFromRecommendationsError(t *testing.T) {
	ctx := context.Background()
	m := meta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28})
	wantErr := errors.New("recs boom")
	src := &recsErrorSource{
		fakeSource: &fakeSource{metas: map[catalog.ID]*catalog.Meta{m.ID: m}},
		err:        wantErr,
	}

	_, err := From(ctx, src, m.ID, FromOptions{Type: catalog.Movie, Limit: 10})
	if !errors.Is(err, wantErr) {
		t.Errorf("err = %v, want %v", err, wantErr)
	}
}

func TestCrossPicksFilterAndRank(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12},
		kw(1, "one"), kw(2, "two"), kw(3, "three"), kw(4, "four"), kw(5, "five"))
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			m.ID:     m,
			tvID(10): namedMeta(tvID(10), "C10", []int{10765}, kw(1, "one")),
			tvID(11): namedMeta(tvID(11), "C11", []int{10765}, kw(1, "one"), kw(2, "two")),
			tvID(12): namedMeta(tvID(12), "C12", []int{10759, 10765}, kw(1, "one")),
			tvID(13): namedMeta(tvID(13), "C13", []int{18}, kw(1, "one"), kw(2, "two"), kw(3, "three")),
			tvID(14): namedMeta(tvID(14), "C14", []int{10765}, kw(1, "one"), kw(2, "two")),
			tvID(15): namedMeta(tvID(15), "C15", []int{10765}, kw(1, "one"), kw(2, "two")),
		},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			if len(q.KeywordIDs) > 0 {
				return []catalog.Related{
					relG(tvID(10), "C10", 100, 10765),
					relG(tvID(11), "C11", 100, 10765),
					relG(tvID(12), "C12", 100, 10759, 10765),
				}
			}
			return []catalog.Related{
				relG(tvID(13), "C13", 100, 18),
				relG(tvID(14), "C14", 100, 10765),
				relG(tvID(15), "C15", 100, 10765),
			}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// C10 shares one keyword and one genre, so it is dropped. The rest are
	// ranked by shared keywords, then shared genres, then pool order.
	wantIDs := []catalog.ID{tvID(13), tvID(11), tvID(14), tvID(15), tvID(12)}
	if got := pickIDs(res.Picks); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("ids = %v, want %v", got, wantIDs)
	}
	wantReasons := []string{
		"shares keywords with X (one, two)",
		"shares keywords with X (one, two)",
		"shares keywords with X (one, two)",
		"shares keywords with X (one, two)",
		"shares keywords with X (one)",
	}
	for i, p := range res.Picks {
		if p.Reason != wantReasons[i] {
			t.Errorf("reason %d = %q, want %q", i, p.Reason, wantReasons[i])
		}
	}
}

func TestCrossPicksQuery2NotCutOff(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, kw(1, "one"), kw(2, "two"))
	metas := map[catalog.ID]*catalog.Meta{m.ID: m}
	var keywordResults, genreResults []catalog.Related
	for i := int64(0); i < 20; i++ {
		id := tvID(100 + i)
		metas[id] = namedMeta(id, "K", []int{18}, kw(999, "other"))
		keywordResults = append(keywordResults, relG(id, "K", 100, 18))
	}
	for i := int64(0); i < 2; i++ {
		id := tvID(200 + i)
		metas[id] = namedMeta(id, "G", []int{10759, 10765}, kw(1, "one"), kw(2, "two"))
		genreResults = append(genreResults, relG(id, "G", 100, 10759, 10765))
	}
	src := &fakeSource{
		metas: metas,
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			if len(q.KeywordIDs) > 0 {
				return keywordResults
			}
			return genreResults
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	wantIDs := []catalog.ID{tvID(200), tvID(201)}
	if got := pickIDs(res.Picks); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("ids = %v, want %v (query 2 must not be cut off by a full query 1)", got, wantIDs)
	}
}

func TestCrossPicksEnrichTopCap(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, kw(1, "one"), kw(2, "two"))
	metas := map[catalog.ID]*catalog.Meta{m.ID: m}
	var keywordResults, genreResults []catalog.Related
	for i := int64(0); i < 20; i++ {
		id := tvID(100 + i)
		metas[id] = namedMeta(id, "K", []int{18}, kw(999, "other"))
		keywordResults = append(keywordResults, relG(id, "K", 100, 18))
	}
	for i := int64(0); i < 5; i++ {
		id := tvID(200 + i)
		metas[id] = namedMeta(id, "G", []int{10759, 10765}, kw(1, "one"), kw(2, "two"))
		genreResults = append(genreResults, relG(id, "G", 100, 10759, 10765))
	}
	src := &fakeSource{
		metas: metas,
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			if len(q.KeywordIDs) > 0 {
				return keywordResults
			}
			return genreResults
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	// 1 source meta + 2 discover + at most CrossEnrichTop candidate meta.
	if want := 1 + 2 + CrossEnrichTop; src.calls != want {
		t.Errorf("calls = %d, want %d", src.calls, want)
	}
	wantIDs := []catalog.ID{tvID(200), tvID(201), tvID(202), tvID(203), tvID(204)}
	if got := pickIDs(res.Picks); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("ids = %v, want %v (only the candidates with the most shared genres are enriched)", got, wantIDs)
	}
}

func TestCrossPicksMetaErrorDrops(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, kw(1, "one"), kw(2, "two"))
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{
			m.ID:     m,
			tvID(10): namedMeta(tvID(10), "C10", []int{10765}, kw(1, "one"), kw(2, "two")),
			// tv:11 has no meta, so its lookup is a 404 and only it is dropped.
			tvID(12): namedMeta(tvID(12), "C12", []int{10765}, kw(1, "one"), kw(2, "two")),
		},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{
				relG(tvID(10), "C10", 100, 10765),
				relG(tvID(11), "C11", 100, 10765),
				relG(tvID(12), "C12", 100, 10765),
			}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	wantIDs := []catalog.ID{tvID(10), tvID(12)}
	if got := pickIDs(res.Picks); !reflect.DeepEqual(got, wantIDs) {
		t.Errorf("ids = %v, want %v", got, wantIDs)
	}
}

// cancelMetaSource cancels its context and returns its error when Meta is
// called for failID, so a cancellation during enrichment can be tested.
type cancelMetaSource struct {
	*fakeSource
	cancel context.CancelFunc
	failID catalog.ID
}

func (s *cancelMetaSource) Meta(ctx context.Context, id catalog.ID) (*catalog.Meta, error) {
	if id == s.failID {
		s.cancel()
		return nil, context.Canceled
	}
	return s.fakeSource.Meta(ctx, id)
}

func TestCrossPicksCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := namedMeta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, kw(1, "one"), kw(2, "two"))
	src := &cancelMetaSource{
		fakeSource: &fakeSource{
			metas: map[catalog.ID]*catalog.Meta{
				m.ID:     m,
				tvID(10): namedMeta(tvID(10), "C10", []int{10765}, kw(1, "one"), kw(2, "two")),
			},
			discover: func(q catalog.DiscoverQuery) []catalog.Related {
				return []catalog.Related{
					relG(tvID(10), "C10", 100, 10765),
					relG(tvID(11), "C11", 100, 10765),
				}
			},
		},
		cancel: cancel,
		failID: tvID(11),
	}

	_, err := From(ctx, src, m.ID, FromOptions{Type: catalog.TV, Limit: 10})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
}

func TestCrossPicksNoKeywords(t *testing.T) {
	ctx := context.Background()
	m := namedMeta(tvID(2), "Y", []int{10759, 18})
	src := &fakeSource{
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{relG(catalog.ID{Type: catalog.Movie, N: 50}, "M50", 100, 28, 12)}
		},
	}

	picks, err := crossPicks(ctx, src, m, catalog.Movie, 10)
	if err != nil {
		t.Fatalf("crossPicks: %v", err)
	}
	if src.calls != 1 {
		t.Errorf("calls = %d, want 1 (only the genre discover query)", src.calls)
	}
	if len(picks) != 1 {
		t.Fatalf("len = %d, want 1", len(picks))
	}
	if picks[0].Reason != "shares genres with Y" {
		t.Errorf("reason = %q, want %q", picks[0].Reason, "shares genres with Y")
	}
}
