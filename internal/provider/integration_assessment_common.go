package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

// iaUnofficialNote opens the description of every Integration Assessment
// type: SAP documents the service and its entities, but the field-level
// contract comes from the service's $metadata.
const iaUnofficialNote = "UNOFFICIAL: needs enable_unofficial = true in the provider block. SAP documents the " +
	"Integration Assessment Entities API and its entities, but its field-level specification is only " +
	"available behind an SAP login; the requests follow the service's $metadata and were verified on a " +
	"tenant in September 2026. Needs provider.integration_assessment.\n\n"

// integrationAssessmentClient checks the provider data of an Integration
// Assessment resource or data source and builds its client.
func integrationAssessmentClient(providerData any, typeName, noun string, diags *diag.Diagnostics) *integrationassessment.Client {
	if providerData == nil {
		return nil
	}
	data, ok := providerData.(*Data)
	if !ok {
		diags.AddError("Unexpected provider data type", "Expected *provider.Data")
		return nil
	}
	if !requireOptIn(data, typeName, diags) || !requireIntegrationAssessmentHTTPClient(data, noun, diags) {
		return nil
	}
	return integrationassessment.New(data.IntegrationAssessmentHTTPClient, data.IntegrationAssessmentURL)
}

// iaIDAttribute is the Id the service assigns on create.
func iaIDAttribute(object string) schema.StringAttribute {
	return schema.StringAttribute{
		Computed:      true,
		Description:   "Id the service assigns to the " + object + " on create (a UUID). Use it to link other objects and to import.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

// iaLinkAttribute is a required link to another object that an update
// changes in place (PATCH with the new {"Id": ...}, confirmed on a tenant
// on 2026-09-28).
func iaLinkAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: description,
	}
}

// iaReplacingLinkAttribute is a required link whose change in place was not
// tested; changing it creates a new object.
func iaReplacingLinkAttribute(description string) schema.StringAttribute {
	a := iaLinkAttribute(description)
	a.PlanModifiers = []planmodifier.String{stringplanmodifier.RequiresReplace()}
	return a
}

// stringOrEmpty returns the value, or "" for null and unknown.
func stringOrEmpty(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

// optionalString returns a pointer to the value, or nil for null.
func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// stringPtrOrNull is the inverse of optionalString.
func stringPtrOrNull(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}
