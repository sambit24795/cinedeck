package indexer

import (
	"testing"
)

func TestNewIndexer(t *testing.T) {
	t.Parallel()

	i := NewIndexer()
	if i == nil {
		t.Fatal("expected non-nil Indexer")
	}
	if i.trackerURL != "" {
		t.Errorf("expected empty tracker URL, got %q", i.trackerURL)
	}
	if i.dhtEnabled {
		t.Error("expected DHT disabled by default")
	}
	if i.privateKey != "" {
		t.Error("expected empty private key by default")
	}
}

func TestWithTrackerURL(t *testing.T) {
	t.Parallel()

	i := NewIndexer(
		WithTrackerURL("http://tracker.example.com"),
	)
	if i.trackerURL != "http://tracker.example.com" {
		t.Errorf("expected tracker URL, got %q", i.trackerURL)
	}
}

func TestWithDHT(t *testing.T) {
	t.Parallel()

	i1 := NewIndexer(WithDHT(true))
	if !i1.dhtEnabled {
		t.Error("expected DHT enabled")
	}

	i2 := NewIndexer(WithDHT(false))
	if i2.dhtEnabled {
		t.Error("expected DHT disabled")
	}
}

func TestWithPrivateKey(t *testing.T) {
	t.Parallel()

	i := NewIndexer(WithPrivateKey("secret-key-12345"))
	if i.privateKey != "secret-key-12345" {
		t.Errorf("expected private key, got %q", i.privateKey)
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
	infoHash, title, err := i.ParseMagnet(nil, magnet)
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
