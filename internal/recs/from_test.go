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
	m := meta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, 11, 12, 13, 14, 15, 16, 17)
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{m.ID: m},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			if len(q.KeywordIDs) > 0 {
				return []catalog.Related{
					rel(catalog.ID{Type: catalog.TV, N: 50}, "T1", 900),
					rel(catalog.ID{Type: catalog.TV, N: 51}, "T2", 0),
				}
			}
			return []catalog.Related{
				rel(catalog.ID{Type: catalog.TV, N: 51}, "T2", 0),
				rel(catalog.ID{Type: catalog.TV, N: 52}, "T3", 0),
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
	wantIDs := []catalog.ID{{Type: catalog.TV, N: 50}, {Type: catalog.TV, N: 51}, {Type: catalog.TV, N: 52}}
	if !reflect.DeepEqual(pickIDs(res.Picks), wantIDs) {
		t.Errorf("ids = %v, want %v", pickIDs(res.Picks), wantIDs)
	}
	wantReasons := []string{"shares keywords with X", "shares keywords with X", "shares genres with X"}
	for i, p := range res.Picks {
		if p.Reason != wantReasons[i] {
			t.Errorf("reason %d = %q, want %q", i, p.Reason, wantReasons[i])
		}
	}
}

func TestFromBoth(t *testing.T) {
	ctx := context.Background()
	m := meta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, 11, 12, 13, 14, 15, 16, 17)
	recs := []catalog.Related{rel(catalog.ID{Type: catalog.Movie, N: 2}, "R2", 5000)}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{m.ID: m},
		recs:  map[catalog.ID][]catalog.Related{m.ID: recs},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{rel(catalog.ID{Type: catalog.TV, N: 50}, "T1", 900)}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if src.calls != 5 {
		t.Errorf("calls = %d, want 5", src.calls)
	}
	wantIDs := []catalog.ID{{Type: catalog.Movie, N: 2}, {Type: catalog.TV, N: 50}}
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
	m := meta(catalog.ID{Type: catalog.Movie, N: 1}, "X", []int{28, 878, 12}, 11, 12)
	recs := []catalog.Related{
		rel(catalog.ID{Type: catalog.Movie, N: 2}, "R2", 5000),
		rel(catalog.ID{Type: catalog.Movie, N: 3}, "R3", 5000),
	}
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{m.ID: m},
		recs:  map[catalog.ID][]catalog.Related{m.ID: recs},
		discover: func(q catalog.DiscoverQuery) []catalog.Related {
			return []catalog.Related{
				rel(catalog.ID{Type: catalog.TV, N: 50}, "T1", 900),
				rel(catalog.ID{Type: catalog.TV, N: 51}, "T2", 900),
			}
		},
	}

	res, err := From(ctx, src, m.ID, FromOptions{Limit: 1})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	wantIDs := []catalog.ID{{Type: catalog.Movie, N: 2}, {Type: catalog.TV, N: 50}}
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
