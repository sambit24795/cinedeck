package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// EZTV queries the EZTV TV-show API.
type EZTV struct {
	BaseURL string
	HTTP    *http.Client
}

// NewEZTV returns an EZTV provider with defaults.
func NewEZTV() *EZTV {
	return &EZTV{
		BaseURL: "https://eztv.re",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (e *EZTV) Name() string { return "eztv" }

type eztvResponse struct {
	Torrents []struct {
		Title   string `json:"title"`
		Hash    string `json:"hash"`
		Magnet  string `json:"magnet"`
		SizeRaw string `json:"size_bytes"`
		Seeds   int    `json:"seeds"`
		Peers   int    `json:"peers"`
		Season  int    `json:"season"`
		Episode int    `json:"episode"`
	} `json:"torrents"`
}

func (e *EZTV) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if q.Type == indexer.Movie {
		return []indexer.Release{}, nil
	}
	if e.BaseURL == "" {
		e.BaseURL = "https://eztv.re"
	}
	if e.HTTP == nil {
		e.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 100
	}
	endpoint := fmt.Sprintf("%s/api/get-torrents?limit=%d&page=1", strings.TrimRight(e.BaseURL, "/"), limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := e.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eztv: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("eztv: status %d", resp.StatusCode)
	}

	var body eztvResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("eztv: decode: %w", err)
	}

	needle := strings.ToLower(strings.TrimSpace(q.Text))
	out := make([]indexer.Release, 0)
	for _, t := range body.Torrents {
		if needle != "" && !strings.Contains(strings.ToLower(t.Title), needle) {
			continue
		}
		if q.MinSeeders > 0 && t.Seeds < q.MinSeeders {
			continue
		}
		size, _ := strconv.ParseUint(t.SizeRaw, 10, 64)
		magnet := t.Magnet
		if magnet == "" {
			magnet = magnetFromHash(t.Hash, t.Title)
		}
		out = append(out, indexer.Release{
			Provider: e.Name(),
			Title:    t.Title,
			InfoHash: t.Hash,
			Magnet:   magnet,
			Size:     size,
			Seeders:  t.Seeds,
			Leechers: t.Peers,
			Category: fmt.Sprintf("S%02dE%02d", t.Season, t.Episode),
		})
	}
	return out, nil
}
