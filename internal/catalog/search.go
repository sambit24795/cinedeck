package catalog

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// Result is one entry in a search, discover or trending list.
type Result struct {
	ID       ID      `json:"id"`
	Title    string  `json:"title"`
	Year     int     `json:"year"`   // 0 when the date is empty
	Rating   float64 `json:"rating"` // TMDB vote_average
	Overview string  `json:"overview"`
	Poster   string  `json:"poster,omitempty"`
}

// item is one entry from a TMDB list response (search, discover or trending).
type item struct {
	ID           int64   `json:"id"`
	MediaType    string  `json:"media_type"`
	Title        string  `json:"title"`
	Name         string  `json:"name"`
	ReleaseDate  string  `json:"release_date"`
	FirstAirDate string  `json:"first_air_date"`
	VoteAverage  float64 `json:"vote_average"`
	Overview     string  `json:"overview"`
	PosterPath   string  `json:"poster_path"`
}

// result converts it to a Result. mt is used as the media type; names and
// dates are read from the fields that match it.
func (it item) result(mt MediaType) Result {
	r := Result{
		ID:       ID{Type: mt, N: it.ID},
		Rating:   it.VoteAverage,
		Overview: it.Overview,
		Poster:   imageURL(posterSize, it.PosterPath),
	}
	if mt == TV {
		r.Title = it.Name
		r.Year = year(it.FirstAirDate)
	} else {
		r.Title = it.Title
		r.Year = year(it.ReleaseDate)
	}
	return r
}

// year parses the leading four digits of a TMDB date, or 0 when there are none.
func year(date string) int {
	if len(date) < 4 {
		return 0
	}
	n, err := strconv.Atoi(date[:4])
	if err != nil {
		return 0
	}
	return n
}

// Search queries TMDB for movies and TV shows matching query.
func (c *Client) Search(ctx context.Context, query string) ([]Result, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("empty query")
	}

	q := url.Values{}
	q.Set("query", query)
	q.Set("include_adult", "false")

	var resp struct {
		Results []item `json:"results"`
	}
	if err := c.get(ctx, "/search/multi", q, &resp); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(resp.Results))
	for _, it := range resp.Results {
		var mt MediaType
		switch it.MediaType {
		case string(Movie):
			mt = Movie
		case string(TV):
			mt = TV
		default:
			continue
		}
		results = append(results, it.result(mt))
	}
	return results, nil
}
