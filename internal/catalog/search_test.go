package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
)

func TestSearch(t *testing.T) {
	body, err := os.ReadFile("../../testdata/search_multi.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var query string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.RawQuery
		w.Write(body)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	got, err := c.Search(context.Background(), "inception")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if !strings.Contains(query, "query=inception") {
		t.Errorf("query = %q, want it to contain %q", query, "query=inception")
	}
	if !strings.Contains(query, "include_adult=false") {
		t.Errorf("query = %q, want it to contain %q", query, "include_adult=false")
	}

	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	want := []Result{
		{ID: ID{Movie, 27205}, Title: "Inception", Year: 2010, Rating: 8.4},
		{ID: ID{TV, 1399}, Title: "Game of Thrones", Year: 2011, Rating: 8.5},
	}
	for i, w := range want {
		if got[i].ID != w.ID || got[i].Title != w.Title || got[i].Year != w.Year || got[i].Rating != w.Rating {
			t.Errorf("got[%d] = %+v, want %+v", i, got[i], w)
		}
	}
	if got[2].Year != 0 {
		t.Errorf("got[2].Year = %d, want 0", got[2].Year)
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	if _, err := c.Search(context.Background(), "  "); err == nil {
		t.Fatal("Search: want error")
	}
	if got := atomic.LoadInt32(&calls); got != 0 {
		t.Errorf("calls = %d, want 0", got)
	}
}
