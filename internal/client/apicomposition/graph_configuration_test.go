package apicomposition

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

func TestClient_CreateGraphConfiguration(t *testing.T) {
	var postBody []byte
	getCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/configuration/v1/sap.graph/GraphConfiguration":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("reading POST body: %v", err)
			}
			postBody = body
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/configuration/v1/sap.graph/GraphConfiguration/my-bdg":
			getCount++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "DEPLOYMENT_INITIATED"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	created, err := client.CreateGraphConfiguration(context.Background(), GraphConfigurationInput{
		BusinessDataGraphIdentifier: "my-bdg",
		DataSources: []DataSource{
			{Name: "s4", Services: []DataSourceService{{DestinationName: "s4-business-partner"}}},
		},
		LocatingPolicy: LocatingPolicy{
			Rules: []LocatingRule{{Name: "sap.s4.*", Leading: "s4"}},
		},
	})
	if err != nil {
		t.Fatalf("CreateGraphConfiguration() error: %v", err)
	}
	if created.Status != StatusDeploymentInitiated {
		t.Errorf("Status = %q, want %q", created.Status, StatusDeploymentInitiated)
	}
	if getCount == 0 {
		t.Error("expected CreateGraphConfiguration to poll GET at least once to leave PROCESSING")
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(postBody, &decoded); err != nil {
		t.Fatalf("decoding POST body: %v", err)
	}
	if decoded["businessDataGraphIdentifier"] != "my-bdg" {
		t.Errorf("businessDataGraphIdentifier = %v, want my-bdg", decoded["businessDataGraphIdentifier"])
	}
}

