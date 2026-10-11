package library

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// openTest opens a library in a temporary directory and closes it in cleanup.
func openTest(t *testing.T) *Library {
	t.Helper()
	l, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if err := l.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	return l
}

// inception returns a fully filled Meta for movie:27205.
func inception() *catalog.Meta {
	return &catalog.Meta{
		ID:         catalog.ID{Type: catalog.Movie, N: 27205},
		Title:      "Inception",
		Year:       2010,
		Overview:   "A thief who steals corporate secrets through dream-sharing technology.",
		Language:   "en",
		Rating:     8.373,
		Votes:      40395,
		Popularity: 51.8728,
		GenreIDs:   []int{28, 878},
		Genres:     []string{"Action", "Science Fiction"},
		Keywords:   []catalog.Keyword{{ID: 10364, Name: "mission"}, {ID: 1566, Name: "dreams"}},
		Cast:       []catalog.Person{{ID: 6193, Name: "Leonardo DiCaprio"}, {ID: 3895, Name: "Joseph Gordon-Levitt"}},
		Directors:  []catalog.Person{{ID: 525, Name: "Christopher Nolan"}},
	}
}

func TestUpsertItemRoundTrip(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	want := inception()
	if err := l.UpsertItem(ctx, want); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	got, err := l.Item(ctx, want.ID)
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Item = %+v, want %+v", got, want)
	}
}

func TestUpsertItemReplaces(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	first := inception()
	if err := l.UpsertItem(ctx, first); err != nil {
		t.Fatalf("UpsertItem first: %v", err)
	}

	second := inception()
	second.Title = "Inception (replaced)"
	second.Keywords = []catalog.Keyword{{ID: 1, Name: "heist"}}
	if err := l.UpsertItem(ctx, second); err != nil {
		t.Fatalf("UpsertItem second: %v", err)
	}

	got, err := l.Item(ctx, second.ID)
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got.Title != "Inception (replaced)" {
		t.Errorf("Title = %q, want %q", got.Title, "Inception (replaced)")
	}
	if !reflect.DeepEqual(got.Keywords, second.Keywords) {
		t.Errorf("Keywords = %v, want %v", got.Keywords, second.Keywords)
	}

	var count int
	if err := l.db.QueryRow("SELECT COUNT(*) FROM items").Scan(&count); err != nil {
		t.Fatalf("COUNT: %v", err)
	}
	if count != 1 {
		t.Errorf("item count = %d, want 1", count)
	}
}

func TestItemNilLists(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	m := &catalog.Meta{
		ID:    catalog.ID{Type: catalog.Movie, N: 1},
		Title: "Bare",
	}
	if err := l.UpsertItem(ctx, m); err != nil {
		t.Fatalf("UpsertItem: %v", err)
	}
	got, err := l.Item(ctx, m.ID)
	if err != nil {
		t.Fatalf("Item: %v", err)
	}
	if got.GenreIDs == nil {
		t.Error("GenreIDs = nil, want non-nil")
	}
	if got.Genres == nil {
		t.Error("Genres = nil, want non-nil")
	}
	if got.Keywords == nil {
		t.Error("Keywords = nil, want non-nil")
	}
	if got.Cast == nil {
		t.Error("Cast = nil, want non-nil")
	}
	if got.Directors == nil {
		t.Error("Directors = nil, want non-nil")
	}
}

func TestItemNotFound(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	_, err := l.Item(ctx, catalog.ID{Type: catalog.Movie, N: 999})
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
}

func TestUpsertItemInvalidType(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	m := &catalog.Meta{ID: catalog.ID{Type: catalog.MediaType("person"), N: 1}}
	err := l.UpsertItem(ctx, m)
	if err == nil {
		t.Fatal("UpsertItem = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid item type") {
		t.Errorf("error = %q, want it to contain %q", err, "invalid item type")
	}
}

func TestItemNonASCII(t *testing.T) {
	ctx := context.Background()
	l := openTest(t)

	titles := []string{"Amélie", "千と千尋の神隠し"}
	for i, title := range titles {
		m := &catalog.Meta{
			ID:    catalog.ID{Type: catalog.Movie, N: int64(i + 1)},
			Title: title,
		}
		if err := l.UpsertItem(ctx, m); err != nil {
			t.Fatalf("UpsertItem %q: %v", title, err)
		}
		got, err := l.Item(ctx, m.ID)
		if err != nil {
			t.Fatalf("Item %q: %v", title, err)
		}
		if got.Title != title {
			t.Errorf("Title = %q, want %q", got.Title, title)
		}
	}
}
