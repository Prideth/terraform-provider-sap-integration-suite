package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// fakeVirtualHosts answers like the API portal of the probe of 2026-10-06:
// requests to Configuration.svc/VirtualHostRequests take effect at once and
// show in Management.svc/VirtualHosts, where hosts can only be listed.
type fakeVirtualHosts struct {
	mu       sync.Mutex
	hosts    map[string]fakeVirtualHost
	requests []map[string]any
}

// fakeVirtualHost is what the fake keeps of a host.
type fakeVirtualHost struct {
	alias      string
	clientAuth bool
	trustStore string
}

const fakeVirtualHostDomain = ".mysubaccount.apimanagement.eu10.hana.ondemand.com"

func (f *fakeVirtualHosts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/Management.svc/VirtualHosts"):
		f.list(w)
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/Configuration.svc/VirtualHostRequests"):
		var req map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		f.requests = append(f.requests, req)
		f.request(w, req)
	default:
		w.WriteHeader(http.StatusForbidden)
	}
}

func (f *fakeVirtualHosts) list(w http.ResponseWriter) {
	var rows []string
	for id, h := range f.hosts {
		trustStore := "null"
		if h.trustStore != "" {
			trustStore = fmt.Sprintf("%q", h.trustStore)
		}
		rows = append(rows, fmt.Sprintf(`{"id":%q,"name":%q,"isDefault":false,"isSSL":true,"isClientAuthEnabled":%t,"trustStore":%s,"virtual_host":%q,"virtual_port":443}`,
			id, id, h.clientAuth, trustStore, h.alias+fakeVirtualHostDomain))
	}
	_, _ = w.Write([]byte(`{"d":{"results":[` + strings.Join(rows, ",") + `]}}`))
}

func (f *fakeVirtualHosts) request(w http.ResponseWriter, req map[string]any) {
	id, _ := req["virtualHostId"].(string)
	alias, _ := req["virtualHostUrl"].(string)
	clientAuth, _ := req["isClientAuthEnabled"].(bool)
	trustStore, _ := req["trustStore"].(string)
	host := fakeVirtualHost{alias: alias, clientAuth: clientAuth, trustStore: trustStore}
	current, known := f.hosts[id]
	switch {
	case req["operation"] == "CREATE":
		id = fmt.Sprintf("00000000-0000-4000-8000-%012d", len(f.requests))
		f.hosts[id] = host
	case !known:
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":{"code":"VHR_NO_COMPLETED_RECORD_FOUND","message":{"lang":"en","value":"Could not locate previous status for the virtual host id ` + id + `."}}}`))
		return
	case req["operation"] == "DELETE":
		delete(f.hosts, id)
		alias = ""
	default:
		if _, sent := req["isClientAuthEnabled"]; !sent {
			// Without the TLS fields the fake keeps the host's setting.
			host.clientAuth, host.trustStore = current.clientAuth, current.trustStore
		}
		f.hosts[id] = host
	}
	w.WriteHeader(http.StatusCreated)
	_, _ = fmt.Fprintf(w, `{"d":{"id":"request-%d","virtualHostId":%q,"virtualHostUrl":%q,"allocationStatus":"COMPLETE","allocatedPort":443,"operation":%q}}`,
		len(f.requests), id, alias+fakeVirtualHostDomain, req["operation"])
}

func virtualHostTestResource(t *testing.T, fake *fakeVirtualHosts) (*apiManagementVirtualHostResource, func()) {
	t.Helper()
	server := httptest.NewServer(fake)
	return &apiManagementVirtualHostResource{
		client:    apimanagementclassic.New(http.DefaultClient, server.URL),
		subdomain: "mysubaccount",
	}, server.Close
}

func virtualHostSchema(r *apiManagementVirtualHostResource) schema.Schema {
	var schemaResp resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &schemaResp)
	return schemaResp.Schema
}

