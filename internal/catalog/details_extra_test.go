package catalog

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestDetailsImagesAndCast(t *testing.T) {
	srv, _, _ := detailsServer(t, "tv_94796.json")

	c := New("tok", WithBaseURL(srv.URL))
	d, err := c.Details(context.Background(), ID{TV, 94796})
	if err != nil {
		t.Fatalf("Details: %v", err)
	}

	if want := "https://image.tmdb.org/t/p/w342/fgBNLPr6mC8pxuR79ENAJY4nBmj.jpg"; d.Poster != want {
		t.Errorf("Poster = %q, want %q", d.Poster, want)
	}
	if want := "https://image.tmdb.org/t/p/w780/3yEHM2HT2vrUtO93YzTJNgEfiZG.jpg"; d.Backdrop != want {
		t.Errorf("Backdrop = %q, want %q", d.Backdrop, want)
	}
	if d.Status != "Ended" {
		t.Errorf("Status = %q, want %q", d.Status, "Ended")
	}
	if d.Language != "ko" {
		t.Errorf("Language = %q, want %q", d.Language, "ko")
	}
	if d.Votes != 977 {
		t.Errorf("Votes = %d, want %d", d.Votes, 977)
	}
	if d.Released != "2019-12-14" {
		t.Errorf("Released = %q, want %q", d.Released, "2019-12-14")
	}
	if len(d.Networks) != 1 || d.Networks[0] != "tvN" {
		t.Errorf("Networks = %v, want [tvN]", d.Networks)
	}
	if d.Tagline != "" {
		t.Errorf("Tagline = %q, want \"\"", d.Tagline)
	}

	if len(d.CastDetails) != 10 {
		t.Fatalf("len(CastDetails) = %d, want 10", len(d.CastDetails))
	}
	wantFirst := CastMember{
		Name:      "Hyun Bin",
		Character: "Ri Jeong-hyeok",
		Photo:     "https://image.tmdb.org/t/p/w185/JQFzhO9j8HRiyr7leGPj6cqhvM.jpg",
	}
	if d.CastDetails[0] != wantFirst {
		t.Errorf("CastDetails[0] = %+v, want %+v", d.CastDetails[0], wantFirst)
	}
	if d.CastDetails[1].Name != "Son Ye-jin" {
		t.Errorf("CastDetails[1].Name = %q, want %q", d.CastDetails[1].Name, "Son Ye-jin")
	}

	if len(d.Cast) != 5 {
		t.Fatalf("len(Cast) = %d, want 5", len(d.Cast))
	}
	if d.Cast[0] != "Hyun Bin" {
		t.Errorf("Cast[0] = %q, want %q", d.Cast[0], "Hyun Bin")
	}
}

func TestDetailsMovieWithoutImages(t *testing.T) {
	srv, _, _ := detailsServer(t, "movie_27205.json")

	c := New("tok", WithBaseURL(srv.URL))
	d, err := c.Details(context.Background(), ID{Movie, 27205})
	if err != nil {
		t.Fatalf("Details: %v", err)
	}

	if d.Poster != "" {
		t.Errorf("Poster = %q, want \"\"", d.Poster)
	}
	if d.Networks != nil {
		t.Errorf("Networks = %v, want nil", d.Networks)
	}
	if d.CastDetails == nil {
		t.Fatal("CastDetails = nil, want non-nil")
	}
	for i, m := range d.CastDetails {
		if m.Photo != "" {
			t.Errorf("CastDetails[%d].Photo = %q, want \"\"", i, m.Photo)
		}
	}

	b, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"cast_details"`) {
		t.Errorf("json = %s, want it to contain %q", b, "cast_details")
	}
	if strings.Contains(string(b), `"poster"`) {
		t.Errorf("json = %s, want it not to contain %q", b, "poster")
	}
}
