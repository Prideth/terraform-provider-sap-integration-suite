package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// NewOAuth2ClientCredentialResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_oauth2_client_credential.
func NewOAuth2ClientCredentialResource() resource.Resource {
	return &oauth2ClientCredentialResource{}
}

type oauth2ClientCredentialResource struct {
	client *securitycontent.Client
	// allowUnofficial allows custom_parameters, which only the tenant
	// $metadata describes (enable_unofficial).
	allowUnofficial bool
}

// oauth2ClientCredentialModel follows the same write-only pattern as
// userCredentialModel; see that type's doc comment for why ClientSecretWO
// exists as a struct field despite never being read from it.
type oauth2ClientCredentialModel struct {
	ID                    types.String `tfsdk:"id"`
	Description           types.String `tfsdk:"description"`
	TokenServiceURL       types.String `tfsdk:"token_service_url"`
	ClientID              types.String `tfsdk:"client_id"`
	Scope                 types.String `tfsdk:"scope"`
	ClientAuthentication  types.String `tfsdk:"client_authentication"`
	ScopeContentType      types.String `tfsdk:"scope_content_type"`
	Resource              types.String `tfsdk:"resource"`
	Audience              types.String `tfsdk:"audience"`
	ClientSecretWO        types.String `tfsdk:"client_secret_wo"`
	ClientSecretWOVersion types.String `tfsdk:"client_secret_wo_version"`
	RuntimeLocationID     types.String `tfsdk:"runtime_location_id"`
	CustomParameters      types.Set    `tfsdk:"custom_parameters"`
}

var customParameterAttrTypes = map[string]attr.Type{
	"key":             types.StringType,
	"value":           types.StringType,
	"send_as_part_of": types.StringType,
}

func (r *oauth2ClientCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth2_client_credential"
}

func (r *oauth2ClientCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Security Content \"OAuth2 Client Credentials\" artifact: the client " +
			"ID, client secret, and token service URL an integration flow adapter uses to obtain an " +
			"OAuth2 access token for outbound requests (RFC 6749 client credentials grant). Backed by " +
			"the public Security Content OData V2 API (OAuth2ClientCredentials). This provider manages " +
			"the artifact's scalar fields (name, description, token " +
			"service URL, client ID, client secret, scope, client authentication, content type, " +
			"resource, audience) and, with enable_unofficial, its custom parameters; the grant-type " +
			"placement has no API property — see docs/guides/security-content.md. The client secret is a write-only " +
			"attribute: Terraform never stores it in plan or state, and SAP documents that it must be " +
			"re-entered on every edit, so this provider resends it on every apply that touches the " +
			"resource. Requires Terraform CLI 1.11 or later for write-only attribute support.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Required: true,
				Description: "The credential artifact's name, also called its alias when used in an " +
					"adapter. This is also this resource's OData key. Immutable: SAP does not " +
					"document renaming a security material artifact, only deleting and recreating it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A free-text description of the credential artifact.",
			},
			"token_service_url": schema.StringAttribute{
				Required:    true,
				Description: "URL of the OAuth2 authorization server that issues the access token.",
			},
			"client_id": schema.StringAttribute{
				Required:    true,
				Description: "The OAuth2 client ID registered with the token service.",
			},
			"scope": schema.StringAttribute{
				Optional:    true,
				Description: "OAuth2 scope to request, if the token service requires one.",
			},
			"client_authentication": oauth2PassThroughAttribute("How the client ID and secret are sent to the token service, as SAP stores it in ClientAuthentication. The UI offers \"Send as Body Parameter\" (default) and \"Send as Request Header\"; the API constants for these are not documented."),
			"scope_content_type":    oauth2PassThroughAttribute("Content type of the token request, as SAP stores it in ScopeContentType (the UI's \"Content Type\" field)."),
			"resource":              oauth2PassThroughAttribute("Resource identifier sent to the token service, for services that require one."),
			"audience":              oauth2PassThroughAttribute("Audience identifier sent to the token service, for services that require one."),
			"client_secret_wo": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "The OAuth2 client secret. Write-only: Terraform never stores this " +
					"value in plan or state, and it is never read back from SAP. Required on every " +
					"apply that creates or redeploys this resource, since SAP documents that editing " +
					"an OAuth2 Client Credentials artifact requires re-entering the client secret " +
					"every time.",
			},
			"custom_parameters": schema.SetNestedAttribute{
				Optional: true,
				Description: "Custom parameters of the token request, each sent in the body, as a header " +
					"or in the URL. Unofficial: needs enable_unofficial = true, because the parameters " +
					"are known only from the tenant $metadata. SAP takes them only when the credential is " +
					"created and deletes them with every update, so with custom_parameters set, every " +
					"change of this resource, a secret rotation included, replaces the credential: it is " +
					"deleted and created again, and flows that use it fail in between. Leave the attribute " +
					"out to leave parameters set in SAP's UI alone; the plan of an update then warns that " +
					"SAP deletes them. Values are stored in plan and state; do not put secrets into them.",
				Validators: []validator.Set{setvalidator.SizeAtLeast(1)},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.RequiresReplace(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Required: true, Description: "The parameter name.", Validators: []validator.String{stringvalidator.LengthBetween(1, 1024)}},
						"value": schema.StringAttribute{Required: true, Description: "The parameter value.", Validators: []validator.String{stringvalidator.LengthBetween(1, 1024)}},
						"send_as_part_of": schema.StringAttribute{
							Required:    true,
							Description: "Where the parameter goes: \"body\", \"header\" or \"url\", the values SAP accepts.",
							Validators:  []validator.String{stringvalidator.OneOf("body", "header", "url")},
						},
					},
				},
			},
			"client_secret_wo_version": schema.StringAttribute{
				Required: true,
				Description: "An arbitrary value (for example a counter or timestamp) that a " +
					"practitioner changes to signal that client_secret_wo's value has changed and " +
					"the credential should be rotated. Changing it redeploys the credential in " +
					"place (SAP's documented \"Edit and deploy\" action) rather than replacing the " +
					"resource.",
			},
		},
	}
}