// virtualHostPlan plans a host; a trustStore switches mutual TLS on.
func virtualHostPlan(t *testing.T, r *apiManagementVirtualHostResource, id, alias, trustStore string) tfsdk.Plan {
	t.Helper()
	s := virtualHostSchema(r)
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	var idValue, trustStoreValue any
	if id != "" {
		idValue = id
	}
	if trustStore != "" {
		trustStoreValue = trustStore
	}
	return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":                  tftypes.NewValue(tftypes.String, idValue),
		"alias":               tftypes.NewValue(tftypes.String, alias),
		"host_name":           tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"port":                tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		"default":             tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"ssl":                 tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"client_auth_enabled": tftypes.NewValue(tftypes.Bool, trustStore != ""),
		"trust_store":         tftypes.NewValue(tftypes.String, trustStoreValue),
	})}
}

func TestAPIManagementVirtualHostResource_Configure(t *testing.T) {
	r := NewAPIManagementVirtualHostResource().(resource.ResourceWithConfigure)
	var resp resource.ConfigureResponse
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &Data{}}, &resp)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "api_management_self_service") {
		t.Fatalf("Configure() without the self-service block = %v, want an error naming api_management_self_service", resp.Diagnostics)
	}
}

func TestAPIManagementVirtualHostResource_Lifecycle(t *testing.T) {
	fake := &fakeVirtualHosts{hosts: map[string]fakeVirtualHost{}}
	r, closeServer := virtualHostTestResource(t, fake)
	defer closeServer()
	ctx := context.Background()

	createResp := &resource.CreateResponse{State: newTestState(t, virtualHostSchema(r))}
	r.Create(ctx, resource.CreateRequest{Plan: virtualHostPlan(t, r, "", "prod-apis", "")}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", createResp.Diagnostics)
	}
	var created apiManagementVirtualHostModel
	createResp.State.Get(ctx, &created)
	if created.HostName.ValueString() != "prod-apis"+fakeVirtualHostDomain || created.Port.ValueInt64() != 443 || !created.SSL.ValueBool() {
		t.Errorf("state after create = %+v", created)
	}
	id := created.ID.ValueString()

	updateResp := &resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: virtualHostPlan(t, r, id, "prod-apis-2", ""), State: createResp.State}, updateResp)
	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update() produced diagnostics: %v", updateResp.Diagnostics)
	}
	var updated apiManagementVirtualHostModel
	updateResp.State.Get(ctx, &updated)
	if updated.ID.ValueString() != id || updated.HostName.ValueString() != "prod-apis-2"+fakeVirtualHostDomain {
		t.Errorf("state after update = %+v", updated)
	}

	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, &resource.DeleteResponse{})
	readResp := &resource.ReadResponse{State: updateResp.State}
	r.Read(ctx, resource.ReadRequest{State: updateResp.State}, readResp)
	if readResp.Diagnostics.HasError() || !readResp.State.Raw.IsNull() {
		t.Errorf("Read() after delete = %v, state null %v; want the resource removed", readResp.Diagnostics, readResp.State.Raw.IsNull())
	}
	// A second delete meets SAP's "unknown ID" answer and succeeds.
	deleteResp := &resource.DeleteResponse{}
	r.Delete(ctx, resource.DeleteRequest{State: updateResp.State}, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Errorf("Delete() of a deleted host produced diagnostics: %v", deleteResp.Diagnostics)
	}

	wantOps := []string{"CREATE", "UPDATE", "DELETE", "DELETE"}
	for i, req := range fake.requests {
		if req["operation"] != wantOps[i] {
			t.Errorf("request %d operation = %v, want %s", i, req["operation"], wantOps[i])
		}
	}
	if fake.requests[0]["accountId"] != "mysubaccount" || fake.requests[1]["isDefaultVirtualHostRequest"] != false {
		t.Errorf("requests = %v; want the subdomain as accountId and the host's default flag kept", fake.requests)
	}
}

