package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"testing"
)

// metaServer serves the named fixtures and records the last request path and query.
func metaServer(t *testing.T, routes map[string]string) (*httptest.Server, *string, *string) {
	t.Helper()
	var path, query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		query = r.URL.RawQuery
		name, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("../../testdata/" + name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv, &path, &query
}

func TestMetaMovie(t *testing.T) {
	srv, path, query := metaServer(t, map[string]string{"/movie/27205": "meta_movie_27205.json"})

	c := New("tok", WithBaseURL(srv.URL))
	m, err := c.Meta(context.Background(), ID{Movie, 27205})
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if *path != "/movie/27205" {
		t.Errorf("path = %q, want %q", *path, "/movie/27205")
	}
	values, err := url.ParseQuery(*query)
	if err != nil {
		t.Fatalf("ParseQuery: %v", err)
	}
	if got := values.Get("append_to_response"); got != "keywords,credits" {
		t.Errorf("append_to_response = %q, want %q", got, "keywords,credits")
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"Title", m.Title, "Inception"},
		{"Year", m.Year, 2010},
		{"Language", m.Language, "en"},
		{"Rating", m.Rating, 8.373},
		{"Votes", m.Votes, 40395},
		{"Popularity", m.Popularity, 51.8728},
		{"GenreIDs", m.GenreIDs, []int{28, 878, 12}},
		{"Genres", m.Genres, []string{"Action", "Science Fiction", "Adventure"}},
		{"Directors", m.Directors, []Person{{525, "Christopher Nolan"}}},
	}
	for _, ck := range checks {
		if !reflect.DeepEqual(ck.got, ck.want) {
			t.Errorf("%s = %v, want %v", ck.name, ck.got, ck.want)
		}
	}
	if len(m.Keywords) != 14 {
		t.Errorf("len(Keywords) = %d, want 14", len(m.Keywords))
	}
	if want := (Keyword{10364, "mission"}); m.Keywords[0] != want {
		t.Errorf("Keywords[0] = %v, want %v", m.Keywords[0], want)
	}
	if len(m.Cast) != 5 {
		t.Errorf("len(Cast) = %d, want 5", len(m.Cast))
	}
	if want := (Person{6193, "Leonardo DiCaprio"}); m.Cast[0] != want {
		t.Errorf("Cast[0] = %v, want %v", m.Cast[0], want)
	}
}

func TestMetaTV(t *testing.T) {
	srv, path, _ := metaServer(t, map[string]string{"/tv/1396": "meta_tv_1396.json"})

	c := New("tok", WithBaseURL(srv.URL))
	m, err := c.Meta(context.Background(), ID{TV, 1396})
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if *path != "/tv/1396" {
		t.Errorf("path = %q, want %q", *path, "/tv/1396")
	}
	if m.Title != "Breaking Bad" {
		t.Errorf("Title = %q, want %q", m.Title, "Breaking Bad")
	}
	if m.Year != 2008 {
		t.Errorf("Year = %d, want %d", m.Year, 2008)
	}
	if want := []int{18, 80}; !reflect.DeepEqual(m.GenreIDs, want) {
		t.Errorf("GenreIDs = %v, want %v", m.GenreIDs, want)
	}
	if len(m.Keywords) != 29 {
		t.Errorf("len(Keywords) = %d, want 29", len(m.Keywords))
	}
	if want := (Keyword{1508, "new mexico"}); m.Keywords[0] != want {
		t.Errorf("Keywords[0] = %v, want %v", m.Keywords[0], want)
	}
	if want := []Person{{66633, "Vince Gilligan"}}; !reflect.DeepEqual(m.Directors, want) {
		t.Errorf("Directors = %v, want %v", m.Directors, want)
	}
	if len(m.Cast) == 0 || m.Cast[0].Name != "Bryan Cranston" {
		t.Errorf("Cast = %v, want first %q", m.Cast, "Bryan Cranston")
	}
}

func TestMetaTVCreators(t *testing.T) {
	srv, _, _ := metaServer(t, map[string]string{"/tv/60059": "meta_tv_60059.json"})

	c := New("tok", WithBaseURL(srv.URL))
	m, err := c.Meta(context.Background(), ID{TV, 60059})
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	want := []Person{{66633, "Vince Gilligan"}, {24951, "Peter Gould"}}
	if !reflect.DeepEqual(m.Directors, want) {
		t.Errorf("Directors = %v, want %v", m.Directors, want)
	}
}

func TestMetaEmptySlices(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"id": 5, "title": "x"}`))
	}))
	t.Cleanup(srv.Close)

	c := New("tok", WithBaseURL(srv.URL))
	m, err := c.Meta(context.Background(), ID{Movie, 5})
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if m.Year != 0 {
		t.Errorf("Year = %d, want 0", m.Year)
	}
	if m.GenreIDs == nil {
		t.Error("GenreIDs = nil, want non-nil")
	}
	if m.Genres == nil {
		t.Error("Genres = nil, want non-nil")
	}
	if m.Keywords == nil {
		t.Error("Keywords = nil, want non-nil")
	}
	if m.Cast == nil {
		t.Error("Cast = nil, want non-nil")
	}
	if m.Directors == nil {
		t.Error("Directors = nil, want non-nil")
	}
}

func TestMetaNotFound(t *testing.T) {
	srv, _, _ := metaServer(t, map[string]string{})

	c := New("tok", WithBaseURL(srv.URL))
	_, err := c.Meta(context.Background(), ID{Movie, 1})
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *APIError", err)
	}
	if apiErr.Status != 404 {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
}
