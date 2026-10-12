package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// serveFixture starts a test server that answers every request with the file at path.
func serveFixture(t *testing.T, path string, lastURL *string) *httptest.Server {
	t.Helper()
	data := readFixture(t, path)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if lastURL != nil {
			*lastURL = r.URL.RequestURI()
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// readFixture reads a testdata file and fails the test on error.
func readFixture(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", path, err)
	}
	return data
}

func TestApibaySearch(t *testing.T) {
	t.Parallel()

	var lastURL string
	srv := serveFixture(t, "../../../testdata/indexers/apibay_inception.json", &lastURL)

	a := &Apibay{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := a.Search(context.Background(), indexer.Query{Text: "inception"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if lastURL != "/q.php?q=inception" {
		t.Errorf("request URI = %q, want /q.php?q=inception", lastURL)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Provider != "apibay" {
		t.Errorf("Provider = %q, want apibay", got[0].Provider)
	}
	if got[0].InfoHash != "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" {
		t.Errorf("InfoHash = %q", got[0].InfoHash)
	}
	if got[0].Seeders != 450 || got[0].Leechers != 120 {
		t.Errorf("seeders/leechers = %d/%d, want 450/120", got[0].Seeders, got[0].Leechers)
	}
	if got[0].Size != 2147483648 {
		t.Errorf("Size = %d, want 2147483648", got[0].Size)
	}
	if got[0].Magnet == "" {
		t.Error("expected a magnet link")
	}
}

func TestApibayMinSeeders(t *testing.T) {
	t.Parallel()

	srv := serveFixture(t, "../../../testdata/indexers/apibay_inception.json", nil)
	a := &Apibay{BaseURL: srv.URL, HTTP: srv.Client()}

	got, err := a.Search(context.Background(), indexer.Query{Text: "inception", MinSeeders: 100})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (only the 450-seeder entry)", len(got))
	}
}

func TestApibayNoResults(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"name":"No results returned","info_hash":"","seeders":"0","leechers":"0","size":"0"}]`))
	}))
	t.Cleanup(srv.Close)

	a := &Apibay{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := a.Search(context.Background(), indexer.Query{Text: "zzzz"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if got == nil {
		t.Fatal("expected a non-nil empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestApibayServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	a := &Apibay{BaseURL: srv.URL, HTTP: srv.Client()}
	if _, err := a.Search(context.Background(), indexer.Query{Text: "x"}); err == nil {
		t.Fatal("expected an error for a 500 response")
	}
}
