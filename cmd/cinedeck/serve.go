package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
	"github.com/sambit24795/cinedeck/internal/indexer"
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
	i := indexer.NewIndexer(indexer.WithTimeout(parseTimeout(cfg.IndexerTimeout)))
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
  :root { color-scheme: dark; }
  * { box-sizing: border-box; }
  body { font-family: system-ui, -apple-system, sans-serif; background: #0f1115; color: #e6e8eb; margin: 0; padding: 24px; }
  h1 { font-size: 22px; margin: 0 0 16px; }
  input { width: 100%; padding: 12px; font-size: 16px; border: 1px solid #2a2e35; border-radius: 8px; background: #171a20; color: #e6e8eb; }
  .tabs { display: flex; gap: 8px; margin: 12px 0; }
  .tabs button { padding: 8px 16px; border: 1px solid #2a2e35; background: #171a20; color: #cbd0d6; border-radius: 6px; cursor: pointer; }
  .tabs button.active { background: #2b6cb0; border-color: #2b6cb0; color: #fff; }
  table { width: 100%; border-collapse: collapse; margin-top: 12px; }
  th, td { text-align: left; padding: 8px 10px; border-bottom: 1px solid #23272e; font-size: 14px; }
  th { color: #8b93a0; font-weight: 500; }
  .muted { color: #8b93a0; }
  .err { color: #f28b82; margin-top: 8px; }
</style>
</head>
<body>
<h1>cinedeck</h1>
<input id="q" placeholder="Search…" autofocus>
<div class="tabs">
  <button data-tab="search" class="active">Movies & TV</button>
  <button data-tab="torrents">Torrents</button>
  <button data-tab="free">Internet Archive</button>
</div>
<div id="out"></div>
<script>
let tab = "search";
const q = document.getElementById("q");
const out = document.getElementById("out");
const tabs = document.querySelectorAll(".tabs button");
tabs.forEach(b => b.onclick = () => { tab = b.dataset.tab; tabs.forEach(x => x.classList.remove("active")); b.classList.add("active"); run(); });

q.onkeydown = e => { if (e.key === "Enter") run(); };

async function run() {
  const query = q.value.trim();
  if (!query) return;
  out.innerHTML = '<div class="muted">Searching…</div>';
  const ep = tab === "search" ? "/api/search" : tab === "torrents" ? "/api/torrents" : "/api/free";
  try {
    const r = await fetch(ep + "?q=" + encodeURIComponent(query));
    const data = await r.json();
    if (data.error) { out.innerHTML = '<div class="err">' + data.error + '</div>'; return; }
    out.innerHTML = render(data);
  } catch (e) {
    out.innerHTML = '<div class="err">' + e + '</div>';
  }
}

function render(d) {
  if (d.releases) return renderTorrents(d);
  if (Array.isArray(d)) return d.length && d[0].title ? renderTitle(d) : renderArchive(d);
  return "<pre>" + JSON.stringify(d, null, 2) + "</pre>";
}

function esc(s) { return (s ?? "").replace(/[&<>\"]/g, function(c){ var m={"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;"}; return m[c]; }); }

function renderTitle(rows) {
  var h = "<table><tr><th>Type</th><th>Title</th><th>Year</th><th>Rating</th><th>ID</th></tr>";
  rows.forEach(function(r){ h += "<tr><td>" + esc(r.id && r.id.split(":")[0]) + "</td><td>" + esc(r.title) + "</td><td>" + (r.year || "-") + "</td><td>" + (r.rating ? r.rating.toFixed(1) : "-") + "</td><td class='muted'>" + esc(r.id) + "</td></tr>"; });
  return h + "</table>";
}

function renderArchive(rows) {
  var h = "<table><tr><th>Title</th><th>Year</th><th>Details</th></tr>";
  rows.forEach(function(r){ h += "<tr><td>" + esc(r.title) + "</td><td>" + (r.year || "-") + "</td><td class='muted'><a href='" + esc(r.info_url) + "' target='_blank'>details</a></td></tr>"; });
  return h + "</table>";
}

function renderTorrents(d) {
  var rows = d.releases || [];
  if (!rows.length) return '<div class="muted">No results.</div>';
  var h = "<table><tr><th>Provider</th><th>Title</th><th>Seeders</th><th>Leechers</th><th>Size</th><th>Hash</th></tr>";
  rows.forEach(function(r){ h += "<tr><td>" + esc(r.provider) + "</td><td>" + esc(r.title) + "</td><td>" + (r.seeders || "-") + "</td><td>" + (r.leechers || "-") + "</td><td>" + fmtSize(r.size) + "</td><td class='muted'>" + esc(r.info_hash) + "</td></tr>"; });
  return h + "</table>";
}

function fmtSize(n) {
  if (!n) return "-";
  const u = ["B","kB","MB","GB","TB"];
  let i = 0;
  while (n >= 1000 && i < u.length - 1) { n /= 1000; i++; }
  return n.toFixed(1) + " " + u[i];
}
</script>
</body>
</html>`
