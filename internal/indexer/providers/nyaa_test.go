package providers

import (
	"context"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

func TestNyaaSearch(t *testing.T) {
	t.Parallel()

	var lastURI string
	srv := serveFixture(t, "../../../testdata/indexers/nyaa_jujutsu.rss", &lastURI)

	n := &Nyaa{BaseURL: srv.URL, HTTP: srv.Client()}
	got, err := n.Search(context.Background(), indexer.Query{Text: "jujutsu kaisen"})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if !strings.Contains(lastURI, "page=rss") {
		t.Errorf("request URI = %q", lastURI)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Provider != "nyaa" {
		t.Errorf("Provider = %q, want nyaa", got[0].Provider)
	}
	if got[0].Seeders != 350 || got[0].Leechers != 12 {
		t.Errorf("seeders/leechers = %d/%d, want 350/12", got[0].Seeders, got[0].Leechers)
	}
	if got[0].InfoHash != "d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0" {
		t.Errorf("InfoHash = %q", got[0].InfoHash)
	}
	if got[0].Size != 1503238553 { // 1.4 GiB
		t.Errorf("Size = %d, want 1503238553", got[0].Size)
	}
	if !strings.HasPrefix(got[0].Magnet, "magnet:?xt=urn:btih:") {
		t.Errorf("Magnet = %q", got[0].Magnet)
	}
}

func TestParseSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want uint64
	}{
		{"1.4 GiB", 1503238553},
		{"700 MiB", 734003200},
		{"12.3 GB", 12300000000},
		{"500 B", 500},
		{"1 TiB", 1099511627776},
	}
	for _, tt := range tests {
		got, err := parseSize(tt.in)
		if err != nil {
			t.Errorf("parseSize(%q) error = %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSize(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}

	if _, err := parseSize("not a size"); err == nil {
		t.Error("parseSize(\"not a size\") should error")
	}
}
