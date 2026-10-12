package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/indexer/providers"
	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/sambit24795/cinedeck/internal/recs"
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

// serveDetailServer serves the detail fixtures.
func serveDetailServer(t *testing.T) *httptest.Server {
	t.Helper()
	return fixtureServer(t, map[string]string{
		"/movie/27205":  "movie_27205.json",
		"/tv/1399":      "tv_1399.json",
		"/search/multi": "search_multi.json",
	})
}

func TestServeDetailByID(t *testing.T) {
	srv := serveDetailServer(t)
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/detail?id=movie:27205", nil)
	rec := httptest.NewRecorder()
	s.handleDetail(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 (body=%s)", rec.Code, rec.Body.String())
	}
	var got catalog.Details
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if got.ID != (catalog.ID{Type: catalog.Movie, N: 27205}) {
		t.Errorf("ID = %v, want movie:27205", got.ID)
	}
}

func TestServeDetailByTitle(t *testing.T) {
	srv := serveDetailServer(t)
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/detail?q=inception", nil)
	rec := httptest.NewRecorder()
	s.handleDetail(rec, req)

	var got catalog.Details
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if got.ID.N != 27205 {
		t.Errorf("resolved ID = %v, want 27205", got.ID)
	}
}

func TestServeDetailMissing(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/detail", nil)
	rec := httptest.NewRecorder()
	s.handleDetail(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["error"] == nil || got["error"] == "" {
		t.Errorf("expected an error field, got %v", got)
	}
}

func TestServeRecommendFrom(t *testing.T) {
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/recommend/from?from=movie:27205&type=movie&limit=5", nil)
	rec := httptest.NewRecorder()
	s.handleRecommendFrom(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var got struct {
		Picks []recs.Pick `json:"picks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if len(got.Picks) == 0 {
		t.Fatalf("no picks returned: %s", rec.Body.String())
	}
	for _, p := range got.Picks {
		if p.ID.Type != catalog.Movie {
			t.Errorf("pick %v has type %q, want movie", p.ID, p.ID.Type)
		}
	}
}

func TestServeRecommendFromByTitle(t *testing.T) {
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/recommend/from?from=inception&type=any", nil)
	rec := httptest.NewRecorder()
	s.handleRecommendFrom(rec, req)

	var got struct {
		Picks []recs.Pick `json:"picks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if len(got.Picks) == 0 {
		t.Errorf("no picks for title form: %s", rec.Body.String())
	}
}

func TestServeRecommendPersonalEmptyLibrary(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/recommend/personal", nil)
	rec := httptest.NewRecorder()
	s.handleRecommendPersonal(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d (body=%s)", rec.Code, rec.Body.String())
	}
	var got struct {
		Seeds int         `json:"seeds"`
		Picks []recs.Pick `json:"picks"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if got.Seeds != 0 {
		t.Errorf("Seeds = %d, want 0", got.Seeds)
	}
	if got.Picks == nil {
		t.Error("Picks is null, want an empty slice")
	}
}

func TestServeRecommendPersonalBadType(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/recommend/personal?type=shows", nil)
	rec := httptest.NewRecorder()
	s.handleRecommendPersonal(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	msg, _ := got["error"].(string)
	if !strings.Contains(msg, "invalid") || !strings.Contains(msg, "type") {
		t.Errorf("error = %q, want it to mention an invalid type", msg)
	}
}

func postJSON(t *testing.T, s *server, path, body string) map[string]any {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	switch path {
	case "/api/like":
		s.handleLike(rec, req)
	case "/api/watchlist":
		s.handleWatchlist(rec, req)
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal(%s): %v", rec.Body.String(), err)
	}
	return got
}

func TestServeLike(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)
	s := &server{registry: defaultTorrentsRegistry()}

	got := postJSON(t, s, "/api/like", `{"id":"movie:27205","set":true}`)
	if got["changed"] != true {
		t.Fatalf("first like changed = %v, want true (%v)", got["changed"], got)
	}
	got = postJSON(t, s, "/api/like", `{"id":"movie:27205","set":true}`)
	if got["changed"] != false {
		t.Errorf("second like changed = %v, want false", got["changed"])
	}
}

func TestServeWatchlist(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)
	s := &server{registry: defaultTorrentsRegistry()}

	if got := postJSON(t, s, "/api/watchlist", `{"id":"tv:1396","set":true}`); got["changed"] != true {
		t.Fatalf("add to watchlist changed = %v, want true", got["changed"])
	}
	if got := postJSON(t, s, "/api/watchlist", `{"id":"tv:1396","set":false}`); got["changed"] != true {
		t.Errorf("remove from watchlist changed = %v, want true", got["changed"])
	}
}

func TestServeLikeInvalidID(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)
	s := &server{registry: defaultTorrentsRegistry()}

	got := postJSON(t, s, "/api/like", `{"id":"nope","set":true}`)
	msg, _ := got["error"].(string)
	if msg == "" {
		t.Errorf("expected an error for an invalid id, got %v", got)
	}
}

func TestServeLibraryList(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, recommendRoutes())
	setCatalogEnv(t, srv)
	s := &server{registry: defaultTorrentsRegistry()}

	// Seed one watchlist entry.
	postJSON(t, s, "/api/watchlist", `{"id":"movie:27205","set":true}`)

	req := httptest.NewRequest(http.MethodGet, "/api/library?kind=watchlist", nil)
	rec := httptest.NewRecorder()
	s.handleLibraryList(rec, req)

	var got []library.Entry
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
	if !got[0].OnWatchlist {
		t.Errorf("entry not flagged OnWatchlist: %+v", got[0])
	}
}

func TestServeLibraryListBadKind(t *testing.T) {
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/library?kind=nope", nil)
	rec := httptest.NewRecorder()
	s.handleLibraryList(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["error"] == nil {
		t.Errorf("expected an error for an unknown kind, got %v", got)
	}
}

func TestServeIndexHasRecommendAndLibrary(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleIndex(rec, req)
	body := rec.Body.String()
	for _, want := range []string{`data-tab="recommend"`, "Recommend", "/api/recommend/personal", `data-tab="library"`} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestServeIndexerNonNil(t *testing.T) {
	t.Setenv("CINEDECK_CACHE_DIR", t.TempDir())
	cfg := config.Config{IndexerTimeout: "5s"}
	if serveIndexer(cfg) == nil {
		t.Fatal("serveIndexer returned nil")
	}
}

func TestServeWatch(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/movie/27205/watch/providers": "watch_movie_27205.json",
	})
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/watch?id=movie:27205&region=IN", nil)
	rec := httptest.NewRecorder()
	s.handleWatch(rec, req)

	var got struct {
		Region    string `json:"region"`
		Available bool   `json:"available"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if got.Region != "IN" || !got.Available {
		t.Errorf("got %+v, want region IN and available", got)
	}
}

func TestServeWatchNoProviders(t *testing.T) {
	srv := fixtureServer(t, map[string]string{
		"/movie/27205/watch/providers": "watch_movie_27205.json",
	})
	setCatalogEnv(t, srv)

	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/watch?id=movie:27205&region=GB", nil)
	rec := httptest.NewRecorder()
	s.handleWatch(rec, req)

	var got struct {
		Region    string `json:"region"`
		Available bool   `json:"available"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v\n%s", err, rec.Body.String())
	}
	if got.Available {
		t.Errorf("expected unavailable for GB, got %+v", got)
	}
}

func TestServeTorrentsForTitleMissingQ(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/api/torrents/title", nil)
	rec := httptest.NewRecorder()
	s.handleTorrentsForTitle(rec, req)

	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got["error"] == nil {
		t.Errorf("expected an error for missing q, got %v", got)
	}
}

func TestServeIndexHasWatchAndTorrentSections(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleIndex(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"/api/watch", "/api/torrents/title", "Where to watch", "btn-back"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestServeIndexHasMagnetControls(t *testing.T) {
	s := &server{registry: defaultTorrentsRegistry()}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	s.handleIndex(rec, req)
	body := rec.Body.String()
	for _, want := range []string{"magnetCell", "copyMagnet", "rememberMagnets"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}
