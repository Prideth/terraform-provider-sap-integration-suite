package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// redeploy_triggers must be an in-place update: Update redeploys without an
// undeploy, while a replacement would take the flow offline in between.
func TestIntegrationFlowDeploymentResource_RedeployTriggersUpdateInPlace(t *testing.T) {
	var resp resource.SchemaResponse
	NewIntegrationFlowDeploymentResource().Schema(context.Background(), resource.SchemaRequest{}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() diagnostics: %v", resp.Diagnostics)
	}

	attr, ok := resp.Schema.Attributes["redeploy_triggers"].(schema.MapAttribute)
	if !ok {
		t.Fatal("redeploy_triggers is missing or not a map attribute")
	}
	if !attr.Optional || attr.Computed {
		t.Error("redeploy_triggers must be Optional and not Computed")
	}
	if len(attr.PlanModifiers) != 0 {
		t.Error("redeploy_triggers must not carry plan modifiers such as RequiresReplace")
	}
}

// Uploading new content keeps an artifact's version, so every deployment
// resource needs redeploy_triggers to pick up new content without a new
// version. It must update in place (redeploy), never replace (undeploy).
func TestDeploymentResources_RedeployTriggersUpdateInPlace(t *testing.T) {
	for name, r := range map[string]resource.Resource{
		"integration flow":  NewIntegrationFlowDeploymentResource(),
		"message mapping":   NewMessageMappingDeploymentResource(),
		"script collection": NewScriptCollectionDeploymentResource(),
		"value mapping":     NewValueMappingDeploymentResource(),
	} {
		var resp resource.SchemaResponse
		r.Schema(context.Background(), resource.SchemaRequest{}, &resp)
		attr, ok := resp.Schema.Attributes["redeploy_triggers"].(schema.MapAttribute)
		if !ok || !attr.Optional || attr.Computed || len(attr.PlanModifiers) != 0 {
			t.Errorf("%s deployment: redeploy_triggers must be an optional map without plan modifiers", name)
		}
	}
}
