package indexer

import (
	"context"
	"testing"
	"time"
)

func TestNewIndexer(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	if i == nil {
		t.Fatal("expected non-nil Indexer")
	}
	if i.timeout != 10*time.Second {
		t.Errorf("expected default timeout 10s, got %v", i.timeout)
	}
}

func TestWithTimeout(t *testing.T) {
	t.Parallel()

	i := NewIndexer(WithTimeout(3 * time.Second))
	if i.timeout != 3*time.Second {
		t.Errorf("expected 3s timeout, got %v", i.timeout)
	}
}

func TestWithTimeoutIgnoresNonPositive(t *testing.T) {
	t.Parallel()

	i := NewIndexer(WithTimeout(0))
	if i.timeout != 10*time.Second {
		t.Errorf("expected default timeout kept, got %v", i.timeout)
	}
}

func TestParseMagnet(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		magnet    string
		wantHash  string
		wantTitle string
		wantErr   bool
	}{
		{
			name:      "valid magnet with title",
			magnet:    "magnet:?xt=urn:btih:abc123def456&dn=Test+Movie+2023",
			wantHash:  "abc123def456",
			wantTitle: "Test Movie 2023",
		},
		{
			name:      "magnet without title",
			magnet:    "magnet:?xt=urn:btih:def789",
			wantHash:  "def789",
			wantTitle: "unknown",
		},
		{
			name:    "invalid magnet",
			magnet:  "not a magnet link",
			wantErr: true,
		},
		{
			name:    "missing xt parameter",
			magnet:  "magnet:?dn=Test+Movie",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			infoHash, title, err := ParseMagnet(tt.magnet)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMagnet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if infoHash != tt.wantHash {
				t.Errorf("ParseMagnet() infoHash = %q, want %q", infoHash, tt.wantHash)
			}
			if title != tt.wantTitle {
				t.Errorf("ParseMagnet() title = %q, want %q", title, tt.wantTitle)
			}
		})
	}
}

func TestIndexer_ParseMagnet(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	magnet := "magnet:?xt=urn:btih:1234567890abcdef&dn=Sample+Movie"
	infoHash, title, err := i.ParseMagnet(context.Background(), magnet)
	if err != nil {
		t.Fatalf("ParseMagnet() error = %v", err)
	}
	if infoHash != "1234567890abcdef" {
		t.Errorf("expected infoHash 1234567890abcdef, got %q", infoHash)
	}
	if title != "Sample Movie" {
		t.Errorf("expected title Sample Movie, got %q", title)
	}
}
