package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// defaultBaseURL is the TMDB v3 API root.
const defaultBaseURL = "https://api.themoviedb.org/3"

// maxAttempts is the total number of tries for a request, including retries
// after HTTP 429 responses.
const maxAttempts = 3

// Client talks to the TMDB API.
type Client struct {
	token     string
	baseURL   string
	http      *http.Client
	retryWait time.Duration
	cache     *cache
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL sets the API root. The default is https://api.themoviedb.org/3.
func WithBaseURL(u string) Option {
	return func(c *Client) { c.baseURL = u }
}

// WithHTTPClient sets the HTTP client. The default times out after 10 seconds.
func WithHTTPClient(h *http.Client) Option {
	return func(c *Client) { c.http = h }
}

// WithRetryWait sets the wait between retries after HTTP 429 responses. The
// default is 1 second.
func WithRetryWait(d time.Duration) Option {
	return func(c *Client) { c.retryWait = d }
}

// WithCache enables the on-disk response cache in dir with the given ttl.
// Without this option responses are not cached.
func WithCache(dir string, ttl time.Duration) Option {
	return func(c *Client) {
		c.cache = &cache{dir: dir, ttl: ttl, now: time.Now}
	}
}

// New returns a Client that authenticates with token.
func New(token string, opts ...Option) *Client {
	c := &Client{
		token:     token,
		baseURL:   defaultBaseURL,
		http:      &http.Client{Timeout: 10 * time.Second},
		retryWait: 1 * time.Second,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// APIError is a non-2xx response from TMDB.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("tmdb: %d: %s", e.Status, e.Message)
}

// get performs GET baseURL+path with the query values q and decodes a 2xx
// body into out. It retries HTTP 429 up to maxAttempts times.
func (c *Client) get(ctx context.Context, path string, q url.Values, out any) error {
	u := c.baseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}

	key := cacheKey(path, q)
	if c.cache != nil {
		if body, ok := c.cache.get(key); ok {
			return json.Unmarshal(body, out)
		}
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/json")

		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt == maxAttempts {
				return &APIError{Status: resp.StatusCode, Message: statusMessage(body)}
			}
			wait := c.retryWait
			if ra := resp.Header.Get("Retry-After"); ra != "" {
				if secs, err := strconv.Atoi(ra); err == nil {
					wait = time.Duration(secs) * time.Second
				}
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
			continue
		}

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			return &APIError{Status: resp.StatusCode, Message: statusMessage(body)}
		}

		if c.cache != nil {
			if err := c.cache.put(key, body); err != nil {
				return err
			}
		}
		return json.Unmarshal(body, out)
	}
	return nil
}

// statusMessage extracts the JSON "status_message" field, returning "" when
// the body is not JSON.
func statusMessage(body []byte) string {
	var e struct {
		StatusMessage string `json:"status_message"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		return ""
	}
	return e.StatusMessage
}
