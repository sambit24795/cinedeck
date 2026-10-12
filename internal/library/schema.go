package library

// migrations are applied in order; migration i takes the database from
// schema version i to i+1. Never edit an applied migration: add a new one.
var migrations = []string{
	`CREATE TABLE items (
        type       TEXT    NOT NULL CHECK (type IN ('movie','tv')),
        tmdb_id    INTEGER NOT NULL,
        title      TEXT    NOT NULL,
        year       INTEGER NOT NULL DEFAULT 0,
        overview   TEXT    NOT NULL DEFAULT '',
        language   TEXT    NOT NULL DEFAULT '',
        rating     REAL    NOT NULL DEFAULT 0,
        votes      INTEGER NOT NULL DEFAULT 0,
        popularity REAL    NOT NULL DEFAULT 0,
        genre_ids  TEXT    NOT NULL DEFAULT '[]',
        genres     TEXT    NOT NULL DEFAULT '[]',
        keywords   TEXT    NOT NULL DEFAULT '[]',
        cast_json  TEXT    NOT NULL DEFAULT '[]',
        directors  TEXT    NOT NULL DEFAULT '[]',
        fetched_at INTEGER NOT NULL,
        PRIMARY KEY (type, tmdb_id)
    );
    CREATE TABLE ratings (
        type     TEXT    NOT NULL,
        tmdb_id  INTEGER NOT NULL,
        rating   INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 10),
        rated_at INTEGER NOT NULL,
        PRIMARY KEY (type, tmdb_id),
        FOREIGN KEY (type, tmdb_id) REFERENCES items (type, tmdb_id) ON DELETE CASCADE
    );
    CREATE TABLE likes (
        type     TEXT    NOT NULL,
        tmdb_id  INTEGER NOT NULL,
        liked_at INTEGER NOT NULL,
        PRIMARY KEY (type, tmdb_id),
        FOREIGN KEY (type, tmdb_id) REFERENCES items (type, tmdb_id) ON DELETE CASCADE
    );
    CREATE TABLE watchlist (
        type     TEXT    NOT NULL,
        tmdb_id  INTEGER NOT NULL,
        added_at INTEGER NOT NULL,
        PRIMARY KEY (type, tmdb_id),
        FOREIGN KEY (type, tmdb_id) REFERENCES items (type, tmdb_id) ON DELETE CASCADE
    );
    CREATE TABLE watched (
        type       TEXT    NOT NULL,
        tmdb_id    INTEGER NOT NULL,
        watched_at INTEGER NOT NULL,
        PRIMARY KEY (type, tmdb_id),
        FOREIGN KEY (type, tmdb_id) REFERENCES items (type, tmdb_id) ON DELETE CASCADE
    );
    CREATE TABLE cache_meta (
        key        TEXT    PRIMARY KEY,
        value      TEXT    NOT NULL,
        updated_at INTEGER NOT NULL
    );`,
	`CREATE TABLE torrents (
        info_hash TEXT    PRIMARY KEY,
        tmdb_id   TEXT    NOT NULL DEFAULT '',
        title     TEXT    NOT NULL,
        provider  TEXT    NOT NULL,
        size      INTEGER NOT NULL DEFAULT 0,
        seeders   INTEGER NOT NULL DEFAULT 0,
        leechers  INTEGER NOT NULL DEFAULT 0,
        magnet    TEXT    NOT NULL DEFAULT '',
        saved_at  INTEGER NOT NULL
    );
    CREATE INDEX torrents_tmdb_id ON torrents (tmdb_id);`,
}
