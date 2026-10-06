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
	hosts    map[string]string // id -> alias
	requests []map[string]any
}

const fakeVirtualHostDomain = ".mysubaccount.apimanagement.eu10.hana.ondemand.com"

func (f *fakeVirtualHosts) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/Management.svc/VirtualHosts"):
		var rows []string
		for id, alias := range f.hosts {
			rows = append(rows, fmt.Sprintf(`{"id":%q,"name":%q,"isDefault":false,"isSSL":true,"virtual_host":%q,"virtual_port":443}`,
				id, id, alias+fakeVirtualHostDomain))
		}
		_, _ = w.Write([]byte(`{"d":{"results":[` + strings.Join(rows, ",") + `]}}`))
	case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/Configuration.svc/VirtualHostRequests"):
		var req map[string]any
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &req)
		f.requests = append(f.requests, req)
		id, _ := req["virtualHostId"].(string)
		alias, _ := req["virtualHostUrl"].(string)
		switch req["operation"] {
		case "CREATE":
			id = fmt.Sprintf("00000000-0000-4000-8000-%012d", len(f.requests))
			f.hosts[id] = alias
		case "UPDATE", "DELETE":
			if _, ok := f.hosts[id]; !ok {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":{"code":"VHR_NO_COMPLETED_RECORD_FOUND","message":{"lang":"en","value":"Could not locate previous status for the virtual host id ` + id + `."}}}`))
				return
			}
			if req["operation"] == "DELETE" {
				delete(f.hosts, id)
				alias = ""
			} else {
				f.hosts[id] = alias
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprintf(w, `{"d":{"id":"request-%d","virtualHostId":%q,"virtualHostUrl":%q,"allocationStatus":"COMPLETE","allocatedPort":443,"operation":%q}}`,
			len(f.requests), id, alias+fakeVirtualHostDomain, req["operation"])
	default:
		w.WriteHeader(http.StatusForbidden)
	}
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

func virtualHostPlan(t *testing.T, r *apiManagementVirtualHostResource, id, alias string) tfsdk.Plan {
	t.Helper()
	s := virtualHostSchema(r)
	objType := s.Type().TerraformType(context.Background()).(tftypes.Object)
	var idValue any
	if id != "" {
		idValue = id
	}
	return tfsdk.Plan{Schema: s, Raw: tftypes.NewValue(objType, map[string]tftypes.Value{
		"id":        tftypes.NewValue(tftypes.String, idValue),
		"alias":     tftypes.NewValue(tftypes.String, alias),
		"host_name": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"port":      tftypes.NewValue(tftypes.Number, tftypes.UnknownValue),
		"default":   tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
		"ssl":       tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue),
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
	fake := &fakeVirtualHosts{hosts: map[string]string{}}
	r, closeServer := virtualHostTestResource(t, fake)
	defer closeServer()
	ctx := context.Background()

	createResp := &resource.CreateResponse{State: newTestState(t, virtualHostSchema(r))}
	r.Create(ctx, resource.CreateRequest{Plan: virtualHostPlan(t, r, "", "prod-apis")}, createResp)
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
	r.Update(ctx, resource.UpdateRequest{Plan: virtualHostPlan(t, r, id, "prod-apis-2"), State: createResp.State}, updateResp)
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

func TestAPIManagementVirtualHostResource_ImportByAlias(t *testing.T) {
	fake := &fakeVirtualHosts{hosts: map[string]string{"00000000-0000-4000-8000-000000000042": "test-apis"}}
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
	for _, h := range []apimanagementclassic.VirtualHost{
		{ID: "id", IsForCustomDomain: true}, {ID: "id", IsClientAuthEnabled: true},
		{ID: "id", KeyStoreName: "ref://keystore"}, {ID: "id", TrustStore: "truststore"},
	} {
		if unsupportedVirtualHost(&h) == "" {
			t.Errorf("host %+v accepted, want it refused", h)
		}
	}
}
