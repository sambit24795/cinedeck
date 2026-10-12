package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/sambit24795/cinedeck/internal/indexer"
)

// Archive queries the Internet Archive for public-domain movies. It is the
// legal provider and reports no seeder/leecher counts.
type Archive struct {
	BaseURL string
	HTTP    *http.Client
}

// NewArchive returns an Internet Archive provider with defaults.
func NewArchive() *Archive {
	return &Archive{
		BaseURL: "https://archive.org",
		HTTP:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *Archive) Name() string { return "archive" }

type archiveResponse struct {
	Response struct {
		Docs []struct {
			Identifier string      `json:"identifier"`
			Title      string      `json:"title"`
			Year       archiveYear `json:"year"`
		} `json:"docs"`
	} `json:"response"`
}

// archiveYear accepts the Internet Archive's year as either a string or a
// number; both appear in the wild.
type archiveYear string

func (y *archiveYear) UnmarshalJSON(data []byte) error {
	if len(data) >= 2 && data[0] == '"' && data[len(data)-1] == '"' {
		*y = archiveYear(strings.Trim(string(data), `"`))
		return nil
	}
	// A bare number (possibly fractional) becomes its string form.
	*y = archiveYear(string(data))
	return nil
}

func (a *Archive) Search(ctx context.Context, q indexer.Query) ([]indexer.Release, error) {
	if a.BaseURL == "" {
		a.BaseURL = "https://archive.org"
	}
	if a.HTTP == nil {
		a.HTTP = &http.Client{Timeout: 15 * time.Second}
	}

	base := strings.TrimRight(a.BaseURL, "/")
	endpoint := fmt.Sprintf(
		"%s/advancedsearch.php?q=%s&fl[]=identifier&fl[]=title&fl[]=year&rows=20&output=json",
		base, url.QueryEscape(q.Text+" AND mediatype:movies"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("archive: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("archive: status %d", resp.StatusCode)
	}

	var body archiveResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("archive: decode: %w", err)
	}

	out := make([]indexer.Release, 0, len(body.Response.Docs))
	for _, d := range body.Response.Docs {
		out = append(out, indexer.Release{
			Provider: a.Name(),
			Title:    d.Title,
			Year:     indexer.ParseYearFromTitle(string(d.Year)),
			Category: "public domain",
			InfoURL:  base + "/details/" + d.Identifier,
		})
	}
	return out, nil
}
