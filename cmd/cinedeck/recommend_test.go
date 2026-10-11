package main

import (
	"bytes"
	"encoding/json"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/recs"
)

func recommendRoutes() map[string]string {
	return map[string]string{
		"/search/multi":                "search_multi.json",
		"/movie/27205":                 "meta_movie_27205.json",
		"/tv/1396":                     "meta_tv_1396.json",
		"/movie/27205/recommendations": "related_recs_movie_27205.json",
		"/movie/27205/similar":         "related_similar_movie_27205.json",
		"/tv/1396/recommendations":     "related_recs_tv_1396.json",
		"/tv/1396/similar":             "related_similar_tv_1396.json",
		"/discover/tv":                 "discover_tv_keywords.json",
		"/discover/movie":              "discover_movie_keywords.json",
	}
}

// setupRecommend points the environment at a fixture server and returns a
// nonexistent config path and the discover query recorder.
func setupRecommend(t *testing.T) (string, *[]url.Values) {
	t.Helper()
	queries := &[]url.Values{}
	srv := discoverServer(t, recommendRoutes(), queries)
	setCatalogEnv(t, srv)
	return filepath.Join(t.TempDir(), "none.toml"), queries
}

// dataRows returns the output lines after the table header.
func dataRows(out string) []string {
	rows := []string{}
	seenHeader := false
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "TYPE") {
			seenHeader = true
			continue
		}
		if !seenHeader || strings.TrimSpace(line) == "" {
			continue
		}
		rows = append(rows, line)
	}
	return rows
}

// rowFor returns the first output line containing id.
func rowFor(out, id string) string {
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, id) {
			return line
		}
	}
	return ""
}

func TestRecommendFromTV(t *testing.T) {
	cfg, queries := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "tv:1396", "--type", "tv", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Titles like Breaking Bad (2008):\n") {
		t.Errorf("output = %q, want it to start with %q", out, "Titles like Breaking Bad (2008):\n")
	}

	wantIDs := []string{"tv:60059", "tv:80968", "tv:71694", "tv:63351", "tv:128904", "tv:66025", "tv:93533", "tv:241501"}
	last := -1
	for _, id := range wantIDs {
		idx := strings.Index(out, id)
		if idx < 0 {
			t.Errorf("output missing %q", id)
			continue
		}
		if idx < last {
			t.Errorf("%q appears out of order", id)
		}
		last = idx
	}
	if strings.Contains(out, "tv:1396") {
		t.Errorf("output = %q, want it not to list the source tv:1396", out)
	}
	if !strings.Contains(rowFor(out, "tv:60059"), "TMDB recommends it for Breaking Bad") {
		t.Errorf("Better Call Saul row = %q, want the recommendation reason", rowFor(out, "tv:60059"))
	}
	if !strings.Contains(rowFor(out, "tv:93533"), "similar to Breaking Bad") {
		t.Errorf("tv:93533 row = %q, want the similar reason", rowFor(out, "tv:93533"))
	}
	if len(*queries) != 0 {
		t.Errorf("discover queries = %v, want none", *queries)
	}
}

func TestRecommendFromMovieLimit(t *testing.T) {
	cfg, _ := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--type", "movie", "--limit", "3", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rows := dataRows(out)
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3", len(rows))
	}
	for i, id := range []string{"movie:9620", "movie:353491", "movie:13455"} {
		if !strings.Contains(rows[i], id) {
			t.Errorf("row %d = %q, want it to contain %q", i, rows[i], id)
		}
	}
}

func TestRecommendFromMovie(t *testing.T) {
	cfg, _ := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--type", "movie", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if rows := dataRows(out); len(rows) != 9 {
		t.Errorf("rows = %d, want 9", len(rows))
	}
}

