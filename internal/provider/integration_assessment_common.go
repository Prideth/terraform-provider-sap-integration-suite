package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

// iaLinkChangeOp is the one operation of the landscape resources that SAP's
// Entities API specification does not document: its update bodies list no
// links, but a tenant accepted a changed link in a PATCH (2026-09-28).
const iaLinkChangeOp = "link change in place (PATCH with a link)"

// iaLinkChangeNote ends the description of a resource whose links change in
// place.
const iaLinkChangeNote = "\n\nChanging a link in place needs enable_unofficial = true: SAP's Entities API " +
	"specification lists no links in its update bodies, but a tenant accepted the change. Names and " +
	"descriptions change without it."

// integrationAssessmentClient checks the provider data of an Integration
// Assessment resource or data source and builds its client.
func integrationAssessmentClient(providerData any, typeName, noun string, diags *diag.Diagnostics) *integrationassessment.Client {
	data := integrationAssessmentData(providerData, diags)
	if data == nil {
		return nil
	}
	if !requireOptIn(data, typeName, diags) || !requireIntegrationAssessmentHTTPClient(data, noun, diags) {
		return nil
	}
	return integrationassessment.New(data.IntegrationAssessmentHTTPClient, data.IntegrationAssessmentURL)
}

// integrationAssessmentData returns the provider data, or nil before the
// provider is configured or when it has the wrong type.
func integrationAssessmentData(providerData any, diags *diag.Diagnostics) *Data {
	if providerData == nil {
		return nil
	}
	data, ok := providerData.(*Data)
	if !ok {
		diags.AddError("Unexpected provider data type", "Expected *provider.Data")
		return nil
	}
	return data
}

// iaLinkChange returns the change of a link from its state to its plan, or
// nil when the link stays. An unknown plan value counts as a change.
func iaLinkChange(state, plan types.String) *integrationassessment.LinkChange {
	if state.Equal(plan) {
		return nil
	}
	return &integrationassessment.LinkChange{To: stringOrEmpty(plan)}
}

// iaRequireLinkChange refuses a plan that changes one of the given links in
// place unless the provider enables unofficial operations. pairs holds the
// state and plan value of each link.
func iaRequireLinkChange(allowed bool, typeName string, diags *diag.Diagnostics, pairs ...[2]types.String) bool {
	for _, p := range pairs {
		if iaLinkChange(p[0], p[1]) != nil {
			return requireUnofficialOperation(allowed, typeName, iaLinkChangeOp, diags)
		}
	}
	return true
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
// on 2026-09-28). The change is gated by iaRequireLinkChange.
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
