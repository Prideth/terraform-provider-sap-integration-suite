package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// maxScriptCollectionContentBytes bounds how large a local script
// collection content file this provider will read, for the same reasons
// documented on maxIntegrationFlowContentBytes.
const maxScriptCollectionContentBytes = 32 * 1024 * 1024 // 32 MiB

// NewScriptCollectionResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_script_collection.
func NewScriptCollectionResource() resource.Resource {
	return &scriptCollectionResource{}
}

type scriptCollectionResource struct {
	client *cloudintegration.Client
}

type scriptCollectionModel struct {
	ID                 types.String `tfsdk:"id"`
	PackageID          types.String `tfsdk:"package_id"`
	ScriptCollectionID types.String `tfsdk:"script_collection_id"`
	Name               types.String `tfsdk:"name"`
	Content            types.String `tfsdk:"content"`
	ContentHash        types.String `tfsdk:"content_hash"`
	Version            types.String `tfsdk:"version"`
	SaveAsVersion      types.String `tfsdk:"save_as_version"`
}

func (r *scriptCollectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_script_collection"
}

func (r *scriptCollectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the design-time content of a reusable Cloud Integration script " +
			"collection, uploaded from a local content file (a ZIP archive of Groovy/JavaScript " +
			"script files). Backed by the public Integration Content OData V2 API " +
			"(ScriptCollectionDesigntimeArtifacts). Deploying to a runtime is handled by the " +
			"separate sapintegrationsuite_script_collection_deployment resource, and is never " +
			"triggered automatically by deploying an integration flow that references this " +
			"collection.\n\n" +
			"An in-place update sends the content with PUT, and save_as_version calls " +
			"ScriptCollectionDesigntimeArtifactSaveAsVersion, both as SAP's Integration Content API " +
			"specification documents them.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<script_collection_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package this script collection belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"script_collection_id": schema.StringAttribute{
				Required: true,
				Description: "The script collection's technical ID. Immutable: changing it " +
					"replaces the script collection. SAP documents this ID as unique across the " +
					"entire tenant, not just the package.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The script collection's display name.",
			},
			"content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Path to the local content file for the script collection project, " +
					"for example \"${path.module}/script-collections/shared-scripts.zip\". Required " +
					"to manage the collection's content; left as-is on import until a matching " +
					"configuration is applied, since SAP does not return a local file path for an " +
					"existing design-time artifact.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_hash": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "SHA-256 hash of the content file, for example " +
					"filesha256(\"${path.module}/script-collections/shared-scripts.zip\"). " +
					"Terraform only re-uploads the file when this hash changes.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP assigned to the most recent upload.",
			},
			"save_as_version": saveAsVersionAttribute("script collection"),
		},
	}
}

func (r *scriptCollectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scriptCollectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scriptCollectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxScriptCollectionContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script collection content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Script collection content hash mismatch", err.Error())
		return
	}
	sc, err := r.client.CreateScriptCollection(ctx, plan.PackageID.ValueString(), plan.ScriptCollectionID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite script collection", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveScriptCollectionAsVersion(ctx, plan.ScriptCollectionID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, scriptCollectionToModel(plan.PackageID.ValueString(), sc, unsaved))...)
			resp.Diagnostics.AddError("Script collection created, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		sc.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptCollectionToModel(plan.PackageID.ValueString(), sc, plan))...)
}

func (r *scriptCollectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scriptCollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sc, err := r.client.GetScriptCollection(ctx, state.PackageID.ValueString(), state.ScriptCollectionID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite script collection", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptCollectionToModel(state.PackageID.ValueString(), sc, state))...)
}

func (r *scriptCollectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan scriptCollectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var prior scriptCollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxScriptCollectionContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read script collection content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Script collection content hash mismatch", err.Error())
		return
	}

	sc, err := r.client.UpdateScriptCollection(ctx, plan.ScriptCollectionID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite script collection", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveScriptCollectionAsVersion(ctx, plan.ScriptCollectionID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			resp.Diagnostics.Append(resp.State.Set(ctx, scriptCollectionToModel(plan.PackageID.ValueString(), sc, unsaved))...)
			resp.Diagnostics.AddError("Script collection updated, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		sc.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, scriptCollectionToModel(plan.PackageID.ValueString(), sc, plan))...)
}

func (r *scriptCollectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scriptCollectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteScriptCollection(ctx, state.ScriptCollectionID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite script collection", diagnosticDetail(err))
	}
}

func (r *scriptCollectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, scriptCollectionID, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("script_collection_id"), scriptCollectionID)...)
}

func scriptCollectionToModel(packageID string, sc *cloudintegration.ScriptCollection, previous scriptCollectionModel) scriptCollectionModel {
	return scriptCollectionModel{
		ID:                 types.StringValue(packageID + "/" + sc.ID),
		PackageID:          types.StringValue(packageID),
		ScriptCollectionID: types.StringValue(sc.ID),
		Name:               types.StringValue(sc.Name),
		Content:            previous.Content,
		ContentHash:        previous.ContentHash,
		Version:            types.StringValue(sc.Version),
		SaveAsVersion:      previous.SaveAsVersion,
	}
}
