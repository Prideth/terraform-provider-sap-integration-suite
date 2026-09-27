package provider

import (
	"context"
	"errors"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// NewAccessPolicyResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_access_policy.
func NewAccessPolicyResource() resource.Resource {
	return &accessPolicyResource{}
}

type accessPolicyResource struct {
	client *cloudintegration.Client
	// allowUnofficial is the provider's enable_unofficial: SAP's tooling
	// creates, reads and deletes access policies, but does not update them.
	allowUnofficial bool
}

type accessPolicyModel struct {
	ID          types.String `tfsdk:"id"`
	RoleName    types.String `tfsdk:"role_name"`
	Description types.String `tfsdk:"description"`
}

func (r *accessPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy"
}

func (r *accessPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An SAP Integration Suite access policy: a named guard, tied to a BTP role, that " +
			"restricts who can work with the artifacts its references match. The policy itself only " +
			"carries the role name and a description; the matching rules live in separate " +
			"sapintegrationsuite_access_policy_reference resources. Backed by the AccessPolicies " +
			"entity of the Security Content OData V2 API.\n\n" +
			"SAP's tooling creates, reads and deletes access policies. Changing the description in " +
			"place uses PATCH, which works on a tenant but is unofficial, so it needs " +
			"enable_unofficial = true in the provider block.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				Description: "Numeric ID SAP assigns to the policy. It differs between tenants, so use " +
					"role_name when you need a portable identifier.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"role_name": schema.StringAttribute{
				Required: true,
				Description: "Role name the policy is associated with. Users only get access to the " +
					"protected artifacts when a BTP custom role carries exactly this string in its " +
					"Values attribute. Unique per tenant. Changing it replaces the policy, because " +
					"SAP does not document renaming a policy in place.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Free-text description shown next to the policy in the Access Policies screen.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
		},
	}
}

func (r *accessPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
	r.allowUnofficial = data.EnableUnofficial
}

func (r *accessPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAccessPolicy(ctx, cloudintegration.AccessPolicy{
		RoleName:    plan.RoleName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite access policy", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, accessPolicyToModel(created))...)
}

func (r *accessPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, err := r.client.GetAccessPolicy(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite access policy", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, accessPolicyToModel(policy))...)
}

// ModifyPlan stops a description change without enable_unofficial: the
// PATCH that applies it works on a tenant but is not documented. role_name
// replaces the policy, so the description is the only in-place change.
func (r *accessPolicyResource) ModifyPlan(_ context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return // provider not configured yet; Update checks again
	}
	if isInPlaceUpdate(req, resp) {
		requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_access_policy", "description update (PATCH)", &resp.Diagnostics)
	}
}

func (r *accessPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_access_policy", "description update (PATCH)", &resp.Diagnostics) {
		return
	}
	var plan accessPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.UpdateAccessPolicy(ctx, plan.ID.ValueString(), cloudintegration.AccessPolicy{
		RoleName:    plan.RoleName.ValueString(),
		Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite access policy", diagnosticDetail(err))
		return
	}

	policy, err := r.client.GetAccessPolicy(ctx, plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read back SAP Integration Suite access policy after update", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, accessPolicyToModel(policy))...)
}

func (r *accessPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAccessPolicy(ctx, state.ID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite access policy", diagnosticDetail(err))
	}
}

func (r *accessPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if _, err := strconv.ParseInt(req.ID, 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected the numeric access policy ID SAP assigned (for example \"1901\"), got "+strconv.Quote(req.ID)+". "+
				"Look it up with the sapintegrationsuite_access_policy data source by role_name.")
		return
	}
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}

func accessPolicyToModel(policy *cloudintegration.AccessPolicy) accessPolicyModel {
	return accessPolicyModel{
		ID:          types.StringValue(policy.ID),
		RoleName:    types.StringValue(policy.RoleName),
		Description: stringOrNull(policy.Description),
	}
}
