package catalog

import (
	"context"
	"net/url"
	"strconv"
)

// Keyword is one TMDB keyword attached to a title.
type Keyword struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Person is one TMDB person (a cast or crew member).
type Person struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Meta is everything cinedeck stores about a title: enough to list it offline
// and to build a taste profile.
type Meta struct {
	ID         ID        `json:"id"`
	Title      string    `json:"title"`
	Year       int       `json:"year"`
	Overview   string    `json:"overview"`
	Language   string    `json:"language"` // original_language
	Rating     float64   `json:"rating"`   // vote_average
	Votes      int       `json:"votes"`    // vote_count
	Popularity float64   `json:"popularity"`
	GenreIDs   []int     `json:"genre_ids"` // TMDB genre ids for ID.Type (movie and TV ids differ)
	Genres     []string  `json:"genres"`    // names, same order as GenreIDs
	Keywords   []Keyword `json:"keywords"`
	Cast       []Person  `json:"cast"`      // first 5 in the order given
	Directors  []Person  `json:"directors"` // movies: crew with job "Director"; TV: created_by
}

// metaResponse mirrors the TMDB detail response with keywords and credits.
type metaResponse struct {
	Title            string  `json:"title"`
	Name             string  `json:"name"`
	ReleaseDate      string  `json:"release_date"`
	FirstAirDate     string  `json:"first_air_date"`
	Overview         string  `json:"overview"`
	OriginalLanguage string  `json:"original_language"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Popularity       float64 `json:"popularity"`
	Genres           []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"genres"`
	Keywords struct {
		Keywords []Keyword `json:"keywords"`
		Results  []Keyword `json:"results"`
	} `json:"keywords"`
	Credits struct {
		Cast []Person `json:"cast"`
		Crew []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
			Job  string `json:"job"`
		} `json:"crew"`
	} `json:"credits"`
	CreatedBy []Person `json:"created_by"`
}

// Meta returns everything cinedeck stores about id, in one request.
func (c *Client) Meta(ctx context.Context, id ID) (*Meta, error) {
	q := url.Values{}
	q.Set("append_to_response", "keywords,credits")

	var resp metaResponse
	if err := c.get(ctx, "/"+string(id.Type)+"/"+strconv.FormatInt(id.N, 10), q, &resp); err != nil {
		return nil, err
	}

	m := &Meta{
		ID:         id,
		Overview:   resp.Overview,
		Language:   resp.OriginalLanguage,
		Rating:     resp.VoteAverage,
		Votes:      resp.VoteCount,
		Popularity: resp.Popularity,
		GenreIDs:   []int{},
		Genres:     []string{},
		Keywords:   []Keyword{},
		Cast:       []Person{},
		Directors:  []Person{},
	}

	for _, g := range resp.Genres {
		m.GenreIDs = append(m.GenreIDs, g.ID)
		m.Genres = append(m.Genres, g.Name)
	}

	if id.Type == TV {
		m.Title = resp.Name
		m.Year = year(resp.FirstAirDate)
		m.Keywords = append(m.Keywords, resp.Keywords.Results...)
		m.Directors = append(m.Directors, resp.CreatedBy...)
	} else {
		m.Title = resp.Title
		m.Year = year(resp.ReleaseDate)
		m.Keywords = append(m.Keywords, resp.Keywords.Keywords...)

		seen := map[int]bool{}
		for _, cr := range resp.Credits.Crew {
			if cr.Job != "Director" || seen[cr.ID] {
				continue
			}
			seen[cr.ID] = true
			m.Directors = append(m.Directors, Person{ID: cr.ID, Name: cr.Name})
		}
	}

	for i, p := range resp.Credits.Cast {
		if i == 5 {
			break
		}
		m.Cast = append(m.Cast, p)
	}

	return m, nil
}
