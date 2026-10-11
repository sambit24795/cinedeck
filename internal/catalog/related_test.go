package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strings"
	"testing"
)

// relatedServer serves the recommendation, similar and discover fixtures. It
// records every request path and the query of the last "/discover/" request.
func relatedServer(t *testing.T, paths *[]string, query *url.Values) *httptest.Server {
	t.Helper()
	files := map[string]string{
		"/movie/27205/recommendations": "related_recs_movie_27205.json",
		"/movie/27205/similar":         "related_similar_movie_27205.json",
		"/tv/1396/recommendations":     "related_recs_tv_1396.json",
		"/tv/1396/similar":             "related_similar_tv_1396.json",
		"/discover/tv":                 "discover_tv_keywords.json",
		"/discover/movie":              "discover_movie_keywords.json",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*paths = append(*paths, r.URL.Path)
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/discover/") {
			*query = r.URL.Query()
		}
		body, err := os.ReadFile("../../testdata/" + name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestRecommendations(t *testing.T) {
	ctx := context.Background()

	t.Run("movie", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		got, err := c.Recommendations(ctx, ID{Movie, 27205})
		if err != nil {
			t.Fatalf("Recommendations: %v", err)
		}
		if len(paths) != 1 || paths[0] != "/movie/27205/recommendations" {
			t.Errorf("paths = %v, want [/movie/27205/recommendations]", paths)
		}
		if len(got) != 6 {
			t.Fatalf("len = %d, want 6", len(got))
		}
		if got[0].ID != (ID{Movie, 9620}) || got[0].Title != "Paycheck" || got[0].Year != 2003 || got[0].Rating != 6.229 {
			t.Errorf("got[0] = %+v, want movie:9620 Paycheck 2003 rating 6.229", got[0])
		}
		if !reflect.DeepEqual(got[0].GenreIDs, []int{28, 878, 53}) {
			t.Errorf("got[0].GenreIDs = %v, want [28 878 53]", got[0].GenreIDs)
		}
		if got[0].Popularity != 7.8359 || got[0].Votes != 2141 {
			t.Errorf("got[0].Popularity/Votes = %v/%d, want 7.8359/2141", got[0].Popularity, got[0].Votes)
		}
	})

	t.Run("tv", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		got, err := c.Recommendations(ctx, ID{TV, 1396})
		if err != nil {
			t.Fatalf("Recommendations: %v", err)
		}
		if len(got) != 6 {
			t.Fatalf("len = %d, want 6", len(got))
		}
		if got[0].ID != (ID{TV, 60059}) || got[0].Title != "Better Call Saul" || got[0].Year != 2015 {
			t.Errorf("got[0] = %+v, want tv:60059 Better Call Saul 2015", got[0])
		}
		if !reflect.DeepEqual(got[0].GenreIDs, []int{80, 18}) {
			t.Errorf("got[0].GenreIDs = %v, want [80 18]", got[0].GenreIDs)
		}
	})

	t.Run("404", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		_, err := c.Recommendations(ctx, ID{Movie, 99999})
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("err = %v, want *APIError", err)
		}
		if apiErr.Status != 404 {
			t.Errorf("Status = %d, want 404", apiErr.Status)
		}
	})
}

func TestSimilar(t *testing.T) {
	ctx := context.Background()

	t.Run("movie", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		got, err := c.Similar(ctx, ID{Movie, 27205})
		if err != nil {
			t.Fatalf("Similar: %v", err)
		}
		if len(paths) != 1 || paths[0] != "/movie/27205/similar" {
			t.Errorf("paths = %v, want [/movie/27205/similar]", paths)
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		if got[0].ID != (ID{Movie, 26834}) || got[0].Title != "Soft Deceit" || got[0].Votes != 5 {
			t.Errorf("got[0] = %+v, want movie:26834 Soft Deceit with 5 votes", got[0])
		}
	})

	t.Run("tv", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		got, err := c.Similar(ctx, ID{TV, 1396})
		if err != nil {
			t.Fatalf("Similar: %v", err)
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		if got[3].ID != (ID{TV, 60059}) {
			t.Errorf("got[3].ID = %v, want tv:60059", got[3].ID)
		}
	})
}

func TestDiscoverBy(t *testing.T) {
	ctx := context.Background()

	t.Run("tv keywords and genres", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		got, err := c.DiscoverBy(ctx, DiscoverQuery{
			Type:       TV,
			GenreIDs:   []int{10759, 10765},
			KeywordIDs: []int{1566, 10937},
			MinVotes:   100,
		})
		if err != nil {
			t.Fatalf("DiscoverBy: %v", err)
		}
		if len(paths) != 1 || paths[0] != "/discover/tv" {
			t.Errorf("paths = %v, want [/discover/tv]", paths)
		}
		for _, p := range paths {
			if strings.HasPrefix(p, "/genre/") {
				t.Errorf("paths = %v, want no genre request", paths)
			}
		}
		for _, want := range [][2]string{
			{"with_genres", "10759|10765"},
			{"with_keywords", "1566|10937"},
			{"vote_count.gte", "100"},
			{"sort_by", "popularity.desc"},
			{"include_adult", "false"},
		} {
			if got := query.Get(want[0]); got != want[1] {
				t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
			}
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		if got[0].ID != (ID{TV, 40075}) || got[0].Title != "Gravity Falls" || got[0].Year != 2012 {
			t.Errorf("got[0] = %+v, want tv:40075 Gravity Falls 2012", got[0])
		}
	})

	t.Run("genres all", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		if _, err := c.DiscoverBy(ctx, DiscoverQuery{Type: TV, GenreIDs: []int{10759, 10765}, GenresAll: true}); err != nil {
			t.Fatalf("DiscoverBy: %v", err)
		}
		if got := query.Get("with_genres"); got != "10759,10765" {
			t.Errorf("with_genres = %q, want %q", got, "10759,10765")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		if _, err := c.DiscoverBy(ctx, DiscoverQuery{}); err != nil {
			t.Fatalf("DiscoverBy: %v", err)
		}
		if len(paths) != 1 || paths[0] != "/discover/movie" {
			t.Errorf("paths = %v, want [/discover/movie]", paths)
		}
		for _, bad := range []string{"with_genres", "with_keywords", "vote_count.gte"} {
			if query.Has(bad) {
				t.Errorf("query = %v, want no %s", query, bad)
			}
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := relatedServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))

		_, err := c.DiscoverBy(ctx, DiscoverQuery{Type: "person"})
		if err == nil || !strings.Contains(err.Error(), `invalid type "person"`) {
			t.Fatalf("err = %v, want it to contain %q", err, `invalid type "person"`)
		}
		if len(paths) != 0 {
			t.Errorf("paths = %v, want no requests", paths)
		}
	})
}
