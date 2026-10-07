package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

func TestClient_CreateValueMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/ValueMappingDesigntimeArtifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var sent ValueMapping
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if sent.Content != base64.StdEncoding.EncodeToString([]byte("mapping-bytes")) {
			t.Errorf("Content was not base64-encoded correctly: %q", sent.Content)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Name": "Company Codes", "PackageId": "UTILITIES", "Version": "1.0.0"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	mapping, err := client.CreateValueMapping(context.Background(), "UTILITIES", "company-codes", "Company Codes", []byte("mapping-bytes"))
	if err != nil {
		t.Fatalf("CreateValueMapping() error: %v", err)
	}
	if mapping.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", mapping.Version)
	}
}

func TestClient_GetValueMapping_UsesActiveVersionKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/ValueMappingDesigntimeArtifacts(Id='company-codes',Version='active')"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Name": "Company Codes", "PackageId": "UTILITIES", "Version": "1.0.0"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if _, err := client.GetValueMapping(context.Background(), "UTILITIES", "company-codes"); err != nil {
		t.Fatalf("GetValueMapping() error: %v", err)
	}
}

func TestClient_GetValueMapping_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "no such value mapping"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	_, err := client.GetValueMapping(context.Background(), "UTILITIES", "missing")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected a not-found *apierror.Error, got %v", err)
	}
}

func TestClient_CreateValueMapping_InvalidArtifactIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"code": "BAD_REQUEST", "message": {"value": "a value mapping must contain at least one entry"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	_, err := client.CreateValueMapping(context.Background(), "UTILITIES", "empty-mapping", "Empty", []byte("no-entries"))

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected a 400 *apierror.Error, got %v", err)
	}
}

func TestClient_GetValueMapping_Forbidden(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error": {"code": "FORBIDDEN", "message": {"value": "missing WorkspacePackagesConfigure role"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	_, err := client.GetValueMapping(context.Background(), "UTILITIES", "company-codes")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected a 403 *apierror.Error, got %v", err)
	}
}

func TestClient_GetValueMapping_MalformedResponseIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if _, err := client.GetValueMapping(context.Background(), "UTILITIES", "company-codes"); err == nil {
		t.Fatal("expected an error decoding a malformed OData response, got nil")
	}
}

func TestClient_DeleteValueMapping_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error": {"code": "CONFLICT", "message": {"value": "still referenced by a deployed integration flow"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	err := client.DeleteValueMapping(context.Background(), "company-codes")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusConflict {
		t.Fatalf("expected a 409 *apierror.Error, got %v", err)
	}
}

func TestClient_DeleteValueMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/ValueMappingDesigntimeArtifacts(Id='company-codes',Version='active')"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if err := client.DeleteValueMapping(context.Background(), "company-codes"); err != nil {
		t.Fatalf("DeleteValueMapping() error: %v", err)
	}
}

func TestClient_DeleteValueMapping_NotFoundIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "gone"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	err := client.DeleteValueMapping(context.Background(), "company-codes")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected a not-found *apierror.Error, got %v", err)
	}
}

func TestClient_DeployValueMapping(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		want := "/api/v1/DeployValueMappingDesigntimeArtifact?Id='company-codes'&Version='1.0.1'"
		if r.URL.String() != want {
			t.Errorf("url = %q, want %q", r.URL.String(), want)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if err := client.DeployValueMapping(context.Background(), "company-codes", "1.0.1"); err != nil {
		t.Fatalf("DeployValueMapping() error: %v", err)
	}
}

func TestClient_DeployValueMapping_NotFoundIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "no such value mapping"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	err := client.DeployValueMapping(context.Background(), "missing", "active")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected a not-found *apierror.Error, got %v", err)
	}
}

// A tenant answered ValueMappingDesigntimeArtifactSaveAsVersion with 200 and
// no body (October 2026); the new label is then read from the active
// artifact.
func TestClient_SaveValueMappingAsVersion(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.RequestURI())
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"d": {"Id": "CompanyCodes", "Version": "1.0.5", "Name": "Company Codes", "PackageId": "P"}}`))
	}))
	defer server.Close()

	mapping, err := New(http.DefaultClient, server.URL).SaveValueMappingAsVersion(context.Background(), "CompanyCodes", "1.0.5")
	if err != nil {
		t.Fatalf("SaveValueMappingAsVersion() error: %v", err)
	}
	if mapping.Version != "1.0.5" {
		t.Errorf("Version = %q, want 1.0.5 from the read-back", mapping.Version)
	}
	want := []string{
		"POST /api/v1/ValueMappingDesigntimeArtifactSaveAsVersion?Id='CompanyCodes'&SaveAsVersion='1.0.5'",
		"GET /api/v1/ValueMappingDesigntimeArtifacts(Id='CompanyCodes',Version='active')",
	}
	if len(requests) != len(want) || requests[0] != want[0] || requests[1] != want[1] {
		t.Errorf("requests = %q, want %q", requests, want)
	}
}
