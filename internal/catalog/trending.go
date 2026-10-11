package catalog

import "context"

// Trending returns the weekly trending movies or TV shows for t.
func (c *Client) Trending(ctx context.Context, t MediaType) ([]Result, error) {
	var resp struct {
		Results []item `json:"results"`
	}
	if err := c.get(ctx, "/trending/"+string(t)+"/week", nil, &resp); err != nil {
		return nil, err
	}

	results := make([]Result, 0, len(resp.Results))
	for _, it := range resp.Results {
		results = append(results, it.result(t))
	}
	return results, nil
}
