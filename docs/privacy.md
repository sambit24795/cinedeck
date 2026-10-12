# Privacy

cinedeck stores everything locally and uploads nothing. This page lists what is
kept on disk and which hosts cinedeck contacts.

## What is stored locally

### Config: `config.toml`

In `$XDG_CONFIG_HOME/cinedeck/` (or the OS equivalent; `cinedeck config path`
prints it). It holds:

- your TMDB token,
- region and download directory,
- the enabled torrent indexer providers, timeout and optional proxy URL.

The file is written with mode `0600` and its directory with `0700`. The token is
never written to any cache, log or output. `CINEDECK_*` environment variables can
override any value.

### Library: `library.db`

In `$XDG_DATA_HOME/cinedeck/` (or the OS equivalent; `CINEDECK_DATA_DIR`
overrides it). One SQLite file holding:

- the titles you liked, rated, watched or added to your watchlist, with their
  TMDB title, year, genres, keywords, top cast and director,
- the torrent releases you saved with `torrents save` (info hash, title,
  provider, size, seeders, leechers, magnet link).

Delete the file to start over. It never leaves your machine.

### Cache

- TMDB responses: `<user cache dir>/cinedeck/tmdb/`, TTL 24 hours.
- Torrent search results: `<user cache dir>/cinedeck/torrents/`, TTL 1 hour.

`CINEDECK_CACHE_DIR` overrides the base. Cache files are mode `0600` and contain
no credentials. The TMDB cache key is derived from a hash of the token, never the
token itself.

## What cinedeck contacts

| Host | When | What is sent |
|---|---|---|
| `api.themoviedb.org` | search, info, where-to-watch, recommendations, library commands that fetch metadata | your token, the search term or title id |
| each enabled indexer | `torrents search`, `torrents info`, `torrents save` | the search term or info hash |
| `archive.org` | `archive` provider, `free` | the search term |

No telemetry, analytics or crash reporting is sent anywhere. Nothing is uploaded:
all data flows from the sources above to your machine.

## Third parties

- TMDB — see https://www.themoviedb.org/documentation/api/terms-of-use
- JustWatch data is surfaced through TMDB for where-to-watch.
- Each torrent indexer has its own terms; enabling it is your choice.
