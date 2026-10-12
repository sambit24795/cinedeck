package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/indexer"
	"github.com/sambit24795/cinedeck/internal/library"
	"github.com/sambit24795/cinedeck/internal/recs"
	"github.com/spf13/cobra"
)

// newServeCmd starts a local web UI for search and torrent browsing.
func newServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a local web UI",
		Args:  cobra.NoArgs,
		RunE:  runServe,
	}
	addServeFlags(cmd)
	return cmd
}

// newRunCmd is an alias for serve, so `cinedeck run` starts the UI too.
func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Start the web UI (alias for serve)",
		Args:  cobra.NoArgs,
		RunE:  runServe,
	}
	addServeFlags(cmd)
	return cmd
}

// addServeFlags registers --port and --local-only.
func addServeFlags(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.Int("port", 8080, "port to listen on")
	flags.Bool("local-only", false, "bind to 127.0.0.1 only")
}

// runServe starts the HTTP server using the --port and --local-only flags.
func runServe(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")
	localOnly, _ := cmd.Flags().GetBool("local-only")

	addr := fmt.Sprintf("0.0.0.0:%d", port)
	if localOnly {
		addr = fmt.Sprintf("127.0.0.1:%d", port)
	}

	s := &server{
		catalog:  nil, // built lazily per request from config
		registry: defaultTorrentsRegistry(),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/search", s.handleSearch)
	mux.HandleFunc("/api/detail", s.handleDetail)
	mux.HandleFunc("/api/watch", s.handleWatch)
	mux.HandleFunc("/api/browse", s.handleBrowse)
	mux.HandleFunc("/api/genres", s.handleGenres)
	mux.HandleFunc("/api/torrents/title", s.handleTorrentsForTitle)
	mux.HandleFunc("/api/recommend/from", s.handleRecommendFrom)
	mux.HandleFunc("/api/recommend/personal", s.handleRecommendPersonal)
	mux.HandleFunc("/api/like", s.handleLike)
	mux.HandleFunc("/api/watchlist", s.handleWatchlist)
	mux.HandleFunc("/api/library", s.handleLibraryList)
	mux.HandleFunc("/api/torrents", s.handleTorrents)
	mux.HandleFunc("/api/free", s.handleFree)

	url := fmt.Sprintf("http://%s:%d", displayHost(addr, localOnly), port)
	fmt.Fprintf(cmd.OutOrStdout(), "cinedeck is running at %s\n", url)
	fmt.Fprintf(cmd.OutOrStdout(), "Press Ctrl+C to stop.\n")

	return http.ListenAndServe(addr, mux)
}

// displayHost returns the host to show users.
func displayHost(addr string, localOnly bool) string {
	if localOnly {
		return "localhost"
	}
	host := "localhost"
	return host
}

// server wires the CLI packages to the HTTP handlers.
type server struct {
	catalog  *catalog.Client
	registry *indexer.Registry
}

// getCatalog lazily builds a catalog client from the config.
func (s *server) getCatalog() (*catalog.Client, error) {
	if s.catalog != nil {
		return s.catalog, nil
	}
	cfg, err := configForServe()
	if err != nil {
		return nil, err
	}
	c := catalog.New(cfg.TMDBToken, catalog.WithBaseURL(cfg.TMDBURL))
	s.catalog = c
	return c, nil
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, indexHTML)
}

// resolveServeID resolves an id from the `id` query, or a title from `q`.
func (s *server) resolveServeID(r *http.Request) (catalog.ID, error) {
	if raw := r.URL.Query().Get("id"); raw != "" {
		return catalog.ParseID(raw)
	}
	q := r.URL.Query().Get("q")
	if q == "" {
		return catalog.ID{}, fmt.Errorf("missing id or q")
	}
	c, err := s.getCatalog()
	if err != nil {
		return catalog.ID{}, err
	}
	results, err := c.Search(r.Context(), q)
	if err != nil {
		return catalog.ID{}, err
	}
	if len(results) == 0 {
		return catalog.ID{}, fmt.Errorf("no match for %q", q)
	}
	return results[0].ID, nil
}

func (s *server) handleDetail(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveServeID(r)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	d, err := c.Details(r.Context(), id)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, d)
}

// handleWatch serves GET /api/watch?id=<id>&region=<cc>. It returns the
// where-to-watch groups for the configured region; no providers is not an error.
func (s *server) handleWatch(w http.ResponseWriter, r *http.Request) {
	id, err := s.resolveServeID(r)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	cfg, err := configForServe()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	region := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("region")))
	if region == "" {
		region = strings.ToUpper(strings.TrimSpace(cfg.Region))
	}

	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	wtw, err := c.WhereToWatch(r.Context(), id, region)
	if errors.Is(err, catalog.ErrNoProviders) {
		writeJSON(w, map[string]any{"region": region, "available": false})
		return
	}
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"region": region, "available": true, "result": wtw})
}

