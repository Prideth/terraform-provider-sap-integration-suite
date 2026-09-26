package provider

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// maxIntegrationFlowContentBytes bounds how large a local iFlow ZIP this
// provider will read, guarding against an oversized file being handed to
// the SAP API (and against unbounded memory use while hashing/encoding it).
const maxIntegrationFlowContentBytes = 32 * 1024 * 1024 // 32 MiB

// NewIntegrationFlowResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_integration_flow.
func NewIntegrationFlowResource() resource.Resource {
	return &integrationFlowResource{}
}

type integrationFlowResource struct {
	client *cloudintegration.Client
}

type integrationFlowModel struct {
	ID            types.String `tfsdk:"id"`
	PackageID     types.String `tfsdk:"package_id"`
	FlowID        types.String `tfsdk:"flow_id"`
	Name          types.String `tfsdk:"name"`
	Content       types.String `tfsdk:"content"`
	ContentHash   types.String `tfsdk:"content_hash"`
	Version       types.String `tfsdk:"version"`
	SaveAsVersion types.String `tfsdk:"save_as_version"`
}

func (r *integrationFlowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_flow"
}

func (r *integrationFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the design-time content of a Cloud Integration integration flow, " +
			"uploaded from a local ZIP file. Backed by the public Integration Content OData V2 API " +
			"(IntegrationDesigntimeArtifacts). Deploying the flow to a runtime is handled by the " +
			"separate sapintegrationsuite_integration_flow_deployment resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<flow_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package this flow belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"flow_id": schema.StringAttribute{
				Required:    true,
				Description: "The flow's technical ID. Immutable: changing it replaces the flow.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The flow's display name.",
			},
			"content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Path to the local ZIP file containing the integration flow project, " +
					"for example \"${path.module}/iflows/metering.zip\". Required to manage the " +
					"flow's content; left as-is on import until a matching configuration is applied, " +
					"since SAP does not return a local file path for an existing design-time artifact. " +
					"The copy that is uploaded carries flow_id as Bundle-SymbolicName in META-INF/MANIFEST.MF " +
					"(the file itself is not changed, and a warning says when this happened): SAP rejects a " +
					"content update whose bundle ID differs from the flow ID, so a ZIP exported under another " +
					"ID could otherwise be created but never updated.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_hash": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "SHA-256 hash of the content file, for example " +
					"filesha256(\"${path.module}/iflows/metering.zip\"). Terraform only re-uploads " +
					"the file when this hash changes.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP assigned to the most recent upload.",
			},
			"save_as_version": saveAsVersionAttribute("integration flow"),
		},
	}
}

func (r *integrationFlowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *integrationFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxIntegrationFlowContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read integration flow content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Integration flow content hash mismatch", err.Error())
		return
	}
	content = alignUploadBundleID(content, plan.FlowID.ValueString(), plan.Content.ValueString(), &resp.Diagnostics)

	flow, err := r.client.CreateIntegrationFlow(ctx, plan.PackageID.ValueString(), plan.FlowID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite integration flow", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveIntegrationFlowAsVersion(ctx, plan.FlowID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, flowToModel(plan.PackageID.ValueString(), flow, unsaved))...)
			resp.Diagnostics.AddError("Integration flow created, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		flow.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, flowToModel(plan.PackageID.ValueString(), flow, plan))...)
}

func (r *integrationFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	flow, err := r.client.GetIntegrationFlow(ctx, state.PackageID.ValueString(), state.FlowID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite integration flow", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, flowToModel(state.PackageID.ValueString(), flow, state))...)
}

func (r *integrationFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan integrationFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var prior integrationFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxIntegrationFlowContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read integration flow content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Integration flow content hash mismatch", err.Error())
		return
	}
	content = alignUploadBundleID(content, plan.FlowID.ValueString(), plan.Content.ValueString(), &resp.Diagnostics)

	flow, err := r.client.UpdateIntegrationFlow(ctx, plan.FlowID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite integration flow", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveIntegrationFlowAsVersion(ctx, plan.FlowID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			resp.Diagnostics.Append(resp.State.Set(ctx, flowToModel(plan.PackageID.ValueString(), flow, unsaved))...)
			resp.Diagnostics.AddError("Integration flow updated, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		flow.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, flowToModel(plan.PackageID.ValueString(), flow, plan))...)
}

func (r *integrationFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteIntegrationFlow(ctx, state.FlowID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite integration flow", diagnosticDetail(err))
	}
}

func (r *integrationFlowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, flowID, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("flow_id"), flowID)...)
}

func flowToModel(packageID string, flow *cloudintegration.IntegrationFlow, previous integrationFlowModel) integrationFlowModel {
	return integrationFlowModel{
		ID:            types.StringValue(packageID + "/" + flow.ID),
		PackageID:     types.StringValue(packageID),
		FlowID:        types.StringValue(flow.ID),
		Name:          types.StringValue(flow.Name),
		Content:       previous.Content,
		ContentHash:   previous.ContentHash,
		Version:       types.StringValue(flow.Version),
		SaveAsVersion: previous.SaveAsVersion,
	}
}

// readBoundedFile reads path, refusing to read more than maxBytes+1 (so an
// oversized file is detected rather than silently truncated). path is a
// Terraform practitioner's own "content" attribute value, not externally
// controlled input, so opening it directly is intentional.
func readBoundedFile(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path) //nolint:gosec // G304: path is an operator-supplied Terraform attribute, not external input
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > maxBytes {
		return nil, errFileTooLarge(path, info.Size(), maxBytes)
	}

	data := make([]byte, info.Size())
	if _, err := io.ReadFull(f, data); err != nil {
		return nil, err
	}
	return data, nil
}

func verifyContentHash(content []byte, expectedHex string) error {
	sum := sha256.Sum256(content)
	actualHex := hex.EncodeToString(sum[:])
	if actualHex != expectedHex {
		return errHashMismatch(expectedHex, actualHex)
	}
	return nil
}
