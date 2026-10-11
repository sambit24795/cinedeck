package catalog

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Related is a list entry with the fields the recommender needs.
type Related struct {
	Result
	GenreIDs   []int   `json:"genre_ids"`
	Popularity float64 `json:"popularity"`
	Votes      int     `json:"votes"` // vote_count
}

// relatedItem is one entry of a recommendations, similar or discover list.
type relatedItem struct {
	item
	GenreIDs   []int   `json:"genre_ids"`
	Popularity float64 `json:"popularity"`
	VoteCount  int     `json:"vote_count"`
}

// related converts it to a Related. mt is the type of the endpoint, whatever
// the media_type field says.
func (it relatedItem) related(mt MediaType) Related {
	genreIDs := it.GenreIDs
	if genreIDs == nil {
		genreIDs = []int{}
	}
	return Related{
		Result:     it.result(mt),
		GenreIDs:   genreIDs,
		Popularity: it.Popularity,
		Votes:      it.VoteCount,
	}
}

// Recommendations returns page 1 of GET /{type}/{N}/recommendations, in TMDB's order.
func (c *Client) Recommendations(ctx context.Context, id ID) ([]Related, error) {
	return c.relatedList(ctx, "/"+string(id.Type)+"/"+strconv.FormatInt(id.N, 10)+"/recommendations", id.Type)
}

// Similar returns page 1 of GET /{type}/{N}/similar, in TMDB's order.
func (c *Client) Similar(ctx context.Context, id ID) ([]Related, error) {
	return c.relatedList(ctx, "/"+string(id.Type)+"/"+strconv.FormatInt(id.N, 10)+"/similar", id.Type)
}

// relatedList fetches a paged list of the given type and converts its results.
func (c *Client) relatedList(ctx context.Context, path string, t MediaType) ([]Related, error) {
	var resp struct {
		Results []relatedItem `json:"results"`
	}
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}

	related := make([]Related, 0, len(resp.Results))
	for _, it := range resp.Results {
		related = append(related, it.related(t))
	}
	return related, nil
}

// DiscoverQuery is a discover request by ids (no name lookups, no genre-list request).
type DiscoverQuery struct {
	Type       MediaType // Movie or TV; "" means Movie
	GenreIDs   []int     // TMDB genre ids of Type
	GenresAll  bool      // false: any of the genres (joined with "|"); true: all of them (joined with ",")
	KeywordIDs []int     // any of these keywords (joined with "|"); keyword ids are shared by movies and TV
	MinVotes   int       // vote_count.gte when > 0
}

// DiscoverBy returns page 1 of GET /discover/{type}.
func (c *Client) DiscoverBy(ctx context.Context, q DiscoverQuery) ([]Related, error) {
	t := q.Type
	if t == "" {
		t = Movie
	}
	if t != Movie && t != TV {
		return nil, fmt.Errorf("invalid type %q: must be movie or tv", t)
	}

	values := url.Values{}
	values.Set("sort_by", "popularity.desc")
	values.Set("include_adult", "false")
	if len(q.GenreIDs) > 0 {
		sep := "|"
		if q.GenresAll {
			sep = ","
		}
		parts := make([]string, len(q.GenreIDs))
		for i, id := range q.GenreIDs {
			parts[i] = strconv.Itoa(id)
		}
		values.Set("with_genres", strings.Join(parts, sep))
	}
	if len(q.KeywordIDs) > 0 {
		parts := make([]string, len(q.KeywordIDs))
		for i, id := range q.KeywordIDs {
			parts[i] = strconv.Itoa(id)
		}
		values.Set("with_keywords", strings.Join(parts, "|"))
	}
	if q.MinVotes > 0 {
		values.Set("vote_count.gte", strconv.Itoa(q.MinVotes))
	}

	var resp struct {
		Results []relatedItem `json:"results"`
	}
	if err := c.get(ctx, "/discover/"+string(t), values, &resp); err != nil {
		return nil, err
	}

	related := make([]Related, 0, len(resp.Results))
	for _, it := range resp.Results {
		related = append(related, it.related(t))
	}
	return related, nil
}
