package indexer

import (
	"context"
	"fmt"
	"sync"
)

// ProviderError records one provider's failure during a search.
type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("%s: %v", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error { return e.Err }

// AggregateResult is the merged output of a multi-provider search.
type AggregateResult struct {
	Releases []Release       `json:"releases"`
	Errors   []ProviderError `json:"errors,omitempty"`
}

// Search runs each provider concurrently and merges the results.
//
// A provider that fails is recorded in Errors and skipped; the call succeeds
// when at least one provider returned results. If every provider fails and no
// releases were found, the first ProviderError is returned as the error.
func (i *Indexer) Search(ctx context.Context, q Query, providers []Provider) (*AggregateResult, error) {
	return i.fanOut(ctx, q, providers, false)
}

// LookupHash looks up one info hash across the providers.
func (i *Indexer) LookupHash(ctx context.Context, infoHash string, providers []Provider) (*AggregateResult, error) {
	q := Query{InfoHash: infoHash, Type: Any}
	return i.fanOut(ctx, q, providers, true)
}

func (i *Indexer) fanOut(ctx context.Context, q Query, providers []Provider, byHash bool) (*AggregateResult, error) {
	if err := q.Validate(); err != nil {
		return nil, err
	}

	type outcome struct {
		provider string
		releases []Release
		err      error
	}

	results := make([]outcome, len(providers))
	var wg sync.WaitGroup
	for idx, p := range providers {
		wg.Add(1)
		go func(idx int, p Provider) {
			defer wg.Done()

			pctx, cancel := context.WithTimeout(ctx, i.timeout)
			defer cancel()

			pq := q
			if byHash {
				pq.Text = ""
				pq.InfoHash = q.InfoHash
			}

			// Hash lookups bypass the cache; text searches use it when enabled.
			if !byHash && i.cache != nil {
				key := cacheKey(pq, p.Name())
				if rel, ok := i.cache.get(key); ok {
					results[idx] = outcome{provider: p.Name(), releases: rel}
					return
				}
				rel, err := p.Search(pctx, pq)
				if err == nil {
					// Best-effort: a cache write failure must not fail the search.
					_ = i.cache.put(key, rel)
				}
				results[idx] = outcome{provider: p.Name(), releases: rel, err: err}
				return
			}

			rel, err := p.Search(pctx, pq)
			results[idx] = outcome{provider: p.Name(), releases: rel, err: err}
		}(idx, p)
	}
	wg.Wait()

	agg := &AggregateResult{Releases: []Release{}, Errors: []ProviderError{}}
	all := make([]Release, 0)
	for _, r := range results {
		if r.err != nil {
			agg.Errors = append(agg.Errors, ProviderError{Provider: r.provider, Err: r.err})
			continue
		}
		all = append(all, r.releases...)
	}

	all = DedupeReleases(all)

	// Apply the query filters defensively so a provider that ignores them
	// cannot leak results past the caller's request.
	filtered := all[:0]
	for _, r := range all {
		if q.MinSeeders > 0 && r.Seeders < q.MinSeeders {
			continue
		}
		filtered = append(filtered, r)
	}
	all = filtered

	sortBy := q.Sort
	if sortBy == "" {
		sortBy = "seeders"
	}
	SortReleases(all, sortBy)
	if q.Limit > 0 && len(all) > q.Limit {
		all = all[:q.Limit]
	}
	agg.Releases = all

	if len(all) == 0 && len(agg.Errors) == len(providers) && len(providers) > 0 {
		first := agg.Errors[0]
		return agg, &first
	}
	return agg, nil
}
