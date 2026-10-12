package indexer

import (
	"context"
	"errors"
	"testing"

	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
)

// buildSingleFileTorrent returns bencoded metainfo bytes for a single-file torrent.
func buildSingleFileTorrent(t *testing.T, name string, length int64) []byte {
	t.Helper()
	info := metainfo.Info{
		Name:        name,
		Length:      length,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	}
	return marshalTorrent(t, info)
}

func buildMultiFileTorrent(t *testing.T, name string, files []metainfo.FileInfo) []byte {
	t.Helper()
	info := metainfo.Info{
		Name:        name,
		Files:       files,
		PieceLength: 16384,
		Pieces:      make([]byte, 20),
	}
	return marshalTorrent(t, info)
}

// marshalTorrent wraps a metainfo.Info in a MetaInfo and bencodes it.
func marshalTorrent(t *testing.T, info metainfo.Info) []byte {
	t.Helper()
	infoBytes, err := bencode.Marshal(info)
	if err != nil {
		t.Fatalf("bencode.Marshal(info): %v", err)
	}
	mi := metainfo.MetaInfo{InfoBytes: infoBytes}
	data, err := bencode.Marshal(mi)
	if err != nil {
		t.Fatalf("bencode.Marshal(metainfo): %v", err)
	}
	return data
}

func TestParseTorrentSingleFile(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	data := buildSingleFileTorrent(t, "Inception 2010", 1024)

	got, err := i.ParseTorrent(context.Background(), data)
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if got.Size != 1024 {
		t.Errorf("Size = %d, want 1024", got.Size)
	}
	if len(got.Files) != 1 {
		t.Fatalf("len(Files) = %d, want 1", len(got.Files))
	}
	if got.Files[0].Name != "Inception 2010" {
		t.Errorf("Files[0].Name = %q, want %q", got.Files[0].Name, "Inception 2010")
	}
	if len(got.InfoHash) != 40 {
		t.Errorf("len(InfoHash) = %d, want 40 (%q)", len(got.InfoHash), got.InfoHash)
	}
	if got.Year != 2010 {
		t.Errorf("Year = %d, want 2010", got.Year)
	}
}

func TestParseTorrentMultiFile(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	data := buildMultiFileTorrent(t, "Show S01", []metainfo.FileInfo{
		{Length: 100, Path: []string{"Season 1", "ep1.mkv"}},
		{Length: 200, Path: []string{"Season 1", "ep2.mkv"}},
	})

	got, err := i.ParseTorrent(context.Background(), data)
	if err != nil {
		t.Fatalf("ParseTorrent() error = %v", err)
	}
	if len(got.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(got.Files))
	}
	if got.Files[0].Name != "Season 1/ep1.mkv" {
		t.Errorf("Files[0].Name = %q, want %q", got.Files[0].Name, "Season 1/ep1.mkv")
	}
	if got.Size != 300 {
		t.Errorf("Size = %d, want 300", got.Size)
	}
}

func TestParseTorrentEmpty(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	_, err := i.ParseTorrent(context.Background(), nil)
	if !errors.Is(err, ErrNoTorrent) {
		t.Errorf("ParseTorrent(nil) error = %v, want ErrNoTorrent", err)
	}
}

func TestParseTorrentGarbage(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	_, err := i.ParseTorrent(context.Background(), []byte("not valid bencode"))
	if !errors.Is(err, ErrParseFailed) {
		t.Errorf("ParseTorrent(garbage) error = %v, want ErrParseFailed", err)
	}
}

func TestParseYearFromTitle(t *testing.T) {
	t.Parallel()

	tests := []struct {
		title string
		want  int
	}{
		{"Inception 2010", 2010},
		{"No Year Here", 0},
		{"1923", 1923},
		{"Some.Show.S01E01.1080p.2021.WEB", 2021},
		{"1800 old", 0},
	}
	for _, tt := range tests {
		if got := ParseYearFromTitle(tt.title); got != tt.want {
			t.Errorf("ParseYearFromTitle(%q) = %d, want %d", tt.title, got, tt.want)
		}
	}
}