func TestAPIManagementVirtualHostResource_MutualTLS(t *testing.T) {
	fake := &fakeVirtualHosts{hosts: map[string]fakeVirtualHost{}}
	r, closeServer := virtualHostTestResource(t, fake)
	defer closeServer()
	ctx := context.Background()

	createResp := &resource.CreateResponse{State: newTestState(t, virtualHostSchema(r))}
	r.Create(ctx, resource.CreateRequest{Plan: virtualHostPlan(t, r, "", "mtls-apis", "clients")}, createResp)
	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create() produced diagnostics: %v", createResp.Diagnostics)
	}
	var created apiManagementVirtualHostModel
	createResp.State.Get(ctx, &created)
	if !created.ClientAuthEnabled.ValueBool() || created.TrustStore.ValueString() != "clients" {
		t.Errorf("state after create = %+v, want client authentication against clients", created)
	}
	id := created.ID.ValueString()

	// Another truststore, through a certificate store reference.
	updateResp := &resource.UpdateResponse{State: createResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: virtualHostPlan(t, r, id, "mtls-apis", "ref://clients-2026"), State: createResp.State}, updateResp)
	var updated apiManagementVirtualHostModel
	updateResp.State.Get(ctx, &updated)
	if updateResp.Diagnostics.HasError() || updated.TrustStore.ValueString() != "ref://clients-2026" {
		t.Fatalf("Update() = %v, state %+v; want the reference", updateResp.Diagnostics, updated)
	}

	// Switched off: the update says so explicitly.
	offResp := &resource.UpdateResponse{State: updateResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: virtualHostPlan(t, r, id, "mtls-apis", ""), State: updateResp.State}, offResp)
	var off apiManagementVirtualHostModel
	offResp.State.Get(ctx, &off)
	if offResp.Diagnostics.HasError() || off.ClientAuthEnabled.ValueBool() || !off.TrustStore.IsNull() {
		t.Fatalf("Update(off) = %v, state %+v; want client authentication off", offResp.Diagnostics, off)
	}

	want := []string{
		`{"accountId":"mysubaccount","isClientAuthEnabled":true,"isDefaultVirtualHostRequest":false,"operation":"CREATE","trustStore":"clients","virtualHostUrl":"mtls-apis"}`,
		`{"accountId":"mysubaccount","isClientAuthEnabled":true,"isDefaultVirtualHostRequest":false,"operation":"UPDATE","trustStore":"ref://clients-2026","virtualHostId":"` + id + `","virtualHostUrl":"mtls-apis"}`,
		`{"accountId":"mysubaccount","isClientAuthEnabled":false,"isDefaultVirtualHostRequest":false,"operation":"UPDATE","virtualHostId":"` + id + `","virtualHostUrl":"mtls-apis"}`,
	}
	for i, req := range fake.requests {
		got, _ := json.Marshal(req)
		if i >= len(want) || string(got) != want[i] {
			t.Errorf("request %d = %s", i, got)
		}
	}

	// An update of a host without mutual TLS leaves the TLS fields out.
	plainResp := &resource.UpdateResponse{State: offResp.State}
	r.Update(ctx, resource.UpdateRequest{Plan: virtualHostPlan(t, r, id, "plain-apis", ""), State: offResp.State}, plainResp)
	if last := fake.requests[len(fake.requests)-1]; plainResp.Diagnostics.HasError() || last["isClientAuthEnabled"] != nil {
		t.Errorf("Update(plain) = %v, request %v; want the default domain body", plainResp.Diagnostics, last)
	}
}

