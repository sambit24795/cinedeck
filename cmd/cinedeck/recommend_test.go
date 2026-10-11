package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/config"
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

// personalRoutes serves the M3b fixtures plus the detail pages used to enrich
// the personal recommendations.
func personalRoutes() map[string]string {
	r := recommendRoutes()
	r["/movie/9620"] = "meta_movie_9620.json"
	r["/tv/60059"] = "meta_tv_60059.json"
	return r
}

// setupPersonal points a test at a fresh data directory and the fixture server.
func setupPersonal(t *testing.T, paths *[]string) string {
	t.Helper()
	t.Setenv("CINEDECK_DATA_DIR", t.TempDir())
	srv := fixtureServer(t, personalRoutes(), paths)
	setCatalogEnv(t, srv)
	return filepath.Join(t.TempDir(), "none.toml")
}

const noTasteText = "Your library has no liked or highly rated titles yet.\n" +
	"Like some first, for example: cinedeck like <id|title>   (or rate one 8 or higher: cinedeck rate <id|title> 9)\n"

func TestRecommendPersonalEmpty(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	out, _, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != noTasteText {
		t.Errorf("stdout = %q, want %q", out, noTasteText)
	}
	if len(paths) != 0 {
		t.Errorf("paths = %v, want none", paths)
	}

	out, errOut, err := runRootCmdSplit(t, "recommend", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("json Execute: %v", err)
	}
	if out != "[]\n" {
		t.Errorf("json stdout = %q, want %q", out, "[]\n")
	}
	if errOut != noTasteText {
		t.Errorf("json stderr = %q, want %q", errOut, noTasteText)
	}
}

func TestRecommendPersonalEmptyCreatesNothing(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)
	dir := filepath.Join(t.TempDir(), "not-yet")
	t.Setenv("CINEDECK_DATA_DIR", dir)
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	for _, args := range [][]string{{"recommend"}, {"recommend", "--json"}} {
		if _, _, err := runRootCmdSplit(t, append(args, "--config", cfg)...); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("Stat(%s) = %v, want os.ErrNotExist (recommend must not create the data directory)", dir, err)
	}
}

func TestRecommendPersonalDislikedOnly(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	lib := openTestLibrary(t)
	ctx := context.Background()
	disliked := &catalog.Meta{ID: catalog.ID{Type: catalog.Movie, N: 1}, Title: "Disliked", GenreIDs: []int{878}}
	if err := lib.UpsertItem(ctx, disliked); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.Rate(ctx, disliked.ID, 3); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	watch := &catalog.Meta{ID: catalog.ID{Type: catalog.Movie, N: 2}, Title: "Watch", GenreIDs: []int{878}}
	if err := lib.UpsertItem(ctx, watch); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.AddToWatchlist(ctx, watch.ID); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}

	out, _, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != noTasteText {
		t.Errorf("stdout = %q, want %q", out, noTasteText)
	}
}

func TestRecommendPersonalTwoLikes(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil

	out, errOut, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Recommended for you, based on 2 liked or highly rated titles:\n") {
		t.Errorf("stdout = %q, want the two-title header", out)
	}
	rows := dataRows(out)
	if len(rows) == 0 || !strings.Contains(rows[0], "tv:60059") {
		t.Errorf("first row = %q, want tv:60059", firstOrEmpty(rows))
	}
	if !strings.Contains(rowFor(out, "tv:60059"), "because you liked Breaking Bad (shared: Vince Gilligan, new mexico, dark comedy)") {
		t.Errorf("Better Call Saul row = %q, want the shared-creator reason", rowFor(out, "tv:60059"))
	}
	if !strings.Contains(out, "Ghost in the Shell") {
		t.Errorf("stdout = %q, want it to contain Ghost in the Shell", out)
	}
	if !strings.Contains(rowFor(out, "Ghost in the Shell"), "because you liked Inception") {
		t.Errorf("Ghost in the Shell row = %q, want it to contain the Inception reason", rowFor(out, "Ghost in the Shell"))
	}
	if strings.Contains(out, "movie:27205") || strings.Contains(out, "tv:1396") {
		t.Errorf("stdout = %q, want it not to list the liked titles", out)
	}
	wantTip := "Tip: you have 2 liked or highly rated title(s); recommendations improve with more (like titles, or rate them 8 or higher).\n"
	if errOut != wantTip {
		t.Errorf("stderr = %q, want %q", errOut, wantTip)
	}
}

func firstOrEmpty(rows []string) string {
	if len(rows) == 0 {
		return ""
	}
	return rows[0]
}

func TestRecommendPersonalExclusion(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}

	lib := openTestLibrary(t)
	ctx := context.Background()
	state := &catalog.Meta{ID: catalog.ID{Type: catalog.Movie, N: 777443}, Title: "The Electric State", GenreIDs: []int{878, 12}}
	if err := lib.UpsertItem(ctx, state); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.MarkWatched(ctx, state.ID); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	push := &catalog.Meta{ID: catalog.ID{Type: catalog.Movie, N: 13455}, Title: "Push", GenreIDs: []int{878, 28}}
	if err := lib.UpsertItem(ctx, push); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.Rate(ctx, push.ID, 3); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "The Electric State") || strings.Contains(out, "Push") {
		t.Errorf("stdout = %q, want it to exclude the watched and disliked titles", out)
	}
	if !strings.Contains(out, "Ghost in the Shell") {
		t.Errorf("stdout = %q, want it to contain Ghost in the Shell", out)
	}
}

