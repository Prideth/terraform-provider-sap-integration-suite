package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

const iaVendorType = "sapintegrationsuite_integration_assessment_vendor"

// NewIntegrationAssessmentVendorResource returns the resource for
// sapintegrationsuite_integration_assessment_vendor.
func NewIntegrationAssessmentVendorResource() resource.Resource {
	return &iaVendorResource{}
}

type iaVendorResource struct {
	client *integrationassessment.Client
}

type iaVendorModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

func (r *iaVendorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_vendor"
}

func (r *iaVendorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "A vendor in the Integration Assessment landscape: the supplier of " +
			"applications and technologies. Backed by the Vendor entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id": iaIDAttribute("vendor"),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The vendor's name. Changes in place.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

func (r *iaVendorResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaVendorType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func iaVendorToModel(v *integrationassessment.Vendor) iaVendorModel {
	return iaVendorModel{ID: types.StringValue(v.ID), Name: types.StringValue(v.Name)}
}

func (r *iaVendorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaVendorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, err := r.client.CreateVendor(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment vendor", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaVendorToModel(v))...)
}

func (r *iaVendorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaVendorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, err := r.client.GetVendor(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment vendor", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaVendorToModel(v))...)
}

func (r *iaVendorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaVendorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateVendor(ctx, id, plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment vendor", diagnosticDetail(err))
		return
	}
	v, err := r.client.GetVendor(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment vendor after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaVendorToModel(v))...)
}

func (r *iaVendorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaVendorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteVendor(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment vendor", diagnosticDetail(err))
	}
}

func (r *iaVendorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
