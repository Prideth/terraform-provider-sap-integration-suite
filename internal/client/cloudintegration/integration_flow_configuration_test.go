package cloudintegration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClient_ListIntegrationFlowConfigurations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := "/api/v1/IntegrationDesigntimeArtifacts(Id='Order_Flow',Version='1.0.3')/Configurations"
		if r.Method != http.MethodGet || r.URL.Path != want {
			t.Errorf("got %s %s, want GET %s", r.Method, r.URL.Path, want)
		}
		_, _ = w.Write([]byte(`{"d": {"results": [
			{"ParameterKey": "receiver_host", "ParameterValue": "erp.example.invalid", "DataType": "xsd:string"},
			{"ParameterKey": "batch_size", "ParameterValue": "100", "DataType": "xsd:integer", "Description": "Records per call"}
		]}}`))
	}))
	defer server.Close()

	got, err := New(http.DefaultClient, server.URL).ListIntegrationFlowConfigurations(context.Background(), "Order_Flow", "1.0.3")
	if err != nil {
		t.Fatalf("ListIntegrationFlowConfigurations() error: %v", err)
	}
	if len(got) != 2 || got[0].ParameterKey != "batch_size" || got[0].DataType != "xsd:integer" {
		t.Errorf("configurations = %+v, want sorted by key with data types", got)
	}
}

func TestClient_UpdateIntegrationFlowConfiguration_DocumentedRequest(t *testing.T) {
	var gotMethod, gotPath string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decoding body %s: %v", raw, err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := New(http.DefaultClient, server.URL).UpdateIntegrationFlowConfiguration(
		context.Background(), "Order_Flow", "1.0.3", "receiver_host", "erp-prod.example.invalid", "xsd:string")
	if err != nil {
		t.Fatalf("UpdateIntegrationFlowConfiguration() error: %v", err)
	}

	wantPath := "/api/v1/IntegrationDesigntimeArtifacts(Id='Order_Flow',Version='1.0.3')/$links/Configurations('receiver_host')"
	if gotMethod != http.MethodPut || gotPath != wantPath {
		t.Errorf("got %s %s, want PUT %s", gotMethod, gotPath, wantPath)
	}
	if len(body) != 2 || body["ParameterValue"] != "erp-prod.example.invalid" || body["DataType"] != "xsd:string" {
		t.Errorf("body = %v, want exactly ParameterValue and DataType", body)
	}
}

func TestClient_UpdateIntegrationFlowConfiguration_EscapesKey(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := New(http.DefaultClient, server.URL).UpdateIntegrationFlowConfiguration(
		context.Background(), "Order_Flow", "1.0.3", "it's", "v", ""); err != nil {
		t.Fatalf("UpdateIntegrationFlowConfiguration() error: %v", err)
	}
	if want := "/api/v1/IntegrationDesigntimeArtifacts(Id='Order_Flow',Version='1.0.3')/$links/Configurations('it''s')"; gotPath != want {
		t.Errorf("path = %q, want quote doubled: %q", gotPath, want)
	}
}

func TestClient_SaveIntegrationFlowAsVersion_DocumentedRequest(t *testing.T) {
	var gotMethod, gotURI string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotURI = r.Method, r.URL.RequestURI()
		_, _ = w.Write([]byte(`{"d": {"Id": "Order_Flow", "Version": "1.0.3", "Name": "Order Flow"}}`))
	}))
	defer server.Close()

	flow, err := New(http.DefaultClient, server.URL).SaveIntegrationFlowAsVersion(context.Background(), "Order_Flow", "1.0.3")
	if err != nil {
		t.Fatalf("SaveIntegrationFlowAsVersion() error: %v", err)
	}
	if want := "/api/v1/IntegrationDesigntimeArtifactSaveAsVersion?Id='Order_Flow'&SaveAsVersion='1.0.3'"; gotMethod != http.MethodPost || gotURI != want {
		t.Errorf("got %s %s, want POST %s", gotMethod, gotURI, want)
	}
	if flow.Version != "1.0.3" {
		t.Errorf("Version = %q, want 1.0.3", flow.Version)
	}
}

// A tenant answered SaveAsVersion with 200 and no body (September 2026);
// the saved version is then read from the active artifact.
func TestClient_SaveIntegrationFlowAsVersion_EmptyBodyReadsBack(t *testing.T) {
	var requests []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests = append(requests, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(`{"d": {"Id": "Order_Flow", "Version": "1.0.5", "Name": "Order Flow", "PackageId": "P"}}`))
	}))
	defer server.Close()

	flow, err := New(http.DefaultClient, server.URL).SaveIntegrationFlowAsVersion(context.Background(), "Order_Flow", "1.0.5")
	if err != nil {
		t.Fatalf("SaveIntegrationFlowAsVersion() error: %v", err)
	}
	if flow.Version != "1.0.5" {
		t.Errorf("Version = %q, want 1.0.5 from the read-back", flow.Version)
	}
	want := "GET /api/v1/IntegrationDesigntimeArtifacts(Id='Order_Flow',Version='active')"
	if len(requests) != 2 || requests[1] != want {
		t.Errorf("requests = %v, want the POST followed by %q", requests, want)
	}
}