func TestRecommendFromMovieToTV(t *testing.T) {
	cfg, queries := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--type", "tv", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(*queries) != 2 {
		t.Fatalf("discover queries = %v, want 2", *queries)
	}
	q0, q1 := (*queries)[0], (*queries)[1]
	for _, want := range [][2]string{
		{"with_keywords", "10364|1566|1930|470|1501"},
		{"with_genres", "10759|10765"},
		{"vote_count.gte", "100"},
	} {
		if got := q0.Get(want[0]); got != want[1] {
			t.Errorf("query 0 %s = %q, want %q", want[0], got, want[1])
		}
	}
	for _, want := range [][2]string{
		{"with_genres", "10759,10765"},
		{"vote_count.gte", "500"},
	} {
		if got := q1.Get(want[0]); got != want[1] {
			t.Errorf("query 1 %s = %q, want %q", want[0], got, want[1])
		}
	}

	rows := dataRows(out)
	wantIDs := []string{"tv:40075", "tv:4455", "tv:2473", "tv:95396"}
	if len(rows) != len(wantIDs) {
		t.Fatalf("rows = %d, want %d", len(rows), len(wantIDs))
	}
	for i, id := range wantIDs {
		if !strings.Contains(rows[i], id) {
			t.Errorf("row %d = %q, want it to contain %q", i, rows[i], id)
		}
		if !strings.Contains(rows[i], "shares keywords with Inception") {
			t.Errorf("row %d = %q, want the shares-keywords reason", i, rows[i])
		}
	}
}

func TestRecommendFromMovieAny(t *testing.T) {
	cfg, _ := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	movieIdx := strings.Index(out, "movie:9620")
	tvIdx := strings.Index(out, "tv:40075")
	if movieIdx < 0 || tvIdx < 0 {
		t.Fatalf("output missing movie or tv rows: %q", out)
	}
	if movieIdx > tvIdx {
		t.Errorf("movie row index %d > tv row index %d, want movie first", movieIdx, tvIdx)
	}
}

func TestRecommendFromTitleFirst(t *testing.T) {
	cfg, _ := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "inception", "--first", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Titles like Inception (2010):\n") {
		t.Errorf("output = %q, want it to start with %q", out, "Titles like Inception (2010):\n")
	}
}

func TestRecommendJSON(t *testing.T) {
	cfg, _ := setupRecommend(t)

	out, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--type", "movie", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []recs.Pick
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) != 9 {
		t.Fatalf("len = %d, want 9", len(got))
	}
	if got[0].ID != (catalog.ID{Type: catalog.Movie, N: 9620}) {
		t.Errorf("got[0].ID = %v, want movie:9620", got[0].ID)
	}
	if got[0].Reason == "" {
		t.Error("got[0].Reason = empty, want a reason")
	}
}

func TestRecommendValidationBeforeConfig(t *testing.T) {
	srv := discoverServer(t, recommendRoutes(), nil)
	setCatalogEnv(t, srv)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"recommend", "--from", "movie:27205", "--type", "show"}, `invalid --type "show"`},
		{[]string{"recommend", "--from", "movie:27205", "--limit", "0"}, "invalid --limit"},
		{[]string{"recommend", "--from", "movie:27205", "--limit", "51"}, "invalid --limit"},
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

	_, err := runRootCmd(t, "recommend", "--from", "movie:27205", "--config", filepath.Join(t.TempDir(), "none.toml"))
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Errorf("err = %v, want it to contain %q", err, "config init")
	}
}

func TestRecommendNotFound(t *testing.T) {
	cfg, _ := setupRecommend(t)

	_, err := runRootCmd(t, "recommend", "--from", "movie:99999", "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "tmdb: 404") {
		t.Fatalf("err = %v, want it to contain %q", err, "tmdb: 404")
	}
}

func TestRecommendPersonalNotImplemented(t *testing.T) {
	cfg, _ := setupRecommend(t)

	_, err := runRootCmd(t, "recommend", "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "personal recommendations are not implemented yet") {
		t.Fatalf("err = %v, want the temporary personal-recommendations error", err)
	}
}

func TestRecommendInHelp(t *testing.T) {
	cmd := newRootCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out.String(), "recommend") {
		t.Errorf("help = %q, want it to list recommend", out.String())
	}
}
