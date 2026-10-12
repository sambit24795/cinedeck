package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/indexer/providers"
)

func TestServeIndex(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleIndex(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "<title>cinedeck</title>") {
		t.Errorf("index missing title:\n%s", body[:200])
	}
}

func TestServeFree(t *testing.T) {
	data, err := os.ReadFile("../../testdata/indexers/archive_nosferatu.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	fixture := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(data)
	}))
	t.Cleanup(fixture.Close)

	old := newFreeArchiveProvider
	newFreeArchiveProvider = func() *providers.Archive {
		a := providers.NewArchive()
		a.BaseURL = fixture.URL
		a.HTTP = fixture.Client()
		return a
	}
	t.Cleanup(func() { newFreeArchiveProvider = old })

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/free?q=nosferatu", nil)
	rec := httptest.NewRecorder()
	s.handleFree(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Content-Type"), "application/json") {
		t.Errorf("content-type = %q", rec.Header().Get("Content-Type"))
	}
	if !strings.Contains(rec.Body.String(), "Nosferatu") {
		t.Errorf("body missing results: %s", rec.Body.String())
	}
}
