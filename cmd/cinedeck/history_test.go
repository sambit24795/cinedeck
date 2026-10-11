package main

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestHistory(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.Rate(ctx, movieID, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if _, err := lib.MarkWatched(ctx, tvID); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if _, err := lib.Like(ctx, movieID); err != nil {
		t.Fatalf("Like: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out, err := runRootCmd(t, "history", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("output lines = %d, want 3:\n%s", len(lines), out)
	}
	for _, want := range []string{"WATCHED", "RATED", "LIKED"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("header = %q, want it to contain %q", lines[0], want)
		}
	}
	if !strings.Contains(lines[1], "Breaking Bad") {
		t.Errorf("first row = %q, want Breaking Bad first", lines[1])
	}
	if !strings.Contains(lines[2], "Inception") {
		t.Errorf("second row = %q, want Inception second", lines[2])
	}
	for _, want := range []string{"9", "yes"} {
		if !strings.Contains(lines[2], want) {
			t.Errorf("Inception row = %q, want it to contain %q", lines[2], want)
		}
	}
	// The date column also contains "-", so check the RATED and LIKED cells.
	cells := strings.Fields(lines[1])
	if len(cells) < 3 || cells[len(cells)-3] != "-" || cells[len(cells)-2] != "-" {
		t.Errorf("Breaking Bad row = %q, want rated and liked cells to be %q", lines[1], "-")
	}
}

func TestHistoryLimit(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.Rate(ctx, movieID, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if _, err := lib.MarkWatched(ctx, tvID); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out, err := runRootCmd(t, "history", "--limit", "1", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Breaking Bad") {
		t.Errorf("output = %q, want it to contain %q", out, "Breaking Bad")
	}
	if strings.Contains(out, "Inception") {
		t.Errorf("output = %q, want it not to contain %q", out, "Inception")
	}

	_, err = runRootCmd(t, "history", "--limit", "0", "--config", cfg)
	if err == nil {
		t.Fatal("limit 0 Execute = nil, want error")
	}
	if !strings.Contains(err.Error(), "invalid --limit 0") {
		t.Errorf("error = %q, want it to contain %q", err, "invalid --limit 0")
	}
}

func TestHistoryEmpty(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)

	out, err := runRootCmd(t, "history", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if want := "No watch history yet. Mark titles with: cinedeck watched <id|title>\n"; out != want {
		t.Errorf("stdout = %q, want %q", out, want)
	}

	out, err = runRootCmd(t, "history", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("json Execute: %v", err)
	}
	var rows []libraryRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if rows == nil {
		t.Error("rows = nil, want non-nil empty slice")
	}
	if len(rows) != 0 {
		t.Errorf("len(rows) = %d, want 0", len(rows))
	}
}

func TestHistoryJSON(t *testing.T) {
	cfg, _ := setupLibraryCmd(t)
	ctx := context.Background()

	lib := openTestLibrary(t)
	seedLibrary(t, lib)
	lib.SetClock(libraryDateClock())
	if _, err := lib.Rate(ctx, movieID, 9); err != nil {
		t.Fatalf("Rate: %v", err)
	}
	if _, err := lib.MarkWatched(ctx, tvID); err != nil {
		t.Fatalf("MarkWatched: %v", err)
	}
	if _, err := lib.Like(ctx, movieID); err != nil {
		t.Fatalf("Like: %v", err)
	}
	if err := lib.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	out, err := runRootCmd(t, "history", "--json", "--config", cfg)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	var rows []libraryRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	if rows[0].ID != tvID {
		t.Errorf("rows[0].ID = %v, want %v", rows[0].ID, tvID)
	}
	if rows[0].At != "2026-10-05T10:00:00Z" {
		t.Errorf("rows[0].At = %q, want %q", rows[0].At, "2026-10-05T10:00:00Z")
	}
	var inception *libraryRow
	for i := range rows {
		if rows[i].ID == movieID {
			inception = &rows[i]
		}
	}
	if inception == nil {
		t.Fatal("Inception row not found")
	}
	if inception.YourRating != 9 {
		t.Errorf("YourRating = %d, want 9", inception.YourRating)
	}
	if !inception.Liked {
		t.Error("Liked = false, want true")
	}
}