// handleGenres serves GET /api/genres?type=movie|tv and returns genre names.
func (s *server) handleGenres(w http.ResponseWriter, r *http.Request) {
	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	t := catalog.Movie
	if r.URL.Query().Get("type") == "tv" {
		t = catalog.TV
	}
	genres, err := c.Genres(r.Context(), t)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	names := make([]string, 0, len(genres))
	for name := range genres {
		names = append(names, name)
	}
	sort.Strings(names)
	writeJSON(w, names)
}

// handleBrowse serves GET /api/browse with a named preset and optional filters.
//
//	preset=trending|popular|top | kdrama | anime | shorts
//	type=movie|tv  genre=  lang=  country=  year=
func (s *server) handleBrowse(w http.ResponseWriter, r *http.Request) {
	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	mediaType := catalog.Movie
	if r.URL.Query().Get("type") == "tv" {
		mediaType = catalog.TV
	}
	genre := strings.TrimSpace(r.URL.Query().Get("genre"))
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	year := parseIntDefault(r.URL.Query().Get("year"), 0)
	preset := strings.TrimSpace(r.URL.Query().Get("preset"))

	var results []catalog.Result
	switch preset {
	case "trending":
		results, err = c.Trending(r.Context(), mediaType)
	case "shorts":
		results, err = c.Shorts(r.Context(), catalog.ShortsOptions{Genre: genre, Year: year})
	case "kdrama":
		results, err = c.Discover(r.Context(), catalog.DiscoverOptions{
			Type: catalog.TV, Genres: appendGenre(genre, "drama"), Language: "ko", MinVotes: 50, Year: year,
		})
	case "anime":
		results, err = c.Discover(r.Context(), catalog.DiscoverOptions{
			Type: mediaType, Genres: appendGenre(genre, "animation"), Language: "ja", MinVotes: 50, Year: year,
		})
	default: // "popular" / "" / any filter combination
		results, err = c.Discover(r.Context(), catalog.DiscoverOptions{
			Type: mediaType, Genres: oneGenre(genre), Language: lang, Country: country, Year: year,
		})
	}
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if results == nil {
		results = []catalog.Result{}
	}
	writeJSON(w, results)
}

// oneGenre returns a one-element genre slice, or nil when empty.
func oneGenre(g string) []string {
	if g == "" {
		return nil
	}
	return []string{g}
}

// appendGenre adds g to base when g is non-empty.
func appendGenre(g, base string) []string {
	if g == "" {
		return []string{base}
	}
	return []string{base, g}
}

// handleTorrentsForTitle serves GET /api/torrents/title?id=<id>&q=<title>&type=.
func (s *server) handleTorrentsForTitle(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		writeJSON(w, map[string]any{"error": "missing q"})
		return
	}
	cfg, err := configForServe()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	names := enabledProviderNames(cfg)
	if len(names) == 0 {
		names = []string{"apibay", "yts", "eztv", "nyaa", "archive"}
	}

	mediaType := indexer.Any
	if t := r.URL.Query().Get("type"); t == "movie" || t == "tv" {
		mediaType = indexer.MediaType(t)
	}

	i := serveIndexer(cfg)
	res, err := i.Search(r.Context(), indexer.Query{Text: q, Type: mediaType, Limit: 10}, s.registry.Enabled(names))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, map[string]any{"error": "missing q"})
		return
	}
	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	results, err := c.Search(r.Context(), q)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, results)
}

// handleRecommendFrom serves GET /api/recommend/from?from=<id|title>&type=&limit=.
func (s *server) handleRecommendFrom(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	if from == "" {
		writeJSON(w, map[string]any{"error": "missing from"})
		return
	}
	tf, err := recs.ParseTypeFilter(r.URL.Query().Get("type"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"))
	if err := recs.CheckLimit(limit); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	id, err := catalog.ParseID(from)
	if err != nil {
		id, err = s.resolveServeID(queryWithQ(r, from))
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
	}
	res, err := recs.From(r.Context(), c, id, recs.FromOptions{Type: tf, Limit: limit})
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	// The recs structs have no JSON tags, so return a lowercase shape for the UI.
	writeJSON(w, map[string]any{
		"source": res.Source,
		"picks":  res.Picks,
	})
}

// handleRecommendPersonal serves GET /api/recommend/personal?type=&limit=.
func (s *server) handleRecommendPersonal(w http.ResponseWriter, r *http.Request) {
	tf, err := recs.ParseTypeFilter(r.URL.Query().Get("type"))
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	limit := parseLimit(r.URL.Query().Get("limit"))
	if err := recs.CheckLimit(limit); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}

	lib, err := openLibraryIfExists()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	var entries []library.Entry
	if lib != nil {
		entries, err = lib.Entries(r.Context())
		lib.Close()
		if err != nil {
			writeJSON(w, map[string]any{"error": err.Error()})
			return
		}
	}

	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	res, err := recs.Personal(r.Context(), c, entries, recs.PersonalOptions{Type: tf, Limit: limit})
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{
		"picks":    res.Picks,
		"seeds":    res.Seeds,
		"requests": res.Requests,
	})
}

