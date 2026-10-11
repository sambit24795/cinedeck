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
	Overview         string  `json:"overview"`
	Genres           []struct {
		Name string `json:"name"`
	} `json:"genres"`
	CreatedBy []struct {
		Name string `json:"name"`
	} `json:"created_by"`
	Credits struct {
		Cast []struct {
			Name string `json:"name"`
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
		ID:       id,
		Rating:   resp.VoteAverage,
		Overview: resp.Overview,
		Genres:   []string{}, // never null in JSON
		Cast:     []string{},
	}
	for _, g := range resp.Genres {
		d.Genres = append(d.Genres, g.Name)
	}
	for i, m := range resp.Credits.Cast {
		if i == 5 {
			break
		}
		d.Cast = append(d.Cast, m.Name)
	}

	if id.Type == TV {
		d.Title = resp.Name
		d.Year = year(resp.FirstAirDate)
		d.Seasons = resp.NumberOfSeasons
		d.Episodes = resp.NumberOfEpisodes
		for _, cr := range resp.CreatedBy {
			d.Creators = append(d.Creators, cr.Name)
		}
	} else {
		d.Title = resp.Title
		d.Year = year(resp.ReleaseDate)
		d.Runtime = resp.Runtime
		for _, cr := range resp.Credits.Crew {
			if cr.Job == "Director" {
				d.Directors = append(d.Directors, cr.Name)
			}
		}
	}
	return d, nil
}