func (r *oauth2ClientCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
	r.allowUnofficial = data.EnableUnofficial
}

// ModifyPlan handles custom parameters, which SAP takes only on create and
// deletes with every update (tenant check of 2026-10-04):
//
//   - With custom_parameters configured, any change replaces the credential
//     instead of updating it, so the parameters are sent again.
//   - Creating with custom_parameters needs enable_unofficial.
//   - Without custom_parameters, an update or replacement warns when SAP
//     holds parameters set outside Terraform, because it deletes them.
func (r *oauth2ClientCredentialResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	managed := !plan.CustomParameters.IsNull()
	changed := !req.State.Raw.IsNull() && !req.Plan.Raw.Equal(req.State.Raw)
	if managed && changed {
		resp.RequiresReplace = append(resp.RequiresReplace, changedRootAttributes(req.State.Raw, req.Plan.Raw)...)
	}
	if managed && (req.State.Raw.IsNull() || len(resp.RequiresReplace) > 0) && r.client != nil &&
		!requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_oauth2_client_credential", "custom_parameters", &resp.Diagnostics) {
		return
	}
	if managed || !changed || r.client == nil {
		return
	}
	var state oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if !state.CustomParameters.IsNull() {
		return // removed from the configuration on purpose
	}
	client, err := r.client.AtLocation(state.RuntimeLocationID.ValueString())
	if err != nil {
		return
	}
	params, err := client.ListOAuth2ClientCredentialCustomParameters(ctx, state.ID.ValueString())
	if err != nil || len(params) == 0 {
		return
	}
	resp.Diagnostics.AddWarning("Custom parameters will be deleted",
		fmt.Sprintf("The OAuth2 client credential %q has %d custom parameter(s) that Terraform does not manage. "+
			"SAP deletes all custom parameters of a credential when it is updated or recreated through the API. "+
			"Cancel the apply and add them to custom_parameters (needs enable_unofficial), or set them again "+
			"in SAP's UI (Monitor > Integrations > Security Material) afterwards.", state.ID.ValueString(), len(params)))
}

// changedRootAttributes lists the top-level attributes whose planned value
// differs from the prior state.
func changedRootAttributes(state, plan tftypes.Value) path.Paths {
	var before, after map[string]tftypes.Value
	if state.As(&before) != nil || plan.As(&after) != nil {
		return path.Paths{path.Root("custom_parameters")}
	}
	var out path.Paths
	for name, v := range after {
		if !v.Equal(before[name]) {
			out = append(out, path.Root(name))
		}
	}
	return out
}

