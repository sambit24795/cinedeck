package library

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// fixedClock returns a clock whose first call is start and that advances by
// one second on every call.
func fixedClock(start int64) func() time.Time {
	n := start
	return func() time.Time {
		t := time.Unix(n, 0).UTC()
		n++
		return t
	}
}

func countRows(t *testing.T, l *Library, table string) int {
	t.Helper()
	var n int
	if err := l.db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

func upsertInception(t *testing.T, l *Library) catalog.ID {
	t.Helper()
	m := inception()
	if err := l.UpsertItem(context.Background(), m); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	return m.ID
}

func TestStateRequiresItem(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name string
		run  func(l *Library, id catalog.ID) error
	}{
		{"Like", func(l *Library, id catalog.ID) error {
			_, err := l.Like(ctx, id)
			return err
		}},
		{"Rate", func(l *Library, id catalog.ID) error {
			_, err := l.Rate(ctx, id, 9)
			return err
		}},
		{"MarkWatched", func(l *Library, id catalog.ID) error {
			_, err := l.MarkWatched(ctx, id)
			return err
		}},
		{"AddToWatchlist", func(l *Library, id catalog.ID) error {
			_, err := l.AddToWatchlist(ctx, id)
			return err
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := openTest(t)
			id := catalog.ID{Type: catalog.Movie, N: 27205}
			err := tt.run(l, id)
			if !errors.Is(err, ErrNotFound) {
				t.Errorf("error = %v, want ErrNotFound", err)
			}
			for _, table := range []string{"likes", "ratings", "watched", "watchlist"} {
				if n := countRows(t, l, table); n != 0 {
					t.Errorf("%s rows = %d, want 0", table, n)
				}
			}
		})
	}
}

func TestLikeIdempotent(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)
	l.SetClock(fixedClock(1000))

	changed, err := l.Like(ctx, id)
	if err != nil {
		t.Fatalf("Like: %v", err)
	}
	if !changed {
		t.Error("first Like changed = false, want true")
	}

	changed, err = l.Like(ctx, id)
	if err != nil {
		t.Fatalf("Like second: %v", err)
	}
	if changed {
		t.Error("second Like changed = true, want false")
	}

	var likedAt int64
	if err := l.db.QueryRow("SELECT liked_at FROM likes").Scan(&likedAt); err != nil {
		t.Fatalf("liked_at: %v", err)
	}
	if likedAt != 1000 {
		t.Errorf("liked_at = %d, want 1000", likedAt)
	}
}

func TestUnlike(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	if _, err := l.Like(ctx, id); err != nil {
		t.Fatalf("Like: %v", err)
	}
	changed, err := l.Unlike(ctx, id)
	if err != nil {
		t.Fatalf("Unlike: %v", err)
	}
	if !changed {
		t.Error("Unlike changed = false, want true")
	}
	changed, err = l.Unlike(ctx, id)
	if err != nil {
		t.Fatalf("Unlike second: %v", err)
	}
	if changed {
		t.Error("second Unlike changed = true, want false")
	}

	changed, err = l.Unlike(ctx, catalog.ID{Type: catalog.Movie, N: 999})
	if err != nil {
		t.Fatalf("Unlike unknown: %v", err)
	}
	if changed {
		t.Error("Unlike unknown changed = true, want false")
	}
}

func TestRate(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	l.SetClock(fixedClock(2000))
	id := upsertInception(t, l)

	previous, err := l.Rate(ctx, id, 9)
	if err != nil {
		t.Fatalf("Rate 9: %v", err)
	}
	if previous != 0 {
		t.Errorf("previous = %d, want 0", previous)
	}

	previous, err = l.Rate(ctx, id, 6)
	if err != nil {
		t.Fatalf("Rate 6: %v", err)
	}
	if previous != 9 {
		t.Errorf("previous = %d, want 9", previous)
	}

	var rating int
	if err := l.db.QueryRow("SELECT rating FROM ratings").Scan(&rating); err != nil {
		t.Fatalf("rating: %v", err)
	}
	if rating != 6 {
		t.Errorf("rating = %d, want 6", rating)
	}
	if n := countRows(t, l, "watched"); n != 1 {
		t.Errorf("watched rows = %d, want 1", n)
	}
}

func TestRateInvalid(t *testing.T) {
	ctx := context.Background()
	for _, n := range []int{0, 11} {
		l := openTest(t)
		id := upsertInception(t, l)

		_, err := l.Rate(ctx, id, n)
		if err == nil {
			t.Fatalf("Rate(%d) = nil, want error", n)
		}
		if !strings.Contains(err.Error(), "rating must be between 1 and 10") {
			t.Errorf("error = %q, want it to contain %q", err, "rating must be between 1 and 10")
		}

		// The same error, not ErrNotFound, for an unknown id.
		_, err = l.Rate(ctx, catalog.ID{Type: catalog.Movie, N: 999}, n)
		if !strings.Contains(err.Error(), "rating must be between 1 and 10") {
			t.Errorf("unknown id error = %q, want it to contain %q", err, "rating must be between 1 and 10")
		}

		if got := countRows(t, l, "ratings"); got != 0 {
			t.Errorf("ratings rows = %d, want 0", got)
		}
		if got := countRows(t, l, "watched"); got != 0 {
			t.Errorf("watched rows = %d, want 0", got)
		}
	}
}

