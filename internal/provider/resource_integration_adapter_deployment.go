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

// NewIntegrationAdapterDeploymentResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_integration_adapter_deployment.
func NewIntegrationAdapterDeploymentResource() resource.Resource {
	return &integrationAdapterDeploymentResource{}
}

type integrationAdapterDeploymentResource struct {
	client *cloudintegration.Client
}

type integrationAdapterDeploymentModel struct {
	ID                types.String   `tfsdk:"id"`
	AdapterID         types.String   `tfsdk:"adapter_id"`
	Status            types.String   `tfsdk:"status"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
	RuntimeLocationID types.String   `tfsdk:"runtime_location_id"`
}

func (r *integrationAdapterDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_adapter_deployment"
}

func (r *integrationAdapterDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Expresses the desired runtime deployment state of a custom Integration " +
			"Adapter, independent of its design-time content lifecycle. SAP documents that an " +
			"adapter must be deployed before an integration flow consuming it is deployed; this " +
			"provider does not infer that ordering automatically — depend_on it explicitly from " +
			"the consuming sapintegrationsuite_integration_flow_deployment. Unlike every other " +
			"*_deployment resource in this provider, there is no *_version attribute: SAP's " +
			"confirmed DeployIntegrationAdapterDesigntimeArtifact action takes only an Id query " +
			"parameter, with no Version parameter, consistent with the adapter design-time entity " +
			"having no confirmed composite (Id, Version) key the way integration flows, value " +
			"mappings, message mappings, and script collections all do — see " +
			"docs/sap-api-references.md. Deploying is asynchronous: this resource polls a runtime " +
			"artifact status until the deployment reaches a terminal state or the configured " +
			"timeout elapses, reusing the same shared runtime-artifacts polling every other " +
			"deployment resource in this provider uses; this project could not independently " +
			"confirm that a deployed custom adapter surfaces through that same shared entity as " +
			"opposed to an adapter-specific status mechanism — see " +
			"docs/guides/integration-adapters.md for what is and is not confirmed here.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Equal to adapter_id.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"adapter_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration adapter to deploy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The runtime status SAP reports for this deployment (for example STARTED or ERROR).",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(context.Background(), timeouts.Opts{
				Create: true,
				Delete: true,
			}),
		},
	}
}

func (r *integrationAdapterDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *integrationAdapterDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationAdapterDeploymentModel
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

	if err := client.DeployIntegrationAdapter(ctx, plan.AdapterID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to deploy SAP Integration Suite integration adapter", diagnosticDetail(err))
		return
	}

	artifact, err := waitForRuntimeArtifact(ctx, client, plan.AdapterID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err)+deploymentTaintedNote)
		artifact = unconfirmedDeployment(ctx, client, plan.AdapterID.ValueString(), "")
	}

	m := integrationAdapterDeploymentToModel(artifact, plan.Timeouts)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *integrationAdapterDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationAdapterDeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	artifact, err := client.GetRuntimeArtifact(ctx, state.AdapterID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite integration adapter deployment", diagnosticDetail(err))
		return
	}

	m := integrationAdapterDeploymentToModel(artifact, state.Timeouts)
	m.RuntimeLocationID = state.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

// Update is unreachable in practice: adapter_id is the only non-Computed
// attribute besides timeouts, and it is RequiresReplace — there is nothing
// to redeploy in place, since a new adapter_id always means a different
// design-time artifact (see sapintegrationsuite_integration_adapter, whose
// id is also RequiresReplace).
func (r *integrationAdapterDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"sapintegrationsuite_integration_adapter_deployment does not support in-place updates; "+
			"Terraform should have replaced this resource instead of updating it.",
	)
}

func (r *integrationAdapterDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationAdapterDeploymentModel
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

	err := client.UndeployRuntimeArtifact(ctx, state.AdapterID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to undeploy SAP Integration Suite integration adapter", diagnosticDetail(err))
	}
}

func (r *integrationAdapterDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("adapter_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRootID(), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
}

func integrationAdapterDeploymentToModel(artifact *cloudintegration.RuntimeArtifact, tf timeouts.Value) integrationAdapterDeploymentModel {
	return integrationAdapterDeploymentModel{
		ID:        types.StringValue(artifact.ID),
		AdapterID: types.StringValue(artifact.ID),
		Status:    types.StringValue(artifact.Status),
		Timeouts:  tf,
	}
}
