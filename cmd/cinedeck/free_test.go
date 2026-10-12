package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer"
	"github.com/sambit24795/cinedeck/internal/indexer/providers"
)

func TestFreeRows(t *testing.T) {
	rels := []indexer.Release{
		{Title: "Nosferatu (1922)", Year: 1922, InfoURL: "https://archive.org/details/nosferatu1922"},
		{Title: "No Year", InfoURL: "https://archive.org/details/noyear"},
	}
	rows := freeRows(rels)
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0][0] != "Nosferatu (1922)" || rows[0][1] != "1922" || rows[0][2] != "https://archive.org/details/nosferatu1922" {
		t.Errorf("rows[0] = %+v", rows[0])
	}
	if rows[1][1] != "-" {
		t.Errorf("rows[1][1] = %q, want \"-\" for a 0 year", rows[1][1])
	}
}

func TestFreeRowsEmpty(t *testing.T) {
	if got := freeRows(nil); len(got) != 0 {
		t.Errorf("freeRows(nil) = %d rows, want 0", len(got))
	}
}

func TestFreeCommand(t *testing.T) {
	data, err := os.ReadFile("../../testdata/indexers/archive_nosferatu.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(srv.Close)

	old := newFreeArchiveProvider
	newFreeArchiveProvider = func() *providers.Archive {
		a := providers.NewArchive()
		a.BaseURL = srv.URL
		a.HTTP = srv.Client()
		return a
	}
	t.Cleanup(func() { newFreeArchiveProvider = old })

	cmd := newRootCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetIn(strings.NewReader(""))
	cmd.SetArgs([]string{"free", "nosferatu"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "Nosferatu") || !strings.Contains(got, "1922") {
		t.Errorf("output missing results:\n%s", got)
	}
}