func TestRateRemovesWatchlist(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	if _, err := l.AddToWatchlist(ctx, id); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if _, err := l.Rate(ctx, id, 8); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if n := countRows(t, l, "watchlist"); n != 0 {
		t.Errorf("watchlist rows = %d, want 0", n)
	}
}

func TestRateKeepsWatchedAt(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)
	l.SetClock(fixedClock(3000))

	if _, err := l.MarkWatched(ctx, id); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if _, err := l.Rate(ctx, id, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}

	var watchedAt int64
	if err := l.db.QueryRow("SELECT watched_at FROM watched").Scan(&watchedAt); err != nil {
		t.Fatalf("watched_at: %v", err)
	}
	if watchedAt != 3000 {
		t.Errorf("watched_at = %d, want 3000", watchedAt)
	}
}

func TestUnrate(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	if _, err := l.Rate(ctx, id, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	changed, err := l.Unrate(ctx, id)
	if err != nil {
		t.Fatalf("Unrate: %v", err)
	}
	if !changed {
		t.Error("Unrate changed = false, want true")
	}
	changed, err = l.Unrate(ctx, id)
	if err != nil {
		t.Fatalf("Unrate second: %v", err)
	}
	if changed {
		t.Error("second Unrate changed = true, want false")
	}
	if n := countRows(t, l, "watched"); n != 1 {
		t.Errorf("watched rows = %d, want 1", n)
	}
}

func TestMarkWatched(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	changed, err := l.MarkWatched(ctx, id)
	if err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if !changed {
		t.Error("first MarkWatched changed = false, want true")
	}
	changed, err = l.MarkWatched(ctx, id)
	if err != nil {
		t.Fatalf("MarkWatched second: %v", err)
	}
	if changed {
		t.Error("second MarkWatched changed = true, want false")
	}

	// A watchlisted item is removed on the first and on a repeated call.
	for i := 0; i < 2; i++ {
		if _, err := l.AddToWatchlist(ctx, id); err != nil {
			t.Fatalf("AddToWatchlist: %v", err)
		}
		if _, err := l.MarkWatched(ctx, id); err != nil {
			t.Fatalf("MarkWatched: %v", err)
		}
		if n := countRows(t, l, "watchlist"); n != 0 {
			t.Fatalf("watchlist rows after call %d = %d, want 0", i+1, n)
		}
	}
}

func TestUnwatch(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	if _, err := l.Rate(ctx, id, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	changed, err := l.Unwatch(ctx, id)
	if err != nil {
		t.Fatalf("Unwatch: %v", err)
	}
	if !changed {
		t.Error("first Unwatch changed = false, want true")
	}
	changed, err = l.Unwatch(ctx, id)
	if err != nil {
		t.Fatalf("Unwatch second: %v", err)
	}
	if changed {
		t.Error("second Unwatch changed = true, want false")
	}
	if n := countRows(t, l, "ratings"); n != 1 {
		t.Errorf("ratings rows = %d, want 1", n)
	}
}

func TestWatchlistState(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	changed, err := l.AddToWatchlist(ctx, id)
	if err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if !changed {
		t.Error("first AddToWatchlist changed = false, want true")
	}
	changed, err = l.AddToWatchlist(ctx, id)
	if err != nil {
		t.Fatalf("AddToWatchlist second: %v", err)
	}
	if changed {
		t.Error("second AddToWatchlist changed = true, want false")
	}

	changed, err = l.RemoveFromWatchlist(ctx, id)
	if err != nil {
		t.Fatalf("RemoveFromWatchlist: %v", err)
	}
	if !changed {
		t.Error("first RemoveFromWatchlist changed = false, want true")
	}
	changed, err = l.RemoveFromWatchlist(ctx, id)
	if err != nil {
		t.Fatalf("RemoveFromWatchlist second: %v", err)
	}
	if changed {
		t.Error("second RemoveFromWatchlist changed = true, want false")
	}
}

func TestDeleteItemCascades(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)
	id := upsertInception(t, l)

	if _, err := l.Like(ctx, id); err != nil {
		t.Fatalf("Like: %v", err)
	}
	if _, err := l.Rate(ctx, id, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if _, err := l.AddToWatchlist(ctx, id); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if _, err := l.db.Exec("DELETE FROM items"); err != nil {
		t.Fatalf("DELETE: %v", err)
	}
	for _, table := range []string{"likes", "ratings", "watched", "watchlist"} {
		if n := countRows(t, l, table); n != 0 {
			t.Errorf("%s rows = %d, want 0", table, n)
		}
	}
}

func TestStateConstants(t *testing.T) {
	if DislikeMax != 4 {
		t.Errorf("DislikeMax = %d, want 4", DislikeMax)
	}
	if LikeMin != 8 {
		t.Errorf("LikeMin = %d, want 8", LikeMin)
	}
}
