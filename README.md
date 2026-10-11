# cinedeck

cinedeck is a command-line tool for discovering movies and TV shows. It searches
TMDB for titles, shows details like ratings, genres and cast, and helps you find
where to watch them from legal sources.

## Install

```
go install github.com/sambit24795/cinedeck/cmd/cinedeck@latest
```

## Quickstart

Create a config file (you will be asked for a TMDB token, region and download
directory):

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

See where a title is available (by id or by title). The region comes from your config; `--region` overrides it:

```
cinedeck where movie:27205 --region IN
cinedeck where "game of thrones" --first
```

Results are grouped as Stream, Free, With ads, Rent and Buy, followed by the TMDB watch page link. TMDB does not provide a link per provider.

Add `--json` to any of these for machine-readable output (with `where`, the JustWatch notice is also printed to stderr).

## Attribution

This product uses the TMDB API but is not endorsed or certified by TMDB. Where-to-watch data is provided by JustWatch.
