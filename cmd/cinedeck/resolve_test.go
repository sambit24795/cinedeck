package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/sambit24795/cinedeck/internal/catalog"
	"github.com/spf13/cobra"
)

// fakeSearcher is a test double for the searcher interface.
type fakeSearcher struct {
	results []catalog.Result
	err     error
	calls   int
}

func (f *fakeSearcher) Search(ctx context.Context, query string) ([]catalog.Result, error) {
	f.calls++
	return f.results, f.err
}

// readCounter counts how many times stdin is read.
type readCounter struct {
	*strings.Reader
	reads int
}

func (r *readCounter) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func newResolveCmd(in *readCounter, out *bytes.Buffer) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(out)
	cmd.SetErr(&bytes.Buffer{})
	if in != nil {
		cmd.SetIn(in)
	}
	return cmd
}

var twoResults = []catalog.Result{
	{ID: catalog.ID{Type: catalog.Movie, N: 1}, Title: "A", Year: 2000},
	{ID: catalog.ID{Type: catalog.TV, N: 2}, Title: "B", Year: 2001},
}

func TestResolveIDTyped(t *testing.T) {
	var out bytes.Buffer
	cmd := newResolveCmd(nil, &out)
	f := &fakeSearcher{}

	id, err := resolveID(cmd, f, "movie:27205", false)
	if err != nil {
		t.Fatalf("resolveID: %v", err)
	}
	if id != (catalog.ID{Type: catalog.Movie, N: 27205}) {
		t.Errorf("id = %v, want movie:27205", id)
	}
	if f.calls != 0 {
		t.Errorf("calls = %d, want 0", f.calls)
	}
}

func TestResolveIDSingleResult(t *testing.T) {
	var out bytes.Buffer
	cmd := newResolveCmd(nil, &out)
	f := &fakeSearcher{results: []catalog.Result{twoResults[0]}}

	id, err := resolveID(cmd, f, "a", false)
	if err != nil {
		t.Fatalf("resolveID: %v", err)
	}
	if id != twoResults[0].ID {
		t.Errorf("id = %v, want %v", id, twoResults[0].ID)
	}
}

func TestResolveIDFirst(t *testing.T) {
	var out bytes.Buffer
	in := &readCounter{Reader: strings.NewReader("2\n")}
	cmd := newResolveCmd(in, &out)
	f := &fakeSearcher{results: twoResults}

	id, err := resolveID(cmd, f, "a", true)
	if err != nil {
		t.Fatalf("resolveID: %v", err)
	}
	if id != twoResults[0].ID {
		t.Errorf("id = %v, want %v", id, twoResults[0].ID)
	}
	if in.reads != 0 {
		t.Errorf("stdin reads = %d, want 0", in.reads)
	}
}

func TestResolveIDPicksFromList(t *testing.T) {
	var out bytes.Buffer
	in := &readCounter{Reader: strings.NewReader("2\n")}
	cmd := newResolveCmd(in, &out)
	f := &fakeSearcher{results: twoResults}

	id, err := resolveID(cmd, f, "a", false)
	if err != nil {
		t.Fatalf("resolveID: %v", err)
	}
	if id != (catalog.ID{Type: catalog.TV, N: 2}) {
		t.Errorf("id = %v, want tv:2", id)
	}
	got := out.String()
	for _, want := range []string{"1) A (2000) [movie:1]", "Pick a number: "} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want it to contain %q", got, want)
		}
	}
}

func TestResolveIDNoMatch(t *testing.T) {
	var out bytes.Buffer
	cmd := newResolveCmd(nil, &out)
	f := &fakeSearcher{}

	_, err := resolveID(cmd, f, "zzz", false)
	if err == nil {
		t.Fatal("resolveID = nil, want error")
	}
	if !strings.Contains(err.Error(), `no match for "zzz"`) {
		t.Errorf("error = %q, want it to contain %q", err, `no match for "zzz"`)
	}
}

func TestResolveIDSearchError(t *testing.T) {
	var out bytes.Buffer
	cmd := newResolveCmd(nil, &out)
	sentinel := errors.New("boom")
	f := &fakeSearcher{err: sentinel}

	_, err := resolveID(cmd, f, "a", false)
	if !errors.Is(err, sentinel) {
		t.Errorf("error = %v, want %v", err, sentinel)
	}
}