// likeRequest is the JSON body of /api/like and /api/watchlist.
type likeRequest struct {
	ID  string `json:"id"`
	Set bool   `json:"set"`
}

// handleLike serves POST /api/like with {"id":"movie:1","set":true}.
func (s *server) handleLike(w http.ResponseWriter, r *http.Request) {
	s.applyAction(w, r, func(lib *library.Library, id catalog.ID, set bool) (bool, error) {
		if set {
			return lib.Like(r.Context(), id)
		}
		return lib.Unlike(r.Context(), id)
	})
}

// handleWatchlist serves POST /api/watchlist with {"id":"movie:1","set":true}.
func (s *server) handleWatchlist(w http.ResponseWriter, r *http.Request) {
	s.applyAction(w, r, func(lib *library.Library, id catalog.ID, set bool) (bool, error) {
		if set {
			return lib.AddToWatchlist(r.Context(), id)
		}
		return lib.RemoveFromWatchlist(r.Context(), id)
	})
}

// applyAction parses the request body, upserts the item's metadata and runs fn.
func (s *server) applyAction(w http.ResponseWriter, r *http.Request, fn func(*library.Library, catalog.ID, bool) (bool, error)) {
	var req likeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, map[string]any{"error": "invalid JSON body"})
		return
	}
	id, err := catalog.ParseID(req.ID)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	c, err := s.getCatalog()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	m, err := c.Meta(r.Context(), id)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	lib, err := openLibrary()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	defer lib.Close()
	if err := lib.UpsertItem(r.Context(), m); err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	changed, err := fn(lib, id, req.Set)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"changed": changed})
}

// handleLibraryList serves GET /api/library?kind=watchlist|history|all&n=50.
func (s *server) handleLibraryList(w http.ResponseWriter, r *http.Request) {
	kind := r.URL.Query().Get("kind")
	if kind != "" && kind != "watchlist" && kind != "history" && kind != "all" {
		writeJSON(w, map[string]any{"error": "unknown kind: want watchlist, history or all"})
		return
	}

	lib, err := openLibraryIfExists()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if lib == nil {
		writeJSON(w, []library.Entry{})
		return
	}
	defer lib.Close()

	var entries []library.Entry
	switch kind {
	case "watchlist":
		entries, err = lib.Watchlist(r.Context())
	case "history":
		entries, err = lib.History(r.Context(), parseIntDefault(r.URL.Query().Get("n"), 50))
	default: // "all" or ""
		entries, err = lib.Entries(r.Context())
	}
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	if entries == nil {
		entries = []library.Entry{}
	}
	writeJSON(w, entries)
}

// queryWithQ returns a shallow copy of r with ?q=<value> appended.
func queryWithQ(r *http.Request, value string) *http.Request {
	r2 := r.Clone(r.Context())
	q := r2.URL.Query()
	q.Set("q", value)
	r2.URL.RawQuery = q.Encode()
	return r2
}

// parseLimit returns the limit query value, or recs.DefaultLimit.
func parseLimit(s string) int {
	return parseIntDefault(s, recs.DefaultLimit)
}

// parseIntDefault parses s, returning def when s is empty or invalid.
func parseIntDefault(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return def
	}
	return n
}

// serveIndexer builds an Indexer for the UI with the config timeout and the
// on-disk result cache (so repeated UI searches are cheap).
func serveIndexer(cfg config.Config) *indexer.Indexer {
	opts := []indexer.Option{indexer.WithTimeout(parseTimeout(cfg.IndexerTimeout))}
	if dir := torrentCacheDir(); dir != "" {
		opts = append(opts, indexer.WithCache(dir, time.Hour))
	}
	return indexer.NewIndexer(opts...)
}

func (s *server) handleTorrents(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, map[string]any{"error": "missing q"})
		return
	}
	cfg, err := configForServe()
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	names := enabledProviderNames(cfg)
	if len(names) == 0 {
		names = []string{"apibay", "yts", "eztv", "nyaa", "archive"}
	}
	providers := s.registry.Enabled(names)
	i := serveIndexer(cfg)
	res, err := i.Search(r.Context(), indexer.Query{Text: q}, providers)
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, res)
}

func (s *server) handleFree(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if q == "" {
		writeJSON(w, map[string]any{"error": "missing q"})
		return
	}
	a := newFreeArchiveProvider()
	results, err := a.Search(r.Context(), indexer.Query{Text: q})
	if err != nil {
		writeJSON(w, map[string]any{"error": err.Error()})
		return
	}
	writeJSON(w, results)
}

// configForServe loads the config for the serve process.
func configForServe() (config.Config, error) {
	path, err := config.Path()
	if err != nil {
		return config.Config{}, err
	}
	return config.Load(path)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(v)
	fmt.Fprint(w, string(data))
}