func TestClient_CreateGraphConfiguration_Failed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "FAILED", "statusDetails": "destination not reachable"}`))
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	cfg, err := client.CreateGraphConfiguration(context.Background(), GraphConfigurationInput{BusinessDataGraphIdentifier: "my-bdg"})
	var failed *ProcessingFailedError
	if !errors.As(err, &failed) {
		t.Fatalf("error = %v, want a *ProcessingFailedError", err)
	}
	if cfg == nil || cfg.Status != StatusFailed {
		t.Fatalf("expected the failed graph to be returned alongside the error, got %+v", cfg)
	}
	if !strings.Contains(err.Error(), "destination not reachable") {
		t.Errorf("error %q does not carry statusDetails", err)
	}
}

// The locating policy shapes follow SAP's configuration file page: cues are
// objects, rules reference cues by name, and key mappings carry a format
// strategy.
func TestGraphConfiguration_DecodesDocumentedLocatingPolicy(t *testing.T) {
	body := `{
		"businessDataGraphIdentifier": "my-bdg",
		"dataSources": [{"name": "myS4", "services": [{"destinationName": "sandbox-s4", "path": "/odata/sap/API_BUSINESS_PARTNER"}]}],
		"locatingPolicy": {
			"cues": [{"name": "emea", "description": "European subsidiaries"}],
			"keyMapping": [{
				"foreignKey": {"dataSource": "myC4C", "entityName": "sap.c4c.ProductCollection", "attributes": ["ExternalID"],
					"strategy": {"name": "format", "match": "(?P<id>[0-9]+)", "replace": "$id"}},
				"references": {"dataSource": "myS4", "entityName": "sap.s4.A_Product", "attributes": ["Product"]}
			}],
			"rules": [
				{"name": "sap.graph.Product", "leading": "myS4", "local": ["myC4C"], "cues": ["emea"]},
				{"name": "bestrun.Product", "leading": "custom", "sourceEntity": "company.custom.CustomProduct"}
			]
		}
	}`
	var cfg GraphConfiguration
	if err := json.Unmarshal([]byte(body), &cfg); err != nil {
		t.Fatalf("decoding: %v", err)
	}
	p := cfg.LocatingPolicy
	if len(p.Cues) != 1 || p.Cues[0].Description != "European subsidiaries" {
		t.Errorf("cues = %+v", p.Cues)
	}
	if len(p.KeyMapping) != 1 || p.KeyMapping[0].ForeignKey.Strategy == nil || p.KeyMapping[0].ForeignKey.Strategy.Replace != "$id" {
		t.Errorf("keyMapping = %+v", p.KeyMapping)
	}
	if p.KeyMapping[0].References.Strategy != nil {
		t.Error("references.strategy should stay nil when SAP omits it")
	}
	if len(p.Rules) != 2 || p.Rules[0].Cues[0] != "emea" || p.Rules[1].SourceEntity != "company.custom.CustomProduct" {
		t.Errorf("rules = %+v", p.Rules)
	}
}

func TestClient_CreateGraphConfiguration_TimesOut(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.CreateGraphConfiguration(ctx, GraphConfigurationInput{BusinessDataGraphIdentifier: "my-bdg"})
	if err == nil {
		t.Fatal("expected an error when processing never leaves PROCESSING before the context deadline")
	}
}

func TestClient_GetGraphConfiguration_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "404", "message": "not found"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	if _, err := client.GetGraphConfiguration(context.Background(), "does-not-exist"); err == nil {
		t.Fatal("expected an error for a missing business data graph")
	}
}

func TestClient_UpdateGraphConfiguration(t *testing.T) {
	var patchBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/configuration/v1/sap.graph/GraphConfiguration/my-bdg":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("reading PATCH body: %v", err)
			}
			patchBody = body
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "PROCESSING"}`))
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"businessDataGraphIdentifier": "my-bdg", "status": "DEPLOYMENT_INITIATED"}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	updated, err := client.UpdateGraphConfiguration(context.Background(), "my-bdg", GraphConfigurationInput{
		BusinessDataGraphIdentifier: "my-bdg",
	})
	if err != nil {
		t.Fatalf("UpdateGraphConfiguration() error: %v", err)
	}
	if updated.Status != StatusDeploymentInitiated {
		t.Errorf("Status = %q, want %q", updated.Status, StatusDeploymentInitiated)
	}

	var decoded map[string]json.RawMessage
	if err := json.Unmarshal(patchBody, &decoded); err != nil {
		t.Fatalf("decoding PATCH body: %v", err)
	}
	// An emptied exclude list must reach SAP as [] so the PATCH clears it.
	if got := string(decoded["exclude"]); got != "[]" {
		t.Errorf("exclude = %s, want []", got)
	}
	// Read-only and unmanaged properties are never sent.
	for _, key := range []string{"status", "statusDetails", "logMessages", "effectiveGraphModelVersion", "extensions"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("PATCH body contains %q", key)
		}
	}
}

func TestClient_DeleteGraphConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		want := "/configuration/v1/sap.graph/GraphConfiguration/my-bdg"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	if err := client.DeleteGraphConfiguration(context.Background(), "my-bdg"); err != nil {
		t.Fatalf("DeleteGraphConfiguration() error: %v", err)
	}
}

// The 403 body is the one a tenant returned on 2026-10-01, with a made-up
// trace ID.
func TestClient_GetGraphConfiguration_ForbiddenKeepsCodeAndTraceID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"code":"2707","message":"You don't have permission to access this resource. Please check your assigned roles in your SAP BTP subaccount.","@Graph.traceId":"00000000-0000-0000-0000-000000000001","@Common.numericSeverity":4}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	_, err := client.GetGraphConfiguration(context.Background(), "my-bdg")

	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("error = %v, want *apierror.Error", err)
	}
	if apiErr.StatusCode != http.StatusForbidden || apiErr.Code != "2707" {
		t.Errorf("status, code = %d, %q, want 403, \"2707\"", apiErr.StatusCode, apiErr.Code)
	}
	if !strings.HasPrefix(apiErr.Message, "You don't have permission") {
		t.Errorf("Message = %q", apiErr.Message)
	}
	if apiErr.RequestID != "00000000-0000-0000-0000-000000000001" {
		t.Errorf("RequestID = %q, want the @Graph.traceId", apiErr.RequestID)
	}
}
