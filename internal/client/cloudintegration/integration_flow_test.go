package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_CreateIntegrationFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/IntegrationDesigntimeArtifacts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		var sent IntegrationFlow
		if err := json.NewDecoder(r.Body).Decode(&sent); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if sent.Content != base64.StdEncoding.EncodeToString([]byte("zip-bytes")) {
			t.Errorf("Content was not base64-encoded correctly: %q", sent.Content)
		}

		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "metering", "Name": "Metering", "PackageId": "UTILITIES", "Version": "1.0.0"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	flow, err := client.CreateIntegrationFlow(context.Background(), "UTILITIES", "metering", "Metering", []byte("zip-bytes"))
	if err != nil {
		t.Fatalf("CreateIntegrationFlow() error: %v", err)
	}
	if flow.Version != "1.0.0" {
		t.Errorf("Version = %q, want 1.0.0", flow.Version)
	}
}

func TestClient_GetIntegrationFlow_UsesActiveVersionKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/IntegrationDesigntimeArtifacts(Id='metering',Version='active')"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "metering", "Name": "Metering", "PackageId": "UTILITIES", "Version": "1.0.0"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if _, err := client.GetIntegrationFlow(context.Background(), "UTILITIES", "metering"); err != nil {
		t.Fatalf("GetIntegrationFlow() error: %v", err)
	}
}

func TestClient_DeployIntegrationFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		want := "/api/v1/DeployIntegrationDesigntimeArtifact?Id='metering'&Version='active'"
		if r.URL.String() != want {
			t.Errorf("url = %q, want %q", r.URL.String(), want)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if _, err := client.DeployIntegrationFlow(context.Background(), "metering", "active"); err != nil {
		t.Fatalf("DeployIntegrationFlow() error: %v", err)
	}
}

func TestClient_DeployIntegrationFlow_SpecificVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/DeployIntegrationDesigntimeArtifact?Id='metering'&Version='1.0.1'"
		if r.URL.String() != want {
			t.Errorf("url = %q, want %q", r.URL.String(), want)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if _, err := client.DeployIntegrationFlow(context.Background(), "metering", "1.0.1"); err != nil {
		t.Fatalf("DeployIntegrationFlow() error: %v", err)
	}
}

func TestClient_UpdateIntegrationFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		want := "/api/v1/IntegrationDesigntimeArtifacts(Id='metering',Version='active')"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "metering", "Name": "Metering v2", "PackageId": "UTILITIES", "Version": "1.0.1"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	flow, err := client.UpdateIntegrationFlow(context.Background(), "metering", "Metering v2", []byte("new-content"))
	if err != nil {
		t.Fatalf("UpdateIntegrationFlow() error: %v", err)
	}
	if flow.Version != "1.0.1" {
		t.Errorf("Version = %q, want 1.0.1", flow.Version)
	}
}

func TestClient_DeleteIntegrationFlow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/IntegrationDesigntimeArtifacts(Id='metering',Version='active')"
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

	if err := client.DeleteIntegrationFlow(context.Background(), "metering"); err != nil {
		t.Fatalf("DeleteIntegrationFlow() error: %v", err)
	}
}
