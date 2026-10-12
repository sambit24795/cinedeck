# Plan Change Summary: Torrent Engine → Torrent Indexing

## What Changed

### M4: Torrent Engine → Torrent Indexer
**Before:** Download and stream torrents from magnet links
**After:** Parse torrent files and query trackers for metadata (seeder/leecher counts)

### Removed Components
- `internal/engine` — Torrent download engine
- `internal/stream` — HTTP range streaming, transcoding hooks
- DLNA server (`anacrolix/dms`)
- Download dir configuration

### Added Components  
- `internal/indexer` — Torrent metadata extraction
  - Parse magnet links and `.torrent` files using `anacrolix/torrent`
  - Query tracker APIs for seeder/leecher counts, file list, hashes
  - Support public trackers (API-based) and DHT
  - Optional private tracker support with auth keys

### CLI Changes
**Before:**
```bash
cinedeck download "<magnet>"
cinedeck stream "<magnet>" --player mpv
```

**After:**
```bash
cinedeck index "<magnet>"
# Output: title, year, seeders, leechers, total size, file list
```

### New Config Options
- `CINEDECK_TRACKER_URL` — Base URL for public tracker API
- `CINEDECK_DHT_ENABLED=true` — Enable DHT search
- `CINEDECK_PRIVATE_KEY=...` — For private trackers requiring auth

### Scope Rule Updated
**Before:** "torrent engine is general-purpose (magnet links, `.torrent` files and URLs the user provides)"

**After:** "torrent indexing is private-use only. Users are responsible for legal compliance."

### Architecture Changes
```
internal/indexer  # NEW: torrent metadata extraction
# Replaced:
# internal/engine   # OLD: download engine  
# internal/stream   # OLD: streaming/transcoding
```

## Why This Change?

1. **More useful:** Seeing seeder/leecher counts helps users choose quality torrents
2. **Legal:** No downloading/streaming, just metadata extraction
3. **Simpler:** Removes complex streaming/transcoding code
4. **Flexible:** Users configure their own trackers; tool doesn't encourage piracy

## What's Next?

**M4 Tasks (planned):**
- T01: Setup indexer package + basic magnet parser
- T02: Torrent file parsing with `anacrolix/torrent`
- T03: Public tracker API client
- T04: DHT search support
- T05: Private tracker auth (optional)
- T06: CLI `index` command with table output
- T07: JSON output with file list and hashes
- T08: Daemon mode integration for `serve`

**M5 (unchanged):** Internet Archive legal sources
**M6-M8 (updated):** References to "torrent indexing" instead of streaming

## Files Modified

1. `PLAN.md` — Complete rewrite of M4, updated architecture and scope
2. `README.md` — Added torrent indexing section, removed download/stream refs  
3. `CONTRIBUTING.md` — Clarified scope: legal sources + private-use indexer
4. `WORKFLOW.md` — (No changes needed)

---

Ready to start M4 when you are!
