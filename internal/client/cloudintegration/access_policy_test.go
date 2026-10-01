package cloudintegration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// The paths, keys and payload shapes asserted here follow SAP's own access
// policy automation (github.com/SAP/cicd-actions-for-sap-integration-suite).

func decodeBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("reading request body: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("decoding request body %s: %v", raw, err)
	}
	return decoded
}

func TestClient_CreateAccessPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/AccessPolicies" {
			t.Errorf("got %s %s, want POST /api/v1/AccessPolicies", r.Method, r.URL.Path)
		}
		body := decodeBody(t, r)
		if len(body) != 2 || body["RoleName"] != "UTILITIES_ARCHITECT" || body["Description"] != "desc" {
			t.Errorf("create body = %v, want exactly RoleName and Description", body)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "1901", "RoleName": "UTILITIES_ARCHITECT", "Description": "desc"}}`))
	}))
	defer server.Close()

	policy, err := New(http.DefaultClient, server.URL).CreateAccessPolicy(context.Background(),
		AccessPolicy{ID: "ignored", RoleName: "UTILITIES_ARCHITECT", Description: "desc"})
	if err != nil {
		t.Fatalf("CreateAccessPolicy() error: %v", err)
	}
	if policy.ID != "1901" {
		t.Errorf("ID = %q, want 1901", policy.ID)
	}
}

// The client is a faithful wire client: it sends the description exactly as
// given, whatever its length. The 200-character limit SAP applies is checked
// by the Terraform schema, and the client must never shorten a value silently.
func TestClient_CreateAccessPolicy_SendsDescriptionUnchanged(t *testing.T) {
	description := strings.Repeat("0123456789", 100) + " äöü 😀"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := decodeBody(t, r)["Description"]; got != description {
			t.Errorf("Description sent = %q, want the configured value unchanged", got)
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "1901", "RoleName": "R", "Description": "x"}}`))
	}))
	defer server.Close()

	if _, err := New(http.DefaultClient, server.URL).CreateAccessPolicy(context.Background(),
		AccessPolicy{RoleName: "R", Description: description}); err != nil {
		t.Fatalf("CreateAccessPolicy() error: %v", err)
	}
}

func TestClient_GetUpdateDeleteAccessPolicy_UseInt64Key(t *testing.T) {
	var patchBody map[string]any
	var methods []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if want := "/api/v1/AccessPolicies(1901L)"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"d": {"Id": "1901", "RoleName": "UTILITIES_ARCHITECT", "Description": ""}}`))
		case http.MethodPatch:
			patchBody = decodeBody(t, r)
			w.WriteHeader(http.StatusNoContent)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected method %s", r.Method)
		}
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	ctx := context.Background()

	policy, err := client.GetAccessPolicy(ctx, "1901")
	if err != nil {
		t.Fatalf("GetAccessPolicy() error: %v", err)
	}
	if policy.RoleName != "UTILITIES_ARCHITECT" {
		t.Errorf("RoleName = %q", policy.RoleName)
	}

	if err := client.UpdateAccessPolicy(ctx, "1901", AccessPolicy{RoleName: "UTILITIES_ARCHITECT", Description: "new"}); err != nil {
		t.Fatalf("UpdateAccessPolicy() error: %v", err)
	}
	// A tenant answered PUT with 501; PATCH of Description alone succeeded.
	if len(patchBody) != 1 || patchBody["Description"] != "new" {
		t.Errorf("PATCH body = %v, want only Description", patchBody)
	}

	if err := client.DeleteAccessPolicy(ctx, "1901"); err != nil {
		t.Fatalf("DeleteAccessPolicy() error: %v", err)
	}

	want := []string{http.MethodGet, http.MethodPatch, http.MethodDelete}
	if len(methods) != len(want) {
		t.Fatalf("methods = %v, want %v", methods, want)
	}
	for i := range want {
		if methods[i] != want[i] {
			t.Errorf("methods = %v, want %v", methods, want)
			break
		}
	}
}

func TestClient_AccessPolicy_RejectsNonNumericID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected for an invalid ID, got %s %s", r.Method, r.URL.String())
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	ctx := context.Background()

	for _, id := range []string{"", "abc", "1'); DROP", "1/ArtifactReferences"} {
		if _, err := client.GetAccessPolicy(ctx, id); err == nil {
			t.Errorf("GetAccessPolicy(%q) returned no error", id)
		}
		if err := client.DeleteAccessPolicy(ctx, id); err == nil {
			t.Errorf("DeleteAccessPolicy(%q) returned no error", id)
		}
		if err := client.DeleteAccessPolicyReference(ctx, id); err == nil {
			t.Errorf("DeleteAccessPolicyReference(%q) returned no error", id)
		}
		if _, err := client.CreateAccessPolicyReference(ctx, id, AccessPolicyReference{}); err == nil {
			t.Errorf("CreateAccessPolicyReference(%q) returned no error", id)
		}
	}
}

func TestClient_FindAccessPolicyByRoleName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/AccessPolicies" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if got, want := r.URL.Query().Get("$filter"), "RoleName eq 'O''Brien'"; got != want {
			t.Errorf("$filter = %q, want %q (quotes must be doubled)", got, want)
		}
		_, _ = w.Write([]byte(`{"d": {"results": [{"Id": "7", "RoleName": "O'Brien", "Description": "d"}]}}`))
	}))
	defer server.Close()

	policy, err := New(http.DefaultClient, server.URL).FindAccessPolicyByRoleName(context.Background(), "O'Brien")
	if err != nil {
		t.Fatalf("FindAccessPolicyByRoleName() error: %v", err)
	}
	if policy == nil || policy.ID != "7" {
		t.Fatalf("policy = %+v, want ID 7", policy)
	}
}

func TestClient_FindAccessPolicyByRoleName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"d": {"results": []}}`))
	}))
	defer server.Close()

	policy, err := New(http.DefaultClient, server.URL).FindAccessPolicyByRoleName(context.Background(), "MISSING")
	if err != nil {
		t.Fatalf("FindAccessPolicyByRoleName() error: %v", err)
	}
	if policy != nil {
		t.Errorf("policy = %+v, want nil", policy)
	}
}

