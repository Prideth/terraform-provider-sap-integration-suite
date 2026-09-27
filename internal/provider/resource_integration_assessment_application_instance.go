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

const iaApplicationInstanceType = "sapintegrationsuite_integration_assessment_application_instance"

// NewIntegrationAssessmentApplicationInstanceResource returns the resource
// for sapintegrationsuite_integration_assessment_application_instance.
func NewIntegrationAssessmentApplicationInstanceResource() resource.Resource {
	return &iaApplicationInstanceResource{}
}

type iaApplicationInstanceResource struct {
	client *integrationassessment.Client
}

type iaApplicationInstanceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ApplicationID     types.String `tfsdk:"application_id"`
	DeploymentModelID types.String `tfsdk:"deployment_model_id"`
}

func (r *iaApplicationInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_application_instance"
}

func (r *iaApplicationInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "One installation of an application in the Integration Assessment " +
			"landscape, for example the production system, with the deployment model it runs on. Backed " +
			"by the ApplicationInstance entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id": iaIDAttribute("application instance"),
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The instance's name. Changes in place.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "A description of the instance. Changes in place.",
			},
			"application_id": iaLinkAttribute("Id of the application this is an instance of, for example from " +
				"sapintegrationsuite_integration_assessment_application. Changing it creates a new instance: " +
				"moving an instance to another application in place was not tested."),
			"deployment_model_id": iaLinkAttribute("Id of the deployment model, for example from the " +
				"sapintegrationsuite_integration_assessment_deployment_model data source. Changing it creates a " +
				"new instance, for the same reason."),
		},
	}
}

func (r *iaApplicationInstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaApplicationInstanceType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func (r *iaApplicationInstanceResource) read(ctx context.Context, id string) (*iaApplicationInstanceModel, error) {
	i, err := r.client.GetApplicationInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	return &iaApplicationInstanceModel{
		ID:                types.StringValue(i.ID),
		Name:              types.StringValue(i.Name),
		Description:       stringPtrOrNull(i.Description),
		ApplicationID:     stringOrNull(i.ApplicationID()),
		DeploymentModelID: stringOrNull(i.DeploymentModelID()),
	}, nil
}

func (r *iaApplicationInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaApplicationInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	i, err := r.client.CreateApplicationInstance(ctx, plan.Name.ValueString(), optionalString(plan.Description),
		plan.ApplicationID.ValueString(), plan.DeploymentModelID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment application instance", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, i.ID)
	if err != nil {
		plan.ID = types.StringValue(i.ID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Integration Assessment application instance created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaApplicationInstanceModel
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
		resp.Diagnostics.AddError("Failed to read Integration Assessment application instance", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaApplicationInstanceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateApplicationInstance(ctx, id, plan.Name.ValueString(), optionalString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment application instance", diagnosticDetail(err))
		return
	}
	m, err := r.read(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment application instance after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *iaApplicationInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaApplicationInstanceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteApplicationInstance(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment application instance", diagnosticDetail(err))
	}
}

func (r *iaApplicationInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
