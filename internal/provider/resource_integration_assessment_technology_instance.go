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

const iaTechnologyInstanceType = "sapintegrationsuite_integration_assessment_technology_instance"

// NewIntegrationAssessmentTechnologyInstanceResource returns the resource
// for sapintegrationsuite_integration_assessment_technology_instance.
func NewIntegrationAssessmentTechnologyInstanceResource() resource.Resource {
	return &iaTechnologyInstanceResource{}
}

type iaTechnologyInstanceResource struct {
	client *integrationassessment.Client
}

type iaTechnologyInstanceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	TechnologyID      types.String `tfsdk:"technology_id"`
	DeploymentModelID types.String `tfsdk:"deployment_model_id"`
}

func (r *iaTechnologyInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_technology_instance"
}

func (r *iaTechnologyInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "One installation of a technology in the Integration Assessment " +
			"landscape, SAP's or your own, with the deployment model it runs on. Assessments recommend " +
			"technology instances for interfaces. Backed by the TechnologyInstance entity set of the " +
			"Entities API.",
		Attributes: map[string]schema.Attribute{
			"id": iaIDAttribute("technology instance"),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The instance's name. Changes in place.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"technology_id": iaReplacingLinkAttribute("Id of the technology, from sapintegrationsuite_integration_assessment_technology " +
				"or, for one of SAP's technologies, from the data source of the same name. Changing it creates a new " +
				"instance: moving an instance to another technology in place was not tested."),
			"deployment_model_id": iaLinkAttribute("Id of the deployment model, for example from the " +
				"sapintegrationsuite_integration_assessment_deployment_model data source. Changes in place."),
		},
	}
}

func (r *iaTechnologyInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaTechnologyInstanceType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func (r *iaTechnologyInstanceResource) read(ctx context.Context, id string) (*iaTechnologyInstanceModel, error) {
	i, err := r.client.GetTechnologyInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	return &iaTechnologyInstanceModel{
		ID:                types.StringValue(i.ID),
		Name:              types.StringValue(i.Name),
		TechnologyID:      stringOrNull(i.TechnologyID()),
		DeploymentModelID: stringOrNull(i.DeploymentModelID()),
	}, nil
}

func (r *iaTechnologyInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaTechnologyInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	i, err := r.client.CreateTechnologyInstance(ctx, plan.Name.ValueString(), plan.TechnologyID.ValueString(), plan.DeploymentModelID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment technology instance", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, i.ID)
	if err != nil {
		plan.ID = types.StringValue(i.ID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Integration Assessment technology instance created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaTechnologyInstanceModel
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
		resp.Diagnostics.AddError("Failed to read Integration Assessment technology instance", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaTechnologyInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateTechnologyInstance(ctx, id, plan.Name.ValueString(), plan.DeploymentModelID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment technology instance", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment technology instance after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaTechnologyInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaTechnologyInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteTechnologyInstance(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment technology instance", diagnosticDetail(err))
	}
}

func (r *iaTechnologyInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
