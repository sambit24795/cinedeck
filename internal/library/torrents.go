package library

import (
	"context"
	"fmt"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// SavedTorrent is one stored torrent release.
type SavedTorrent struct {
	InfoHash string
	TMDBID   string // "movie:27205", or "" when saved without a title
	Title    string
	Provider string
	Size     uint64
	Seeders  int
	Leechers int
	Magnet   string
	SavedAt  time.Time
}

// SaveRelease upserts a release by info hash. tmdbID may be empty.
func (l *Library) SaveRelease(ctx context.Context, r indexer.Release, tmdbID string) error {
	if r.InfoHash == "" {
		return fmt.Errorf("library: release has no info hash")
	}
	_, err := l.db.ExecContext(ctx,
		`INSERT INTO torrents (info_hash, tmdb_id, title, provider, size, seeders, leechers, magnet, saved_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT (info_hash) DO UPDATE SET
             tmdb_id  = excluded.tmdb_id,
             title    = excluded.title,
             provider = excluded.provider,
             size     = excluded.size,
             seeders  = excluded.seeders,
             leechers = excluded.leechers,
             magnet   = excluded.magnet,
             saved_at = excluded.saved_at`,
		r.InfoHash, tmdbID, r.Title, r.Provider, r.Size, r.Seeders, r.Leechers, r.Magnet, l.now().Unix())
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	return nil
}

// ReleasesFor returns the saved releases for tmdbID, newest first.
func (l *Library) ReleasesFor(ctx context.Context, tmdbID string) ([]indexer.Release, error) {
	rows, err := l.db.QueryContext(ctx,
		`SELECT info_hash, title, provider, size, seeders, leechers, magnet
         FROM torrents WHERE tmdb_id = ? AND tmdb_id <> '' ORDER BY saved_at DESC`, tmdbID)
	if err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	defer rows.Close()

	out := []indexer.Release{}
	for rows.Next() {
		var r indexer.Release
		if err := rows.Scan(&r.InfoHash, &r.Title, &r.Provider, &r.Size, &r.Seeders, &r.Leechers, &r.Magnet); err != nil {
			return nil, fmt.Errorf("library: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	return out, nil
}
