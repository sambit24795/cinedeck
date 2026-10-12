package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
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
	var port int
	var localOnly bool
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start a local web UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
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
		},
	}
	cmd.Flags().IntVar(&port, "port", 8080, "port to listen on")
	cmd.Flags().BoolVar(&localOnly, "local-only", false, "bind to 127.0.0.1 only")
	return cmd
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
  .cast { display:flex; gap:10px; flex-wrap:wrap; margin-top:8px; }
  .cast .person { width:82px; text-align:center; font-size:11px; color:var(--muted); }
  .cast img { width:66px; height:88px; object-fit:cover; border-radius:7px; background:var(--panel); }
  .actions { display:flex; gap:8px; margin:14px 0; }
  .badge { font-size:11px; padding:2px 7px; border-radius:999px; border:1px solid var(--line); color:var(--muted); }
  .reason { color:var(--muted); font-size:13px; }
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
  if (t === "recommend") { showRecommendControls(); }
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
  var query = q.value.trim();
  if (!query) { return; }
  out.innerHTML = '<div class="empty">Searching…</div>';
  var ep = tab === "search" ? "/api/search" : tab === "torrents" ? "/api/torrents" : "/api/free";
  try {
    var data = await getJSON(ep + "?q=" + encodeURIComponent(query));
    if (data && data.error) { out.innerHTML = '<div class="err">' + esc(data.error) + '</div>'; return; }
    if (tab === "search") out.innerHTML = renderTitles(data);
    else if (tab === "torrents") out.innerHTML = renderTorrents(data);
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
  var h = "<table><tr><th>Provider</th><th>Title</th><th>Seeders</th><th>Leechers</th><th>Size</th><th>Hash</th></tr>";
  rows.forEach(function(r){
    h += "<tr><td>" + esc(r.provider) + "</td><td>" + esc(r.title) + "</td><td>" + (r.seeders || "-") + "</td><td>" + (r.leechers || "-") + "</td><td>" + fmtSize(r.size) + "</td><td class='muted'>" + esc(r.info_hash) + "</td></tr>";
  });
  return h + "</table>" + renderErrors(errs);
}

function renderErrors(errs) {
  if (!errs || !errs.length) return "";
  var s = '<div class="muted" style="margin-top:10px">Provider issues:</div>';
  errs.forEach(function(e){ s += '<div class="err">' + esc(e.provider) + ": " + esc(e.error || e.err || "") + "</div>"; });
  return s;
}

function openDetail(id) {
  out.innerHTML = '<div class="empty">Loading…</div>';
  getJSON("/api/detail?id=" + encodeURIComponent(id)).then(function(d){
    if (d && d.error) { out.innerHTML = '<div class="err">' + esc(d.error) + '</div>'; return; }
    out.innerHTML = renderDetail(d);
  });
}

function renderDetail(d) {
  var h = '<div class="detail">';
  h += '<div><img class="poster" src="' + esc(d.poster || "") + '" alt="" onerror="this.style.visibility=\'hidden\'"></div>';
  h += '<div>';
  h += '<h2>' + esc(d.title) + (d.year ? " (" + d.year + ")" : "") + '</h2>';
  h += '<div class="muted">Rating ' + fmtRating(d.rating) + (d.runtime ? " · " + d.runtime + " min" : "") + (d.seasons ? " · " + d.seasons + " seasons" : "") + (d.status ? " · " + esc(d.status) : "") + '</div>';
  if (d.genres && d.genres.length) {
    h += '<div class="chips">';
    d.genres.forEach(function(g){ h += '<span class="chip">' + esc(g) + '</span>'; });
    h += '</div>';
  }
  var dir = (d.directors && d.directors.length) ? d.directors.join(", ") : (d.creators && d.creators.length ? d.creators.join(", ") : "");
  if (dir) h += '<div class="muted">' + esc(dir) + '</div>';
  h += '<div class="actions">';
  h += '<button id="btn-like" class="ghost">Like</button>';
  h += '<button id="btn-watch" class="ghost">Watchlist</button>';
  h += '<a class="badge" href="https://www.themoviedb.org/' + (d.id || "").split(":")[0] + '/' + (d.id || "").split(":")[1] + '" target="_blank">TMDB</a>';
  h += '</div>';
  if (d.overview) h += '<p>' + esc(d.overview) + '</p>';
  if (d.cast_details && d.cast_details.length) {
    h += '<div class="cast">';
    d.cast_details.forEach(function(c){
      h += '<div class="person"><img src="' + esc(c.photo || "") + '" onerror="this.style.visibility=\'hidden\'"><div>' + esc(c.name) + '</div>' + (c.character ? '<div>' + esc(c.character) + '</div>' : '') + '</div>';
    });
    h += '</div>';
  }
  h += '</div></div>';

  document.getElementById("btn-like").onclick = function(){ action("/api/like", d.id, this); };
  document.getElementById("btn-watch").onclick = function(){ action("/api/watchlist", d.id, this); };
  return h;
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
