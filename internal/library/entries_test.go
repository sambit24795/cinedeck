package library

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

var (
	idA = catalog.ID{Type: catalog.Movie, N: 1}
	idB = catalog.ID{Type: catalog.TV, N: 2}
	idC = catalog.ID{Type: catalog.Movie, N: 3}
)

func meta(id catalog.ID, title string) *catalog.Meta {
	return &catalog.Meta{
		ID:         id,
		Title:      title,
		Year:       2000 + int(id.N),
		Overview:   "overview of " + title,
		Language:   "en",
		Rating:     7.5,
		Votes:      100,
		Popularity: 1.5,
		GenreIDs:   []int{18},
		Genres:     []string{"Drama"},
		Keywords:   []catalog.Keyword{{ID: 1, Name: "test"}},
		Cast:       []catalog.Person{{ID: 10, Name: "Actor"}},
		Directors:  []catalog.Person{{ID: 11, Name: "Director"}},
	}
}

func TestEntry(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	stored := meta(idA, "A")
	if err := l.UpsertItem(ctx, stored); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	l.SetClock(fixedClock(100))
	if _, err := l.Like(ctx, idA); err != nil {
		t.Fatalf("Like: %v", err)
	}
	if _, err := l.Rate(ctx, idA, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}

	e, err := l.Entry(ctx, idA)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if !e.Liked {
		t.Error("Liked = false, want true")
	}
	if e.Rating != 9 {
		t.Errorf("Rating = %d, want 9", e.Rating)
	}
	if !e.Watched {
		t.Error("Watched = false, want true")
	}
	if e.OnWatchlist {
		t.Error("OnWatchlist = true, want false")
	}
	if e.LikedAt.IsZero() || e.LikedAt.Location() != time.UTC {
		t.Errorf("LikedAt = %v, want a non-zero UTC time", e.LikedAt)
	}
	if e.RatedAt.IsZero() || e.RatedAt.Location() != time.UTC {
		t.Errorf("RatedAt = %v, want a non-zero UTC time", e.RatedAt)
	}
	if !reflect.DeepEqual(e.Meta, *stored) {
		t.Errorf("Meta = %+v, want %+v", e.Meta, *stored)
	}
}

func TestEntryNotFound(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	_, err := l.Entry(ctx, idA)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestEntryNoFlags(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	stored := meta(idC, "C")
	if err := l.UpsertItem(ctx, stored); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	e, err := l.Entry(ctx, idC)
	if err != nil {
		t.Fatalf("Entry: %v", err)
	}
	if e.Liked || e.Watched || e.OnWatchlist || e.Rating != 0 {
		t.Errorf("Entry = %+v, want no flags", e)
	}
	if !reflect.DeepEqual(e.Meta, *stored) {
		t.Errorf("Meta = %+v, want %+v", e.Meta, *stored)
	}
}

func TestEntries(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	for _, id := range []catalog.ID{idA, idB, idC} {
		if err := l.UpsertItem(ctx, meta(id, id.String())); err != nil {
			t.Fatalf("UpsertItem: %v", err)
		}
	}
	if _, err := l.Like(ctx, idA); err != nil {
		t.Fatalf("Like: %v", err)
	}
	if _, err := l.AddToWatchlist(ctx, idB); err != nil {
		t.Fatalf("AddToWatchlist: %v", err)
	}
	if _, err := l.MarkWatched(ctx, idC); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}

	entries, err := l.Entries(ctx)
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	var got []catalog.ID
	for _, e := range entries {
		got = append(got, e.Meta.ID)
	}
	want := []catalog.ID{idA, idC, idB}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ids = %v, want %v", got, want)
	}
}

func TestEntriesEmpty(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	entries, err := l.Entries(ctx)
	if err != nil {
		t.Fatalf("Entries: %v", err)
	}
	if entries == nil {
		t.Error("Entries = nil, want non-nil empty slice")
	}
	if len(entries) != 0 {
		t.Errorf("len(Entries) = %d, want 0", len(entries))
	}
}

