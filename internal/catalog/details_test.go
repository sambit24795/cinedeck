package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

// detailsServer serves the named fixture and records the request path and query.
func detailsServer(t *testing.T, name string) (*httptest.Server, *string, *string) {
	t.Helper()
	body, err := os.ReadFile("../../testdata/" + name)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var path, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.RawQuery
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &path, &query
}

func TestDetailsMovie(t *testing.T) {
	srv, path, query := detailsServer(t, "movie_27205.json")

	c := New("tok", WithBaseURL(srv.URL))
	d, err := c.Details(context.Background(), ID{Movie, 27205})
	if err != nil {
		t.Fatalf("Details: %v", err)
	}
	if *path != "/movie/27205" {
		t.Errorf("path = %q, want %q", *path, "/movie/27205")
	}
	if !strings.Contains(*query, "append_to_response=credits") {
		t.Errorf("query = %q, want it to contain %q", *query, "append_to_response=credits")
	}
	if d.Title != "Inception" {
		t.Errorf("Title = %q, want %q", d.Title, "Inception")
	}
	if d.Year != 2010 {
		t.Errorf("Year = %d, want %d", d.Year, 2010)
	}
	if d.Runtime != 148 {
		t.Errorf("Runtime = %d, want %d", d.Runtime, 148)
	}
	if want := []string{"Action", "Science Fiction", "Adventure"}; !reflect.DeepEqual(d.Genres, want) {
		t.Errorf("Genres = %v, want %v", d.Genres, want)
	}
	if want := []string{"Christopher Nolan"}; !reflect.DeepEqual(d.Directors, want) {
		t.Errorf("Directors = %v, want %v", d.Directors, want)
	}
	if len(d.Cast) == 0 || d.Cast[0] != "Leonardo DiCaprio" {
		t.Errorf("Cast = %v, want first %q", d.Cast, "Leonardo DiCaprio")
	}
}

func TestDetailsTV(t *testing.T) {
	srv, path, _ := detailsServer(t, "tv_1399.json")

	c := New("tok", WithBaseURL(srv.URL))
	d, err := c.Details(context.Background(), ID{TV, 1399})
	if err != nil {
		t.Fatalf("Details: %v", err)
	}
	if *path != "/tv/1399" {
		t.Errorf("path = %q, want %q", *path, "/tv/1399")
	}
	if d.Title != "Game of Thrones" {
		t.Errorf("Title = %q, want %q", d.Title, "Game of Thrones")
	}
	if d.Seasons != 8 {
		t.Errorf("Seasons = %d, want %d", d.Seasons, 8)
	}
	if d.Episodes != 73 {
		t.Errorf("Episodes = %d, want %d", d.Episodes, 73)
	}
	if want := []string{"David Benioff", "D. B. Weiss"}; !reflect.DeepEqual(d.Creators, want) {
		t.Errorf("Creators = %v, want %v", d.Creators, want)
	}
	if d.Runtime != 0 {
		t.Errorf("Runtime = %d, want 0", d.Runtime)
	}
}
