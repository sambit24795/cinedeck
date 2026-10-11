# cinedeck

cinedeck is a command-line tool for discovering movies and TV shows. It searches
TMDB for titles, shows details like ratings, genres and cast, and helps you find
where to watch them from legal sources.

## Install

```
go install github.com/sambit24795/cinedeck/cmd/cinedeck@latest
```

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

Create a config file (you will be asked for the token from the section above, a region and a download directory):

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

Browse by genre, language, country and year, or use the presets. TMDB has no "K-drama" or "anime" type, so the presets are filters: `kdrama` lists Korean-language TV dramas, and `anime` lists Japanese-language animation (TV by default, `--type movie` for films).

```
cinedeck discover --type tv --lang ko --genre drama --year 2024
cinedeck discover --type movie --country FR
cinedeck kdrama --year 2024
cinedeck anime --type movie --genre action
```

`discover` needs at least one of `--genre`, `--lang` (2-letter code such as `ko`), `--country` (2-letter code such as `KR`) or `--year`. Genre names are TMDB's and differ between movies and TV (for example TV has "Action & Adventure" and no "Romance"); an unknown name lists the valid ones. The presets skip titles with fewer than 50 votes, so very new titles may not show up yet. `shorts` only lists titles whose real runtime is 40 minutes or less.

See where a title is available (by id or by title). The region comes from your config; `--region` overrides it:

```
cinedeck where movie:27205 --region IN
cinedeck where "game of thrones" --first
```

Results are grouped as Stream, Free, With ads, Rent and Buy, followed by the TMDB watch page link. TMDB does not provide a link per provider.

Add `--json` to any of these for machine-readable output (with `where`, the JustWatch notice is also printed to stderr).

## Attribution

This product uses the TMDB API but is not endorsed or certified by TMDB. Where-to-watch data is provided by JustWatch.
