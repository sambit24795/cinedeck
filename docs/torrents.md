# Torrent search

`cinedeck torrents` searches public torrent indexers by title and shows seeder and
leecher counts. It is for personal use only; you are responsible for legal
compliance with your local laws and each indexer's terms.

cinedeck indexes metadata only. It does not host, seed or download content.

## Enabling providers

Providers are off until you enable them. In `config.toml`:

```toml
indexer_providers = "apibay,yts,eztv,nyaa,archive"
indexer_timeout   = "10s"
```

The same settings can come from the environment, which wins over the file:

| Variable | Meaning |
|---|---|
| `CINEDECK_INDEXER_PROVIDERS` | Comma-separated provider names. Empty means none. |
| `CINEDECK_INDEXER_TIMEOUT` | Per-provider timeout, a duration such as `10s`. |
| `CINEDECK_PROXY_URL` | Optional HTTP(S) proxy for provider requests. |

## Providers

| Name | Covers | Source |
|---|---|---|
| `apibay` | Everything | The Pirate Bay's JSON API (`apibay.org`) |
| `yts` | Movies | YTS API (`yts.mx`) |
| `eztv` | TV | EZTV API (`eztv.re`) |
| `nyaa` | Anime | Nyaa RSS feed (`nyaa.si`) |
| `archive` | Public-domain films | Internet Archive (`archive.org`) |

`archive` is the legal provider. The Internet Archive does not report seeder or
leecher counts, so those columns show `-` for its results.

## Commands

```bash
cinedeck torrents search "inception" --type movie --min-seeders 10
cinedeck torrents search "breaking bad" --type tv --provider eztv --sort seeders
cinedeck torrents search "jujutsu kaisen" --json
cinedeck torrents providers
cinedeck torrents info "magnet:?xt=urn:btih:..."
cinedeck torrents info <40-char-info-hash>
cinedeck torrents info ./movie.torrent
cinedeck torrents save <info-hash> --for movie:27205
```

### search flags

| Flag | Default | Meaning |
|---|---|---|
| `--type` | `any` | `movie`, `tv` or `any` |
| `--year` | `0` | Year filter (applied after the provider responds) |
| `--min-seeders` | `0` | Drop releases below this seeder count |
| `--provider` | (all enabled) | Repeatable; restrict to these providers |
| `--sort` | `seeders` | `seeders`, `size` or `date` |
| `--limit` | `20` | Maximum results after merging |
| `--json` | off | Machine-readable output |

## How results are merged

1. Every selected provider runs concurrently with its own timeout.
2. A provider that fails is reported on stderr (`apibay: timeout`) and skipped;
   the command still succeeds when another provider returned results.
3. Releases are de-duplicated by info hash. When two providers know the same
   hash, the entry with the most seeders is kept.
4. The merged list is filtered by `--min-seeders` and sorted by `--sort`.

The first page only is fetched from each provider; the seeder and leecher numbers
are a snapshot from the provider and change over time.

## Caching

Search results are cached on disk for one hour, per provider and query, under
your user cache directory (`CINEDECK_CACHE_DIR` overrides the base). Hash lookups
and provider errors are never cached.

## Saving releases

`cinedeck torrents save <hash> --for <id|title>` stores a release in your local
library. It then appears in `info <id|title>` under `Available torrents`, and in
`info --json` as `saved_torrents`. Saved releases survive restarts; saving the
same hash again updates its seeder count.

Without `--for`, a release is saved with no title association and does not appear
under any title.
