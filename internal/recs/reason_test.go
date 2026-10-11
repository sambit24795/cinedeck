package recs

import (
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func reasonSeeds() (Seed, Seed) {
	a := Seed{
		Meta: catalog.Meta{
			ID:        movieID(1),
			Title:     "Alpha",
			GenreIDs:  []int{878, 28},
			Keywords:  []catalog.Keyword{{ID: 1, Name: "dreams"}, {ID: 2, Name: "heist"}},
			Directors: []catalog.Person{{ID: 10, Name: "Nolan"}},
			Cast:      []catalog.Person{{ID: 100, Name: "Leo"}},
		},
		Weight: 1,
		Liked:  true,
	}
	b := Seed{
		Meta: catalog.Meta{
			ID:        tvID(2),
			Title:     "Beta",
			GenreIDs:  []int{10759},
			Keywords:  []catalog.Keyword{{ID: 2, Name: "heist"}},
			Directors: []catalog.Person{{ID: 11, Name: "Gil"}},
		},
		Weight: 0.9,
		Rating: 9,
	}
	return a, b
}

func TestReason(t *testing.T) {
	a, b := reasonSeeds()
	seeds := []Seed{a, b}

	t.Run("recBy", func(t *testing.T) {
		c1 := Candidate{RecBy: []catalog.ID{tvID(2)}, Genres: []string{"action"}, Keywords: []int{2}}
		want := "because you rated Beta 9/10 (shared: heist, Action)"
		if got := Reason(seeds, c1); got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("shared features", func(t *testing.T) {
		c2 := Candidate{Genres: []string{"scifi"}, Keywords: []int{1, 2}, Directors: []int{10}}
		want := "because you liked Alpha (shared: Nolan, dreams, heist)"
		if got := Reason(seeds, c2); got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("recBy no shared", func(t *testing.T) {
		c3 := Candidate{RecBy: []catalog.ID{movieID(1)}}
		if got, want := Reason(seeds, c3), "because you liked Alpha"; got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("no shared", func(t *testing.T) {
		if got, want := Reason(seeds, Candidate{}), "matches your taste"; got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("nil seeds", func(t *testing.T) {
		c1 := Candidate{RecBy: []catalog.ID{tvID(2)}, Genres: []string{"action"}, Keywords: []int{2}}
		if got, want := Reason(nil, c1), "matches your taste"; got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("two recBy picks heigher weight", func(t *testing.T) {
		c := Candidate{RecBy: []catalog.ID{movieID(1), tvID(2)}}
		if got, want := Reason(seeds, c), "because you liked Alpha"; got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("equal weight tie by id", func(t *testing.T) {
		a2, b2 := a, b
		a2.Weight = 0.9
		b2.Weight = 0.9
		c := Candidate{RecBy: []catalog.ID{movieID(1), tvID(2)}}
		if got, want := Reason([]Seed{a2, b2}, c), "because you liked Alpha"; got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})

	t.Run("recBy not among seeds falls back", func(t *testing.T) {
		c := Candidate{RecBy: []catalog.ID{movieID(99)}, Genres: []string{"scifi"}, Keywords: []int{1}}
		want := "because you liked Alpha (shared: dreams, Sci-Fi)"
		if got := Reason(seeds, c); got != want {
			t.Errorf("Reason = %q, want %q", got, want)
		}
	})
}
