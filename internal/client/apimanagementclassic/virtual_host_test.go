package apimanagementclassic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The answers below follow those of a tenant (October 2026), with neutral
// names.
const (
	testVirtualHostsBody = `{"d":{"results":[
		{"id":"de4d04d6-0000-4000-8000-000000000001","name":"de4d04d6-0000-4000-8000-000000000001","isDefault":true,"isSSL":true,"isForCustomDomain":false,"isClientAuthEnabled":false,"keyStoreName":null,"keyStoreAlias":null,"trustStore":null,"projectPath":null,"virtual_host":"mysubaccount.apimanagement.eu10.hana.ondemand.com","virtual_port":443},
		{"id":"79b619fc-0000-4000-8000-000000000002","name":"79b619fc-0000-4000-8000-000000000002","isDefault":false,"isSSL":true,"isForCustomDomain":false,"isClientAuthEnabled":false,"keyStoreName":null,"keyStoreAlias":null,"trustStore":null,"projectPath":null,"virtual_host":"prod-apis.apimanagement.eu10.hana.ondemand.com","virtual_port":443}
	]}}`
	testVirtualHostRequestBody = `{"d":{"accountId":"mysubaccount","allocatedPort":443,"allocationStatus":"COMPLETE","clusterName":"","id":"439759e8-0000-4000-8000-000000000003","isClientAuthEnabled":false,"isDefaultVirtualHostRequest":false,"isForCustomDomain":false,"isForNonSni":false,"isTLS":false,"keyStoreAlias":null,"keyStoreName":null,"operation":"CREATE","trustStore":null,"virtualHostId":"79b619fc-0000-4000-8000-000000000002","virtualHostUrl":"prod-apis.apimanagement.eu10.hana.ondemand.com","lbHost":null}}`
)

func TestClient_FindVirtualHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/apiportal/api/1.0/Management.svc/VirtualHosts"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		_, _ = w.Write([]byte(testVirtualHostsBody))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	found, err := client.FindVirtualHost(context.Background(), "79b619fc-0000-4000-8000-000000000002")
	if err != nil {
		t.Fatalf("FindVirtualHost() error: %v", err)
	}
	if found == nil || found.Alias() != "prod-apis" || found.Port != 443 || found.IsDefault || !found.IsSSL {
		t.Errorf("found = %+v, want the prod-apis host on port 443", found)
	}
	missing, err := client.FindVirtualHost(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil || missing != nil {
		t.Errorf("FindVirtualHost(unknown) = %+v, %v; want nil, nil", missing, err)
	}
}

func TestClient_VirtualHostRequests(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if want := "/apiportal/operations/1.0/Configuration.svc/VirtualHostRequests"; r.URL.Path != want {
			t.Errorf("path = %q, want %q", r.URL.Path, want)
		}
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, string(body))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(testVirtualHostRequestBody))
	}))
	defer server.Close()

	client := New(http.DefaultClient, server.URL)
	ctx := context.Background()
	created, err := client.CreateVirtualHost(ctx, "mysubaccount", "prod-apis", nil)
	if err != nil {
		t.Fatalf("CreateVirtualHost() error: %v", err)
	}
	if created.VirtualHostID != "79b619fc-0000-4000-8000-000000000002" || created.AllocationStatus != "COMPLETE" {
		t.Errorf("created = %+v", created)
	}
	if _, err := client.UpdateVirtualHost(ctx, "mysubaccount", "79b619fc-0000-4000-8000-000000000002", "prod-apis-2", true, nil); err != nil {
		t.Fatalf("UpdateVirtualHost() error: %v", err)
	}
	// Mutual TLS: switched on at create, then off again by an update.
	if _, err := client.CreateVirtualHost(ctx, "mysubaccount", "mtls-apis", &VirtualHostTLS{ClientAuthEnabled: true, TrustStore: "ref://clients"}); err != nil {
		t.Fatalf("CreateVirtualHost(mutual TLS) error: %v", err)
	}
	if _, err := client.UpdateVirtualHost(ctx, "mysubaccount", "79b619fc-0000-4000-8000-000000000002", "mtls-apis", false, &VirtualHostTLS{}); err != nil {
		t.Fatalf("UpdateVirtualHost(mutual TLS off) error: %v", err)
	}
	if err := client.DeleteVirtualHost(ctx, "79b619fc-0000-4000-8000-000000000002"); err != nil {
		t.Fatalf("DeleteVirtualHost() error: %v", err)
	}

	want := []string{
		`{"accountId":"mysubaccount","virtualHostUrl":"prod-apis","isDefaultVirtualHostRequest":false,"operation":"CREATE"}`,
		`{"accountId":"mysubaccount","virtualHostUrl":"prod-apis-2","isDefaultVirtualHostRequest":true,"operation":"UPDATE","virtualHostId":"79b619fc-0000-4000-8000-000000000002"}`,
		`{"accountId":"mysubaccount","virtualHostUrl":"mtls-apis","isDefaultVirtualHostRequest":false,"isClientAuthEnabled":true,"trustStore":"ref://clients","operation":"CREATE"}`,
		`{"accountId":"mysubaccount","virtualHostUrl":"mtls-apis","isDefaultVirtualHostRequest":false,"isClientAuthEnabled":false,"operation":"UPDATE","virtualHostId":"79b619fc-0000-4000-8000-000000000002"}`,
		`{"operation":"DELETE","virtualHostId":"79b619fc-0000-4000-8000-000000000002"}`,
	}
	if len(bodies) != len(want) {
		t.Fatalf("got %d requests, want %d", len(bodies), len(want))
	}
	for i := range want {
		if bodies[i] != want[i] {
			t.Errorf("request %d body = %s, want %s", i, bodies[i], want[i])
		}
	}
}

func TestIsUnknownVirtualHost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"VHR_NO_COMPLETED_RECORD_FOUND","message":{"lang":"en","value":"Could not locate previous status for the virtual host id 79b619fc-0000-4000-8000-000000000002."}}}`))
	}))
	defer server.Close()

	err := New(http.DefaultClient, server.URL).DeleteVirtualHost(context.Background(), "79b619fc-0000-4000-8000-000000000002")
	if !IsUnknownVirtualHost(err) {
		t.Errorf("IsUnknownVirtualHost(%v) = false, want true", err)
	}
	if IsUnknownVirtualHost(nil) {
		t.Error("IsUnknownVirtualHost(nil) = true")
	}
}
