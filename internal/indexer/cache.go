package indexer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// cache stores provider search results on disk.
type cache struct {
	dir string
	ttl time.Duration
	now func() time.Time
}

// WithCache enables an on-disk results cache for text searches.
func WithCache(dir string, ttl time.Duration) Option {
	return func(i *Indexer) {
		if dir == "" {
			return
		}
		i.cache = &cache{dir: dir, ttl: ttl, now: time.Now}
	}
}

// cacheKey returns the file key for a (provider, query) pair.
func cacheKey(q Query, provider string) string {
	parts := []string{
		provider,
		q.Text,
		string(q.Type),
		strconv.Itoa(q.Year),
		strconv.Itoa(q.MinSeeders),
		strconv.Itoa(q.Limit),
		q.Sort,
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(sum[:])
}

// get returns the cached releases when the file exists and is fresh.
func (c *cache) get(key string) ([]Release, bool) {
	path := filepath.Join(c.dir, key+".json")
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	if c.now().Sub(info.ModTime()) > c.ttl {
		return nil, false
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var releases []Release
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, false
	}
	return releases, true
}

// put stores releases for key, creating the cache directory when needed.
func (c *cache) put(key string, releases []Release) error {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}
	body, err := json.Marshal(releases)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, key+".json"), body, 0o600)
}