func TestClient_CreateAccessPolicyReference_WireFormat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/ArtifactReferences" {
			t.Errorf("got %s %s, want POST /api/v1/ArtifactReferences", r.Method, r.URL.Path)
		}
		body := decodeBody(t, r)
		want := map[string]any{
			"Name":               "metering flows",
			"Description":        "",
			"Type":               "INTEGRATION_FLOW",
			"ConditionAttribute": "Name",
			"ConditionType":      "exactString",
			"ConditionValue":     "Metering",
		}
		for k, v := range want {
			if body[k] != v {
				t.Errorf("body[%q] = %v, want %v", k, body[k], v)
			}
		}
		if _, hasID := body["Id"]; hasID {
			t.Errorf("create body must not carry an Id: %v", body)
		}
		link, ok := body["AccessPolicy"].(map[string]any)
		if !ok || link["Id"] != "1901" {
			t.Errorf("AccessPolicy link = %v, want {\"Id\": \"1901\"}", body["AccessPolicy"])
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"d": {"Id": "55", "Name": "metering flows", "Description": "", "Type": "INTEGRATION_FLOW",
			"ConditionAttribute": "Name", "ConditionType": "exactString", "ConditionValue": "Metering"}}`))
	}))
	defer server.Close()

	ref, err := New(http.DefaultClient, server.URL).CreateAccessPolicyReference(context.Background(), "1901", AccessPolicyReference{
		ID:                 "ignored",
		Name:               "metering flows",
		Type:               "INTEGRATION_FLOW",
		ConditionAttribute: "Name",
		ConditionType:      "exactString",
		ConditionValue:     "Metering",
	})
	if err != nil {
		t.Fatalf("CreateAccessPolicyReference() error: %v", err)
	}
	if ref.ID != "55" || ref.Type != "INTEGRATION_FLOW" {
		t.Errorf("ref = %+v", ref)
	}
}

func TestClient_FindAccessPolicyReference_ReadsThroughPolicy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/api/v1/AccessPolicies(1901L)/ArtifactReferences"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = w.Write([]byte(`{"d": {"results": [
			{"Id": "54", "Name": "a", "Type": "INTEGRATION_FLOW", "ConditionAttribute": "Name", "ConditionType": "exactString", "ConditionValue": "A"},
			{"Id": "55", "Name": "b", "Type": "INTEGRATION_FLOW", "ConditionAttribute": "Name", "ConditionType": "exactString", "ConditionValue": "B"}
		]}}`))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)

	ref, err := client.FindAccessPolicyReference(context.Background(), "1901", "55")
	if err != nil {
		t.Fatalf("FindAccessPolicyReference() error: %v", err)
	}
	if ref == nil || ref.ConditionValue != "B" {
		t.Fatalf("ref = %+v, want reference 55", ref)
	}

	missing, err := client.FindAccessPolicyReference(context.Background(), "1901", "99")
	if err != nil {
		t.Fatalf("FindAccessPolicyReference() error: %v", err)
	}
	if missing != nil {
		t.Errorf("missing reference = %+v, want nil", missing)
	}
}

func TestClient_DeleteAccessPolicyReference_UsesTopLevelEntitySet(t *testing.T) {
	var gotMethod, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := New(http.DefaultClient, server.URL).DeleteAccessPolicyReference(context.Background(), "55"); err != nil {
		t.Fatalf("DeleteAccessPolicyReference() error: %v", err)
	}
	if gotMethod != http.MethodDelete || gotPath != "/api/v1/ArtifactReferences(55L)" {
		t.Errorf("got %s %s, want DELETE /api/v1/ArtifactReferences(55L)", gotMethod, gotPath)
	}
}

func TestClient_ListAccessPolicyRuntimeAssignments(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if want := "/api/v1/AccessPolicies(1901L)/AccessPolicyRuntimeAssignments"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = w.Write([]byte(`{"d": {"results": [
			{"Id": "7", "RuntimeLocationId": "cloudintegration", "TransferStatus": "SUCCESS", "TransferErrors": null, "StatusUpdatedAt": "/Date(1767225600000)/"},
			{"Id": "8", "RuntimeLocationId": "eic-plant-a", "TransferStatus": "PENDING", "TransferErrors": ""}
		]}}`))
	}))
	defer server.Close()

	got, err := New(http.DefaultClient, server.URL).ListAccessPolicyRuntimeAssignments(context.Background(), "1901")
	if err != nil {
		t.Fatalf("ListAccessPolicyRuntimeAssignments() error: %v", err)
	}
	if len(got) != 2 || got[0].RuntimeLocationID != "cloudintegration" || got[1].TransferStatus != "PENDING" {
		t.Errorf("assignments = %+v", got)
	}
	if got[0].StatusUpdatedAt != "/Date(1767225600000)/" {
		t.Errorf("StatusUpdatedAt = %q", got[0].StatusUpdatedAt)
	}
}

func TestClient_ListAccessPolicyRuntimeAssignments_RejectsNonNumericID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		t.Errorf("no request expected, got %s", r.URL.String())
	}))
	defer server.Close()

	if _, err := New(http.DefaultClient, server.URL).ListAccessPolicyRuntimeAssignments(context.Background(), "x"); err == nil {
		t.Error("expected an error for a non-numeric policy ID")
	}
}

// Other Cloud Integration entity sets answer creates with 202 and no body. If
// AccessPolicies does the same, the new policy must be found by its role
// name, since SAP assigns the ID.
func TestCreateAccessPolicy_FindsPolicyAfterEmptyResponse(t *testing.T) {
	var gotFilter string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		gotFilter = r.URL.Query().Get("$filter")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"results":[{"Id":"7","RoleName":"MyRole","Description":"d"}]}}`))
	}))
	defer server.Close()

	created, err := New(http.DefaultClient, server.URL).CreateAccessPolicy(context.Background(), AccessPolicy{RoleName: "MyRole", Description: "d"})
	if err != nil {
		t.Fatalf("CreateAccessPolicy() error: %v", err)
	}
	if created.ID != "7" {
		t.Errorf("ID = %q, want 7", created.ID)
	}
	if gotFilter != "RoleName eq 'MyRole'" {
		t.Errorf("$filter = %q, want the role name lookup", gotFilter)
	}
}

