package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

func TestArchiveSearch(t *testing.T) {
	t.Parallel()

	var lastURI string
	srv := serveFixture(t, "../../../testdata/indexers/archive_nosferatu.json", &lastURI)

	a := &Archive{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := a.Search(context.Background(), indexer.Query{Text: "nosferatu"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !strings.Contains(lastURI, "mediatype%3Amovies") || !strings.Contains(lastURI, "output=json") {
		t.Errorf("request URI = %q", lastURI)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Provider != "archive" {
		t.Errorf("Provider = %q, want archive", got[0].Provider)
	}
	if got[0].InfoHash != "" || got[0].Seeders != 0 || got[0].Leechers != 0 {
		t.Errorf("archive releases must have no hash or seeder counts: %+v", got[0])
	}
	if !strings.HasSuffix(got[0].InfoURL, "/details/nosferatu1922") {
		t.Errorf("InfoURL = %q", got[0].InfoURL)
	}
	if got[0].Year != 1922 {
		t.Errorf("Year = %d, want 1922", got[0].Year)
	}
}

func TestArchiveYearNumber(t *testing.T) {
	t.Parallel()

	body := `{"response":{"docs":[{"identifier":"x","title":"X","year":1984}]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	a := &Archive{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := a.Search(context.Background(), indexer.Query{Text: "x"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 || got[0].Year != 1984 {
		t.Fatalf("got %+v, want one entry with Year 1984", got)
	}
}
