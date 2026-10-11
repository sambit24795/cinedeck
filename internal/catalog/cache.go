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

// cacheKey returns the cache file key for a request. The key is the
// hex-encoded SHA-256 of baseURL, a SHA-256 hash of token, the path and the
// encoded query, joined by newlines:
//
//	sha256(baseURL + "\n" + hex(sha256(token)) + "\n" + path + "?" + q.Encode())
//
// Binding the key to baseURL and the token means a cache entry written for one
// endpoint or credential is never reused for another. The token is hashed here
// so it is never written to any file, file name, output or error message.
func cacheKey(baseURL, token, path string, q url.Values) string {
	tok := sha256.Sum256([]byte(token))
	sum := sha256.Sum256([]byte(baseURL + "\n" + hex.EncodeToString(tok[:]) + "\n" + path + "?" + q.Encode()))
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
