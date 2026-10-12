package providers

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// Nyaa queries the Nyaa anime tracker's RSS feed.
type Nyaa struct {
	BaseURL string
	HTTP    *http.Client
}

// NewNyaa returns a Nyaa provider with defaults.
func NewNyaa() *Nyaa {
	return &Nyaa{
		BaseURL: "https://nyaa.si",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (n *Nyaa) Name() string { return "nyaa" }

// nyaaItem decodes one <item>. Elements in the nyaa namespace are matched by
// their local name, which encoding/xml handles when the struct tags omit the
// namespace prefix.
type nyaaItem struct {
	Title    string `xml:"title"`
	Link     string `xml:"link"`
	GUID     string `xml:"guid"`
	Seeders  string `xml:"seeders"`
	Leechers string `xml:"leechers"`
	InfoHash string `xml:"infoHash"`
	Category string `xml:"category"`
	Size     string `xml:"size"`
}

type nyaaRSS struct {
	Channel struct {
		Items []nyaaItem `xml:"item"`
	} `xml:"channel"`
}

func (n *Nyaa) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if n.BaseURL == "" {
		n.BaseURL = "https://nyaa.si"
	}
	if n.HTTP == nil {
		n.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	endpoint := fmt.Sprintf("%s/?page=rss&q=%s&c=0_0&f=0",
		strings.TrimRight(n.BaseURL, "/"), url.QueryEscape(q.Text))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := n.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("nyaa: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("nyaa: status %d", resp.StatusCode)
	}

	// The feed declares the nyaa namespace; strip it so the local-name tags match.
	dec := xml.NewDecoder(resp.Body)
	var body nyaaRSS
	if err := dec.Decode(&body); err != nil {
		return nil, fmt.Errorf("nyaa: decode: %w", err)
	}

	out := make([]indexer.Release, 0, len(body.Channel.Items))
	for _, it := range body.Channel.Items {
		seeders, _ := strconv.Atoi(it.Seeders)
		leechers, _ := strconv.Atoi(it.Leechers)
		if q.MinSeeders > 0 && seeders < q.MinSeeders {
			continue
		}
		size, _ := parseSize(it.Size)
		out = append(out, indexer.Release{
			Provider: n.Name(),
			Title:    it.Title,
			InfoHash: it.InfoHash,
			Magnet:   magnetFromHash(it.InfoHash, it.Title),
			Size:     size,
			Seeders:  seeders,
			Leechers: leechers,
			Year:     indexer.ParseYearFromTitle(it.Title),
			Category: it.Category,
			InfoURL:  it.GUID,
		})
	}
	return out, nil
}

// parseSize converts strings like "1.4 GiB", "700 MiB", "12.3 GB" to bytes.
func parseSize(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty size")
	}
	parts := strings.Fields(s)
	if len(parts) != 2 {
		return 0, fmt.Errorf("bad size %q", s)
	}
	value, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return 0, fmt.Errorf("bad size %q: %w", s, err)
	}
	mult := map[string]float64{
		"B":  1,
		"KB": 1000, "KIB": 1024,
		"MB": 1000 * 1000, "MIB": 1024 * 1024,
		"GB": 1000 * 1000 * 1000, "GIB": 1024 * 1024 * 1024,
		"TB": 1e12, "TIB": 1024 * 1024 * 1024 * 1024,
	}[strings.ToUpper(parts[1])]
	if mult == 0 {
		return 0, fmt.Errorf("unknown size unit %q", parts[1])
	}
	return uint64(value * mult), nil
}