func TestCreateAccessPolicyReference_FindsReferenceAfterEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		if r.URL.Path != "/api/v1/AccessPolicies(7L)/ArtifactReferences" {
			t.Errorf("path = %q, want the policy's reference list", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"d":{"results":[
			{"Id":"3","Name":"flows","Type":"INTEGRATION_FLOW","ConditionAttribute":"Name","ConditionType":"exactString","ConditionValue":"A"},
			{"Id":"9","Name":"flows","Type":"INTEGRATION_FLOW","ConditionAttribute":"Name","ConditionType":"exactString","ConditionValue":"A"},
			{"Id":"12","Name":"other","Type":"INTEGRATION_FLOW","ConditionAttribute":"Name","ConditionType":"exactString","ConditionValue":"B"}]}}`))
	}))
	defer server.Close()

	created, err := New(http.DefaultClient, server.URL).CreateAccessPolicyReference(context.Background(), "7", AccessPolicyReference{
		Name: "flows", Type: "INTEGRATION_FLOW", ConditionAttribute: "Name", ConditionType: "exactString", ConditionValue: "A",
	})
	if err != nil {
		t.Fatalf("CreateAccessPolicyReference() error: %v", err)
	}
	if created.ID != "9" {
		t.Errorf("ID = %q, want 9, the newest matching reference", created.ID)
	}
}