func TestRecommendPersonalWatchlistNotExcluded(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}

	lib := openTestLibrary(t)
	solo := &catalog.Meta{ID: catalog.ID{Type: catalog.Movie, N: 348350}, Title: "Solo: A Star Wars Story", GenreIDs: []int{878, 12, 28}}
	if err := lib.UpsertItem(context.Background(), solo); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	if _, err := lib.AddToWatchlist(context.Background(), solo.ID); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Solo: A Star Wars Story") {
		t.Errorf("stdout = %q, want it to contain the watchlist-only title", out)
	}
}

func TestRecommendPersonalFewLikes(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like: %v", err)
	}
	paths = nil

	out, errOut, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Recommended for you, based on 1 liked or highly rated title:\n") {
		t.Errorf("stdout = %q, want the one-title header", out)
	}
	if !strings.Contains(errOut, "Tip: you have 1 liked or highly rated title(s)") {
		t.Errorf("stderr = %q, want the one-seed tip", errOut)
	}
}

func TestRecommendPersonalRatingCounts(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "rate", "movie:27205", "9", "--config", cfg); err != nil {
		t.Fatalf("rate: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.HasPrefix(out, "Recommended for you, based on 1 liked or highly rated title:\n") {
		t.Errorf("stdout = %q, want the one-title header", out)
	}
	if !strings.Contains(out, "because you rated Inception 9/10") {
		t.Errorf("stdout = %q, want a rated-based reason", out)
	}
}

func TestRecommendPersonalRequestBound(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil

	if _, _, err := runRootCmdSplit(t, "recommend", "--config", cfg); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(paths) > recs.MaxRequests {
		t.Errorf("requests = %d, want <= %d", len(paths), recs.MaxRequests)
	}
	recCount, discCount := 0, 0
	for _, p := range paths {
		if strings.HasSuffix(p, "/recommendations") {
			recCount++
		}
		if strings.HasPrefix(p, "/discover/") {
			discCount++
		}
		if strings.HasPrefix(p, "/genre/") || strings.HasPrefix(p, "/search/") {
			t.Errorf("path = %q, want no genre or search request", p)
		}
	}
	if recCount != 2 {
		t.Errorf("recommendation paths = %d, want 2", recCount)
	}
	if discCount != 4 {
		t.Errorf("discover paths = %d, want 4", discCount)
	}
}

func TestRecommendPersonalTypeTV(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--type", "tv", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(out, "movie:") {
		t.Errorf("stdout = %q, want no movie rows", out)
	}
	for _, p := range paths {
		if strings.HasPrefix(p, "/discover/") && p != "/discover/tv" {
			t.Errorf("discover path = %q, want /discover/tv", p)
		}
		if p == "/movie/27205/recommendations" {
			t.Errorf("paths = %v, want no movie recommendations request", paths)
		}
	}
}

func TestRecommendPersonalTVJSON(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--type", "tv", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var got []recs.Pick
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", out, err)
	}
	if len(got) == 0 {
		t.Fatal("picks = empty, want at least one")
	}
	if got[0].ID != (catalog.ID{Type: catalog.TV, N: 60059}) {
		t.Errorf("got[0].ID = %v, want tv:60059", got[0].ID)
	}
	if !(got[0].Score > 0) || got[0].Reason == "" {
		t.Errorf("got[0] = %+v, want a positive score and a reason", got[0])
	}
}

func TestRecommendPersonalLimit(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil

	out, _, err := runRootCmdSplit(t, "recommend", "--limit", "3", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if rows := dataRows(out); len(rows) != 3 {
		t.Errorf("rows = %d, want 3", len(rows))
	}
}

func TestRecommendPersonalValidationUntouched(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	cases := []struct {
		args []string
		want string
	}{
		{[]string{"recommend", "--type", "show"}, `invalid --type "show"`},
		{[]string{"recommend", "--limit", "0"}, "invalid --limit"},
	}
	for _, tc := range cases {
		args := append(tc.args, "--config", cfg)
		_, err := runRootCmd(t, args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%v: err = %v, want it to contain %q", tc.args, err, tc.want)
		}
	}

	path, err := config.LibraryPath()
	if err != nil {
		t.Fatalf("LibraryPath: %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Errorf("Stat = %v, want os.ErrNotExist", statErr)
	}

	// With a seeded library and no token, a token check placed before the flag
	// validation would return the "config init" error instead of the flag error.
	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	t.Setenv("CINEDECK_TMDB_TOKEN", "")
	for _, tc := range cases {
		args := append(tc.args, "--config", cfg)
		_, err := runRootCmd(t, args...)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("seeded %v: err = %v, want it to contain %q", tc.args, err, tc.want)
		}
		if err != nil && strings.Contains(err.Error(), "config init") {
			t.Errorf("seeded %v: err = %v, want the flag error, not the config error", tc.args, err)
		}
	}
}

func TestRecommendPersonalTokenMissing(t *testing.T) {
	var paths []string
	cfg := setupPersonal(t, &paths)

	if _, err := runRootCmd(t, "like", "movie:27205", "--config", cfg); err != nil {
		t.Fatalf("like movie: %v", err)
	}
	if _, err := runRootCmd(t, "like", "tv:1396", "--config", cfg); err != nil {
		t.Fatalf("like tv: %v", err)
	}
	paths = nil
	// A typed id would not need the token, but a seed search does.
	t.Setenv("CINEDECK_TMDB_TOKEN", "")

	_, err := runRootCmd(t, "recommend", "--config", cfg)
	if err == nil || !strings.Contains(err.Error(), "config init") {
		t.Fatalf("err = %v, want it to contain %q", err, "config init")
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
