package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
)

func readWatchFixture(t *testing.T, name string) []byte {
	t.Helper()
	body, err := os.ReadFile("../../testdata/" + name)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	return body
}

func providerNames(ps []Provider) []string {
	names := make([]string, 0, len(ps))
	for _, p := range ps {
		names = append(names, p.Name)
	}
	return names
}

func newWatchClient(t *testing.T, body []byte, gotPath *string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotPath != nil {
			*gotPath = r.URL.Path
		}
		w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return New("tok", WithBaseURL(srv.URL))
}

func TestWhereToWatchMovieIN(t *testing.T) {
	var path string
	c := newWatchClient(t, readWatchFixture(t, "watch_movie_27205.json"), &path)

	got, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, "IN")
	if err != nil {
		t.Fatalf("WhereToWatch: %v", err)
	}
	if path != "/movie/27205/watch/providers" {
		t.Errorf("path = %q, want %q", path, "/movie/27205/watch/providers")
	}
	if got.Region != "IN" {
		t.Errorf("Region = %q, want %q", got.Region, "IN")
	}
	if !strings.HasSuffix(got.Link, "locale=IN") {
		t.Errorf("Link = %q, want it to end with %q", got.Link, "locale=IN")
	}
	if want := []string{"JioHotstar", "Netflix", "Amazon Prime Video"}; !reflect.DeepEqual(providerNames(got.Stream), want) {
		t.Errorf("Stream = %v, want %v", providerNames(got.Stream), want)
	}
	if want := []string{"Apple TV", "Google Play Movies"}; !reflect.DeepEqual(providerNames(got.Rent), want) {
		t.Errorf("Rent = %v, want %v", providerNames(got.Rent), want)
	}
	if want := []string{"Apple TV"}; !reflect.DeepEqual(providerNames(got.Buy), want) {
		t.Errorf("Buy = %v, want %v", providerNames(got.Buy), want)
	}
	if got.Free == nil || len(got.Free) != 0 {
		t.Errorf("Free = %#v, want non-nil and empty", got.Free)
	}
	if got.Ads == nil || len(got.Ads) != 0 {
		t.Errorf("Ads = %#v, want non-nil and empty", got.Ads)
	}
}

func TestWhereToWatchRegionNormalised(t *testing.T) {
	c := newWatchClient(t, readWatchFixture(t, "watch_movie_27205.json"), nil)

	in, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, "IN")
	if err != nil {
		t.Fatalf("WhereToWatch(IN): %v", err)
	}
	spaces, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, " in ")
	if err != nil {
		t.Fatalf("WhereToWatch(\" in \"): %v", err)
	}
	if !reflect.DeepEqual(in, spaces) {
		t.Errorf("WhereToWatch(\" in \") = %#v, want %#v", spaces, in)
	}
}

func TestWhereToWatchMovieUS(t *testing.T) {
	c := newWatchClient(t, readWatchFixture(t, "watch_movie_27205.json"), nil)

	got, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, "US")
	if err != nil {
		t.Fatalf("WhereToWatch: %v", err)
	}
	if want := []string{"Tubi TV", "Pluto TV"}; !reflect.DeepEqual(providerNames(got.Ads), want) {
		t.Errorf("Ads = %v, want %v", providerNames(got.Ads), want)
	}
	if want := []string{"Amazon Video", "Apple TV"}; !reflect.DeepEqual(providerNames(got.Rent), want) {
		t.Errorf("Rent = %v, want %v", providerNames(got.Rent), want)
	}
}

func TestWhereToWatchMovieDE(t *testing.T) {
	c := newWatchClient(t, readWatchFixture(t, "watch_movie_27205.json"), nil)

	got, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, "DE")
	if err != nil {
		t.Fatalf("WhereToWatch: %v", err)
	}
	if got.Link == "" {
		t.Error("Link is empty, want non-empty")
	}
	if len(got.Stream) != 0 || len(got.Free) != 0 || len(got.Ads) != 0 || len(got.Rent) != 0 {
		t.Errorf("groups = stream %v, free %v, ads %v, rent %v, want all empty",
			got.Stream, got.Free, got.Ads, got.Rent)
	}
	if len(got.Buy) != 1 {
		t.Errorf("Buy = %v, want exactly one provider", got.Buy)
	}
}

func TestWhereToWatchMovieGB(t *testing.T) {
	c := newWatchClient(t, readWatchFixture(t, "watch_movie_27205.json"), nil)

	got, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, "GB")
	if err != ErrNoProviders {
		t.Errorf("err = %v, want ErrNoProviders", err)
	}
	if got != nil {
		t.Errorf("got = %#v, want nil", got)
	}
}

func TestWhereToWatchTVUS(t *testing.T) {
	var path string
	c := newWatchClient(t, readWatchFixture(t, "watch_tv_1399.json"), &path)

	got, err := c.WhereToWatch(context.Background(), ID{TV, 1399}, "US")
	if err != nil {
		t.Fatalf("WhereToWatch: %v", err)
	}
	if path != "/tv/1399/watch/providers" {
		t.Errorf("path = %q, want %q", path, "/tv/1399/watch/providers")
	}
	if want := []string{"Max", "Amazon Prime Video"}; !reflect.DeepEqual(providerNames(got.Stream), want) {
		t.Errorf("Stream = %v, want %v", providerNames(got.Stream), want)
	}
	if want := []string{"Tubi TV"}; !reflect.DeepEqual(providerNames(got.Free), want) {
		t.Errorf("Free = %v, want %v", providerNames(got.Free), want)
	}
	if want := []string{"Amazon Video", "Apple TV"}; !reflect.DeepEqual(providerNames(got.Buy), want) {
		t.Errorf("Buy = %v, want %v", providerNames(got.Buy), want)
	}
}

func TestWhereToWatchInvalidRegion(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.Write([]byte(`{"results":{}}`))
	}))
	t.Cleanup(srv.Close)
	c := New("tok", WithBaseURL(srv.URL))

	for _, region := range []string{"X", "USA", "1N", ""} {
		got, err := c.WhereToWatch(context.Background(), ID{Movie, 27205}, region)
		if err == nil || !strings.Contains(err.Error(), "invalid region") {
			t.Errorf("WhereToWatch(%q) err = %v, want it to contain %q", region, err, "invalid region")
		}
		if got != nil {
			t.Errorf("WhereToWatch(%q) = %#v, want nil", region, got)
		}
	}
	if called {
		t.Error("server was called for an invalid region")
	}
}
