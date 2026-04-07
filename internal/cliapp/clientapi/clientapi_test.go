package clientapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDoJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/data" {
			t.Fatalf("expected path /data, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET request, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"value":"ok"}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	var resp struct {
		Value string `json:"value"`
	}
	if err := c.doJSON(http.MethodGet, "/data", "", nil, &resp); err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
	if resp.Value != "ok" {
		t.Fatalf("expected value to be 'ok', got %s", resp.Value)
	}
}

func TestDoJSON_JSONError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"error":"bad request"}`)
	}))
	defer srv.Close()

	c := New(srv.URL)
	err := c.doJSON(http.MethodGet, "/oops", "", nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "server GET /oops: bad request" {
		t.Fatalf("unexpected error message: %s", err.Error())
	}
}

func TestDoJSON_PlainError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "boom")
	}))
	defer srv.Close()

	c := New(srv.URL)
	err := c.doJSON(http.MethodGet, "/err", "", nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
	if err.Error() != "server GET /err: boom" {
		t.Fatalf("unexpected error: %s", err.Error())
	}
}

func TestDoJSON_AuthorizationHeader(t *testing.T) {
	wantToken := "abc123"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+wantToken {
			t.Fatalf("Authorization header not set, got %s", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	c := New(srv.URL)
	if err := c.doJSON(http.MethodGet, "/auth", wantToken, nil, nil); err != nil {
		t.Fatalf("doJSON returned error: %v", err)
	}
}
