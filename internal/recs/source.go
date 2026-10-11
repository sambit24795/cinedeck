package recs

import (
	"context"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// Source is the metadata the recommender needs. *catalog.Client implements it.
type Source interface {
	Meta(ctx context.Context, id catalog.ID) (*catalog.Meta, error)
	Recommendations(ctx context.Context, id catalog.ID) ([]catalog.Related, error)
	Similar(ctx context.Context, id catalog.ID) ([]catalog.Related, error)
	DiscoverBy(ctx context.Context, q catalog.DiscoverQuery) ([]catalog.Related, error)
}

var _ Source = (*catalog.Client)(nil)

// Pick is one recommendation.
type Pick struct {
	ID     catalog.ID `json:"id"`
	Title  string     `json:"title"`
	Year   int        `json:"year"`
	Rating float64    `json:"rating"` // TMDB vote_average
	Reason string     `json:"reason"`
	Score  float64    `json:"score,omitempty"` // personal recommendations only
}
