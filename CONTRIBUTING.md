# Contributing

## Checks

Before opening a pull request, run the full check suite and make sure all three
commands pass. `gofmt -l .` must print nothing.

```
gofmt -l . && go vet ./... && go test ./...
```

## Scope

cinedeck includes:
- TMDB API integration (legal, free)
- JustWatch where-to-watch data (legal attribution required)
- Internet Archive search (public domain/legal sources only)
- Torrent indexing (metadata extraction only; private use by user)

Do not add: built-in scrapers for piracy indexers or any source that encourages
copyright infringement. Torrent indexing is configured per-user and remains
their responsibility for legal compliance.
