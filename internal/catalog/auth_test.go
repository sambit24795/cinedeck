package catalog

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestCheckTokenOK(t *testing.T) {
	body, err := os.ReadFile("../../testdata/authentication_ok.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	c := New("tok", WithBaseURL(srv.URL))
	if err := c.CheckToken(context.Background()); err != nil {
		t.Fatalf("CheckToken: %v", err)
	}
	if gotPath != "/authentication" {
		t.Errorf("path = %q, want %q", gotPath, "/authentication")
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer tok")
	}
}

func TestCheckTokenInvalid(t *testing.T) {
	body, err := os.ReadFile("../../testdata/error_401.json")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(body)
	}))
	t.Cleanup(srv.Close)

	c := New("tok", WithBaseURL(srv.URL))
	err = c.CheckToken(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *APIError", err)
	}
	if apiErr.Status != 401 {
		t.Errorf("Status = %d, want 401", apiErr.Status)
	}
	if !strings.Contains(apiErr.Message, "Invalid API key") {
		t.Errorf("Message = %q, want it to contain %q", apiErr.Message, "Invalid API key")
	}
}

func TestCheckTokenNotAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"success": false}`))
	}))
	t.Cleanup(srv.Close)

	c := New("tok", WithBaseURL(srv.URL))
	err := c.CheckToken(context.Background())
	if err == nil || !strings.Contains(err.Error(), "token not accepted") {
		t.Fatalf("err = %v, want it to contain %q", err, "token not accepted")
	}
}

func TestCheckTokenServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	c := New("tok", WithBaseURL(srv.URL))
	err := c.CheckToken(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want an *APIError", err)
	}
	if apiErr.Status != 500 {
		t.Errorf("Status = %d, want 500", apiErr.Status)
	}
}

func TestCheckTokenTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	c := New("tok", WithBaseURL(srv.URL))
	err := c.CheckToken(context.Background())
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		t.Fatalf("err = %v, want not an *APIError", err)
	}
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Fatalf("err = %v, want it to contain %q", err, "request failed")
	}
}
