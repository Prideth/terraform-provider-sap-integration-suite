package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync/atomic"
	"testing"
)

func TestConfig_HTTPClient_AcquiresAndCachesToken(t *testing.T) {
	var tokenRequests int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tokenRequests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "bearer", "expires_in": 3600}`))
	}))
	defer server.Close()

	cfg := Config{
		TokenURL:     server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	client, _, err := cfg.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("HTTPClient() error: %v", err)
	}

	var apiCalls int32
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&apiCalls, 1)
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer test-token")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()

	for i := 0; i < 3; i++ {
		resp, err := client.Get(api.URL)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		_ = resp.Body.Close()
	}

	if got := atomic.LoadInt32(&tokenRequests); got != 1 {
		t.Errorf("expected the token to be fetched once and cached, got %d token requests", got)
	}
	if got := atomic.LoadInt32(&apiCalls); got != 3 {
		t.Errorf("expected 3 API calls, got %d", got)
	}
}

func TestConfig_HTTPClient_RejectsIncompleteConfig(t *testing.T) {
	cases := []Config{
		{ClientID: "id", ClientSecret: "secret"},
		{TokenURL: "https://example.com/token", ClientSecret: "secret"},
		{TokenURL: "https://example.com/token", ClientID: "id"},
	}

	for _, cfg := range cases {
		if _, _, err := cfg.HTTPClient(context.Background(), http.DefaultClient); err == nil {
			t.Errorf("expected an error for incomplete config %+v", cfg)
		}
	}
}

func TestConfig_HTTPClient_InvalidateForcesFreshToken(t *testing.T) {
	var tokenRequests int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&tokenRequests, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "token-` + strconv.Itoa(int(n)) + `", "token_type": "bearer", "expires_in": 3600}`))
	}))
	defer server.Close()

	cfg := Config{
		TokenURL:     server.URL,
		ClientID:     "client-id",
		ClientSecret: "client-secret",
	}

	client, invalidate, err := cfg.HTTPClient(context.Background(), http.DefaultClient)
	if err != nil {
		t.Fatalf("HTTPClient() error: %v", err)
	}

	var gotAuth []string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = append(gotAuth, r.Header.Get("Authorization"))
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()

	for i := 0; i < 2; i++ {
		resp, err := client.Get(api.URL)
		if err != nil {
			t.Fatalf("Get() error: %v", err)
		}
		_ = resp.Body.Close()
	}
	if got := atomic.LoadInt32(&tokenRequests); got != 1 {
		t.Fatalf("expected 1 token request before Invalidate, got %d", got)
	}

	invalidate()

	resp, err := client.Get(api.URL)
	if err != nil {
		t.Fatalf("Get() error: %v", err)
	}
	_ = resp.Body.Close()

	if got := atomic.LoadInt32(&tokenRequests); got != 2 {
		t.Fatalf("expected a fresh token request after Invalidate, got %d total", got)
	}
	if gotAuth[0] != gotAuth[1] {
		t.Errorf("expected the first two requests to reuse the cached token, got %q then %q", gotAuth[0], gotAuth[1])
	}
	if gotAuth[2] == gotAuth[1] {
		t.Errorf("expected Invalidate to force a different token, got the same one: %q", gotAuth[2])
	}
}

// Terraform cancels the ConfigureProvider context as soon as the call
// returns; the provider builds this client there and fetches the first token
// much later, in a resource call. The first acceptance run failed every
// request with "context canceled" on the token URL.
func TestConfig_HTTPClient_OutlivesConfigureContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "bearer", "expires_in": 3600}`))
	}))
	defer server.Close()
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer api.Close()

	configureCtx, cancel := context.WithCancel(context.Background())
	client, _, err := Config{TokenURL: server.URL, ClientID: "id", ClientSecret: "secret"}.HTTPClient(configureCtx, http.DefaultClient)
	if err != nil {
		t.Fatalf("HTTPClient() error: %v", err)
	}
	cancel()

	resp, err := client.Get(api.URL)
	if err != nil {
		t.Fatalf("request after the configure context ended: %v", err)
	}
	_ = resp.Body.Close()
}
