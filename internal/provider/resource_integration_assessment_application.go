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

const iaApplicationType = "sapintegrationsuite_integration_assessment_application"

// NewIntegrationAssessmentApplicationResource returns the resource for
// sapintegrationsuite_integration_assessment_application.
func NewIntegrationAssessmentApplicationResource() resource.Resource {
	return &iaApplicationResource{}
}

type iaApplicationResource struct {
	client *integrationassessment.Client
}

type iaApplicationModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	VendorID types.String `tfsdk:"vendor_id"`
}

func (r *iaApplicationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_application"
}

func (r *iaApplicationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "A business application in the Integration Assessment landscape, " +
			"optionally of a vendor. Application instances describe where it runs. Backed by the " +
			"Application entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id": iaIDAttribute("application"),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The application's name. Changes in place.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"vendor_id": schema.StringAttribute{
				Optional: true,
				Description: "Id of the vendor, for example from sapintegrationsuite_integration_assessment_vendor. " +
					"Setting, changing and removing it changes the application in place.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

func (r *iaApplicationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaApplicationType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func iaApplicationToModel(a *integrationassessment.Application) iaApplicationModel {
	return iaApplicationModel{ID: types.StringValue(a.ID), Name: types.StringValue(a.Name), VendorID: stringOrNull(a.VendorID())}
}

// read reads the application back with its vendor link; the create
// response carries the link only as a deferred URI.
func (r *iaApplicationResource) read(ctx context.Context, id string) (*iaApplicationModel, error) {
	a, err := r.client.GetApplication(ctx, id)
	if err != nil {
		return nil, err
	}
	m := iaApplicationToModel(a)
	return &m, nil
}

func (r *iaApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	a, err := r.client.CreateApplication(ctx, plan.Name.ValueString(), stringOrEmpty(plan.VendorID))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment application", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, a.ID)
	if err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, iaApplicationModel{ID: types.StringValue(a.ID), Name: plan.Name, VendorID: plan.VendorID})...)
		resp.Diagnostics.AddError("Integration Assessment application created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaApplicationModel
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
		resp.Diagnostics.AddError("Failed to read Integration Assessment application", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaApplicationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateApplication(ctx, id, plan.Name.ValueString(), stringOrEmpty(plan.VendorID)); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment application", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment application after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaApplicationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApplication(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment application", diagnosticDetail(err))
	}
}

func (r *iaApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
