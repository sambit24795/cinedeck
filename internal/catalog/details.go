package catalog

import (
	"context"
	"net/url"
	"strconv"
)

// Details is the full information for one movie or TV show.
type Details struct {
	ID        ID       `json:"id"`
	Title     string   `json:"title"`
	Year      int      `json:"year"`
	Rating    float64  `json:"rating"`
	Overview  string   `json:"overview"`
	Genres    []string `json:"genres"`
	Runtime   int      `json:"runtime,omitempty"`   // minutes, movies only
	Seasons   int      `json:"seasons,omitempty"`   // TV only
	Episodes  int      `json:"episodes,omitempty"`  // TV only
	Cast      []string `json:"cast"`                // first 5 names, in the order given
	Directors []string `json:"directors,omitempty"` // movies: crew entries with job == "Director"
	Creators  []string `json:"creators,omitempty"`  // TV: created_by names

	Poster      string       `json:"poster,omitempty"`   // w342 URL
	Backdrop    string       `json:"backdrop,omitempty"` // w780 URL
	Tagline     string       `json:"tagline,omitempty"`
	Status      string       `json:"status,omitempty"`   // e.g. "Released", "Ended", "Returning Series"
	Language    string       `json:"language,omitempty"` // TMDB original_language, e.g. "ko"
	Released    string       `json:"released,omitempty"` // release_date (movie) or first_air_date (TV), as given
	Votes       int          `json:"votes,omitempty"`    // vote_count
	Networks    []string     `json:"networks,omitempty"` // TV only
	CastDetails []CastMember `json:"cast_details"`       // first 10, in the order given; never null
}

// CastMember is one cast entry with the character played and a photo URL.
type CastMember struct {
	Name      string `json:"name"`
	Character string `json:"character,omitempty"`
	Photo     string `json:"photo,omitempty"` // w185 URL
}

// detailsResponse mirrors the TMDB movie/tv detail response fields used here.
type detailsResponse struct {
	Title            string  `json:"title"`
	Name             string  `json:"name"`
	ReleaseDate      string  `json:"release_date"`
	FirstAirDate     string  `json:"first_air_date"`
	Runtime          int     `json:"runtime"`
	NumberOfSeasons  int     `json:"number_of_seasons"`
	NumberOfEpisodes int     `json:"number_of_episodes"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Overview         string  `json:"overview"`
	PosterPath       string  `json:"poster_path"`
	BackdropPath     string  `json:"backdrop_path"`
	Tagline          string  `json:"tagline"`
	Status           string  `json:"status"`
	OriginalLanguage string  `json:"original_language"`
	Networks         []struct {
		Name string `json:"name"`
	} `json:"networks"`
	Genres []struct {
		Name string `json:"name"`
	} `json:"genres"`
	CreatedBy []struct {
		Name string `json:"name"`
	} `json:"created_by"`
	Credits struct {
		Cast []struct {
			Name        string `json:"name"`
			Character   string `json:"character"`
			ProfilePath string `json:"profile_path"`
		} `json:"cast"`
		Crew []struct {
			Name string `json:"name"`
			Job  string `json:"job"`
		} `json:"crew"`
	} `json:"credits"`
}

// Details returns the full information for id.
func (c *Client) Details(ctx context.Context, id ID) (*Details, error) {
	q := url.Values{}
	q.Set("append_to_response", "credits")

	var resp detailsResponse
	if err := c.get(ctx, "/"+string(id.Type)+"/"+strconv.FormatInt(id.N, 10), q, &resp); err != nil {
		return nil, err
	}

	d := &Details{
		ID:          id,
		Rating:      resp.VoteAverage,
		Overview:    resp.Overview,
		Genres:      []string{}, // never null in JSON
		Cast:        []string{},
		Poster:      imageURL(posterSize, resp.PosterPath),
		Backdrop:    imageURL(backdropSize, resp.BackdropPath),
		Tagline:     resp.Tagline,
		Status:      resp.Status,
		Language:    resp.OriginalLanguage,
		Votes:       resp.VoteCount,
		CastDetails: []CastMember{},
	}
	for _, g := range resp.Genres {
		d.Genres = append(d.Genres, g.Name)
	}
	for i, m := range resp.Credits.Cast {
		if i >= 10 {
			break
		}
		if i < 5 {
			d.Cast = append(d.Cast, m.Name)
		}
		d.CastDetails = append(d.CastDetails, CastMember{
			Name:      m.Name,
			Character: m.Character,
			Photo:     imageURL(profileSize, m.ProfilePath),
		})
	}

	if id.Type == TV {
		d.Title = resp.Name
		d.Year = year(resp.FirstAirDate)
		d.Seasons = resp.NumberOfSeasons
		d.Episodes = resp.NumberOfEpisodes
		d.Released = resp.FirstAirDate
		for _, cr := range resp.CreatedBy {
			d.Creators = append(d.Creators, cr.Name)
		}
		for _, n := range resp.Networks {
			d.Networks = append(d.Networks, n.Name)
		}
	} else {
		d.Title = resp.Title
		d.Year = year(resp.ReleaseDate)
		d.Runtime = resp.Runtime
		d.Released = resp.ReleaseDate
		for _, cr := range resp.Credits.Crew {
			if cr.Job == "Director" {
				d.Directors = append(d.Directors, cr.Name)
			}
		}
	}
	return d, nil
}