func (r *oauth2ClientCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	var clientSecret types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("client_secret_wo"), &clientSecret)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := client.CreateOAuth2ClientCredential(ctx, securitycontent.OAuth2ClientCredential{
		Name:                 plan.ID.ValueString(),
		Description:          plan.Description.ValueString(),
		TokenServiceURL:      plan.TokenServiceURL.ValueString(),
		ClientID:             plan.ClientID.ValueString(),
		Scope:                plan.Scope.ValueString(),
		ClientAuthentication: plan.ClientAuthentication.ValueString(),
		ScopeContentType:     plan.ScopeContentType.ValueString(),
		Resource:             plan.Resource.ValueString(),
		Audience:             plan.Audience.ValueString(),
		CustomParameters:     customParametersFromSet(ctx, plan.CustomParameters, &resp.Diagnostics),
	}, clientSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite OAuth2 client credential", diagnosticDetail(err))
		return
	}

	m := oauth2ClientCredentialToModel(created, plan.ClientSecretWOVersion)
	m.RuntimeLocationID = plan.RuntimeLocationID
	m.CustomParameters = plan.CustomParameters
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *oauth2ClientCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	cred, err := client.GetOAuth2ClientCredential(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite OAuth2 client credential", diagnosticDetail(err))
		return
	}

	m := oauth2ClientCredentialToModel(cred, state.ClientSecretWOVersion)
	m.RuntimeLocationID = state.RuntimeLocationID
	m.CustomParameters = types.SetNull(types.ObjectType{AttrTypes: customParameterAttrTypes})
	// Managed parameters, or after an import (marked by an empty set), are
	// read back; an empty result stays null, like a configuration without
	// the attribute.
	if !state.CustomParameters.IsNull() {
		params, err := client.ListOAuth2ClientCredentialCustomParameters(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read the custom parameters of SAP Integration Suite OAuth2 client credential", diagnosticDetail(err))
			return
		}
		if len(params) > 0 {
			m.CustomParameters = customParametersToSet(params, &resp.Diagnostics)
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func customParametersFromSet(ctx context.Context, set types.Set, diags *diag.Diagnostics) []securitycontent.CustomParameter {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	var items []struct {
		Key          types.String `tfsdk:"key"`
		Value        types.String `tfsdk:"value"`
		SendAsPartOf types.String `tfsdk:"send_as_part_of"`
	}
	diags.Append(set.ElementsAs(ctx, &items, false)...)
	out := make([]securitycontent.CustomParameter, 0, len(items))
	for _, i := range items {
		out = append(out, securitycontent.CustomParameter{Key: i.Key.ValueString(), Value: i.Value.ValueString(), SendAsPartOf: i.SendAsPartOf.ValueString()})
	}
	return out
}

func customParametersToSet(params []securitycontent.CustomParameter, diags *diag.Diagnostics) types.Set {
	elems := make([]attr.Value, 0, len(params))
	for _, p := range params {
		elems = append(elems, types.ObjectValueMust(customParameterAttrTypes, map[string]attr.Value{
			"key":             types.StringValue(p.Key),
			"value":           types.StringValue(p.Value),
			"send_as_part_of": types.StringValue(p.SendAsPartOf),
		}))
	}
	set, d := types.SetValue(types.ObjectType{AttrTypes: customParameterAttrTypes}, elems)
	diags.Append(d...)
	return set
}

func (r *oauth2ClientCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	var clientSecret types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("client_secret_wo"), &clientSecret)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.UpdateOAuth2ClientCredential(ctx, securitycontent.OAuth2ClientCredential{
		Name:                 plan.ID.ValueString(),
		Description:          plan.Description.ValueString(),
		TokenServiceURL:      plan.TokenServiceURL.ValueString(),
		ClientID:             plan.ClientID.ValueString(),
		Scope:                plan.Scope.ValueString(),
		ClientAuthentication: plan.ClientAuthentication.ValueString(),
		ScopeContentType:     plan.ScopeContentType.ValueString(),
		Resource:             plan.Resource.ValueString(),
		Audience:             plan.Audience.ValueString(),
	}, clientSecret.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite OAuth2 client credential", diagnosticDetail(err))
		return
	}

	cred, err := client.GetOAuth2ClientCredential(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back SAP Integration Suite OAuth2 client credential after update", diagnosticDetail(err))
		return
	}

	m := oauth2ClientCredentialToModel(cred, plan.ClientSecretWOVersion)
	m.RuntimeLocationID = plan.RuntimeLocationID
	m.CustomParameters = plan.CustomParameters
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *oauth2ClientCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oauth2ClientCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := client.DeleteOAuth2ClientCredential(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite OAuth2 client credential", diagnosticDetail(err))
	}
}

// ImportState only recovers id: client_secret_wo can never be recovered,
// and client_secret_wo_version is a practitioner-chosen marker with no
// server-side equivalent — see userCredentialResource.ImportState's doc
// comment, which applies identically here.
func (r *oauth2ClientCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRootID(), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
	// An empty set makes the next read fetch the custom parameters.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("custom_parameters"),
		types.SetValueMust(types.ObjectType{AttrTypes: customParameterAttrTypes}, nil))...)
}

func oauth2ClientCredentialToModel(cred *securitycontent.OAuth2ClientCredential, clientSecretWOVersion types.String) oauth2ClientCredentialModel {
	return oauth2ClientCredentialModel{
		ID:                    types.StringValue(cred.Name),
		Description:           stringOrNull(cred.Description),
		TokenServiceURL:       types.StringValue(cred.TokenServiceURL),
		ClientID:              types.StringValue(cred.ClientID),
		Scope:                 stringOrNull(cred.Scope),
		ClientAuthentication:  stringOrNull(cred.ClientAuthentication),
		ScopeContentType:      stringOrNull(cred.ScopeContentType),
		Resource:              stringOrNull(cred.Resource),
		Audience:              stringOrNull(cred.Audience),
		ClientSecretWO:        types.StringNull(),
		ClientSecretWOVersion: clientSecretWOVersion,
		CustomParameters:      types.SetNull(types.ObjectType{AttrTypes: customParameterAttrTypes}),
	}
}

// oauth2PassThroughAttribute builds an Optional+Computed attribute for a
// token-request setting SAP may fill with its own default. The prior state is
// kept when the configuration omits the attribute, so every PUT (which
// replaces the whole entity) resends the value SAP currently holds instead of
// silently clearing one set in the UI.
func oauth2PassThroughAttribute(description string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true,
		Computed: true,
		Description: description + " Passed through unchanged. Omitting the attribute keeps " +
			"whatever value SAP currently holds; it cannot be cleared from Terraform.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
}
