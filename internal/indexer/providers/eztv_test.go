package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

func TestEZTVSearchFiltersByText(t *testing.T) {
	t.Parallel()

	var lastURI string
	srv := serveFixture(t, "../../../testdata/indexers/eztv_breaking_bad.json", &lastURI)

	e := &EZTV{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := e.Search(context.Background(), indexer.Query{Text: "breaking bad"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !strings.HasPrefix(lastURI, "/api/get-torrents") {
		t.Errorf("request URI = %q", lastURI)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2 (Better Call Saul filtered out)", len(got))
	}
	if got[0].Provider != "eztv" {
		t.Errorf("Provider = %q, want eztv", got[0].Provider)
	}
	if got[0].Seeders != 500 {
		t.Errorf("Seeders = %d, want 500", got[0].Seeders)
	}
}

func TestEZTVMinSeeders(t *testing.T) {
	t.Parallel()

	srv := serveFixture(t, "../../../testdata/indexers/eztv_breaking_bad.json", nil)
	e := &EZTV{BaseURL: srv.URL, HTTP: srv.Client()}

	got, err := e.Search(context.Background(), indexer.Query{Text: "breaking", MinSeeders: 200})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestEZTVTypeMovieMakesNoRequest(t *testing.T) {
	t.Parallel()

	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
	}))
	t.Cleanup(srv.Close)

	e := &EZTV{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := e.Search(context.Background(), indexer.Query{Text: "x", Type: indexer.Movie})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 0 || calls != 0 {
		t.Errorf("len = %d, calls = %d; want 0 and 0", len(got), calls)
	}
}
