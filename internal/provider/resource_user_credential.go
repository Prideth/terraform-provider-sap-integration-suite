package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// NewUserCredentialResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_user_credential.
func NewUserCredentialResource() resource.Resource {
	return &userCredentialResource{}
}

type userCredentialResource struct {
	client *securitycontent.Client
}

// userCredentialModel carries a PasswordWO field only because the
// framework's reflection-based Plan/State.Get requires every schema
// attribute to have a corresponding struct field to decode into — but since
// password_wo is WriteOnly, Terraform always supplies it as null there
// regardless of what the practitioner configured, and this code never reads
// PasswordWO's value from this struct. The real value is read once,
// directly out of Config (the only place Terraform actually sends it), in
// Create and Update; see the pathRoot("password_wo") lookups below.
// password_wo_version is the plain, stored companion attribute that makes
// password rotation a detectable, plannable change: unlike
// sapintegrationsuite_partner_user_credential_parameter, changing it here
// triggers an in-place Update (a redeploy), not a replacement, since SAP's
// Manage Security Material UI documents "Edit" as a supported action for a
// User Credentials artifact.
type userCredentialModel struct {
	ID                types.String `tfsdk:"id"`
	Kind              types.String `tfsdk:"kind"`
	Description       types.String `tfsdk:"description"`
	User              types.String `tfsdk:"user"`
	CompanyID         types.String `tfsdk:"company_id"`
	PasswordWO        types.String `tfsdk:"password_wo"`
	PasswordWOVersion types.String `tfsdk:"password_wo_version"`
	RuntimeLocationID types.String `tfsdk:"runtime_location_id"`
}

func (r *userCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_credential"
}

func (r *userCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Security Content \"User Credentials\" artifact: a username/password " +
			"credential integration flow adapters use for outbound basic or username-token " +
			"authentication. Backed by the public Security Content OData V2 API (UserCredentials). " +
			"The password is a write-only attribute: Terraform never stores it in plan or state, this " +
			"provider never requests or reads a password back from SAP, and SAP does not document " +
			"returning one. Requires Terraform CLI 1.11 or later for write-only attribute support. " +
			"See docs/guides/security-content.md for the full security analysis, including which " +
			"fields this provider could and could not confirm against SAP's documentation.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Required: true,
				Description: "The credential artifact's name. SAP's documentation states the name " +
					"is used as the alias integration flow adapters reference, so it is also this " +
					"resource's identifier and OData key. Immutable: SAP does not document renaming " +
					"a security material artifact, only deleting and recreating it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"kind": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(securitycontent.DefaultUserCredentialKind),
				Description: "The credential's system-specific type, as selected by SAP's \"Type\" " +
					"UI field: \"default\" (the default) for a generic Basic/username-token " +
					"credential, \"successfactors\" (needs company_id), or \"openconnectors\". SAP " +
					"accepts only these values, in lower case. Immutable: SAP's UI " +
					"does not document changing an artifact's kind via Edit, only via delete and " +
					"recreate, and this provider is conservative about a field that changes which " +
					"other fields (for example company_id) are meaningful.",
				Validators: []validator.String{userCredentialKindValidator{}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A free-text description of the credential artifact.",
			},
			"user": schema.StringAttribute{
				Required:    true,
				Description: "The username that authenticates to the receiver system.",
			},
			"company_id": schema.StringAttribute{
				Optional: true,
				Description: "The SuccessFactors company ID (client instance) this credential " +
					"connects to. Required when kind is \"successfactors\"; SAP's UI hides this " +
					"field for every other kind.",
			},
			"password_wo": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "The password. Write-only: Terraform never stores this value in plan " +
					"or state, and it is never read back from SAP. Required on every apply that " +
					"creates or redeploys this resource: SAP documents that editing a sibling " +
					"artifact type (OAuth2 Client Credentials) requires re-entering its secret on " +
					"every edit, and this provider assumes the same holds here, so the configured " +
					"value is resent whenever any other attribute changes too, not only when " +
					"password_wo_version changes.",
			},
			"password_wo_version": schema.StringAttribute{
				Required: true,
				Description: "An arbitrary value (for example a counter or timestamp) that a " +
					"practitioner changes to signal that password_wo's value has changed and the " +
					"credential should be rotated. Changing it redeploys the credential in place " +
					"(SAP's documented \"Edit\" action) rather than replacing the resource.",
			},
		},
	}
}

