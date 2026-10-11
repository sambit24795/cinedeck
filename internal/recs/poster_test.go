package recs

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/sambit24795/cinedeck/internal/library"
)

const testPoster = "https://image.tmdb.org/t/p/w342/x.jpg"

func TestFromCopiesPoster(t *testing.T) {
	ctx := context.Background()
	id := movieID(1)
	m := meta(id, "X", []int{28})
	rec := rel(movieID(2), "R2", 5000)
	rec.Poster = testPoster
	src := &fakeSource{
		metas: map[catalog.ID]*catalog.Meta{id: m},
		recs:  map[catalog.ID][]catalog.Related{id: {rec}},
	}

	res, err := From(ctx, src, id, FromOptions{Type: catalog.Movie, Limit: 10})
	if err != nil {
		t.Fatalf("From: %v", err)
	}
	if len(res.Picks) != 1 {
		t.Fatalf("len(Picks) = %d, want 1", len(res.Picks))
	}
	if res.Picks[0].Poster != testPoster {
		t.Errorf("Poster = %q, want %q", res.Picks[0].Poster, testPoster)
	}
}

func TestPersonalCopiesPoster(t *testing.T) {
	ctx := context.Background()
	seed := likedEntry(movieID(1), "Seed", []int{878}, catalog.Keyword{ID: 5, Name: "k5"})
	rec := relG(movieID(10), "R1", 900, 878)
	rec.Poster = testPoster
	src := &fakeSource{
		recs: map[catalog.ID][]catalog.Related{movieID(1): {rec}},
	}

	res, err := Personal(ctx, src, []library.Entry{seed}, PersonalOptions{Limit: 10})
	if err != nil {
		t.Fatalf("Personal: %v", err)
	}
	if len(res.Picks) != 1 {
		t.Fatalf("len(Picks) = %d, want 1", len(res.Picks))
	}
	if res.Picks[0].Poster != testPoster {
		t.Errorf("Poster = %q, want %q", res.Picks[0].Poster, testPoster)
	}
}

func TestPickEmptyPosterOmitted(t *testing.T) {
	b, err := json.Marshal(Pick{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "poster") {
		t.Errorf("json = %s, want it not to contain %q", b, "poster")
	}
}
