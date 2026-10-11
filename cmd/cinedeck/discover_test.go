package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// discoverServer serves testdata files by request path (404 for other paths)
// and appends the query of every request whose path starts with "/discover/" to queries.
func discoverServer(t *testing.T, routes map[string]string, queries *[]url.Values) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if queries != nil && strings.HasPrefix(r.URL.Path, "/discover/") {
			*queries = append(*queries, r.URL.Query())
		}
		name, ok := routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		body, err := os.ReadFile("../../testdata/" + name)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func discoverRoutes() map[string]string {
	return map[string]string{
		"/discover/tv":      "discover_tv_kdrama.json",
		"/discover/movie":   "discover_movie_anime.json",
		"/genre/tv/list":    "genre_tv_list.json",
		"/genre/movie/list": "genre_movie_list.json",
	}
}

func TestDiscoverCommand(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, discoverRoutes(), &queries)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "discover", "--type", "tv", "--lang", "ko", "--genre", "drama", "--year", "2024", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Squid Game", "tv:93405", "2021"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	for _, want := range [][2]string{{"with_original_language", "ko"}, {"with_genres", "18"}, {"first_air_date_year", "2024"}} {
		if got := q.Get(want[0]); got != want[1] {
			t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
		}
	}
	if q.Has("vote_count.gte") {
		t.Errorf("query = %v, want no vote_count.gte", q)
	}
}

func TestDiscoverCommandNormalizesFlags(t *testing.T) {
	var queries []url.Values
	// No genre route: a genre lookup would 404 and fail the command.
	srv := discoverServer(t, map[string]string{"/discover/tv": "discover_tv_kdrama.json"}, &queries)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "discover", "--type", "TV", "--lang", "KO", "--country", "kr", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1 (only /discover/tv)", queries)
	}
	q := queries[0]
	if got := q.Get("with_original_language"); got != "ko" {
		t.Errorf("with_original_language = %q, want %q", got, "ko")
	}
	if got := q.Get("with_origin_country"); got != "KR" {
		t.Errorf("with_origin_country = %q, want %q", got, "KR")
	}
	if q.Has("with_genres") {
		t.Errorf("query = %v, want no with_genres", q)
	}
}

func TestDiscoverCommandJSON(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, discoverRoutes(), &queries)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "discover", "--genre", "animation", "--lang", "ja", "--json", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	if got[0].ID != "movie:1311031" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "movie:1311031")
	}
	if got[0].Title != "Demon Slayer: Kimetsu no Yaiba Infinity Castle" {
		t.Errorf("got[0].Title = %q", got[0].Title)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	if got := queries[0].Get("with_genres"); got != "16" {
		t.Errorf("with_genres = %q, want %q", got, "16")
	}
}

func TestDiscoverCommandYear(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, discoverRoutes(), &queries)
	setCatalogEnv(t, srv)

	if _, err := runRootCmd(t, "discover", "--year", "2023", "--config", filepath.Join(t.TempDir(), "none.toml")); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	if got := q.Get("primary_release_year"); got != "2023" {
		t.Errorf("primary_release_year = %q, want %q", got, "2023")
	}
	if q.Has("first_air_date_year") {
		t.Errorf("query = %v, want no first_air_date_year", q)
	}
}

func TestDiscoverCommandNeedsAFilter(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, discoverRoutes(), &queries)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "discover", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "at least one filter") {
		t.Fatalf("err = %v, want it to contain %q", err, "at least one filter")
	}
	if len(queries) != 0 {
		t.Errorf("queries = %v, want no request", queries)
	}
}

func TestDiscoverCommandValidatesBeforeConfig(t *testing.T) {
	srv := discoverServer(t, discoverRoutes(), nil)
	setCatalogEnv(t, srv)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"discover", "--type", "person", "--lang", "ko"}, `invalid type "person"`},
		{[]string{"discover", "--lang", "korean"}, `invalid --lang "korean"`},
		{[]string{"discover", "--lang", "k1"}, `invalid --lang "k1"`},
		{[]string{"discover", "--country", "KOR"}, `invalid --country "KOR"`},
		{[]string{"discover", "--year", "1500"}, "invalid --year 1500"},
	}
	for _, tc := range cases {
		args := append(tc.args, "--config", filepath.Join(t.TempDir(), "none.toml"))
		_, err := runRootCmd(t, args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err = %v, want it to contain %q", tc.args, err, tc.want)
		}
		if err != nil && strings.Contains(err.Error(), "config init") {
			t.Errorf("%v: err = %v, want the flag error, not the config error", tc.args, err)
		}
	}
}

func TestDiscoverCommandUnknownGenre(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, discoverRoutes(), &queries)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "discover", "--type", "tv", "--genre", "action", "--lang", "ja", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), `unknown genre "action" for tv`) {
		t.Fatalf("err = %v, want it to contain unknown genre for tv", err)
	}
}

func TestDiscoverCommandNoToken(t *testing.T) {
	srv := discoverServer(t, discoverRoutes(), nil)
	setCatalogEnv(t, srv)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	_, err := runRootCmd(t, "discover", "--lang", "ko", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Fatalf("err = %v, want it to contain %q", err, "config init")
	}
}

func TestDiscoverHelpers(t *testing.T) {
	if got, err := normalizeLang(" KO "); err != nil || got != "ko" {
		t.Errorf("normalizeLang(%q) = %q, %v; want ko, nil", " KO ", got, err)
	}
	if got, err := normalizeLang(""); err != nil || got != "" {
		t.Errorf("normalizeLang(\"\") = %q, %v; want \"\", nil", got, err)
	}
	if got, err := normalizeCountry("kr"); err != nil || got != "KR" {
		t.Errorf("normalizeCountry(%q) = %q, %v; want KR, nil", "kr", got, err)
	}
	if got, err := parseMediaType(" TV "); err != nil || got != catalog.TV {
		t.Errorf("parseMediaType(%q) = %q, %v; want tv, nil", " TV ", got, err)
	}
	if err := checkYear(0); err != nil {
		t.Errorf("checkYear(0) = %v, want nil", err)
	}
	if err := checkYear(1999); err != nil {
		t.Errorf("checkYear(1999) = %v, want nil", err)
	}
	if err := checkYear(2101); err == nil {
		t.Errorf("checkYear(2101) = nil, want error")
	}
}
