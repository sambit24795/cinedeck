package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestTrending(t *testing.T) {
	body, err := os.ReadFile("../../testdata/trending_movie.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write(body)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Trending(context.Background(), Movie)
	if err != nil {
		t.Fatalf("Trending: %v", err)
	}
	if path != "/trending/movie/week" {
		t.Errorf("path = %q, want %q", path, "/trending/movie/week")
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].ID != (ID{Movie, 27205}) {
		t.Errorf("got[0].ID = %v, want %v", got[0].ID, ID{Movie, 27205})
	}
}
