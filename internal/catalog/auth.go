package catalog

import (
	"context"
	"errors"
)

// CheckToken makes one request to GET /authentication and reports whether
// TMDB accepts the client's token. Use a client built without WithCache: a
// cached answer would hide a revoked token.
func (c *Client) CheckToken(ctx context.Context) error {
	var resp struct {
		Success bool `json:"success"`
	}
	if err := c.get(ctx, "/authentication", nil, &resp); err != nil {
		return err
	}
	if !resp.Success {
		return errors.New("tmdb: token not accepted")
	}
	return nil
}
