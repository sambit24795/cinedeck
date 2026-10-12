package indexer

import "errors"

var (
	ErrNoTorrent     = errors.New("no torrent data")
	ErrParseFailed   = errors.New("failed to parse torrent")
	ErrTrackerFailed = errors.New("tracker query failed")
)
