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

const iaTechnologyType = "sapintegrationsuite_integration_assessment_technology"

// NewIntegrationAssessmentTechnologyResource returns the resource for
// sapintegrationsuite_integration_assessment_technology.
func NewIntegrationAssessmentTechnologyResource() resource.Resource {
	return &iaTechnologyResource{}
}

type iaTechnologyResource struct {
	client *integrationassessment.Client
}

type iaTechnologyModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	VendorID types.String `tfsdk:"vendor_id"`
}

func (r *iaTechnologyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_technology"
}

func (r *iaTechnologyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "An integration technology of your own in the Integration Assessment " +
			"landscape, for example a third-party middleware, of a vendor. SAP delivers its own " +
			"technologies; look them up with the sapintegrationsuite_integration_assessment_technology data " +
			"source instead of managing them. Backed by the Technology entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id": iaIDAttribute("technology"),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The technology's name. Changes in place.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"vendor_id": iaLinkAttribute("Id of the vendor, for example from " +
				"sapintegrationsuite_integration_assessment_vendor. Changing it creates a new technology: " +
				"changing the vendor of a technology in place was not tested."),
		},
	}
}

func (r *iaTechnologyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaTechnologyType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func (r *iaTechnologyResource) read(ctx context.Context, id string) (*iaTechnologyModel, error) {
	t, err := r.client.GetTechnology(ctx, id)
	if err != nil {
		return nil, err
	}
	return &iaTechnologyModel{ID: types.StringValue(t.ID), Name: types.StringValue(t.Name), VendorID: stringOrNull(t.VendorID())}, nil
}

func (r *iaTechnologyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaTechnologyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	t, err := r.client.CreateTechnology(ctx, plan.Name.ValueString(), plan.VendorID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment technology", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, t.ID)
	if err != nil {
		plan.ID = types.StringValue(t.ID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Integration Assessment technology created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaTechnologyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := r.read(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment technology", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaTechnologyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateTechnology(ctx, id, plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment technology", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment technology after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaTechnologyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTechnology(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment technology", diagnosticDetail(err))
	}
}

func (r *iaTechnologyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
