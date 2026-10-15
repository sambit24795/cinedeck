# cinedeck

cinedeck is a command-line tool for discovering movies and TV shows. It searches
TMDB for titles, shows details like ratings, genres and cast, finds where to
watch them legally, and searches public torrent indexers for releases with
seeder/leecher counts.

**Note:** Torrent search and indexing are for personal use only. Users are
responsible for legal compliance with local laws and indexer terms of service.

## Install

```
go install github.com/sambit24795/cinedeck/cmd/cinedeck@latest
```

Or install the prebuilt binary with npm:

```
npm install -g cinedeck
```

On Windows, `npm install -g` adds the binary directory to your user Path
environment variable. Open a new terminal if `cinedeck` is not recognized in
the one you installed from.

## Get a TMDB token

cinedeck needs a free TMDB token. Each user brings their own; none is built into cinedeck.

1. Make a free account at https://www.themoviedb.org, then open https://www.themoviedb.org/settings/api.
2. Request an API key. In TMDB's application form enter:
   - Application Name: `cinedeck (personal use)`
   - Application URL: `https://github.com/sambit24795/cinedeck`
   - Type of Use: `Desktop Application`
   - Application Summary: `A command-line tool for personal use to search movies and TV shows and see where to watch them.`
3. On the same page copy the long **API Read Access Token** (it starts with `eyJ`). Do not use the short **API Key**.
4. Run `cinedeck config init` and paste the token when asked. cinedeck checks it with TMDB before saving, and refuses a token that TMDB rejects. If TMDB cannot be reached it saves the token anyway and prints a warning.

The token is stored in your config file (`cinedeck config path` prints where). You can also set it with the environment variable `CINEDECK_TMDB_TOKEN`.

## Quickstart

Create a config file (you will be asked for the token from the section above, a region and optionally tracker settings):

```
cinedeck config init
```

Search for a movie or show:

```
cinedeck search inception
```

Show details for a title, by id or by title:

```
cinedeck info movie:27205
cinedeck info inception --first
```

Find short films (40 minutes or less) and see what is trending:

```
cinedeck shorts --genre animation --year 2008
cinedeck trending --type tv
```

Index a torrent to see metadata (title, seeder/leecher counts, file list):

```
cinedeck index "magnet:?xt=urn:btih:..."
cinedeck index "movie.torrent"
```

Browse by genre, language, country and year, or use the presets. TMDB has no "K-drama" or "anime" type, so the presets are filters: `kdrama` lists Korean-language TV dramas, and `anime` lists Japanese-language animation (TV by default, `--type movie` for films).

```
cinedeck discover --type tv --lang ko --genre drama --year 2024
cinedeck discover --type movie --country FR
cinedeck kdrama --year 2024
cinedeck anime --type movie --genre action
```

`discover` needs at least one of `--genre`, `--lang` (2-letter code such as `ko`), `--country` (2-letter code such as `KR`) or `--year`. Genre names are TMDB's and differ between movies and TV (for example TV has "Action & Adventure" and no "Romance"); an unknown name lists the valid ones. The presets skip titles with fewer than 50 votes, so very new titles may not show up yet. `shorts` only lists titles whose real runtime is 40 minutes or less.

Keep track of what you like, rate and want to watch. Everything is stored locally in a SQLite file (see below), so `watchlist ls` and `history` work offline. Each command takes an id or a title; a title is searched on TMDB (`--first` picks the first hit).

```
cinedeck like movie:27205
cinedeck rate "the matrix" 9 --first
cinedeck watchlist add tv:1399
cinedeck watchlist ls
cinedeck watched movie:27205
cinedeck history
```

`rate` takes a number from 1 to 10 and also marks the title as watched. A rating of 4 or less counts as "disliked" (the title will not be recommended), and 8 or more counts like a like. Undo with `unlike`, `unrate`, `unwatch` and `watchlist rm`; repeating a command is harmless. Marking a title as watched removes it from the watchlist.

