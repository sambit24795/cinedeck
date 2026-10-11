package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
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