// ValidateConfig requires company_id for a SuccessFactors credential, which
// SAP otherwise refuses at apply (500 "Property 'CompanyId' must not be empty
// or null", tenant check of 2026-10-04).
func (r *userCredentialResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var kind, companyID types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("kind"), &kind)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("company_id"), &companyID)...)
	if resp.Diagnostics.HasError() || kind.ValueString() != "successfactors" || companyID.IsUnknown() {
		return
	}
	if companyID.ValueString() == "" {
		resp.Diagnostics.AddAttributeError(pathRoot("company_id"), "Missing company_id",
			"A credential with kind = \"successfactors\" needs the SuccessFactors company ID; SAP refuses it without one.")
	}
}

// userCredentialKindValidator accepts only the kinds SAP takes and names the
// right spelling for a value that differs only in letter case, such as
// "SuccessFactors", which earlier releases documented.
type userCredentialKindValidator struct{}

func (userCredentialKindValidator) Description(context.Context) string {
	return "must be one of " + strings.Join(securitycontent.UserCredentialKinds, ", ")
}

func (v userCredentialKindValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (userCredentialKindValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if containsString(securitycontent.UserCredentialKinds, value) {
		return
	}
	detail := fmt.Sprintf("SAP accepts only %s as kind, in lower case.", strings.Join(securitycontent.UserCredentialKinds, ", "))
	for _, k := range securitycontent.UserCredentialKinds {
		if strings.EqualFold(k, value) {
			detail = fmt.Sprintf("Use %q. ", k) + detail
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Unsupported user credential kind", fmt.Sprintf("%q is not a kind SAP accepts. ", value)+detail)
}

func (r *userCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *userCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	var password types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := client.CreateUserCredential(ctx, securitycontent.UserCredential{
		Name:        plan.ID.ValueString(),
		Kind:        plan.Kind.ValueString(),
		Description: plan.Description.ValueString(),
		User:        plan.User.ValueString(),
		CompanyID:   plan.CompanyID.ValueString(),
	}, password.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite user credential", diagnosticDetail(err))
		return
	}

	m := userCredentialToModel(created, plan.PasswordWOVersion)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *userCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	cred, err := client.GetUserCredential(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite user credential", diagnosticDetail(err))
		return
	}

	m := userCredentialToModel(cred, state.PasswordWOVersion)
	m.RuntimeLocationID = state.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *userCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	var password types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.UpdateUserCredential(ctx, securitycontent.UserCredential{
		Name:        plan.ID.ValueString(),
		Kind:        plan.Kind.ValueString(),
		Description: plan.Description.ValueString(),
		User:        plan.User.ValueString(),
		CompanyID:   plan.CompanyID.ValueString(),
	}, password.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite user credential", diagnosticDetail(err))
		return
	}

	cred, err := client.GetUserCredential(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back SAP Integration Suite user credential after update", diagnosticDetail(err))
		return
	}

	m := userCredentialToModel(cred, plan.PasswordWOVersion)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *userCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := client.DeleteUserCredential(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite user credential", diagnosticDetail(err))
	}
}

// ImportState only recovers id (the credential's name): password_wo can
// never be recovered (SAP does not return it, and it is write-only in this
// provider's own model even if it did), and password_wo_version is a
// practitioner-chosen marker with no server-side equivalent to read back. A
// configuration applied right after import must supply both, which will
// plan as an update to password_wo_version even though nothing server-side
// actually changes — an inherent, documented limitation of importing a
// write-only-secret resource, not a bug. Until the practitioner supplies a
// deliberate password_wo_version, treat the imported credential's password
// as owned by whatever process created it outside Terraform.
func (r *userCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRootID(), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
}

func userCredentialToModel(cred *securitycontent.UserCredential, passwordWOVersion types.String) userCredentialModel {
	return userCredentialModel{
		ID:                types.StringValue(cred.Name),
		Kind:              stringOrNull(cred.Kind),
		Description:       stringOrNull(cred.Description),
		User:              types.StringValue(cred.User),
		CompanyID:         stringOrNull(cred.CompanyID),
		PasswordWO:        types.StringNull(),
		PasswordWOVersion: passwordWOVersion,
	}
}