// indexHTML is the single-page UI, served as-is.
const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>cinedeck</title>
<style>
  :root { color-scheme: dark; --bg:#0d1017; --panel:#151a22; --line:#232a35; --text:#e6e8eb; --muted:#8b93a0; --accent:#3b82f6; }
  * { box-sizing: border-box; }
  body { font-family: system-ui, -apple-system, Segoe UI, sans-serif; background: var(--bg); color: var(--text); margin: 0; }
  header { display:flex; align-items:center; gap:16px; padding:14px 24px; border-bottom:1px solid var(--line); position:sticky; top:0; background:rgba(13,16,23,.95); backdrop-filter:blur(6px); z-index:5; }
  header h1 { font-size:18px; margin:0; letter-spacing:.3px; }
  header .spacer { flex:1; }
  main { max-width:1100px; margin:0 auto; padding:20px 24px 60px; }
  input, select, button { font:inherit; }
  input, select { width:100%; padding:12px 14px; border:1px solid var(--line); border-radius:9px; background:var(--panel); color:var(--text); }
  .row { display:flex; gap:10px; }
  .row > input { flex:1; }
  button { padding:10px 16px; border:1px solid var(--line); background:var(--panel); color:var(--text); border-radius:9px; cursor:pointer; }
  button.primary { background:var(--accent); border-color:var(--accent); color:#fff; }
  button.ghost { background:transparent; }
  button.tiny { padding:2px 8px; font-size:12px; border-radius:6px; }
  button:hover { border-color:#39424f; }
  .tabs { display:flex; gap:8px; margin:16px 0; flex-wrap:wrap; }
  .tabs button.active { background:var(--accent); border-color:var(--accent); color:#fff; }
  table { width:100%; border-collapse:collapse; margin-top:12px; }
  th, td { text-align:left; padding:9px 10px; border-bottom:1px solid var(--line); font-size:14px; vertical-align:top; }
  th { color:var(--muted); font-weight:500; font-size:12px; text-transform:uppercase; letter-spacing:.4px; }
  tr.clickable:hover { background:var(--panel); cursor:pointer; }
  .muted { color:var(--muted); }
  .err { color:#f28b82; margin-top:10px; }
  .empty { color:var(--muted); margin-top:16px; }
  a { color:#7ab8ff; text-decoration:none; }
  a:hover { text-decoration:underline; }
  .detail { display:grid; grid-template-columns:220px 1fr; gap:22px; margin-top:16px; }
  .detail img.poster { width:220px; border-radius:10px; background:var(--panel); }
  .detail h2 { margin:0 0 4px; font-size:24px; }
  .chips { display:flex; gap:6px; flex-wrap:wrap; margin:8px 0; }
  .chip { font-size:12px; padding:3px 9px; border:1px solid var(--line); border-radius:999px; color:var(--muted); }
  .clickchip { cursor:pointer; }
  .clickchip:hover { border-color:var(--accent); color:var(--text); }
  .cast { display:flex; gap:10px; flex-wrap:wrap; margin-top:8px; }
  .cast .person { width:82px; text-align:center; font-size:11px; color:var(--muted); }
  .cast img { width:66px; height:88px; object-fit:cover; border-radius:7px; background:var(--panel); }
  .actions { display:flex; gap:8px; margin:14px 0; }
  .badge { font-size:11px; padding:2px 7px; border-radius:999px; border:1px solid var(--line); color:var(--muted); }
  .reason { color:var(--muted); font-size:13px; }
  h3 { margin:20px 0 6px; font-size:15px; }
  @media (max-width:640px){ .detail { grid-template-columns:1fr; } .detail img.poster { width:140px; } }
</style>
</head>
<body>
<header>
  <h1>cinedeck</h1>
  <div class="spacer"></div>
  <button class="ghost" id="nav-library">Library</button>
</header>
<main>
  <div class="row">
    <input id="q" placeholder="Search movies and TV shows…" autofocus>
    <button class="primary" id="go">Search</button>
  </div>
  <div class="tabs">
    <button data-tab="search" class="active">Movies &amp; TV</button>
    <button data-tab="browse">Browse</button>
    <button data-tab="recommend">Recommend</button>
    <button data-tab="torrents">Torrents</button>
    <button data-tab="free">Internet Archive</button>
    <button data-tab="library">Library</button>
  </div>
  <div id="controls"></div>
  <div id="out"></div>
</main>
<script>
var tab = "search";
var q = document.getElementById("q");
var out = document.getElementById("out");
var controls = document.getElementById("controls");
var tabs = document.querySelectorAll(".tabs button");
var goBtn = document.getElementById("go");

function setTab(t) {
  tab = t;
  tabs.forEach(function(b){ b.classList.toggle("active", b.dataset.tab === t); });
  if (t === "browse") { showBrowseControls(); }
  else if (t === "recommend") { showRecommendControls(); }
  else if (t === "library") { run(); }
  else { controls.innerHTML = ""; if (q.value.trim()) run(); }
}
tabs.forEach(function(b){ b.onclick = function(){ setTab(b.dataset.tab); }; });
document.getElementById("nav-library").onclick = function(){ setTab("library"); };
goBtn.onclick = function(){ run(); };
q.onkeydown = function(e){ if (e.key === "Enter") run(); };

function esc(s) { return (s == null ? "" : String(s)).replace(/[&<>"]/g, function(c){ var m={"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}; return m[c]; }); }
function fmtSize(n) { if (!n) return "-"; var u=["B","kB","MB","GB","TB"], i=0; while (n >= 1000 && i < u.length-1){ n/=1000; i++; } return n.toFixed(1)+" "+u[i]; }
function fmtYear(y) { return y ? y : "-"; }
function fmtRating(r) { return r ? Number(r).toFixed(1) : "-"; }

function showBrowseControls() {
  controls.innerHTML =
    '<div class="chips" style="margin-top:10px">' +
      '<span class="chip clickchip" onclick="browse(\'trending\')">🔥 Trending</span>' +
      '<span class="chip clickchip" onclick="browse(\'popular\')">Popular</span>' +
      '<span class="chip clickchip" onclick="browse(\'kdrama\')">K-drama</span>' +
      '<span class="chip clickchip" onclick="browse(\'anime\')">Anime</span>' +
      '<span class="chip clickchip" onclick="browse(\'shorts\')">Shorts</span>' +
    '</div>' +
    '<div class="row" style="margin-top:8px;flex-wrap:wrap">' +
      '<select id="br-type" style="max-width:120px"><option value="movie">movies</option><option value="tv">tv</option></select>' +
      '<select id="br-genre" style="max-width:200px"><option value="">any genre</option></select>' +
      '<input id="br-lang" placeholder="lang (ko)" style="max-width:110px">' +
      '<input id="br-country" placeholder="country (KR)" style="max-width:130px">' +
      '<input id="br-year" placeholder="year" type="number" style="max-width:100px">' +
      '<button class="primary" id="br-go">Go</button>' +
    '</div>';
  document.getElementById("br-go").onclick = function(){ browse("popular"); };
  document.getElementById("br-type").onchange = loadGenres;
  loadGenres();
}

// loadGenres fills the genre dropdown for the selected type.
function loadGenres() {
  var type = document.getElementById("br-type").value;
  getJSON("/api/genres?type=" + type).then(function(names){
    var sel = document.getElementById("br-genre");
    if (!sel) return;
    var cur = sel.value;
    sel.innerHTML = '<option value="">any genre</option>';
    (names || []).forEach(function(n){ sel.innerHTML += '<option value="' + esc(n) + '">' + esc(n) + '</option>'; });
    sel.value = cur;
  });
}

// browse runs a browse preset with the current filters.
function browse(preset) {
  var type = document.getElementById("br-type").value;
  var genre = document.getElementById("br-genre").value;
  var lang = document.getElementById("br-lang").value.trim();
  var country = document.getElementById("br-country").value.trim();
  var year = document.getElementById("br-year").value.trim();
  var url = "/api/browse?preset=" + encodeURIComponent(preset) + "&type=" + encodeURIComponent(type) +
    "&genre=" + encodeURIComponent(genre) + "&lang=" + encodeURIComponent(lang) +
    "&country=" + encodeURIComponent(country) + "&year=" + encodeURIComponent(year);
  out.innerHTML = '<div class="empty">Loading…</div>';
  getJSON(url).then(function(data){
    if (data && data.error) { out.innerHTML = '<div class="err">' + esc(data.error) + '</div>'; return; }
    var label = preset === "popular" ? "Popular" : preset.charAt(0).toUpperCase() + preset.slice(1);
    out.innerHTML = '<div class="muted" style="margin-top:8px">' + esc(label) + (genre ? " · " + esc(genre) : "") + '</div>' + renderTitles(data);
  });
}

function showRecommendControls() {
  controls.innerHTML =
    '<div class="row" style="margin-top:8px">' +
      '<input id="rec-from" placeholder="From a title or id (e.g. Inception or movie:27205)">' +
      '<select id="rec-type" style="max-width:140px"><option value="any">movie + tv</option><option value="movie">movies</option><option value="tv">tv</option></select>' +
      '<button class="primary" id="rec-from-go">Go</button>' +
      '<button class="ghost" id="rec-personal">Personal</button>' +
    '</div>';
  document.getElementById("rec-from-go").onclick = recommendFrom;
  document.getElementById("rec-personal").onclick = recommendPersonal;
  document.getElementById("rec-from").onkeydown = function(e){ if (e.key === "Enter") recommendFrom(); };
}

async function getJSON(url) {
  var r = await fetch(url);
  return r.json();
}

async function run() {
  if (tab === "library") {
    out.innerHTML = '<div class="empty">Loading…</div>';
    var entries = await getJSON("/api/library?kind=all");
    if (entries && entries.error) { out.innerHTML = '<div class="err">' + esc(entries.error) + '</div>'; return; }
    out.innerHTML = renderLibrary(entries);
    return;
  }
  if (tab === "recommend") { return; }
  if (tab === "browse") { return; }
  var query = q.value.trim();
  if (!query) { return; }
  out.innerHTML = '<div class="empty">Searching…</div>';
  var ep = tab === "search" ? "/api/search" : tab === "torrents" ? "/api/torrents" : "/api/free";
  try {
    var data = await getJSON(ep + "?q=" + encodeURIComponent(query));
    if (data && data.error) { out.innerHTML = '<div class="err">' + esc(data.error) + '</div>'; return; }
    if (tab === "search") out.innerHTML = renderTitles(data);
    else if (tab === "torrents") { rememberMagnets(data.releases || []); out.innerHTML = renderTorrents(data); }
    else out.innerHTML = renderArchive(data);
  } catch (e) { out.innerHTML = '<div class="err">' + esc(e) + '</div>'; }
}

function renderTitles(rows) {
  if (!rows || !rows.length) return '<div class="empty">No results.</div>';
  var h = "<table><tr><th>Type</th><th>Title</th><th>Year</th><th>Rating</th><th>ID</th></tr>";
  rows.forEach(function(r){
    var t = (r.id || "").split(":")[0];
    h += '<tr class="clickable" onclick="openDetail(\'' + esc(r.id) + '\')"><td>' + esc(t) + '</td><td>' + esc(r.title) + '</td><td>' + fmtYear(r.year) + '</td><td>' + fmtRating(r.rating) + '</td><td class="muted">' + esc(r.id) + '</td></tr>';
  });
  return h + "</table>";
}

function renderArchive(rows) {
  if (!rows || !rows.length) return '<div class="empty">No results.</div>';
  var h = "<table><tr><th>Title</th><th>Year</th><th>Details</th></tr>";
  rows.forEach(function(r){ h += "<tr><td>" + esc(r.title) + "</td><td>" + fmtYear(r.year) + "</td><td class='muted'><a href='" + esc(r.info_url) + "' target='_blank'>archive.org</a></td></tr>"; });
  return h + "</table>";
}

function renderTorrents(d) {
  var rows = (d && d.releases) || [];
  var errs = (d && d.errors) || [];
  if (!rows.length) return '<div class="empty">No results.</div>' + renderErrors(errs);
  var h = "<table><tr><th>Provider</th><th>Title</th><th>Seeders</th><th>Leechers</th><th>Size</th><th>Magnet</th></tr>";
  rows.forEach(function(r){
    h += "<tr><td>" + esc(r.provider) + "</td><td>" + esc(r.title) + "</td><td>" + (r.seeders || "-") + "</td><td>" + (r.leechers || "-") + "</td><td>" + fmtSize(r.size) + "</td><td>" + magnetCell(r) + "</td></tr>";
  });
  return h + "</table>" + renderErrors(errs);
}

// magnetCell renders the magnet link plus a copy button for one release.
function magnetCell(r) {
  if (r.magnet) {
    return '<a href="' + esc(r.magnet) + '" title="Open in your torrent client">🧲 magnet</a> ' +
      '<button class="ghost tiny" onclick="copyMagnet(\'' + esc(r.info_hash) + '\', this)" title="Copy magnet link">copy</button>';
  }
  if (r.info_url) return '<a href="' + esc(r.info_url) + '" target="_blank">open</a>';
  return '<span class="muted">-</span>';
}

// magnetByHash holds the magnet links of the last rendered torrent list, keyed
// by info hash, so the copy button can look one up without escaping issues.
var magnetByHash = {};
function rememberMagnets(rows) { magnetByHash = {}; (rows || []).forEach(function(r){ if (r.magnet) magnetByHash[r.info_hash] = r.magnet; }); }
function copyMagnet(hash, el) {
  var m = magnetByHash[hash];
  if (!m) return;
  var done = function(){ if (el) { var o = el.textContent; el.textContent = "copied"; setTimeout(function(){ el.textContent = o; }, 1200); } };
  if (navigator.clipboard && navigator.clipboard.writeText) { navigator.clipboard.writeText(m).then(done, function(){ prompt("Magnet link", m); }); }
  else { prompt("Magnet link", m); }
}

function renderErrors(errs) {
  if (!errs || !errs.length) return "";
  var s = '<div class="muted" style="margin-top:10px">Provider issues:</div>';
  errs.forEach(function(e){ s += '<div class="err">' + esc(e.provider) + ": " + esc(e.error || e.err || "") + "</div>"; });
  return s;
}

function openDetail(id) {
  rememberList();
  out.innerHTML = '<div class="empty">Loading…</div>';
  getJSON("/api/detail?id=" + encodeURIComponent(id)).then(function(d){
    if (d && d.error) { out.innerHTML = '<div class="err">' + esc(d.error) + '</div>'; return; }
    out.innerHTML = renderDetail(d);
    wireDetailActions(d);
    loadWatch(d.id);
    loadTitleTorrents(d);
  });
}

// back button restores the last list view.
var lastList = { html: '', tab: 'search' };
function rememberList() { lastList = { html: out.innerHTML, tab: tab }; }

function renderDetail(d) {
  var h = '<div style="margin-bottom:10px"><button class="ghost" id="btn-back">← Back</button></div>';
  h += '<div class="detail">';
  h += '<div><img class="poster" src="' + esc(d.poster || "") + '" alt="" onerror="this.style.visibility=\'hidden\'"></div>';
  h += '<div>';
  h += '<h2>' + esc(d.title) + (d.year ? " (" + d.year + ")" : "") + '</h2>';
  if (d.tagline) h += '<div class="muted" style="font-style:italic">' + esc(d.tagline) + '</div>';
  h += '<div class="muted">Rating ' + fmtRating(d.rating) + (d.votes ? " (" + d.votes + " votes)" : "") + (d.runtime ? " · " + d.runtime + " min" : "") + (d.seasons ? " · " + d.seasons + " seasons" : "") + (d.episodes ? " · " + d.episodes + " episodes" : "") + (d.status ? " · " + esc(d.status) : "") + (d.released ? " · " + esc(d.released) : "") + '</div>';
  if (d.networks && d.networks.length) h += '<div class="muted">' + esc(d.networks.join(", ")) + '</div>';
  if (d.genres && d.genres.length) {
    h += '<div class="chips">';
    d.genres.forEach(function(g){ h += '<span class="chip">' + esc(g) + '</span>'; });
    h += '</div>';
  }
  var dir = (d.directors && d.directors.length) ? d.directors.join(", ") : (d.creators && d.creators.length ? d.creators.join(", ") : "");
  if (dir) h += '<div class="muted">' + esc(dir) + '</div>';
  h += '<div class="actions">';
  h += '<button id="btn-like" class="ghost">☆ Like</button>';
  h += '<button id="btn-watch" class="ghost">＋ Watchlist</button>';
  h += '<button class="ghost" id="btn-recs">Similar titles</button>';
  h += '<a class="badge" href="https://www.themoviedb.org/' + (d.id || "").split(":")[0] + '/' + (d.id || "").split(":")[1] + '" target="_blank">TMDB</a>';
  h += '</div>';
  if (d.overview) h += '<p>' + esc(d.overview) + '</p>';
  if (d.cast_details && d.cast_details.length) {
    h += '<div class="cast">';
    d.cast_details.forEach(function(c){
      h += '<div class="person"><img src="' + esc(c.photo || "") + '" onerror="this.style.visibility=\'hidden\'"><div>' + esc(c.name) + '</div>' + (c.character ? '<div class="muted">' + esc(c.character) + '</div>' : '') + '</div>';
    });
    h += '</div>';
  }
  h += '</div></div>';
  h += '<div id="watch-section"></div>';
  h += '<div id="torrent-section"></div>';
  return h;
}

// wireDetailActions attaches the detail view's buttons.
function wireDetailActions(d) {
  document.getElementById("btn-back").onclick = function(){ out.innerHTML = lastList.html; setTab(lastList.tab); };
  document.getElementById("btn-like").onclick = function(){ action("/api/like", d.id, this); };
  document.getElementById("btn-watch").onclick = function(){ action("/api/watchlist", d.id, this); };
  document.getElementById("btn-recs").onclick = function(){ setTab("recommend"); document.getElementById("rec-from").value = d.title; recommendFrom(); };
}

function loadWatch(id) {
  var el = document.getElementById("watch-section");
  if (!el) return;
  getJSON("/api/watch?id=" + encodeURIComponent(id)).then(function(res){
    if (!el) return;
    if (res && res.error) { el.innerHTML = ""; return; }
    if (!res.available) { el.innerHTML = '<h3>Where to watch</h3><div class="muted">No providers found for region ' + esc(res.region) + '.</div>'; return; }
    var w = res.result;
    var groups = [["Stream", w.stream], ["Free", w.free], ["With ads", w.ads], ["Rent", w.rent], ["Buy", w.buy]];
    var h = '<h3>Where to watch <span class="muted">(' + esc(res.region) + ')</span></h3>';
    var any = false;
    groups.forEach(function(g){
      if (g[1] && g[1].length) {
        any = true;
        h += '<div style="margin:6px 0"><span class="chip">' + g[0] + '</span> ';
        h += g[1].map(function(p){ return '<span class="muted">' + esc(p.name) + '</span>'; }).join(' · ');
        h += '</div>';
      }
    });
    if (!any) h += '<div class="muted">No providers found for region ' + esc(res.region) + '.</div>';
    if (w.link) h += '<div class="muted"><a href="' + esc(w.link) + '" target="_blank">TMDB watch page</a></div>';
    el.innerHTML = h;
  });
}

function loadTitleTorrents(d) {
  var el = document.getElementById("torrent-section");
  if (!el) return;
  var type = (d.id || "").split(":")[0];
  var title = d.title + (d.year ? " " + d.year : "");
  getJSON("/api/torrents/title?q=" + encodeURIComponent(title) + "&type=" + encodeURIComponent(type)).then(function(res){
    if (!el) return;
    if (!res || res.error) { el.innerHTML = '<h3>Torrents</h3><div class="muted">' + esc((res && res.error) || "unavailable") + '</div>'; return; }
    var rows = res.releases || [];
    rememberMagnets(rows);
    var h = '<h3>Torrents <span class="muted">(' + rows.length + ')</span></h3>';
    if (!rows.length) h += '<div class="muted">No releases found.</div>';
    else {
      h += "<table><tr><th>Provider</th><th>Title</th><th>Seeders</th><th>Leechers</th><th>Size</th><th>Magnet</th></tr>";
      rows.forEach(function(r){
        h += "<tr><td>" + esc(r.provider) + "</td><td>" + esc(r.title) + "</td><td>" + (r.seeders || "-") + "</td><td>" + (r.leechers || "-") + "</td><td>" + fmtSize(r.size) + "</td><td>" + magnetCell(r) + "</td></tr>";
      });
      h += "</table>";
    }
    el.innerHTML = h;
  });
}

function action(ep, id, el, set) {
  if (set === undefined) set = true;
  fetch(ep, { method:"POST", headers:{"Content-Type":"application/json"}, body: JSON.stringify({id:id, set:set}) })
    .then(function(r){ return r.json(); })
    .then(function(res){
      if (el && !res.error) el.textContent = (el.textContent || "").replace("✓ ", "") + (res.changed ? " ✓" : "");
      if (res.error) alert(res.error);
    });
}

function recommendFrom() {
  var from = document.getElementById("rec-from").value.trim();
  if (!from) return;
  var type = document.getElementById("rec-type").value;
  out.innerHTML = '<div class="empty">Finding titles…</div>';
  getJSON("/api/recommend/from?from=" + encodeURIComponent(from) + "&type=" + encodeURIComponent(type)).then(function(res){
    if (res && res.error) { out.innerHTML = '<div class="err">' + esc(res.error) + '</div>'; return; }
    out.innerHTML = renderPicks(res.picks || [], res.source);
  });
}

function recommendPersonal() {
  var type = document.getElementById("rec-type").value;
  out.innerHTML = '<div class="empty">Building your recommendations…</div>';
  getJSON("/api/recommend/personal?type=" + encodeURIComponent(type)).then(function(res){
    if (res && res.error) { out.innerHTML = '<div class="err">' + esc(res.error) + '</div>'; return; }
    if (res.seeds === 0) { out.innerHTML = '<div class="empty">Your library has no liked or highly rated titles yet. Open a title and like it (or rate it 8+).</div>'; return; }
    out.innerHTML = '<div class="muted">Based on ' + res.seeds + ' liked or highly rated title(s).</div>' + renderPicks(res.picks || [], null);
  });
}

function renderPicks(picks, source) {
  if (!picks.length) return '<div class="empty">No recommendations found.</div>';
  var h = source ? '<div class="muted" style="margin-top:8px">Titles like ' + esc(source.title) + (source.year ? " (" + source.year + ")" : "") + '</div>' : "";
  h += "<table><tr><th></th><th>Title</th><th>Year</th><th>Rating</th><th>Why</th></tr>";
  picks.forEach(function(p){
    h += '<tr class="clickable" onclick="openDetail(\'' + esc(p.id) + '\')">' +
      '<td><img src="' + esc(p.poster || "") + '" style="width:42px;border-radius:5px" onerror="this.style.visibility=\'hidden\'"></td>' +
      '<td>' + esc(p.title) + '</td><td>' + fmtYear(p.year) + '</td><td>' + fmtRating(p.rating) + '</td>' +
      '<td class="reason">' + esc(p.reason || "") + '</td></tr>';
  });
  return h + "</table>";
}

function renderLibrary(entries) {
  if (!entries || !entries.length) return '<div class="empty">Your library is empty. Like a title to get started.</div>';
  var h = "<table><tr><th>Type</th><th>Title</th><th>Year</th><th>Rating</th><th>Status</th></tr>";
  entries.forEach(function(e){
    var m = e.meta || e.Meta || {};
    var id = m.id || ((m.id && m.id.Type) ? "" : "");
    var flags = [];
    if (e.liked) flags.push("liked");
    if (e.rating) flags.push("rated " + e.rating);
    if (e.watched) flags.push("watched");
    if (e.on_watchlist) flags.push("watchlist");
    h += '<tr><td>' + esc((m.id || "").split(":")[0]) + '</td><td>' + esc(m.title) + '</td><td>' + fmtYear(m.year) + '</td><td>' + fmtRating(m.rating) + '</td><td class="muted">' + esc(flags.join(", ")) + '</td></tr>';
  });
  return h + "</table>";
}
</script>
</body>
</html>`
