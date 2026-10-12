package indexer

import (
	"context"
	"time"
)

// Indexer holds the shared configuration for torrent parsing and provider search.
type Indexer struct {
	timeout time.Duration
	cache   *cache
}

// Option configures an Indexer.
type Option func(*Indexer)

// NewIndexer builds an Indexer with the given options.
func NewIndexer(opts ...Option) *Indexer {
	i := &Indexer{timeout: 10 * time.Second}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

// WithTimeout sets the per-provider timeout (default 10s).
func WithTimeout(d time.Duration) Option {
	return func(i *Indexer) {
		if d > 0 {
			i.timeout = d
		}
	}
}

// ParseMagnet extracts info hash and title from a magnet link.
func (i *Indexer) ParseMagnet(ctx context.Context, magnet string) (infoHash, title string, err error) {
	return ParseMagnet(magnet)
}
