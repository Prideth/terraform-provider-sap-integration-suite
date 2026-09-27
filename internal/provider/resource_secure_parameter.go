package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// NewSecureParameterResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_secure_parameter.
func NewSecureParameterResource() resource.Resource {
	return &secureParameterResource{}
}

type secureParameterResource struct {
	client *securitycontent.Client
}

// secureParameterModel carries SecureParamWO only because the framework
// needs a field for every attribute; Terraform always passes a write-only
// attribute as null in plan and state, so the value is read from Config in
// Create and Update.
type secureParameterModel struct {
	ID                   types.String `tfsdk:"id"`
	Description          types.String `tfsdk:"description"`
	SecureParamWO        types.String `tfsdk:"secure_param_wo"`
	SecureParamWOVersion types.String `tfsdk:"secure_param_wo_version"`
	Status               types.String `tfsdk:"status"`
	DeployedBy           types.String `tfsdk:"deployed_by"`
	DeployedOn           types.String `tfsdk:"deployed_on"`
}

func (r *secureParameterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secure_parameter"
}

func (r *secureParameterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "UNOFFICIAL: needs enable_unofficial = true in the provider block. It works and was " +
			"verified on a tenant, but SAP does not document the " +
			"API behind it, so SAP may change it without notice.\n\n" +
			"Manages a Security Content \"Secure Parameter\" artifact: a confidential value, " +
			"for example for a custom adapter or a script, deployed under an alias that integration " +
			"flows reference. Backed by the SecureParameters entity set of the Security Content OData " +
			"V2 API. SAP Help documents the artifact only in the Monitor UI, and SecureParameters is " +
			"not among the API resources SAP lists; the entity set comes from the service's $metadata. " +
			"Create, read, update and delete were verified on a tenant in September 2026. The value is a write-only attribute: " +
			"Terraform never stores it, and SAP returns it as null. Requires Terraform CLI 1.11 or later.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required: true,
				Description: "The artifact's name, which integration flows use as the alias for the " +
					"value. SAP's OData key, so changing it replaces the secure parameter. At most 150 " +
					"characters.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, securitycontent.MaxSecureParameterNameLength),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A free-text description of the artifact, at most 1024 characters.",
				Validators:  []validator.String{stringvalidator.LengthBetween(1, 1024)},
			},
			"secure_param_wo": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "The confidential value, at most 4096 characters. Write-only: Terraform " +
					"never stores it in plan or state, and SAP never returns it. It is sent on create " +
					"and on every update, because an update replaces the artifact and the Monitor UI " +
					"requires re-entering the value on every edit.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, securitycontent.MaxSecureParameterValueLength),
				},
			},
			"secure_param_wo_version": schema.StringAttribute{
				Required: true,
				Description: "An arbitrary marker (for example a counter or a date) that you change " +
					"whenever secure_param_wo changes. Terraform cannot see the write-only value, so " +
					"changing this marker is what triggers the update that sends the new value.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Deployment status SAP reports, for example \"DEPLOYED\".",
			},
			"deployed_by": schema.StringAttribute{
				Computed:    true,
				Description: "User or client that last deployed the artifact.",
			},
			"deployed_on": schema.StringAttribute{
				Computed:    true,
				Description: "When the artifact was last deployed, RFC 3339 in UTC.",
			},
		},
	}
}

func (r *secureParameterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, "sapintegrationsuite_secure_parameter", &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
}

func secureParameterToModel(sp *securitycontent.SecureParameter, version types.String) secureParameterModel {
	return secureParameterModel{
		ID:                   types.StringValue(sp.Name),
		Description:          stringOrNull(sp.Description),
		SecureParamWO:        types.StringNull(),
		SecureParamWOVersion: version,
		Status:               stringOrNull(sp.Status),
		DeployedBy:           stringOrNull(sp.DeployedBy),
		DeployedOn:           odataDateToRFC3339(sp.DeployedOn),
	}
}

func (r *secureParameterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan secureParameterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var value types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("secure_param_wo"), &value)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.ID.ValueString()

	// SAP does not document what a create does for a name that exists. Stop
	// rather than risk replacing a secret this configuration does not own.
	if _, err := r.client.GetSecureParameter(ctx, name); err == nil {
		resp.Diagnostics.AddError(
			"Secure parameter already exists",
			"A secure parameter named "+name+" already exists on the tenant. Import it with "+
				"terraform import, then apply to set the configured value.",
		)
		return
	} else if !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to check for an existing SAP Integration Suite secure parameter", diagnosticDetail(err))
		return
	}

	created, err := r.client.CreateSecureParameter(ctx, name, plan.Description.ValueString(), value.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite secure parameter", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, secureParameterToModel(created, plan.SecureParamWOVersion))...)
}

func (r *secureParameterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state secureParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, state.ID.ValueString(), state.SecureParamWOVersion, &resp.State, &resp.Diagnostics)
}

func (r *secureParameterResource) readInto(ctx context.Context, name string, version types.String, state *tfsdk.State, diags *diag.Diagnostics) {
	sp, err := r.client.GetSecureParameter(ctx, name)
	if err != nil {
		if isNotFound(err) {
			state.RemoveResource(ctx)
			return
		}
		diags.AddError("Failed to read SAP Integration Suite secure parameter", diagnosticDetail(err))
		return
	}
	diags.Append(state.Set(ctx, secureParameterToModel(sp, version))...)
}

func (r *secureParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan secureParameterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var value types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("secure_param_wo"), &value)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.ID.ValueString()
	if err := r.client.UpdateSecureParameter(ctx, name, plan.Description.ValueString(), value.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite secure parameter", diagnosticDetail(err))
		return
	}
	r.readInto(ctx, name, plan.SecureParamWOVersion, &resp.State, &resp.Diagnostics)
}

func (r *secureParameterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state secureParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteSecureParameter(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite secure parameter", diagnosticDetail(err))
	}
}

// ImportState imports by name. The value cannot be recovered, so the first
// apply after an import plans an update that sends the configured value.
func (r *secureParameterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
