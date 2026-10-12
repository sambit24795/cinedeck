package indexer

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvider struct {
	name     string
	releases []Release
	err      error
	delay    time.Duration
	calls    int
	lastQ    Query
}

func (f *fakeProvider) Name() string { return f.name }

func (f *fakeProvider) Search(ctx context.Context, q Query) ([]Release, error) {
	f.calls++
	f.lastQ = q
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.releases, nil
}

func TestSearchMergesAndDedupes(t *testing.T) {
	t.Parallel()

	a := &fakeProvider{name: "a", releases: []Release{
		{Provider: "a", Title: "one", InfoHash: "h1", Seeders: 10},
		{Provider: "a", Title: "two", InfoHash: "h2", Seeders: 30},
	}}
	b := &fakeProvider{name: "b", releases: []Release{
		{Provider: "b", Title: "one", InfoHash: "h1", Seeders: 99},
		{Provider: "b", Title: "three", InfoHash: "h3", Seeders: 20},
	}}

	i := NewIndexer()
	got, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{a, b})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got.Releases) != 3 {
		t.Fatalf("len(Releases) = %d, want 3: %+v", len(got.Releases), got.Releases)
	}
	if got.Releases[0].InfoHash != "h1" || got.Releases[0].Seeders != 99 {
		t.Errorf("h1 should keep the higher-seeded entry and sort first: %+v", got.Releases[0])
	}
	if len(got.Errors) != 0 {
		t.Errorf("expected no errors, got %+v", got.Errors)
	}
}

func TestSearchOneProviderFails(t *testing.T) {
	t.Parallel()

	bad := &fakeProvider{name: "bad", err: errors.New("boom")}
	good := &fakeProvider{name: "good", releases: []Release{
		{Provider: "good", Title: "ok", InfoHash: "h1", Seeders: 5},
	}}

	i := NewIndexer()
	got, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{bad, good})
	if err != nil {
		t.Fatalf("Search() error = %v, want nil", err)
	}
	if len(got.Releases) != 1 {
		t.Fatalf("len(Releases) = %d, want 1", len(got.Releases))
	}
	if len(got.Errors) != 1 || got.Errors[0].Provider != "bad" {
		t.Fatalf("Errors = %+v, want one entry for bad", got.Errors)
	}
}

func TestSearchAllProvidersFail(t *testing.T) {
	t.Parallel()

	bad1 := &fakeProvider{name: "bad1", err: errors.New("boom1")}
	bad2 := &fakeProvider{name: "bad2", err: errors.New("boom2")}

	i := NewIndexer()
	_, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{bad1, bad2})
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("Search() error = %v, want *ProviderError", err)
	}
	if pe.Provider != "bad1" {
		t.Errorf("ProviderError.Provider = %q, want bad1", pe.Provider)
	}
}

func TestSearchTimeout(t *testing.T) {
	t.Parallel()

	slow := &fakeProvider{name: "slow", delay: 100 * time.Millisecond, releases: []Release{{InfoHash: "h", Seeders: 1}}}
	fast := &fakeProvider{name: "fast", releases: []Release{{Provider: "fast", InfoHash: "h2", Seeders: 2}}}

	i := NewIndexer(WithTimeout(10 * time.Millisecond))
	got, err := i.Search(context.Background(), Query{Text: "x"}, []Provider{slow, fast})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got.Releases) != 1 {
		t.Fatalf("expected only the fast provider's release, got %+v", got.Releases)
	}
	if len(got.Errors) != 1 || got.Errors[0].Provider != "slow" {
		t.Fatalf("expected a timeout error for slow, got %+v", got.Errors)
	}
}

func TestSearchInvalidQuery(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "a"}
	i := NewIndexer()
	_, err := i.Search(context.Background(), Query{}, []Provider{p})
	if err == nil {
		t.Fatal("expected an error for an empty query")
	}
	if p.calls != 0 {
		t.Errorf("provider should not be called for an invalid query, got %d calls", p.calls)
	}
}

func TestLookupHash(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "a", releases: []Release{{Provider: "a", InfoHash: "h1", Seeders: 3}}}
	i := NewIndexer()
	got, err := i.LookupHash(context.Background(), "h1", []Provider{p})
	if err != nil {
		t.Fatalf("LookupHash() error = %v", err)
	}
	if len(got.Releases) != 1 {
		t.Fatalf("len(Releases) = %d, want 1", len(got.Releases))
	}
	if p.lastQ.InfoHash != "h1" {
		t.Errorf("provider query InfoHash = %q, want h1", p.lastQ.InfoHash)
	}
}

func TestSearchLimit(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "a", releases: []Release{
		{Provider: "a", InfoHash: "h1", Seeders: 3},
		{Provider: "a", InfoHash: "h2", Seeders: 2},
		{Provider: "a", InfoHash: "h3", Seeders: 1},
	}}
	i := NewIndexer()
	got, err := i.Search(context.Background(), Query{Text: "x", Limit: 2}, []Provider{p})
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(got.Releases) != 2 {
		t.Fatalf("len(Releases) = %d, want 2", len(got.Releases))
	}
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	a := &fakeProvider{name: "a"}
	b := &fakeProvider{name: "b"}
	r.Register(a)
	r.Register(b)

	if len(r.Providers()) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(r.Providers()))
	}
	if got := r.Enabled(nil); len(got) != 2 {
		t.Fatalf("Enabled(nil) = %d providers, want 2", len(got))
	}
	if got := r.Enabled([]string{"a"}); len(got) != 1 || got[0].Name() != "a" {
		t.Fatalf("Enabled([a]) = %+v", got)
	}
	if got := r.Enabled([]string{"nope"}); len(got) != 0 {
		t.Fatalf("Enabled([nope]) = %d providers, want 0", len(got))
	}
}

func TestRegistryRegisterReplaces(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	r.Register(&fakeProvider{name: "a", releases: []Release{{Seeders: 1}}})
	r.Register(&fakeProvider{name: "b"})
	r.Register(&fakeProvider{name: "a", releases: []Release{{Seeders: 2}}})

	got := r.Providers()
	if len(got) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(got))
	}
	if got[0].Name() != "a" || got[1].Name() != "b" {
		t.Fatalf("order should keep a first: %v", r.Names())
	}
}

func TestLookupHashBypassesText(t *testing.T) {
	t.Parallel()

	p := &fakeProvider{name: "a"}
	i := NewIndexer()
	if _, err := i.LookupHash(context.Background(), "h1", []Provider{p}); err != nil {
		t.Fatalf("LookupHash() error = %v", err)
	}
	if p.lastQ.Text != "" {
		t.Errorf("LookupHash should clear Text, got %q", p.lastQ.Text)
	}
}