Get recommendations. `--from` finds titles like one you name (TMDB's own recommendations plus similar titles; for the other type, such as "I liked this movie, which shows?", it matches shared keywords and genres). Without `--from`, cinedeck builds your taste from the titles you liked or rated 8 or higher, finds candidates on TMDB, scores them locally, and tells you why each was picked:

```
cinedeck recommend --from movie:27205 --type tv
cinedeck recommend --from "breaking bad" --first
cinedeck recommend
cinedeck recommend --type movie --limit 5
```

Personal recommendations skip everything you liked, rated or watched, and never suggest a title you rated 4 or lower. They use at most 32 TMDB requests per run (all cached for 24 hours) and work best after you have liked or rated a handful of titles. With `--from`, titles you rated 4 or lower are hidden, and titles you liked, rated, watched or put on the watchlist are marked in a status column.

See where a title is available (by id or by title). The region comes from your config; `--region` overrides it:

```
cinedeck where movie:27205 --region IN
cinedeck where "game of thrones" --first
```

Results are grouped as Stream, Free, With ads, Rent and Buy, followed by the TMDB watch page link. TMDB does not provide a link per provider.

Add `--json` to any of these for machine-readable output (with `where`, the JustWatch notice is also printed to stderr).

Search the Internet Archive for free, public-domain media (no TMDB token needed):

```
cinedeck free "night of the living dead"
cinedeck free "nosferatu" --json
```

## Torrent search (private use only)

Search public torrent indexers by title and see seeder and leecher counts. Providers are off until you enable them in your config:

```
# config.toml
indexer_providers = "apibay,yts,eztv,nyaa,archive"
indexer_timeout   = "10s"
```

Or with environment variables: `CINEDECK_INDEXER_PROVIDERS`, `CINEDECK_INDEXER_TIMEOUT` and `CINEDECK_PROXY_URL`.

```
cinedeck torrents search "inception" --type movie --min-seeders 10
cinedeck torrents search "breaking bad" --type tv --provider eztv --sort seeders
cinedeck torrents search "jujutsu kaisen" --json
```

Results are merged from every enabled provider, de-duplicated by info hash (the entry with the most seeders wins) and sorted. A provider that is down is reported on stderr and does not fail the command. Use `torrents providers` to list providers and which are enabled.

Work with a single torrent:

```
cinedeck torrents info "magnet:?xt=urn:btih:..."
cinedeck torrents info <40-char-info-hash>
cinedeck torrents info ./movie.torrent
```

Save a release to your library, optionally associating it with a title. Saved releases then appear in `info` under `Available torrents`:

```
cinedeck torrents save <info-hash> --for movie:27205
cinedeck info movie:27205
```

Providers: `apibay` (The Pirate Bay), `yts` (movies), `eztv` (TV), `nyaa` (anime), `archive` (Internet Archive, public-domain films with no seeder counts). See `docs/torrents.md` for details and `docs/privacy.md` for what is stored and what is called.

## Where your data is stored

Your library is one SQLite file, `library.db`, in:

- Linux and Termux: `$XDG_DATA_HOME/cinedeck/` (usually `~/.local/share/cinedeck/`)
- macOS: `~/Library/Application Support/cinedeck/`
- Windows: `%LOCALAPPDATA%\cinedeck\`

Set `CINEDECK_DATA_DIR` to use another directory. The file holds the titles you liked, rated, watched or added to your watchlist, with their TMDB title, year, genres, keywords, top cast and director, plus any torrent releases you saved. It never leaves your machine. Delete the file to start over.

Torrent search results are cached for one hour under your user cache directory (`CINEDECK_CACHE_DIR` overrides it).

## Toolchain

cinedeck builds with **Go 1.24** or newer. It is a single static binary; no cgo is required.

## Torrent indexing (private use only)

Torrent search and indexing are for personal use only. cinedeck queries the public
indexers you enable and parses magnet links or `.torrent` files; it does not host,
seed or download content. Enable providers with `indexer_providers` in your config
(see the Torrent search section above). No providers are enabled by default because
indexer legality varies by country.

## Attribution

This product uses the TMDB API but is not endorsed or certified by TMDB. Where-to-watch data is provided by JustWatch.

**Torrent search:** cinedeck indexes torrent metadata only; it does not download any content.
