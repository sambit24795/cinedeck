package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

func TestGenres(t *testing.T) {
	body, err := os.ReadFile("../../testdata/genre_movie_list.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(body)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Genres(context.Background(), Movie)
	if err != nil {
		t.Fatalf("Genres: %v", err)
	}
	if got["animation"] != 16 {
		t.Errorf("genres[animation] = %d, want 16", got["animation"])
	}
	if got["science fiction"] != 878 {
		t.Errorf("genres[science fiction] = %d, want 878", got["science fiction"])
	}
}

func TestShorts(t *testing.T) {
	genreBody, err := os.ReadFile("../../testdata/genre_movie_list.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	discoverBody, err := os.ReadFile("../../testdata/discover_shorts.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var discoverQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/genre/movie/list":
			w.Write(genreBody)
		case "/discover/movie":
			discoverQuery = r.URL.RawQuery
			w.Write(discoverBody)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Shorts(context.Background(), ShortsOptions{Genre: "Animation", Year: 2008})
	if err != nil {
		t.Fatalf("Shorts: %v", err)
	}
	for _, want := range []string{"with_runtime.lte=40", "with_genres=16", "primary_release_year=2008"} {
		if !strings.Contains(discoverQuery, want) {
			t.Errorf("query = %q, want it to contain %q", discoverQuery, want)
		}
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != (ID{Movie, 13042}) || got[0].Title != "Presto" || got[0].Year != 2008 {
		t.Errorf("got[0] = %+v, want {ID:{movie 13042} Presto 2008}", got[0])
	}
}

// genreServer serves the genre list fixtures and records every request path.
func genreServer(t *testing.T, paths *[]string) *httptest.Server {
	t.Helper()
	tvBody, err := os.ReadFile("../../testdata/genre_tv_list.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	movieBody, err := os.ReadFile("../../testdata/genre_movie_list.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*paths = append(*paths, r.URL.Path)
		switch r.URL.Path {
		case "/genre/tv/list":
			w.Write(tvBody)
		case "/genre/movie/list":
			w.Write(movieBody)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestGenreIDs(t *testing.T) {
	ctx := context.Background()

	t.Run("tv names", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.genreIDs(ctx, TV, []string{"Animation", " drama "})
		if err != nil {
			t.Fatalf("genreIDs: %v", err)
		}
		if len(got) != 2 || got[0] != 16 || got[1] != 18 {
			t.Errorf("got = %v, want [16 18]", got)
		}
	})

	t.Run("tv duplicate", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.genreIDs(ctx, TV, []string{"Drama", "drama"})
		if err != nil {
			t.Fatalf("genreIDs: %v", err)
		}
		if len(got) != 1 || got[0] != 18 {
			t.Errorf("got = %v, want [18]", got)
		}
	})

	t.Run("movie path", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.genreIDs(ctx, Movie, []string{"Animation", "Drama"})
		if err != nil {
			t.Fatalf("genreIDs: %v", err)
		}
		if len(got) != 2 || got[0] != 16 || got[1] != 18 {
			t.Errorf("got = %v, want [16 18]", got)
		}
		if len(paths) != 1 || paths[0] != "/genre/movie/list" {
			t.Errorf("paths = %v, want [/genre/movie/list]", paths)
		}
	})

	t.Run("unknown tv", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		_, err := c.genreIDs(ctx, TV, []string{"Action"})
		if err == nil {
			t.Fatal("err = nil, want error")
		}
		for _, want := range []string{`unknown genre "Action" for tv`, "action & adventure"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("err = %v, want it to contain %q", err, want)
			}
		}
	})

	t.Run("unknown movie", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		_, err := c.genreIDs(ctx, Movie, []string{"Sci-Fi & Fantasy"})
		if err == nil || !strings.Contains(err.Error(), "unknown genre") {
			t.Fatalf("err = %v, want it to contain %q", err, "unknown genre")
		}
	})

	t.Run("no names", func(t *testing.T) {
		var paths []string
		srv := genreServer(t, &paths)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.genreIDs(ctx, TV, nil)
		if err != nil {
			t.Fatalf("genreIDs: %v", err)
		}
		if len(got) != 0 {
			t.Errorf("got = %v, want empty", got)
		}
		if len(paths) != 0 {
			t.Errorf("paths = %v, want no requests", paths)
		}
	})
}

