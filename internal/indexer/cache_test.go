package indexer

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCacheTwoIdenticalSearchesHitProviderOnce(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "fake", releases: []Release{
		{Provider: "fake", Title: "Inception", InfoHash: "h1", Seeders: 10},
	}}
	dir := t.TempDir()
	i := NewIndexer(WithCache(dir, time.Hour))

	q := Query{Text: "inception"}
	if _, err := i.Search(context.Background(), q, []Provider{p}); err != nil {
		t.Fatalf("first Search: %v", err)
	}
	if _, err := i.Search(context.Background(), q, []Provider{p}); err != nil {
		t.Fatalf("second Search: %v", err)
	}
	if p.calls != 1 {
		t.Errorf("provider calls = %d, want 1 (second call served from cache)", p.calls)
	}
}

func TestCacheDifferentQueryHitsProviderTwice(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "fake", releases: []Release{}}
	i := NewIndexer(WithCache(t.TempDir(), time.Hour))

	if _, err := i.Search(context.Background(), Query{Text: "a"}, []Provider{p}); err != nil {
		t.Fatalf("Search a: %v", err)
	}
	if _, err := i.Search(context.Background(), Query{Text: "b"}, []Provider{p}); err != nil {
		t.Fatalf("Search b: %v", err)
	}
	if p.calls != 2 {
		t.Errorf("provider calls = %d, want 2", p.calls)
	}
}

func TestCacheErrorNotCached(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "fake", err: errors.New("boom")}
	i := NewIndexer(WithCache(t.TempDir(), time.Hour))

	_, _ = i.Search(context.Background(), Query{Text: "x"}, []Provider{p})
	_, _ = i.Search(context.Background(), Query{Text: "x"}, []Provider{p})
	if p.calls != 2 {
		t.Errorf("provider calls = %d, want 2 (errors must not be cached)", p.calls)
	}
}

func TestCacheExpires(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "fake", releases: []Release{}}
	dir := t.TempDir()
	i := NewIndexer(WithCache(dir, time.Minute))

	if _, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{p}); err != nil {
		t.Fatalf("first Search: %v", err)
	}
	// Move the cache's clock forward past the TTL.
	i.cache.now = func() time.Time { return time.Now().Add(time.Hour) }
	if _, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{p}); err != nil {
		t.Fatalf("second Search: %v", err)
	}
	if p.calls != 2 {
		t.Errorf("provider calls = %d, want 2 after TTL expiry", p.calls)
	}
}

func TestCacheSkipsHashLookup(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "fake", releases: []Release{{InfoHash: "h1", Seeders: 1}}}
	i := NewIndexer(WithCache(t.TempDir(), time.Hour))

	if _, err := i.LookupHash(context.Background(), "h1", []Provider{p}); err != nil {
		t.Fatalf("first LookupHash: %v", err)
	}
	if _, err := i.LookupHash(context.Background(), "h1", []Provider{p}); err != nil {
		t.Fatalf("second LookupHash: %v", err)
	}
	if p.calls != 2 {
		t.Errorf("provider calls = %d, want 2 (hash lookups bypass the cache)", p.calls)
	}
}
