package securitycontent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// syntheticTestSecret is an unmistakably synthetic secret used only against
// an in-process httptest server; it is never a real credential.
const syntheticTestSecret = "tf-acc-synthetic-client-secret-not-real" // #nosec G101 -- synthetic test fixture, not a real credential

func TestClient_CreateOAuth2ClientCredential(t *testing.T) {
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/OAuth2ClientCredentials" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("reading POST body: %v", err)
		}
		gotBody = body
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Name": "BACKEND_OAUTH", "TokenServiceUrl": "https://auth.example.com/oauth/token", "ClientId": "integration-client", "Scope": "read write"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	created, err := client.CreateOAuth2ClientCredential(context.Background(), OAuth2ClientCredential{
		Name:            "BACKEND_OAUTH",
		TokenServiceURL: "https://auth.example.com/oauth/token",
		ClientID:        "integration-client",
		Scope:           "read write",
	}, syntheticTestSecret)
	if err != nil {
		t.Fatalf("CreateOAuth2ClientCredential() error: %v", err)
	}
	if created.Name != "BACKEND_OAUTH" {
		t.Errorf("Name = %q, want BACKEND_OAUTH", created.Name)
	}

	var decoded map[string]any
	if err := json.Unmarshal(gotBody, &decoded); err != nil {
		t.Fatalf("decoding POST body: %v", err)
	}
	if decoded["ClientSecret"] != syntheticTestSecret {
		t.Errorf("POST body ClientSecret = %v, want the synthetic test secret", decoded["ClientSecret"])
	}
}

func TestClient_CreateOAuth2ClientCredential_ResponseNeverExposesSecret(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Name": "BACKEND_OAUTH", "ClientId": "integration-client", "ClientSecret": "` + syntheticTestSecret + `"}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	created, err := client.CreateOAuth2ClientCredential(context.Background(), OAuth2ClientCredential{Name: "BACKEND_OAUTH", ClientID: "integration-client"}, syntheticTestSecret)
	if err != nil {
		t.Fatalf("CreateOAuth2ClientCredential() error: %v", err)
	}

	body, err := json.Marshal(created)
	if err != nil {
		t.Fatalf("marshaling created OAuth2ClientCredential: %v", err)
	}
	if strings.Contains(string(body), syntheticTestSecret) {
		t.Errorf("CreateOAuth2ClientCredential() result serializes a client secret: %s", body)
	}
}

func TestClient_GetUpdateDeleteOAuth2ClientCredential(t *testing.T) {
	var lastMethod string
	var putBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastMethod = r.Method
		want := "/api/v1/OAuth2ClientCredentials('BACKEND_OAUTH')"
		if r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"d": {"Name": "BACKEND_OAUTH", "TokenServiceUrl": "https://auth.example.com/oauth/token", "ClientId": "integration-client", "Scope": "read"}}`))
		case http.MethodPut:
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("reading PUT body: %v", err)
			}
			putBody = body
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	cred, err := client.GetOAuth2ClientCredential(context.Background(), "BACKEND_OAUTH")
	if err != nil {
		t.Fatalf("GetOAuth2ClientCredential() error: %v", err)
	}
	if cred.ClientID != "integration-client" {
		t.Errorf("ClientID = %q, want integration-client", cred.ClientID)
	}

	err = client.UpdateOAuth2ClientCredential(context.Background(), OAuth2ClientCredential{
		Name: "BACKEND_OAUTH", TokenServiceURL: "https://auth.example.com/oauth/token",
		ClientID: "integration-client", Scope: "read write",
	}, syntheticTestSecret)
	if err != nil {
		t.Fatalf("UpdateOAuth2ClientCredential() error: %v", err)
	}
	if lastMethod != http.MethodPut {
		t.Errorf("last method = %q, want PUT", lastMethod)
	}

	var decoded map[string]any
	if err := json.Unmarshal(putBody, &decoded); err != nil {
		t.Fatalf("decoding PUT body: %v", err)
	}
	if decoded["ClientSecret"] != syntheticTestSecret {
		t.Errorf("PUT body ClientSecret = %v, want the synthetic test secret", decoded["ClientSecret"])
	}
	if decoded["Scope"] != "read write" {
		t.Errorf("PUT body Scope = %v, want \"read write\"", decoded["Scope"])
	}

	if err := client.DeleteOAuth2ClientCredential(context.Background(), "BACKEND_OAUTH"); err != nil {
		t.Fatalf("DeleteOAuth2ClientCredential() error: %v", err)
	}
}

// A PUT replaces the whole entity, so the token-request settings must be
// resent on every update or SAP would reset values maintained in the UI.
func TestClient_UpdateOAuth2ClientCredential_ResendsTokenRequestSettings(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		raw, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Fatalf("decoding PUT body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := New(http.DefaultClient, server.URL).UpdateOAuth2ClientCredential(context.Background(), OAuth2ClientCredential{
		Name:                 "tf-acc-oauth",
		TokenServiceURL:      "https://auth.example.invalid/oauth/token",
		ClientID:             "client",
		ClientAuthentication: "header-constant",
		ScopeContentType:     "application/json",
		Audience:             "https://api.example.invalid",
	}, "synthetic-secret")
	if err != nil {
		t.Fatalf("UpdateOAuth2ClientCredential() error: %v", err)
	}

	want := map[string]string{
		"ClientAuthentication": "header-constant",
		"ScopeContentType":     "application/json",
		"Audience":             "https://api.example.invalid",
	}
	for k, v := range want {
		if body[k] != v {
			t.Errorf("PUT body %s = %v, want %q", k, body[k], v)
		}
	}
	if _, ok := body["Resource"]; ok {
		t.Errorf("an empty Resource must be omitted, got %v", body["Resource"])
	}
}

// The count reads the expanded navigation; SAP deletes custom parameters
// with every PUT, so the resource warns when there are any.
func TestClient_CountOAuth2ClientCredentialCustomParameters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/OAuth2ClientCredentials('BACKEND_OAUTH')" || r.URL.Query().Get("$expand") != "CustomParameters" {
			t.Errorf("request = %s?%s, want the credential with $expand=CustomParameters", r.URL.Path, r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`{"d": {"Name": "BACKEND_OAUTH", "CustomParameters": {"results": [
			{"Key": "resource", "Value": "x", "SendAsPartOf": "body"},
			{"Key": "tenant", "Value": "y", "SendAsPartOf": "header"}]}}}`))
	}))
	defer server.Close()

	n, err := New(http.DefaultClient, server.URL).CountOAuth2ClientCredentialCustomParameters(context.Background(), "BACKEND_OAUTH")
	if err != nil {
		t.Fatalf("CountOAuth2ClientCredentialCustomParameters() error: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}
