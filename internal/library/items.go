package library

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// ErrNotFound means the item (or entry) is not in the library.
var ErrNotFound = errors.New("library: not found")

// rowScanner is implemented by *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

// UpsertItem stores m, replacing the stored metadata of the same id. Likes,
// ratings, watched and watchlist rows of the item are not touched.
func (l *Library) UpsertItem(ctx context.Context, m *catalog.Meta) error {
	if m.ID.Type != catalog.Movie && m.ID.Type != catalog.TV {
		return fmt.Errorf("library: invalid item type %q", m.ID.Type)
	}

	genreIDs, err := marshalList(m.GenreIDs)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	genres, err := marshalList(m.Genres)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	keywords, err := marshalList(m.Keywords)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	cast, err := marshalList(m.Cast)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	directors, err := marshalList(m.Directors)
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}

	_, err = l.db.ExecContext(ctx,
		`INSERT INTO items (type, tmdb_id, title, year, overview, language, rating, votes, popularity,
                            genre_ids, genres, keywords, cast_json, directors, fetched_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT (type, tmdb_id) DO UPDATE SET
             title = excluded.title, year = excluded.year, overview = excluded.overview,
             language = excluded.language, rating = excluded.rating, votes = excluded.votes,
             popularity = excluded.popularity, genre_ids = excluded.genre_ids, genres = excluded.genres,
             keywords = excluded.keywords, cast_json = excluded.cast_json, directors = excluded.directors,
             fetched_at = excluded.fetched_at`,
		string(m.ID.Type), m.ID.N, m.Title, m.Year, m.Overview, m.Language, m.Rating, m.Votes,
		m.Popularity, genreIDs, genres, keywords, cast, directors, l.now().Unix())
	if err != nil {
		return fmt.Errorf("library: %w", err)
	}
	return nil
}

// Item returns the stored metadata for id, or ErrNotFound.
func (l *Library) Item(ctx context.Context, id catalog.ID) (*catalog.Meta, error) {
	row := l.db.QueryRowContext(ctx,
		`SELECT type, tmdb_id, title, year, overview, language, rating, votes, popularity,
                genre_ids, genres, keywords, cast_json, directors
         FROM items WHERE type = ? AND tmdb_id = ?`,
		string(id.Type), id.N)

	m := &catalog.Meta{}
	if err := scanMeta(row, m); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("library: %w", err)
	}
	return m, nil
}

// scanMeta scans the 14 item columns into m, then any extra columns into extra.
func scanMeta(s rowScanner, m *catalog.Meta, extra ...any) error {
	var typ string
	var tmdbID int64
	var genreIDs, genres, keywords, cast, directors string

	dest := []any{
		&typ, &tmdbID, &m.Title, &m.Year, &m.Overview, &m.Language,
		&m.Rating, &m.Votes, &m.Popularity,
		&genreIDs, &genres, &keywords, &cast, &directors,
	}
	dest = append(dest, extra...)
	if err := s.Scan(dest...); err != nil {
		return err
	}

	m.ID = catalog.ID{Type: catalog.MediaType(typ), N: tmdbID}
	var err error
	if m.GenreIDs, err = decodeJSONList[int](genreIDs); err != nil {
		return err
	}
	if m.Genres, err = decodeJSONList[string](genres); err != nil {
		return err
	}
	if m.Keywords, err = decodeJSONList[catalog.Keyword](keywords); err != nil {
		return err
	}
	if m.Cast, err = decodeJSONList[catalog.Person](cast); err != nil {
		return err
	}
	if m.Directors, err = decodeJSONList[catalog.Person](directors); err != nil {
		return err
	}
	return nil
}

// marshalList encodes v as JSON, storing a nil slice as [].
func marshalList(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if string(b) == "null" {
		return "[]", nil
	}
	return string(b), nil
}

// decodeJSONList decodes a JSON list column into a non-nil slice.
func decodeJSONList[T any](s string) ([]T, error) {
	if s == "" {
		return []T{}, nil
	}
	var v []T
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, err
	}
	if v == nil {
		v = []T{}
	}
	return v, nil
}
