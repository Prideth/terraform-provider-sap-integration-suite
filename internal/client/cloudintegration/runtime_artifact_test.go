package cloudintegration

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

func TestClient_GetRuntimeArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "metering", "Version": "1.0.0", "Name": "Metering", "Type": "INTEGRATION_FLOW", "Status": "STARTED", "DeployedOn": "/Date(1790424405369)/", "ErrorInformation": {"__deferred": {"uri": "x"}}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	artifact, err := client.GetRuntimeArtifact(context.Background(), "metering")
	if err != nil {
		t.Fatalf("GetRuntimeArtifact() error: %v", err)
	}
	if artifact.Status != StatusStarted {
		t.Errorf("Status = %q, want %q", artifact.Status, StatusStarted)
	}
}

// TestClient_GetRuntimeArtifact_NotFoundIsError covers what a
// *_deployment resource's Read sees after an external undeploy: something
// outside Terraform (SAP Cloud Integration UI, another automation) removed
// the runtime deployment, and GetRuntimeArtifact must surface that as a
// plain 404 rather than any special-cased error, so the resource's Read can
// treat it exactly like "already gone" and drop it from state.
func TestClient_GetRuntimeArtifact_NotFoundIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "not deployed"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	_, err := client.GetRuntimeArtifact(context.Background(), "company-codes")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
		t.Fatalf("expected a not-found *apierror.Error, got %v", err)
	}
}

func TestClient_UndeployRuntimeArtifact(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/IntegrationRuntimeArtifacts('metering')"
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

	if err := client.UndeployRuntimeArtifact(context.Background(), "metering"); err != nil {
		t.Fatalf("UndeployRuntimeArtifact() error: %v", err)
	}
}

func TestClient_UndeployRuntimeArtifact_NotFoundIsError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "gone"}}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	if err := client.UndeployRuntimeArtifact(context.Background(), "metering"); err == nil {
		t.Fatal("expected an error for a 404 response")
	}
}

func TestClient_GetRuntimeArtifactErrorInformation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/IntegrationRuntimeArtifacts('metering')/ErrorInformation/$value"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("  Mapping step failed\n"))
	}))
	defer server.Close()

	info, err := New(http.DefaultClient, server.URL).GetRuntimeArtifactErrorInformation(context.Background(), "metering")
	if err != nil {
		t.Fatalf("GetRuntimeArtifactErrorInformation() error: %v", err)
	}
	if info != "Mapping step failed" {
		t.Errorf("info = %q, want the trimmed text", info)
	}
}

func TestDeployTaskID(t *testing.T) {
	cases := map[string]string{
		"2f93475d-9c44-4175-6b13-3f0b2c26e246":                     "2f93475d-9c44-4175-6b13-3f0b2c26e246", // tenant answer, September 2026
		"\"2f93475d-9c44-4175-6b13-3f0b2c26e246\"\n":               "2f93475d-9c44-4175-6b13-3f0b2c26e246",
		`{"d": {"DeployIntegrationDesigntimeArtifact": "task-9"}}`: "task-9",
		"":                                   "",
		"<html><body>Accepted</body></html>": "",
	}
	for body, want := range cases {
		if got := deployTaskID([]byte(body)); got != want {
			t.Errorf("deployTaskID(%q) = %q, want %q", body, got, want)
		}
	}
}

func TestClient_GetBuildAndDeployStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/api/v1/BuildAndDeployStatus('task-1')"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		// Shape a tenant returned, September 2026.
		_, _ = w.Write([]byte(`{"d": {"__metadata": {"type": "com.sap.hci.api.BuildAndDeployStatus"}, "TaskId": "task-1", "Status": "SUCCESS"}}`))
	}))
	defer server.Close()

	status, err := New(http.DefaultClient, server.URL).GetBuildAndDeployStatus(context.Background(), "task-1")
	if err != nil || status.Status != "SUCCESS" || status.TaskID != "task-1" {
		t.Fatalf("status %+v, err %v", status, err)
	}
}
