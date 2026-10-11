package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
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
