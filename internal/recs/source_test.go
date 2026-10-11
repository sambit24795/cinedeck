package recs

import (
	"context"
	"errors"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
)

// fakeSource is the Source used by every recs test. It records every
// DiscoverBy argument and the total number of method calls.
type fakeSource struct {
	metas    map[catalog.ID]*catalog.Meta
	recs     map[catalog.ID][]catalog.Related
	sims     map[catalog.ID][]catalog.Related
	discover func(q catalog.DiscoverQuery) []catalog.Related // nil means no results
	err      error                                           // returned by every method when set
	calls    int                                             // total number of method calls
	queries  []catalog.DiscoverQuery                         // every DiscoverBy argument, in order
}

var _ Source = (*fakeSource)(nil)

func (f *fakeSource) Meta(_ context.Context, id catalog.ID) (*catalog.Meta, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	if m, ok := f.metas[id]; ok {
		return m, nil
	}
	return nil, &catalog.APIError{Status: 404, Message: "not found"}
}

func (f *fakeSource) Recommendations(_ context.Context, id catalog.ID) ([]catalog.Related, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.recs[id], nil
}

func (f *fakeSource) Similar(_ context.Context, id catalog.ID) ([]catalog.Related, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.sims[id], nil
}

func (f *fakeSource) DiscoverBy(_ context.Context, q catalog.DiscoverQuery) ([]catalog.Related, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	f.queries = append(f.queries, q)
	if f.discover == nil {
		return nil, nil
	}
	return f.discover(q), nil
}

func TestFakeSource(t *testing.T) {
	var _ Source = (*fakeSource)(nil)

	f := &fakeSource{}
	_, err := f.Meta(context.Background(), catalog.ID{Type: catalog.Movie, N: 1})
	var apiErr *catalog.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want *catalog.APIError", err)
	}
	if apiErr.Status != 404 {
		t.Errorf("Status = %d, want 404", apiErr.Status)
	}
}
