package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// YTS queries the YTS movie API.
type YTS struct {
	BaseURL string
	HTTP    *http.Client
}

// NewYTS returns a YTS provider with defaults.
func NewYTS() *YTS {
	return &YTS{
		BaseURL: "https://yts.mx",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (y *YTS) Name() string { return "yts" }

type ytsResponse struct {
	Data struct {
		Movies []struct {
			Title    string `json:"title"`
			Year     int    `json:"year"`
			Torrents []struct {
				Hash             string `json:"hash"`
				Quality          string `json:"quality"`
				Type             string `json:"type"`
				Seeds            int    `json:"seeds"`
				Peers            int    `json:"peers"`
				SizeBytes        int64  `json:"size_bytes"`
				DateUploadedUnix int64  `json:"date_uploaded_unix"`
			} `json:"torrents"`
		} `json:"movies"`
	} `json:"data"`
}

func (y *YTS) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if q.Type == indexer.TV {
		return []indexer.Release{}, nil
	}
	if y.BaseURL == "" {
		y.BaseURL = "https://yts.mx"
	}
	if y.HTTP == nil {
		y.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	limit := q.Limit
	if limit <= 0 {
		limit = 20
	}
	endpoint := fmt.Sprintf("%s/api/v2/list_movies.json?query_term=%s&limit=%d",
		strings.TrimRight(y.BaseURL, "/"), url.QueryEscape(q.Text), limit)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := y.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("yts: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("yts: status %d", resp.StatusCode)
	}

	var body ytsResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("yts: decode: %w", err)
	}

	out := make([]indexer.Release, 0)
	for _, m := range body.Data.Movies {
		for _, tr := range m.Torrents {
			if q.MinSeeders > 0 && tr.Seeds < q.MinSeeders {
				continue
			}
			title := m.Title
			if m.Year > 0 {
				title = fmt.Sprintf("%s %d", m.Title, m.Year)
			}
			if tr.Quality != "" {
				title = title + " " + tr.Quality
			}
			out = append(out, indexer.Release{
				Provider: y.Name(),
				Title:    title,
				InfoHash: tr.Hash,
				Magnet:   magnetFromHash(tr.Hash, title),
				Size:     uint64(tr.SizeBytes),
				Seeders:  tr.Seeds,
				Leechers: tr.Peers,
				Year:     m.Year,
				Category: tr.Type,
				InfoURL:  strings.TrimRight(y.BaseURL, "/") + "/movies/" + slugify(m.Title) + "-" + strconv.Itoa(m.Year),
			})
		}
	}
	return out, nil
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
