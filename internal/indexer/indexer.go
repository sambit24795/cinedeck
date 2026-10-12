package indexer

import (
	"context"
)

type Indexer struct {
	trackerURL string
	dhtEnabled bool
	privateKey string
}

func NewIndexer(opts ...Option) *Indexer {
	i := &Indexer{}
	for _, opt := range opts {
		opt(i)
	}
	return i
}

type Option func(*Indexer)

func WithTrackerURL(url string) Option {
	return func(i *Indexer) { i.trackerURL = url }
}

func WithDHT(enabled bool) Option {
	return func(i *Indexer) { i.dhtEnabled = enabled }
}

func WithPrivateKey(key string) Option {
	return func(i *Indexer) { i.privateKey = key }
}

// ParseMagnet extracts info hash and title from a magnet link.
func (i *Indexer) ParseMagnet(ctx context.Context, magnet string) (infoHash, title string, err error) {
	return ParseMagnet(magnet)
}

// ParseTorrent parses a .torrent file into TorrentInfo.
func (i *Indexer) ParseTorrent(ctx context.Context, data []byte) (*TorrentInfo, error) {
	return nil, ErrNoTorrent
}

// QueryTracker queries a configured tracker API for metadata.
func (i *Indexer) QueryTracker(ctx context.Context, infoHash string) (*TrackerResult, error) {
	return nil, ErrTrackerFailed
}

// TorrentInfo contains parsed torrent metadata.
type TorrentInfo struct {
	InfoHash string // hex lowercase, 40 chars
	Title    string // from info or name field
	Year     int    // parsed from title if present (TMDB-style)
	Size     uint64 // total size in bytes
	Files    []File // list of files with names and sizes
}

// File represents a file within the torrent.
type File struct {
	Name string
	Size uint64
	Hash string // piece hash for verification
}

// TrackerResult contains metadata from a tracker API query.
type TrackerResult struct {
	Title     string
	Year      int
	Seeders   int
	Leechers  int
	TotalSize uint64
	FileList  []string // simplified: just filenames
}
