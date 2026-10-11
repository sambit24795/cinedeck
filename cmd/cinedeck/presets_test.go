package main

import (
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
)

func kdramaRoutes() map[string]string {
	return map[string]string{
		"/discover/tv":   "discover_tv_kdrama.json",
		"/genre/tv/list": "genre_tv_list.json",
	}
}

func animeTVRoutes() map[string]string {
	return map[string]string{
		"/discover/tv":   "discover_tv_anime.json",
		"/genre/tv/list": "genre_tv_list.json",
	}
}

func animeMovieRoutes() map[string]string {
	return map[string]string{
		"/discover/movie":   "discover_movie_anime.json",
		"/genre/movie/list": "genre_movie_list.json",
	}
}

func TestKdramaCommand(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, kdramaRoutes(), &queries)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "kdrama", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Squid Game", "tv:93405"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	for _, want := range [][2]string{{"with_original_language", "ko"}, {"with_genres", "18"}, {"vote_count.gte", "50"}, {"sort_by", "popularity.desc"}} {
		if got := q.Get(want[0]); got != want[1] {
			t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
		}
	}
	if q.Has("with_origin_country") {
		t.Errorf("query = %v, want no with_origin_country", q)
	}
}

func TestKdramaCommandExtraGenreAndYear(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, kdramaRoutes(), &queries)
	setCatalogEnv(t, srv)

	if _, err := runRootCmd(t, "kdrama", "--genre", "comedy", "--year", "2024", "--config", filepath.Join(t.TempDir(), "none.toml")); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	if got := q.Get("with_genres"); got != "18,35" {
		t.Errorf("with_genres = %q, want %q", got, "18,35")
	}
	if got := q.Get("first_air_date_year"); got != "2024" {
		t.Errorf("first_air_date_year = %q, want %q", got, "2024")
	}
}

func TestKdramaCommandDuplicateGenre(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, kdramaRoutes(), &queries)
	setCatalogEnv(t, srv)

	if _, err := runRootCmd(t, "kdrama", "--genre", "Drama", "--config", filepath.Join(t.TempDir(), "none.toml")); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	if got := queries[0].Get("with_genres"); got != "18" {
		t.Errorf("with_genres = %q, want %q", got, "18")
	}
}

func TestKdramaCommandUnknownGenre(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, kdramaRoutes(), &queries)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "kdrama", "--genre", "romance", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), `unknown genre "romance" for tv`) {
		t.Fatalf("err = %v, want it to contain unknown genre for tv", err)
	}
}

func TestKdramaCommandBadYear(t *testing.T) {
	srv := discoverServer(t, kdramaRoutes(), nil)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "kdrama", "--year", "1500", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "invalid --year 1500") {
		t.Fatalf("err = %v, want it to contain invalid --year 1500", err)
	}
}

func TestKdramaCommandJSON(t *testing.T) {
	srv := discoverServer(t, kdramaRoutes(), nil)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "kdrama", "--json", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	if got[0].ID != "tv:93405" {
		t.Errorf("got[0].ID = %q, want %q", got[0].ID, "tv:93405")
	}
}

func TestAnimeCommand(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, animeTVRoutes(), &queries)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "anime", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Naruto", "tv:46260"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	for _, want := range [][2]string{{"with_original_language", "ja"}, {"with_genres", "16"}, {"vote_count.gte", "50"}} {
		if got := q.Get(want[0]); got != want[1] {
			t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
		}
	}
}

func TestAnimeCommandMovie(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, animeMovieRoutes(), &queries)
	setCatalogEnv(t, srv)

	out, err := runRootCmd(t, "anime", "--type", "movie", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"Spirited Away", "movie:129"} {
		if !strings.Contains(out, want) {
			t.Errorf("output = %q, want it to contain %q", out, want)
		}
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	q := queries[0]
	for _, want := range [][2]string{{"with_genres", "16"}, {"with_original_language", "ja"}, {"vote_count.gte", "50"}} {
		if got := q.Get(want[0]); got != want[1] {
			t.Errorf("query[%s] = %q, want %q", want[0], got, want[1])
		}
	}
	if q.Has("first_air_date_year") {
		t.Errorf("query = %v, want no first_air_date_year", q)
	}
}

func TestAnimeCommandMovieYear(t *testing.T) {
	var queries []url.Values
	srv := discoverServer(t, animeMovieRoutes(), &queries)
	setCatalogEnv(t, srv)

	if _, err := runRootCmd(t, "anime", "--type", "movie", "--year", "2023", "--config", filepath.Join(t.TempDir(), "none.toml")); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(queries) != 1 {
		t.Fatalf("queries = %v, want exactly 1", queries)
	}
	if got := queries[0].Get("primary_release_year"); got != "2023" {
		t.Errorf("primary_release_year = %q, want %q", got, "2023")
	}
}

func TestAnimeCommandBadType(t *testing.T) {
	srv := discoverServer(t, animeTVRoutes(), nil)
	setCatalogEnv(t, srv)

	_, err := runRootCmd(t, "anime", "--type", "person", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), `invalid type "person"`) {
		t.Fatalf("err = %v, want it to contain invalid type", err)
	}
}

func TestPresetsNoToken(t *testing.T) {
	srv := discoverServer(t, kdramaRoutes(), nil)
	setCatalogEnv(t, srv)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	_, err := runRootCmd(t, "kdrama", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Fatalf("kdrama err = %v, want it to contain %q", err, "config init")
	}

	_, err = runRootCmd(t, "anime", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Fatalf("anime err = %v, want it to contain %q", err, "config init")
	}
}

func TestPresetsInHelp(t *testing.T) {
	out, err := runRootCmd(t, "--help")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	for _, want := range []string{"discover", "kdrama", "anime"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output = %q, want it to contain %q", out, want)
		}
	}
}