// discoverCatalogServer serves the discover and genre fixtures by path. It
// records every requested path and the query of the last "/discover/" request.
func discoverCatalogServer(t *testing.T, paths *[]string, query *url.Values) *httptest.Server {
	t.Helper()
	files := map[string]string{
		"/discover/tv":      "discover_tv_kdrama.json",
		"/discover/movie":   "discover_movie_anime.json",
		"/genre/tv/list":    "genre_tv_list.json",
		"/genre/movie/list": "genre_movie_list.json",
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

func TestDiscover(t *testing.T) {
	ctx := context.Background()

	t.Run("tv", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.Discover(ctx, DiscoverOptions{Type: TV, Genres: []string{"Drama"}, Language: "ko", MinVotes: 50})
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(paths) == 0 || paths[len(paths)-1] != "/discover/tv" {
			t.Errorf("paths = %v, want the last one to be /discover/tv", paths)
		}
		for _, want := range [][2]string{{"with_original_language", "ko"}, {"with_genres", "18"}, {"vote_count.gte", "50"}, {"sort_by", "popularity.desc"}, {"include_adult", "false"}} {
			if got := query.Get(want[0]); got != want[1] {
				t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
			}
		}
		for _, bad := range []string{"with_origin_country", "first_air_date_year", "primary_release_year"} {
			if query.Has(bad) {
				t.Errorf("query = %v, want no %s", query, bad)
			}
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		want := Result{ID: ID{TV, 93405}, Title: "Squid Game", Year: 2021, Rating: 7.853}
		if got[0].ID != want.ID || got[0].Title != want.Title || got[0].Year != want.Year || got[0].Rating != want.Rating {
			t.Errorf("got[0] = %+v, want %+v", got[0], want)
		}
	})

	t.Run("movie", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		got, err := c.Discover(ctx, DiscoverOptions{Type: Movie, Genres: []string{"animation"}, Language: "ja", Country: "JP", Year: 2023})
		if err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(paths) == 0 || paths[len(paths)-1] != "/discover/movie" {
			t.Errorf("paths = %v, want the last one to be /discover/movie", paths)
		}
		for _, want := range [][2]string{{"with_genres", "16"}, {"with_original_language", "ja"}, {"with_origin_country", "JP"}, {"primary_release_year", "2023"}} {
			if got := query.Get(want[0]); got != want[1] {
				t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
			}
		}
		for _, bad := range []string{"first_air_date_year", "vote_count.gte"} {
			if query.Has(bad) {
				t.Errorf("query = %v, want no %s", query, bad)
			}
		}
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		if got[0].ID != (ID{Movie, 1311031}) || got[0].Title != "Demon Slayer: Kimetsu no Yaiba Infinity Castle" || got[0].Year != 2025 {
			t.Errorf("got[0] = %+v, want movie:1311031 at 2025", got[0])
		}
	})

	t.Run("tv year", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		if _, err := c.Discover(ctx, DiscoverOptions{Type: TV, Year: 2024}); err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if got := query.Get("first_air_date_year"); got != "2024" {
			t.Errorf("first_air_date_year = %q, want 2024", got)
		}
		if query.Has("primary_release_year") {
			t.Errorf("query = %v, want no primary_release_year", query)
		}
		if query.Has("with_genres") {
			t.Errorf("query = %v, want no with_genres", query)
		}
		for _, p := range paths {
			if p == "/genre/tv/list" {
				t.Errorf("paths = %v, want no genre request", paths)
			}
		}
	})

	t.Run("empty type", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		if _, err := c.Discover(ctx, DiscoverOptions{Language: "ja"}); err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if len(paths) == 0 || paths[len(paths)-1] != "/discover/movie" {
			t.Errorf("paths = %v, want the last one to be /discover/movie", paths)
		}
	})

	t.Run("two genres", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		if _, err := c.Discover(ctx, DiscoverOptions{Type: TV, Genres: []string{"Animation", "Drama"}}); err != nil {
			t.Fatalf("Discover: %v", err)
		}
		if got := query.Get("with_genres"); got != "16,18" {
			t.Errorf("with_genres = %q, want %q", got, "16,18")
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		_, err := c.Discover(ctx, DiscoverOptions{Type: MediaType("person")})
		if err == nil || !strings.Contains(err.Error(), `invalid type "person"`) {
			t.Fatalf("err = %v, want it to contain invalid type", err)
		}
		if len(paths) != 0 {
			t.Errorf("paths = %v, want no requests", paths)
		}
	})

	t.Run("unknown genre", func(t *testing.T) {
		var paths []string
		var query url.Values
		srv := discoverCatalogServer(t, &paths, &query)
		c := New("tok", WithBaseURL(srv.URL))
		_, err := c.Discover(ctx, DiscoverOptions{Type: TV, Genres: []string{"Action"}})
		if err == nil || !strings.Contains(err.Error(), "unknown genre") {
			t.Fatalf("err = %v, want it to contain unknown genre", err)
		}
		for _, p := range paths {
			if p == "/discover/tv" {
				t.Errorf("paths = %v, want no /discover/tv request", paths)
			}
		}
	})
}

func TestShortsUnknownGenre(t *testing.T) {
	genreBody, err := os.ReadFile("../../testdata/genre_movie_list.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(genreBody)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	_, err = c.Shorts(context.Background(), ShortsOptions{Genre: "nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown genre") {
		t.Fatalf("err = %v, want it to contain %q", err, "unknown genre")
	}
}
