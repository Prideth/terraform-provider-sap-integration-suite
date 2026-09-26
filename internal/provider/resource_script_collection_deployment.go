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

// NewScriptCollectionDeploymentResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_script_collection_deployment.
func NewScriptCollectionDeploymentResource() resource.Resource {
	return &scriptCollectionDeploymentResource{}
}

type scriptCollectionDeploymentResource struct {
	client *cloudintegration.Client
}

type scriptCollectionDeploymentModel struct {
	ID                      types.String   `tfsdk:"id"`
	PackageID               types.String   `tfsdk:"package_id"`
	ScriptCollectionID      types.String   `tfsdk:"script_collection_id"`
	ScriptCollectionVersion types.String   `tfsdk:"script_collection_version"`
	Status                  types.String   `tfsdk:"status"`
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
	RedeployTriggers        types.Map      `tfsdk:"redeploy_triggers"`
	RuntimeLocationID       types.String   `tfsdk:"runtime_location_id"`
}

func (r *scriptCollectionDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_collection_deployment"
}

func (r *scriptCollectionDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Expresses the desired runtime deployment state of a Cloud Integration " +
			"script collection, independent of its design-time content lifecycle. Deploying is " +
			"asynchronous: this resource polls SAP's runtime artifact status until the deployment " +
			"reaches a terminal state (STARTED or ERROR) or the configured timeout elapses. This " +
			"resource is never created implicitly by deploying an integration flow that " +
			"references the collection.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<script_collection_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package the script collection belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"script_collection_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the script collection to deploy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"script_collection_version": schema.StringAttribute{
				Required: true,
				Description: "The design-time version to deploy, typically " +
					"sapintegrationsuite_script_collection.<name>.version. Changing it redeploys " +
					"the script collection. After refresh this also reflects whatever version SAP " +
					"reports as actually deployed, so a redeploy performed outside Terraform (or a " +
					"failed/stale deployment) shows up as drift on the next plan.",
			},
			"redeploy_triggers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Arbitrary values that redeploy the script collection in place whenever they change, for " +
					"example its content_hash. Uploading new content does not change the script collection's " +
					"version (tenant test, September 2026), so without this or a new script_collection_version the " +
					"runtime keeps the previously deployed content.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "The runtime status SAP reports for this deployment (for example STARTED or ERROR).",
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

func (r *scriptCollectionDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scriptCollectionDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scriptCollectionDeploymentModel
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

	if err := client.DeployScriptCollection(ctx, plan.ScriptCollectionID.ValueString(), plan.ScriptCollectionVersion.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to deploy SAP Integration Suite script collection", diagnosticDetail(err))
		return
	}

	artifact, err := waitForRuntimeArtifact(ctx, client, plan.ScriptCollectionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err)+deploymentTaintedNote)
		artifact = unconfirmedDeployment(ctx, client, plan.ScriptCollectionID.ValueString(), plan.ScriptCollectionVersion.ValueString())
	}

	m := scriptCollectionDeploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts)
	m.RedeployTriggers = plan.RedeployTriggers
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *scriptCollectionDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scriptCollectionDeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	artifact, err := client.GetRuntimeArtifact(ctx, state.ScriptCollectionID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite script collection deployment", diagnosticDetail(err))
		return
	}

	// artifact.Version reflects whatever SAP actually has deployed right now,
	// which may differ from state.ScriptCollectionVersion if someone
	// redeployed a different version outside Terraform, or if a deployment
	// failed part-way. Writing it into script_collection_version (a
	// Required, non-Computed attribute) is what makes that visible as
	// drift on the next plan.
	m := scriptCollectionDeploymentToModel(state.PackageID.ValueString(), artifact, state.Timeouts)
	m.RedeployTriggers = state.RedeployTriggers
	m.RuntimeLocationID = state.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *scriptCollectionDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scriptCollectionDeploymentModel
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

	if err := client.DeployScriptCollection(ctx, plan.ScriptCollectionID.ValueString(), plan.ScriptCollectionVersion.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to redeploy SAP Integration Suite script collection", diagnosticDetail(err))
		return
	}

	artifact, err := waitForRuntimeArtifact(ctx, client, plan.ScriptCollectionID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err))
		return
	}

	m := scriptCollectionDeploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts)
	m.RedeployTriggers = plan.RedeployTriggers
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *scriptCollectionDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scriptCollectionDeploymentModel
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

	err := client.UndeployRuntimeArtifact(ctx, state.ScriptCollectionID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to undeploy SAP Integration Suite script collection", diagnosticDetail(err))
	}
}

func (r *scriptCollectionDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	packageID, scriptCollectionID := parts[0], parts[1]
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("script_collection_id"), scriptCollectionID)...)
}

func scriptCollectionDeploymentToModel(packageID string, artifact *cloudintegration.RuntimeArtifact, tf timeouts.Value) scriptCollectionDeploymentModel {
	return scriptCollectionDeploymentModel{
		ID:                      types.StringValue(packageID + "/" + artifact.ID),
		PackageID:               types.StringValue(packageID),
		ScriptCollectionID:      types.StringValue(artifact.ID),
		ScriptCollectionVersion: types.StringValue(artifact.Version),
		Status:                  types.StringValue(artifact.Status),
		Timeouts:                tf,
	}
}
