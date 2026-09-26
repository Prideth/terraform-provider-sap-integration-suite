package apidiscovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

func fakeService(t *testing.T, metadataStatus int) (*httptest.Server, Service) {
	t.Helper()
	metadata, err := os.ReadFile(filepath.Join("..", "apimeta", "testdata", "v2.xml"))
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"access_token":"x","token_type":"bearer","expires_in":3600}`))
	})
	mux.HandleFunc("/api/v1/$metadata", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer x" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(metadataStatus)
		_, _ = w.Write(metadata)
	})
	mux.HandleFunc("/api/v1/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"d":{"EntitySets":["Policies","References","Resources","Extra"]}}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	env := map[string]string{
		"SAP_INTEGRATION_SUITE_HOST":          srv.URL,
		"SAP_INTEGRATION_SUITE_TOKEN_URL":     srv.URL + "/oauth/token",
		"SAP_INTEGRATION_SUITE_CLIENT_ID":     "id",
		"SAP_INTEGRATION_SUITE_CLIENT_SECRET": "secret",
	}
	svc := Service{ID: "fake", Protocol: apimeta.ProtocolODataV2, Resolve: func(func(string) string) (string, Credentials, []string) {
		return mainService("/api/v1")(func(k string) string { return env[k] })
	}}
	return srv, svc
}

func TestFetchReadsMetadataAndServiceDocument(t *testing.T) {
	_, svc := fakeService(t, http.StatusOK)
	res, err := Fetch(context.Background(), svc)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Service.EntitySets) == 0 {
		t.Fatal("no entity sets parsed")
	}
	joined := strings.Join(res.ServiceDocumentIssues, "\n")
	if !strings.Contains(joined, "service document lists Extra") {
		t.Errorf("expected the undeclared collection to be reported, got:\n%s", joined)
	}
	if res.Service.Source != "live /$metadata" {
		t.Errorf("source %q", res.Service.Source)
	}
}

func TestFetchErrorsDoNotContainTheServiceRoot(t *testing.T) {
	srv, svc := fakeService(t, http.StatusForbidden)
	_, err := Fetch(context.Background(), svc)
	if err == nil {
		t.Fatal("expected an error")
	}
	host := strings.TrimPrefix(srv.URL, "http://")
	if strings.Contains(err.Error(), host) {
		t.Errorf("error leaks the host: %v", err)
	}
	if !strings.Contains(err.Error(), "HTTP 403") {
		t.Errorf("error should carry the status: %v", err)
	}
}

func TestFetchUnconfigured(t *testing.T) {
	svc := Service{ID: "fake", Resolve: func(func(string) string) (string, Credentials, []string) {
		return "", Credentials{}, []string{"SAP_INTEGRATION_SUITE_HOST"}
	}}
	_, err := Fetch(context.Background(), svc)
	if err == nil || !strings.Contains(err.Error(), "SAP_INTEGRATION_SUITE_HOST") {
		t.Fatalf("want a missing-variable error, got %v", err)
	}
}
