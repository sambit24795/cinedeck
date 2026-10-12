package library

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

func openTorrentsTestLibrary(t *testing.T) *Library {
	t.Helper()
	l, err := Open(filepath.Join(t.TempDir(), "library.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { l.Close() })
	return l
}

func TestSaveReleaseUpserts(t *testing.T) {
	ctx := context.Background()
	l := openTorrentsTestLibrary(t)

	r := indexer.Release{InfoHash: "abc", Title: "One", Provider: "apibay", Seeders: 5}
	if err := l.SaveRelease(ctx, r, "movie:1"); err != nil {
		t.Fatalf("SaveRelease: %v", err)
	}
	r.Seeders = 99
	if err := l.SaveRelease(ctx, r, "movie:1"); err != nil {
		t.Fatalf("SaveRelease (update): %v", err)
	}

	got, err := l.ReleasesFor(ctx, "movie:1")
	if err != nil {
		t.Fatalf("ReleasesFor: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 (upsert by hash)", len(got))
	}
	if got[0].Seeders != 99 {
		t.Errorf("Seeders = %d, want 99", got[0].Seeders)
	}
}

func TestReleasesForFiltersAndOrders(t *testing.T) {
	ctx := context.Background()
	l := openTorrentsTestLibrary(t)

	base := l.now()
	l.SetClock(func() time.Time { return base })
	if err := l.SaveRelease(ctx, indexer.Release{InfoHash: "h1", Title: "old", Provider: "p"}, "movie:1"); err != nil {
		t.Fatalf("SaveRelease h1: %v", err)
	}
	l.SetClock(func() time.Time { return base.Add(time.Hour) })
	if err := l.SaveRelease(ctx, indexer.Release{InfoHash: "h2", Title: "new", Provider: "p"}, "movie:1"); err != nil {
		t.Fatalf("SaveRelease h2: %v", err)
	}
	if err := l.SaveRelease(ctx, indexer.Release{InfoHash: "h3", Title: "other", Provider: "p"}, "movie:2"); err != nil {
		t.Fatalf("SaveRelease h3: %v", err)
	}

	got, err := l.ReleasesFor(ctx, "movie:1")
	if err != nil {
		t.Fatalf("ReleasesFor: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].InfoHash != "h2" {
		t.Errorf("newest first: got %q, want h2", got[0].InfoHash)
	}
}

func TestReleaseRoundTripSurvivesReopen(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	path := filepath.Join(dir, "library.db")

	l, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := l.SaveRelease(ctx, indexer.Release{InfoHash: "h1", Title: "x", Provider: "p", Seeders: 3}, "movie:1"); err != nil {
		t.Fatalf("SaveRelease: %v", err)
	}
	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	l2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer l2.Close()
	got, err := l2.ReleasesFor(ctx, "movie:1")
	if err != nil {
		t.Fatalf("ReleasesFor: %v", err)
	}
	if len(got) != 1 || got[0].InfoHash != "h1" {
		t.Fatalf("got %+v, want one entry h1", got)
	}
}

func TestSaveReleaseNoHash(t *testing.T) {
	ctx := context.Background()
	l := openTorrentsTestLibrary(t)
	if err := l.SaveRelease(ctx, indexer.Release{Title: "x"}, ""); err == nil {
		t.Fatal("expected an error for a release with no info hash")
	}
}

func TestSaveReleaseWithoutTMDBIDHidesFromTitle(t *testing.T) {
	ctx := context.Background()
	l := openTorrentsTestLibrary(t)
	if err := l.SaveRelease(ctx, indexer.Release{InfoHash: "h1", Title: "x", Provider: "p"}, ""); err != nil {
		t.Fatalf("SaveRelease: %v", err)
	}
	got, err := l.ReleasesFor(ctx, "")
	if err != nil {
		t.Fatalf("ReleasesFor: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0 (no title association)", len(got))
	}
}