func TestAPIManagementVirtualHostResource_ValidateConfig(t *testing.T) {
	r := &apiManagementVirtualHostResource{}
	for _, tc := range []struct {
		clientAuth bool
		trustStore string
		wantError  bool
	}{
		{false, "", false},
		{true, "clients", false},
		{true, "", true},
		{false, "clients", true},
	} {
		plan := virtualHostPlan(t, r, "", "apis", tc.trustStore)
		raw := map[string]tftypes.Value{}
		_ = plan.Raw.As(&raw)
		raw["client_auth_enabled"] = tftypes.NewValue(tftypes.Bool, tc.clientAuth)
		objType := plan.Schema.Type().TerraformType(context.Background()).(tftypes.Object)
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(context.Background(), resource.ValidateConfigRequest{
			Config: tfsdk.Config{Schema: plan.Schema, Raw: tftypes.NewValue(objType, raw)},
		}, resp)
		if resp.Diagnostics.HasError() != tc.wantError {
			t.Errorf("client_auth_enabled = %v, trust_store = %q: diagnostics %v, want error %v",
				tc.clientAuth, tc.trustStore, resp.Diagnostics, tc.wantError)
		}
	}
}

func TestAPIManagementVirtualHostResource_ImportByAlias(t *testing.T) {
	fake := &fakeVirtualHosts{hosts: map[string]fakeVirtualHost{"00000000-0000-4000-8000-000000000042": {alias: "test-apis"}}}
	r, closeServer := virtualHostTestResource(t, fake)
	defer closeServer()
	ctx := context.Background()

	for _, importID := range []string{"test-apis", "TEST-APIS", "test-apis" + fakeVirtualHostDomain, "00000000-0000-4000-8000-000000000042"} {
		resp := &resource.ImportStateResponse{State: newTestState(t, virtualHostSchema(r))}
		r.ImportState(ctx, resource.ImportStateRequest{ID: importID}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState(%q) produced diagnostics: %v", importID, resp.Diagnostics)
		}
		var state apiManagementVirtualHostModel
		resp.State.Get(ctx, &state)
		if state.ID.ValueString() != "00000000-0000-4000-8000-000000000042" || state.Alias.ValueString() != "test-apis" {
			t.Errorf("ImportState(%q) state = %+v", importID, state)
		}
	}
	resp := &resource.ImportStateResponse{State: newTestState(t, virtualHostSchema(r))}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "missing"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("ImportState(missing) produced no error")
	}
}

func TestVirtualHostAliasRule(t *testing.T) {
	for alias, want := range map[string]bool{
		"prod-apis": true, "a": true, "Prod1": true, "a-b-c": true,
		"-prod": false, "prod-": false, "prod_apis": false, "prod.apis": false, "": false,
	} {
		if got := virtualHostAlias.MatchString(alias); got != want {
			t.Errorf("virtualHostAlias(%q) = %v, want %v", alias, got, want)
		}
	}
}

func TestVirtualHostModel_KeepsConfiguredCase(t *testing.T) {
	host := &apimanagementclassic.VirtualHost{ID: "id", HostName: "prod-apis" + fakeVirtualHostDomain, Port: 443}
	if got := virtualHostModel(host, types.StringValue("Prod-APIs")).Alias.ValueString(); got != "Prod-APIs" {
		t.Errorf("alias = %q, want the configured spelling", got)
	}
	if got := virtualHostModel(host, types.StringValue("other")).Alias.ValueString(); got != "prod-apis" {
		t.Errorf("alias = %q, want SAP's alias after a change outside Terraform", got)
	}
}

func TestUnsupportedVirtualHost(t *testing.T) {
	if msg := unsupportedVirtualHost(&apimanagementclassic.VirtualHost{ID: "id"}); msg != "" {
		t.Errorf("default domain host refused: %s", msg)
	}
	if msg := unsupportedVirtualHost(&apimanagementclassic.VirtualHost{ID: "id", IsClientAuthEnabled: true, TrustStore: "clients"}); msg != "" {
		t.Errorf("mutual TLS host refused: %s", msg)
	}
	for _, h := range []apimanagementclassic.VirtualHost{
		{ID: "id", IsForCustomDomain: true}, {ID: "id", KeyStoreName: "ref://keystore"}, {ID: "id", KeyStoreAlias: "server"},
	} {
		if unsupportedVirtualHost(&h) == "" {
			t.Errorf("host %+v accepted, want it refused", h)
		}
	}
}