func TestWatchlistOrder(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	for _, id := range []catalog.ID{idA, idB} {
		if err := l.UpsertItem(ctx, meta(id, id.String())); err != nil {
			t.Fatalf("UpsertItem: %v", err)
		}
	}
	l.SetClock(fixedClock(50))
	if _, err := l.AddToWatchlist(ctx, idA); err != nil {
		t.Fatalf("AddToWatchlist A: %v", err)
	}
	if _, err := l.AddToWatchlist(ctx, idB); err != nil {
		t.Fatalf("AddToWatchlist B: %v", err)
	}

	entries, err := l.Watchlist(ctx)
	if err != nil {
		t.Fatalf("Watchlist: %v", err)
	}
	var got []catalog.ID
	for _, e := range entries {
		got = append(got, e.Meta.ID)
	}
	if want := []catalog.ID{idB, idA}; !reflect.DeepEqual(got, want) {
		t.Errorf("watchlist ids = %v, want %v", got, want)
	}

	// A watched item leaves the watchlist.
	if _, err := l.MarkWatched(ctx, idA); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	entries, err = l.Watchlist(ctx)
	if err != nil {
		t.Fatalf("Watchlist second: %v", err)
	}
	for _, e := range entries {
		if e.Meta.ID == idA {
			t.Error("watched item is still on the watchlist")
		}
	}
}

func TestHistory(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	for _, id := range []catalog.ID{idA, idB, idC} {
		if err := l.UpsertItem(ctx, meta(id, id.String())); err != nil {
			t.Fatalf("UpsertItem: %v", err)
		}
	}
	l.SetClock(fixedClock(1))
	if _, err := l.MarkWatched(ctx, idA); err != nil {
		t.Fatalf("MarkWatched A: %v", err)
	}
	if _, err := l.Rate(ctx, idB, 9); err != nil {
		t.Fatalf("Rate B: %v", err)
	}
	if _, err := l.MarkWatched(ctx, idC); err != nil {
		t.Fatalf("MarkWatched C: %v", err)
	}

	all, err := l.History(ctx, 0)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	var got []catalog.ID
	for _, e := range all {
		got = append(got, e.Meta.ID)
	}
	if want := []catalog.ID{idC, idB, idA}; !reflect.DeepEqual(got, want) {
		t.Errorf("history ids = %v, want %v", got, want)
	}

	limited, err := l.History(ctx, 2)
	if err != nil {
		t.Fatalf("History limit: %v", err)
	}
	if len(limited) != 2 {
		t.Fatalf("len(History(2)) = %d, want 2", len(limited))
	}
	if limited[0].Meta.ID != idC || limited[1].Meta.ID != idB {
		t.Errorf("History(2) = [%v %v], want [%v %v]", limited[0].Meta.ID, limited[1].Meta.ID, idC, idB)
	}
}

func TestHistoryTieBreak(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	for _, id := range []catalog.ID{idA, idB} {
		if err := l.UpsertItem(ctx, meta(id, id.String())); err != nil {
			t.Fatalf("UpsertItem: %v", err)
		}
	}
	l.SetClock(func() time.Time { return time.Unix(5, 0).UTC() })
	if _, err := l.MarkWatched(ctx, idB); err != nil {
		t.Fatalf("MarkWatched B: %v", err)
	}
	if _, err := l.MarkWatched(ctx, idA); err != nil {
		t.Fatalf("MarkWatched A: %v", err)
	}

	entries, err := l.History(ctx, 0)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("len(History) = %d, want 2", len(entries))
	}
	if entries[0].Meta.ID != idA || entries[1].Meta.ID != idB {
		t.Errorf("ids = [%v %v], want [%v %v]", entries[0].Meta.ID, entries[1].Meta.ID, idA, idB)
	}
}

func TestDisliked(t *testing.T) {
	tests := []struct {
		rating int
		want   bool
	}{
		{1, true}, {2, true}, {3, true}, {4, true},
		{0, false}, {5, false}, {6, false}, {7, false}, {8, false}, {9, false}, {10, false},
	}
	for _, tt := range tests {
		e := Entry{Rating: tt.rating}
		if got := e.Disliked(); got != tt.want {
			t.Errorf("Rating %d: Disliked = %v, want %v", tt.rating, got, tt.want)
		}
	}
}

func TestPositive(t *testing.T) {
	tests := []struct {
		name string
		e    Entry
		want bool
	}{
		{"liked no rating", Entry{Liked: true}, true},
		{"rating 8", Entry{Rating: 8}, true},
		{"rating 9", Entry{Rating: 9}, true},
		{"rating 10", Entry{Rating: 10}, true},
		{"rating 7", Entry{Rating: 7}, false},
		{"liked and rating 4", Entry{Liked: true, Rating: 4}, false},
		{"rating 3 not liked", Entry{Rating: 3}, false},
		{"nothing", Entry{}, false},
	}
	for _, tt := range tests {
		if got := tt.e.Positive(); got != tt.want {
			t.Errorf("%s: Positive = %v, want %v", tt.name, got, tt.want)
		}
	}
}
