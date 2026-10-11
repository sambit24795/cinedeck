package recs

import (
	"reflect"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

func TestCanonGenres(t *testing.T) {
	cases := []struct {
		t    catalog.MediaType
		ids  []int
		want []string
	}{
		{catalog.Movie, []int{28, 878, 12}, []string{"action", "adventure", "scifi"}},
		{catalog.TV, []int{10759, 18}, []string{"action", "adventure", "drama"}},
		{catalog.TV, []int{10765}, []string{"fantasy", "scifi"}},
		{catalog.Movie, []int{10770, 999}, []string{}},
		{catalog.Movie, nil, []string{}},
		{catalog.Movie, []int{28, 28, 878, 878, 12}, []string{"action", "adventure", "scifi"}},
	}
	for _, tc := range cases {
		got := CanonGenres(tc.t, tc.ids)
		if len(tc.want) == 0 {
			if len(got) != 0 {
				t.Errorf("CanonGenres(%q, %v) = %v, want empty", tc.t, tc.ids, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("CanonGenres(%q, %v) = %v, want %v", tc.t, tc.ids, got, tc.want)
		}
	}
}

func TestGenreIDsFor(t *testing.T) {
	cases := []struct {
		t    catalog.MediaType
		keys []string
		want []int
	}{
		{catalog.TV, []string{"action", "adventure", "scifi"}, []int{10759, 10765}},
		{catalog.TV, []string{"thriller", "drama"}, []int{18}},
		{catalog.Movie, []string{"crime", "drama"}, []int{80, 18}},
		{catalog.Movie, []string{"kids"}, []int{}},
		{catalog.MediaType("person"), []string{"drama"}, nil},
	}
	for _, tc := range cases {
		got := GenreIDsFor(tc.t, tc.keys)
		if len(tc.want) == 0 {
			if len(got) != 0 {
				t.Errorf("GenreIDsFor(%q, %v) = %v, want empty", tc.t, tc.keys, got)
			}
			continue
		}
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("GenreIDsFor(%q, %v) = %v, want %v", tc.t, tc.keys, got, tc.want)
		}
	}
}

func TestGenreIDsRoundTrip(t *testing.T) {
	keys := CanonGenres(catalog.Movie, []int{16, 35, 80})
	if got, want := GenreIDsFor(catalog.TV, keys), []int{16, 35, 80}; !reflect.DeepEqual(got, want) {
		t.Errorf("round trip = %v, want %v", got, want)
	}
}

func TestGenreName(t *testing.T) {
	cases := map[string]string{
		"scifi": "Sci-Fi",
		"drama": "Drama",
		"zzz":   "zzz",
	}
	for key, want := range cases {
		if got := GenreName(key); got != want {
			t.Errorf("GenreName(%q) = %q, want %q", key, got, want)
		}
	}
}
