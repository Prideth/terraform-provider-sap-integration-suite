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

// NewMessageMappingDeploymentResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_message_mapping_deployment.
func NewMessageMappingDeploymentResource() resource.Resource {
	return &messageMappingDeploymentResource{}
}

type messageMappingDeploymentResource struct {
	client *cloudintegration.Client
}

type messageMappingDeploymentModel struct {
	ID                types.String   `tfsdk:"id"`
	PackageID         types.String   `tfsdk:"package_id"`
	MappingID         types.String   `tfsdk:"mapping_id"`
	MappingVersion    types.String   `tfsdk:"mapping_version"`
	Status            types.String   `tfsdk:"status"`
	Timeouts          timeouts.Value `tfsdk:"timeouts"`
	RuntimeLocationID types.String   `tfsdk:"runtime_location_id"`
}

func (r *messageMappingDeploymentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_mapping_deployment"
}

func (r *messageMappingDeploymentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Expresses the desired runtime deployment state of a Cloud Integration " +
			"message mapping, independent of its design-time content lifecycle. Deploying is " +
			"asynchronous: this resource polls SAP's runtime artifact status until the deployment " +
			"reaches a terminal state (STARTED or ERROR) or the configured timeout elapses. This " +
			"resource is never created implicitly by deploying an integration flow that " +
			"references the mapping — SAP does not deploy referenced message mappings " +
			"automatically either, so a configuration must create this resource explicitly.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<mapping_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package the message mapping belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mapping_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the message mapping to deploy.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mapping_version": schema.StringAttribute{
				Required: true,
				Description: "The design-time version to deploy, typically " +
					"sapintegrationsuite_message_mapping.<name>.version. Changing it redeploys the " +
					"message mapping. After refresh this also reflects whatever version SAP reports " +
					"as actually deployed, so a redeploy performed outside Terraform (or a " +
					"failed/stale deployment) shows up as drift on the next plan.",
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

func (r *messageMappingDeploymentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *messageMappingDeploymentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageMappingDeploymentModel
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

	if err := client.DeployMessageMapping(ctx, plan.MappingID.ValueString(), plan.MappingVersion.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to deploy SAP Integration Suite message mapping", diagnosticDetail(err))
		return
	}

	artifact, err := waitForRuntimeArtifact(ctx, client, plan.MappingID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err)+deploymentTaintedNote)
		artifact = unconfirmedDeployment(ctx, client, plan.MappingID.ValueString(), plan.MappingVersion.ValueString())
	}

	m := messageMappingDeploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *messageMappingDeploymentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state messageMappingDeploymentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	artifact, err := client.GetRuntimeArtifact(ctx, state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite message mapping deployment", diagnosticDetail(err))
		return
	}

	// artifact.Version reflects whatever SAP actually has deployed right now,
	// which may differ from state.MappingVersion if someone redeployed a
	// different version outside Terraform, or if a deployment failed
	// part-way. Writing it into mapping_version (a Required, non-Computed
	// attribute) is what makes that visible as drift on the next plan.
	m := messageMappingDeploymentToModel(state.PackageID.ValueString(), artifact, state.Timeouts)
	m.RuntimeLocationID = state.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *messageMappingDeploymentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageMappingDeploymentModel
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

	if err := client.DeployMessageMapping(ctx, plan.MappingID.ValueString(), plan.MappingVersion.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to redeploy SAP Integration Suite message mapping", diagnosticDetail(err))
		return
	}

	artifact, err := waitForRuntimeArtifact(ctx, client, plan.MappingID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Deployment did not reach a ready state", diagnosticDetail(err))
		return
	}

	m := messageMappingDeploymentToModel(plan.PackageID.ValueString(), artifact, plan.Timeouts)
	m.RuntimeLocationID = plan.RuntimeLocationID
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}

func (r *messageMappingDeploymentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageMappingDeploymentModel
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

	err := client.UndeployRuntimeArtifact(ctx, state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to undeploy SAP Integration Suite message mapping", diagnosticDetail(err))
	}
}

func (r *messageMappingDeploymentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	packageID, mappingID := parts[0], parts[1]
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("mapping_id"), mappingID)...)
}

func messageMappingDeploymentToModel(packageID string, artifact *cloudintegration.RuntimeArtifact, tf timeouts.Value) messageMappingDeploymentModel {
	return messageMappingDeploymentModel{
		ID:             types.StringValue(packageID + "/" + artifact.ID),
		PackageID:      types.StringValue(packageID),
		MappingID:      types.StringValue(artifact.ID),
		MappingVersion: types.StringValue(artifact.Version),
		Status:         types.StringValue(artifact.Status),
		Timeouts:       tf,
	}
}
