package provider

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// NewIntegrationFlowDeploymentResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_integration_flow_deployment.
func NewIntegrationFlowDeploymentResource() resource.Resource {
	return &integrationFlowDeploymentResource{}
}

type integrationFlowDeploymentResource struct {
	client *cloudintegration.Client
}

type integrationFlowDeploymentModel struct {
	ID                types.String   `tfsdk:"id"`
	PackageID         types.String   `tfsdk:"package_id"`
	FlowID            types.String   `tfsdk:"flow_id"`
	FlowVersion       types.String   `tfsdk:"flow_version"`
	Status            types.String   `tfsdk:"status"`
	RedeployTriggers  types.Map      `tfsdk:"redeploy_triggers"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
	RuntimeLocationID types.String   `tfsdk:"runtime_location_id"`
}

func (r *integrationFlowDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_flow_deployment"
}

func (r *integrationFlowDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Expresses the desired runtime deployment state of a Cloud Integration " +
			"integration flow, independent of its design-time content lifecycle. Deploying is " +
			"asynchronous: this resource polls SAP's runtime artifact status until the deployment " +
			"reaches a terminal state (STARTED or ERROR) or the configured timeout elapses. It also " +
			"follows the task SAP returns for the deploy (BuildAndDeployStatus): if the task fails, or " +
			"succeeds while the flow does not appear in the Cloud Integration runtime, it stops at once. " +
			"The second case means the flow went to another runtime profile: a flow whose externalized " +
			"parameter SAP_ProfileId is \"integrationcell\" deploys to Integration Cell, which this " +
			"provider does not support; set it to \"iflmap\" with " +
			"sapintegrationsuite_integration_flow_configuration.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<flow_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package the flow belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flow_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration flow to deploy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flow_version": schema.StringAttribute{
				Required: true,
				Description: "The design-time version to deploy, typically " +
					"sapintegrationsuite_integration_flow.<name>.version. Changing it redeploys the " +
					"flow. After refresh this also reflects whatever version SAP reports as actually " +
					"deployed, so a redeploy performed outside Terraform (or a failed/stale " +
					"deployment) shows up as drift on the next plan.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The runtime status SAP reports for this deployment (for example STARTED or ERROR).",
			},
			"redeploy_triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary values that redeploy the flow in place whenever they change, for " +
					"example the parameters of a sapintegrationsuite_integration_flow_configuration. " +
					"Changed externalized parameters only take effect at runtime after a redeploy, and " +
					"flow_version does not change when only parameters change.",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Create: true,
				Update: true,
				Delete: true,
			}),
		},
	}
}

func (r *integrationFlowDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
}

func (r *integrationFlowDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationFlowDeploymentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	taskID, err := client.DeployIntegrationFlow(ctx, plan.FlowID.ValueString(), plan.FlowVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to deploy SAP Integration Suite integration flow", diagnosticDetail(err))
		return
	}

	artifact, err := waitForDeployment(ctx, client, plan.FlowID.ValueString(), taskID)
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err)+deploymentTaintedNote)
		artifact = unconfirmedDeployment(ctx, client, plan.FlowID.ValueString(), plan.FlowVersion.ValueString())
	}

	m := deploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts, plan.RedeployTriggers)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *integrationFlowDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationFlowDeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	artifact, err := client.GetRuntimeArtifact(ctx, state.FlowID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite integration flow deployment", diagnosticDetail(err))
		return
	}

	// artifact.Version reflects whatever SAP actually has deployed right now,
	// which may differ from state.FlowVersion if someone redeployed a
	// different version outside Terraform, or if a deployment failed
	// part-way. Writing it into flow_version (a Required, non-Computed
	// attribute) is what makes that visible as drift on the next plan.
	m := deploymentToModel(state.PackageID.ValueString(), artifact, state.Timeouts, state.RedeployTriggers)
	m.RuntimeLocationID = state.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *integrationFlowDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan integrationFlowDeploymentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	taskID, err := client.DeployIntegrationFlow(ctx, plan.FlowID.ValueString(), plan.FlowVersion.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to redeploy SAP Integration Suite integration flow", diagnosticDetail(err))
		return
	}

	artifact, err := waitForDeployment(ctx, client, plan.FlowID.ValueString(), taskID)
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err))
		return
	}

	m := deploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts, plan.RedeployTriggers)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *integrationFlowDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationFlowDeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	timeout, diags := state.Timeouts.Delete(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	err := client.UndeployRuntimeArtifact(ctx, state.FlowID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to undeploy SAP Integration Suite integration flow", diagnosticDetail(err))
	}
}

func (r *integrationFlowDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	packageID, flowID := parts[0], parts[1]
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("flow_id"), flowID)...)
}

func deploymentToModel(packageID string, artifact *cloudintegration.RuntimeArtifact, tf timeouts.Value, triggers types.Map) integrationFlowDeploymentModel {
	return integrationFlowDeploymentModel{
		ID:               types.StringValue(packageID + "/" + artifact.ID),
		PackageID:        types.StringValue(packageID),
		FlowID:           types.StringValue(artifact.ID),
		FlowVersion:      types.StringValue(artifact.Version),
		Status:           types.StringValue(artifact.Status),
		Timeouts:         tf,
		RedeployTriggers: triggers,
	}
}
