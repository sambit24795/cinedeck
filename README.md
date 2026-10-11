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

The commands below are planned for the next milestone and are not available
yet. Search for a movie or show:

```
cinedeck search inception
```

Show details for a title:

```
cinedeck info movie:27205
```

## Attribution

This product uses the TMDB API but is not endorsed or certified by TMDB. Where-to-watch data is provided by JustWatch.
