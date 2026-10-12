# cinedeck

Find movies and TV shows, get "if you liked X" recommendations, see where to
watch them, and search torrents — from the command line or a local web UI.

This npm package wraps the prebuilt Go binary. It downloads the correct binary
for your platform on install; there is no Node runtime required to use the tool
itself.

## Install

```bash
npm install -g cinedeck
```

## Quickstart

```bash
cinedeck config init                 # set your free TMDB token (optional for torrents/free)
cinedeck search "inception"
cinedeck info movie:27205
cinedeck where movie:27205 --region IN
cinedeck recommend --from movie:27205 --type tv
cinedeck recommend                   # personal, based on your likes/ratings
cinedeck torrents search "inception" --min-seeders 10
cinedeck free "night of the living dead"
cinedeck run                         # start the local web UI (http://localhost:8080)
```

A TMDB token is only needed for movie/TV search, details and recommendations.
Torrent search and the Internet Archive `free` command work without one.

## License

MIT. See the [repository](https://github.com/sambit24795/cinedeck) for the Go
source and full documentation.

This product uses the TMDB API but is not endorsed or certified by TMDB.
Where-to-watch data is provided by JustWatch.
