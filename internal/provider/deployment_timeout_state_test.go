package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// A deployment SAP accepted but that did not reach STARTED in time stays in
// state (Terraform marks it tainted), so destroy undeploys it. Before, it was
// dropped and kept running untracked: an acceptance run left a value mapping
// deployed after its five-minute timeout.
func TestMessageMappingDeployment_TimeoutKeepsState(t *testing.T) {
	fastDeploymentPolls(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusAccepted)
		default:
			// Deployed, but still starting, as seen on the tenant.
			_, _ = w.Write([]byte(`{"d": {"Id": "Map_A", "Version": "1.0.5", "Status": "STARTING"}}`))
		}
	}))
	defer server.Close()

	ctx := context.Background()
	r := &messageMappingDeploymentResource{client: cloudintegration.New(http.DefaultClient, server.URL)}
	var sresp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sresp)
	schemaType := sresp.Schema.Type().TerraformType(ctx)

	timeoutTypes := map[string]attr.Type{"create": types.StringType, "update": types.StringType, "delete": types.StringType}
	plan := tfsdk.Plan{Schema: sresp.Schema, Raw: tftypes.NewValue(schemaType, nil)}
	plan.Set(ctx, messageMappingDeploymentModel{
		ID: types.StringUnknown(), PackageID: types.StringValue("P"), MappingID: types.StringValue("Map_A"),
		MappingVersion: types.StringValue("1.0.5"), Status: types.StringUnknown(), RuntimeLocationID: types.StringNull(),
		Timeouts: timeouts.Value{Object: types.ObjectValueMust(timeoutTypes, map[string]attr.Value{
			"create": types.StringValue("200ms"), "update": types.StringNull(), "delete": types.StringNull(),
		})},
	})
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: sresp.Schema, Raw: tftypes.NewValue(schemaType, nil)}}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("want an error for the timeout")
	}
	if !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "tainted") {
		t.Errorf("error should explain the tainted state: %s", resp.Diagnostics.Errors()[0].Detail())
	}
	var got messageMappingDeploymentModel
	resp.Diagnostics.Append(resp.State.Get(ctx, &got)...)
	if got.MappingID.ValueString() != "Map_A" || got.MappingVersion.ValueString() != "1.0.5" || got.Status.ValueString() != "STARTING" {
		t.Errorf("state = id %q, version %q, status %q; want the deployment kept with its last status",
			got.MappingID.ValueString(), got.MappingVersion.ValueString(), got.Status.ValueString())
	}
}
