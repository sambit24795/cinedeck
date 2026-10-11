package library

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// Entry is one item with everything the user did with it. Time fields are
// zero when the flag is not set.
type Entry struct {
	Meta        catalog.Meta
	Liked       bool
	LikedAt     time.Time
	Rating      int // 0 = not rated
	RatedAt     time.Time
	Watched     bool
	WatchedAt   time.Time
	OnWatchlist bool
	AddedAt     time.Time
}

// Disliked reports a rating from 1 to DislikeMax.
func (e Entry) Disliked() bool {
	return e.Rating >= 1 && e.Rating <= DislikeMax
}

// Positive reports a signal that the user enjoyed the title: liked or rated
// LikeMin or more, and not Disliked.
func (e Entry) Positive() bool {
	return !e.Disliked() && (e.Liked || e.Rating >= LikeMin)
}

// entryQuery is the shared base query for the entry read functions.
const entryQuery = `
SELECT i.type, i.tmdb_id, i.title, i.year, i.overview, i.language, i.rating, i.votes, i.popularity,
       i.genre_ids, i.genres, i.keywords, i.cast_json, i.directors,
       l.liked_at, r.rating, r.rated_at, w.watched_at, wl.added_at
FROM items i
LEFT JOIN likes     l  ON l.type  = i.type AND l.tmdb_id  = i.tmdb_id
LEFT JOIN ratings   r  ON r.type  = i.type AND r.tmdb_id  = i.tmdb_id
LEFT JOIN watched   w  ON w.type  = i.type AND w.tmdb_id  = i.tmdb_id
LEFT JOIN watchlist wl ON wl.type = i.type AND wl.tmdb_id = i.tmdb_id`

// Entry returns the stored item and the user's flags for it, or ErrNotFound.
func (l *Library) Entry(ctx context.Context, id catalog.ID) (*Entry, error) {
	entries, err := l.selectEntries(ctx, "i.type = ? AND i.tmdb_id = ?", "", 0, string(id.Type), id.N)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, ErrNotFound
	}
	return &entries[0], nil
}

// Entries returns every item with at least one flag, ordered by type, tmdb_id.
func (l *Library) Entries(ctx context.Context) ([]Entry, error) {
	const where = "l.liked_at IS NOT NULL OR r.rating IS NOT NULL OR w.watched_at IS NOT NULL OR wl.added_at IS NOT NULL"
	return l.selectEntries(ctx, where, "i.type, i.tmdb_id", 0)
}

// Watchlist returns the items on the watchlist, newest added first.
func (l *Library) Watchlist(ctx context.Context) ([]Entry, error) {
	return l.selectEntries(ctx, "wl.added_at IS NOT NULL", "wl.added_at DESC, i.type, i.tmdb_id", 0)
}

// History returns watched items, newest first; limit <= 0 means no limit.
func (l *Library) History(ctx context.Context, limit int) ([]Entry, error) {
	return l.selectEntries(ctx, "w.watched_at IS NOT NULL", "w.watched_at DESC, i.type, i.tmdb_id", limit)
}

// selectEntries runs the shared query with the given WHERE and ORDER BY clauses.
func (l *Library) selectEntries(ctx context.Context, where, orderBy string, limit int, args ...any) ([]Entry, error) {
	query := entryQuery
	if where != "" {
		query += " WHERE " + where
	}
	if orderBy != "" {
		query += " ORDER BY " + orderBy
	}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	defer rows.Close()

	entries := []Entry{}
	for rows.Next() {
		var e Entry
		var likedAt, rating, ratedAt, watchedAt, addedAt sql.NullInt64
		if err := scanMeta(rows, &e.Meta, &likedAt, &rating, &ratedAt, &watchedAt, &addedAt); err != nil {
			return nil, fmt.Errorf("library: %w", err)
		}
		if likedAt.Valid {
			e.Liked = true
			e.LikedAt = time.Unix(likedAt.Int64, 0).UTC()
		}
		if rating.Valid {
			e.Rating = int(rating.Int64)
		}
		if ratedAt.Valid {
			e.RatedAt = time.Unix(ratedAt.Int64, 0).UTC()
		}
		if watchedAt.Valid {
			e.Watched = true
			e.WatchedAt = time.Unix(watchedAt.Int64, 0).UTC()
		}
		if addedAt.Valid {
			e.OnWatchlist = true
			e.AddedAt = time.Unix(addedAt.Int64, 0).UTC()
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("library: %w", err)
	}
	return entries, nil
}
