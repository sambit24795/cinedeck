package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestImageURL(t *testing.T) {
	if got, want := imageURL("w342", "/a.jpg"), "https://image.tmdb.org/t/p/w342/a.jpg"; got != want {
		t.Errorf("imageURL(w342, /a.jpg) = %q, want %q", got, want)
	}
	if got := imageURL("w342", ""); got != "" {
		t.Errorf("imageURL(w342, \"\") = %q, want \"\"", got)
	}
}

// imageServer serves the named fixture for the given path.
func imageServer(t *testing.T, files map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name, ok := files[r.URL.Path]
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
	return srv
}

func TestDiscoverPoster(t *testing.T) {
	srv := imageServer(t, map[string]string{
		"/discover/tv":   "discover_tv_kdrama.json",
		"/genre/tv/list": "genre_tv_list.json",
	})

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Discover(context.Background(), DiscoverOptions{Type: TV, Language: "ko"})
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Discover returned no results")
	}
	want := "https://image.tmdb.org/t/p/w342/1QdXdRYfktUSONkl1oD5gc6Be0s.jpg"
	if got[0].Poster != want {
		t.Errorf("Poster = %q, want %q", got[0].Poster, want)
	}
}

func TestSearchHasNoPoster(t *testing.T) {
	srv := imageServer(t, map[string]string{"/search/multi": "search_multi.json"})

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Search(context.Background(), "inception")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(got) == 0 {
		t.Fatal("Search returned no results")
	}
	for i, r := range got {
		if r.Poster != "" {
			t.Errorf("got[%d].Poster = %q, want \"\"", i, r.Poster)
		}
	}
	b, err := json.Marshal(got[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "poster") {
		t.Errorf("json = %s, want it not to contain %q", b, "poster")
	}
}
