package catalog

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"os"
	"path/filepath"
	"time"
)

// cache stores raw API response bodies on disk.
type cache struct {
	dir string
	ttl time.Duration
	now func() time.Time
}

// cacheKey returns the hex-encoded SHA-256 of path+"?"+q.Encode().
func cacheKey(path string, q url.Values) string {
	sum := sha256.Sum256([]byte(path + "?" + q.Encode()))
	return hex.EncodeToString(sum[:])
}

// get returns the cached body for key when the file exists and is fresh.
func (c *cache) get(key string) ([]byte, bool) {
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
	return body, true
}

// put stores body for key, creating the cache directory when needed.
func (c *cache) put(key string, body []byte) error {
	if err := os.MkdirAll(c.dir, 0o700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dir, key+".json"), body, 0o600)
}
