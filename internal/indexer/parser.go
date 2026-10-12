package indexer

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// TorrentInfo contains parsed torrent metadata.
type TorrentInfo struct {
	InfoHash string // hex lowercase, 40 chars
	Title    string // from info["name"], "unknown" when empty
	Year     int    // best-effort from the title, 0 when absent
	Size     uint64 // total length in bytes
	Files    []File // never nil
}

// File represents a file within the torrent.
type File struct {
	Name string
	Size uint64
	Hash string // info hash of the torrent
}

// yearRe matches a 4-digit year between 1870 and 2100.
var yearRe = regexp.MustCompile(`\b(18[7-9]\d|19\d{2}|20\d{2}|2100)\b`)

// ParseTorrent parses a .torrent file into TorrentInfo.
func (i *Indexer) ParseTorrent(ctx context.Context, data []byte) (*TorrentInfo, error) {
	if len(strings.TrimSpace(string(data))) == 0 {
		return nil, ErrNoTorrent
	}

	mi := &metainfo.MetaInfo{}
	if err := bencode.Unmarshal(data, mi); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	info := &metainfo.Info{}
	if err := bencode.Unmarshal(mi.InfoBytes, info); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParseFailed, err)
	}

	title := strings.TrimSpace(info.Name)
	if title == "" {
		title = "unknown"
	}

	hash := mi.HashInfoBytes().HexString()

	return &TorrentInfo{
		InfoHash: hash,
		Title:    title,
		Year:     ParseYearFromTitle(title),
		Size:     uint64(info.TotalLength()),
		Files:    parseFiles(info, hash),
	}, nil
}

// ParseYearFromTitle returns the 4-digit year found in title, or 0.
func ParseYearFromTitle(title string) int {
	m := yearRe.FindString(title)
	if m == "" {
		return 0
	}
	y, err := strconv.Atoi(m)
	if err != nil {
		return 0
	}
	return y
}

func parseFiles(info *metainfo.Info, infoHash string) []File {
	files := make([]File, 0)

	if len(info.Files) == 0 {
		// Single-file torrent.
		return append(files, File{
			Name: info.Name,
			Size: uint64(info.Length),
			Hash: infoHash,
		})
	}

	for _, fi := range info.Files {
		files = append(files, File{
			Name: strings.Join(fi.Path, "/"),
			Size: uint64(fi.Length),
			Hash: infoHash,
		})
	}

	return files
}
