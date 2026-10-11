package catalog

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// Provider is one streaming service in a region.
type Provider struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// WhereToWatch is the availability of one title in one region.
type WhereToWatch struct {
	ID     ID         `json:"id"`
	Region string     `json:"region"`
	Link   string     `json:"link"`
	Stream []Provider `json:"stream"` // TMDB "flatrate"
	Free   []Provider `json:"free"`
	Ads    []Provider `json:"ads"`
	Rent   []Provider `json:"rent"`
	Buy    []Provider `json:"buy"`
}

// ErrNoProviders means the region has no entry, or an entry with no providers.
var ErrNoProviders = errors.New("no watch providers")

// watchProvider mirrors one provider in a TMDB watch/providers response.
type watchProvider struct {
	ProviderID      int    `json:"provider_id"`
	ProviderName    string `json:"provider_name"`
	DisplayPriority int    `json:"display_priority"`
}

// watchRegion mirrors one region in a TMDB watch/providers response.
type watchRegion struct {
	Link     string          `json:"link"`
	Flatrate []watchProvider `json:"flatrate"`
	Free     []watchProvider `json:"free"`
	Ads      []watchProvider `json:"ads"`
	Rent     []watchProvider `json:"rent"`
	Buy      []watchProvider `json:"buy"`
}

// WhereToWatch returns the watch providers for id in region. The TMDB
// endpoint returns every region at once, so the region is picked client-side.
func (c *Client) WhereToWatch(ctx context.Context, id ID, region string) (*WhereToWatch, error) {
	norm := strings.ToUpper(strings.TrimSpace(region))
	if len(norm) != 2 || norm[0] < 'A' || norm[0] > 'Z' || norm[1] < 'A' || norm[1] > 'Z' {
		return nil, fmt.Errorf("invalid region %q: want a 2-letter code like IN", region)
	}

	var resp struct {
		Results map[string]watchRegion `json:"results"`
	}
	path := "/" + string(id.Type) + "/" + strconv.FormatInt(id.N, 10) + "/watch/providers"
	if err := c.get(ctx, path, nil, &resp); err != nil {
		return nil, err
	}

	r, ok := resp.Results[norm]
	if !ok || (len(r.Flatrate) == 0 && len(r.Free) == 0 && len(r.Ads) == 0 && len(r.Rent) == 0 && len(r.Buy) == 0) {
		return nil, ErrNoProviders
	}

	return &WhereToWatch{
		ID:     id,
		Region: norm,
		Link:   r.Link,
		Stream: toProviders(r.Flatrate),
		Free:   toProviders(r.Free),
		Ads:    toProviders(r.Ads),
		Rent:   toProviders(r.Rent),
		Buy:    toProviders(r.Buy),
	}, nil
}

// toProviders converts TMDB providers to []Provider, sorted by display
// priority ascending, then lower-case name, then id. The result is never nil.
func toProviders(ps []watchProvider) []Provider {
	sort.Slice(ps, func(i, j int) bool {
		if ps[i].DisplayPriority != ps[j].DisplayPriority {
			return ps[i].DisplayPriority < ps[j].DisplayPriority
		}
		li, lj := strings.ToLower(ps[i].ProviderName), strings.ToLower(ps[j].ProviderName)
		if li != lj {
			return li < lj
		}
		return ps[i].ProviderID < ps[j].ProviderID
	})

	out := make([]Provider, 0, len(ps))
	for _, p := range ps {
		out = append(out, Provider{ID: p.ProviderID, Name: p.ProviderName})
	}
	return out
}
