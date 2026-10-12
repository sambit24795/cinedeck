package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

func TestYTSSearch(t *testing.T) {
	t.Parallel()

	var lastURI string
	var calls int
	data := readFixture(t, "../../../testdata/indexers/yts_inception.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		lastURI = r.URL.RequestURI()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	y := &YTS{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := y.Search(context.Background(), indexer.Query{Text: "inception"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !strings.HasPrefix(lastURI, "/api/v2/list_movies.json?") || !strings.Contains(lastURI, "query_term=inception") {
		t.Errorf("request URI = %q", lastURI)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (one per quality)", len(got))
	}
	if got[0].Provider != "yts" || got[1].Provider != "yts" {
		t.Errorf("providers = %q/%q", got[0].Provider, got[1].Provider)
	}
	if got[0].InfoHash != got[1].InfoHash {
		t.Errorf("both qualities should share the hash: %q vs %q", got[0].InfoHash, got[1].InfoHash)
	}
	if got[0].Seeders != 300 {
		t.Errorf("Seeders = %d, want 300", got[0].Seeders)
	}
	if got[0].Year != 2010 {
		t.Errorf("Year = %d, want 2010", got[0].Year)
	}
}

func TestYTSTypeTVMakesNoRequest(t *testing.T) {
	t.Parallel()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	t.Cleanup(srv.Close)

	y := &YTS{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := y.Search(context.Background(), indexer.Query{Text: "x", Type: indexer.TV})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
	if calls != 0 {
		t.Errorf("server saw %d requests, want 0", calls)
	}
}
