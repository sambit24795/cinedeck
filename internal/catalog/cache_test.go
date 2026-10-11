package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestCacheRepeatedGet(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL), WithCache(t.TempDir(), time.Hour))
	q := url.Values{"query": {"abc"}}
	for i := 0; i < 2; i++ {
		var out map[string]bool
		if err := c.get(context.Background(), "/x", q, &out); err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}

func TestCacheDifferentQueries(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL), WithCache(t.TempDir(), time.Hour))
	for _, v := range []string{"a", "b"} {
		var out map[string]any
		if err := c.get(context.Background(), "/x", url.Values{"query": {v}}, &out); err != nil {
			t.Fatalf("get %q: %v", v, err)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestCacheExpires(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL), WithCache(t.TempDir(), time.Minute))
	base := time.Now()
	c.cache.now = func() time.Time { return base }

	var out map[string]any
	if err := c.get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("first get: %v", err)
	}
	c.cache.now = func() time.Time { return base.Add(2 * time.Minute) }
	if err := c.get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("second get: %v", err)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestCacheDoesNotStoreErrors(t *testing.T) {
	body, err := os.ReadFile("../../testdata/error_401.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(body)
	}))
	defer srv.Close()

	c := New("tok", WithBaseURL(srv.URL), WithCache(t.TempDir(), time.Hour))
	for i := 0; i < 2; i++ {
		if err := c.get(context.Background(), "/x", nil, nil); err == nil {
			t.Fatalf("get %d: want error", i)
		}
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Errorf("calls = %d, want 2", got)
	}
}

func TestCacheKeyIncludesToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer b" {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"status_message":"Invalid API key"}`))
			return
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	// Warm the cache with token a.
	a := New("a", WithBaseURL(srv.URL), WithCache(dir, time.Hour))
	var out map[string]bool
	if err := a.get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("warm get: %v", err)
	}

	// Token b must not reuse a's cache entry: the server answers 401 for b.
	b := New("b", WithBaseURL(srv.URL), WithCache(dir, time.Hour))
	err := b.get(context.Background(), "/x", nil, &out)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusUnauthorized {
		t.Fatalf("get with token b: err = %v, want a 401 APIError", err)
	}
}

func TestCacheWriteFailureDoesNotFailRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	// A regular file where the cache directory should be makes every write fail.
	dir := filepath.Join(t.TempDir(), "notadir")
	if err := os.WriteFile(dir, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := New("tok", WithBaseURL(srv.URL), WithCache(dir, time.Hour))
	var out map[string]bool
	if err := c.get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("get: %v, want nil when only the cache write fails", err)
	}
	if !out["ok"] {
		t.Errorf("out = %v, want ok=true", out)
	}
}

func TestCacheCorruptEntryIsRefetched(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, cacheKey(srv.URL, "tok", "/x", nil)+".json"), []byte("{truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	c := New("tok", WithBaseURL(srv.URL), WithCache(dir, time.Hour))
	var out map[string]bool
	if err := c.get(context.Background(), "/x", nil, &out); err != nil {
		t.Fatalf("get: %v, want a refetch after a corrupt cache entry", err)
	}
	if !out["ok"] || atomic.LoadInt32(&calls) != 1 {
		t.Errorf("out = %v, calls = %d, want ok=true and 1 call", out, calls)
	}
}
