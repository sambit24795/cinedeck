package library

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

const (
	DislikeMax = 4 // a rating from 1 to DislikeMax counts as disliked
	LikeMin    = 8 // a rating of LikeMin or more counts as a positive signal
)

// tx runs fn inside a transaction, rolling back on error and committing on success.
func (l *Library) tx(ctx context.Context, fn func(*sql.Tx) error) error {
	tx, err := l.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("library: %w", err)
	}
	return nil
}

// requireItem returns ErrNotFound when id is not in items.
func requireItem(ctx context.Context, tx *sql.Tx, id catalog.ID) error {
	var one int
	err := tx.QueryRowContext(ctx, "SELECT 1 FROM items WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	return nil
}

// Like adds a like for id. An existing like keeps its original liked_at.
func (l *Library) Like(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		if err := requireItem(ctx, tx, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO likes (type, tmdb_id, liked_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
			string(id.Type), id.N, l.now().Unix())
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n == 1
		return nil
	})
	return changed, err
}

// Unlike removes the like for id.
func (l *Library) Unlike(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"DELETE FROM likes WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N)
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n > 0
		return nil
	})
	return changed, err
}

// Rate stores rating n for id and marks the item watched. It returns the
// previous rating, or 0 when it was not rated. Rating implies watched and
// removes the item from the watchlist.
func (l *Library) Rate(ctx context.Context, id catalog.ID, n int) (previous int, err error) {
	if n < 1 || n > 10 {
		return 0, errors.New("rating must be between 1 and 10")
	}
	err = l.tx(ctx, func(tx *sql.Tx) error {
		if err := requireItem(ctx, tx, id); err != nil {
			return err
		}
		err := tx.QueryRowContext(ctx,
			"SELECT rating FROM ratings WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N).Scan(&previous)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("library: %w", err)
		}

		now := l.now().Unix()
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO ratings (type, tmdb_id, rating, rated_at) VALUES (?, ?, ?, ?)
             ON CONFLICT (type, tmdb_id) DO UPDATE SET rating = excluded.rating, rated_at = excluded.rated_at`,
			string(id.Type), id.N, n, now); err != nil {
			return fmt.Errorf("library: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO watched (type, tmdb_id, watched_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
			string(id.Type), id.N, now); err != nil {
			return fmt.Errorf("library: %w", err)
		}
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM watchlist WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N); err != nil {
			return fmt.Errorf("library: %w", err)
		}
		return nil
	})
	return previous, err
}

// Unrate removes the rating for id, leaving the watched mark.
func (l *Library) Unrate(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"DELETE FROM ratings WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N)
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n > 0
		return nil
	})
	return changed, err
}

// MarkWatched marks id as watched and removes it from the watchlist.
func (l *Library) MarkWatched(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		if err := requireItem(ctx, tx, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO watched (type, tmdb_id, watched_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
			string(id.Type), id.N, l.now().Unix())
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n == 1

		if _, err := tx.ExecContext(ctx,
			"DELETE FROM watchlist WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N); err != nil {
			return fmt.Errorf("library: %w", err)
		}
		return nil
	})
	return changed, err
}

// Unwatch removes the watched mark for id.
func (l *Library) Unwatch(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"DELETE FROM watched WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N)
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n > 0
		return nil
	})
	return changed, err
}

// AddToWatchlist adds id to the watchlist.
func (l *Library) AddToWatchlist(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		if err := requireItem(ctx, tx, id); err != nil {
			return err
		}
		res, err := tx.ExecContext(ctx,
			"INSERT INTO watchlist (type, tmdb_id, added_at) VALUES (?, ?, ?) ON CONFLICT DO NOTHING",
			string(id.Type), id.N, l.now().Unix())
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n == 1
		return nil
	})
	return changed, err
}

// RemoveFromWatchlist removes id from the watchlist.
func (l *Library) RemoveFromWatchlist(ctx context.Context, id catalog.ID) (changed bool, err error) {
	err = l.tx(ctx, func(tx *sql.Tx) error {
		res, err := tx.ExecContext(ctx,
			"DELETE FROM watchlist WHERE type = ? AND tmdb_id = ?", string(id.Type), id.N)
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("library: %w", err)
		}
		changed = n > 0
		return nil
	})
	return changed, err
}
