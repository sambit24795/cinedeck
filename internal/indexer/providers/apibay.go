// Package providers holds the built-in torrent indexer provider implementations.
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

// Apibay queries The Pirate Bay's public JSON API (apibay.org).
type Apibay struct {
	BaseURL string
	HTTP    *http.Client
}

// NewApibay returns an Apibay provider with defaults.
func NewApibay() *Apibay {
	return &Apibay{
		BaseURL: "https://apibay.org",
		HTTP:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (a *Apibay) Name() string { return "apibay" }

type apibayItem struct {
	Name     string `json:"name"`
	InfoHash string `json:"info_hash"`
	Seeders  string `json:"seeders"`
	Leechers string `json:"leechers"`
	Size     string `json:"size"`
	Category string `json:"category"`
	Username string `json:"username"`
}

func (a *Apibay) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if a.BaseURL == "" {
		a.BaseURL = "https://apibay.org"
	}
	if a.HTTP == nil {
		a.HTTP = &http.Client{Timeout: 10 * time.Second}
	}

	term := q.Text
	if q.InfoHash != "" {
		term = q.InfoHash
	}

	endpoint := strings.TrimRight(a.BaseURL, "/") + "/q.php?q=" + url.QueryEscape(term)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("apibay: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("apibay: status %d", resp.StatusCode)
	}

	var items []apibayItem
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("apibay: decode: %w", err)
	}

	// The API answers one sentinel object when there are no results.
	if len(items) == 1 && strings.EqualFold(items[0].Name, "No results returned") {
		return []indexer.Release{}, nil
	}

	out := make([]indexer.Release, 0, len(items))
	for _, it := range items {
		seeders, _ := strconv.Atoi(it.Seeders)
		leechers, _ := strconv.Atoi(it.Leechers)
		size, _ := strconv.ParseUint(it.Size, 10, 64)
		if q.MinSeeders > 0 && seeders < q.MinSeeders {
			continue
		}
		out = append(out, indexer.Release{
			Provider: a.Name(),
			Title:    it.Name,
			InfoHash: it.InfoHash,
			Magnet:   magnetFromHash(it.InfoHash, it.Name),
			Size:     size,
			Seeders:  seeders,
			Leechers: leechers,
			Year:     indexer.ParseYearFromTitle(it.Name),
			Uploader: it.Username,
			Category: it.Category,
			InfoURL:  strings.TrimRight(a.BaseURL, "/") + "/torrent/" + it.InfoHash,
		})
	}
	return out, nil
}

// magnetFromHash builds a magnet link for an info hash.
func magnetFromHash(hash, name string) string {
	if hash == "" {
		return ""
	}
	m := "magnet:?xt=urn:btih:" + hash
	if name != "" {
		m += "&dn=" + url.QueryEscape(name)
	}
	return m
}
