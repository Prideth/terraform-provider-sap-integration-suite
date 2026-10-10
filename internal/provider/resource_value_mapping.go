package provider

import (
	"context"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// maxValueMappingContentBytes bounds how large a local value mapping content
// file this provider will read, for the same reasons documented on
// maxIntegrationFlowContentBytes.
const maxValueMappingContentBytes = 32 * 1024 * 1024 // 32 MiB

// NewValueMappingResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_value_mapping.
func NewValueMappingResource() resource.Resource {
	return &valueMappingResource{}
}

type valueMappingResource struct {
	client *cloudintegration.Client
}

type valueMappingModel struct {
	ID          types.String `tfsdk:"id"`
	PackageID   types.String `tfsdk:"package_id"`
	MappingID   types.String `tfsdk:"mapping_id"`
	Name        types.String `tfsdk:"name"`
	Content     types.String `tfsdk:"content"`
	ContentHash types.String `tfsdk:"content_hash"`
	Version     types.String `tfsdk:"version"`

	SaveAsVersion types.String `tfsdk:"save_as_version"`
}

func (r *valueMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_mapping"
}

func (r *valueMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the design-time content of a Cloud Integration value mapping, " +
			"uploaded from a local content file (SAP's own design-time export/import format for " +
			"this artifact type — the same ZIP-style package produced by exporting a value " +
			"mapping from the Integration Suite UI). Backed by the public Integration Content " +
			"OData V2 API (ValueMappingDesigntimeArtifacts). A value mapping cannot be saved " +
			"without at least one mapping entry, so the uploaded content must already contain " +
			"one. Deploying to a runtime is handled by the separate " +
			"sapintegrationsuite_value_mapping_deployment resource. Individual mapping entries " +
			"are not yet independently manageable through this provider — see " +
			"docs/resource-design.md for why. Changing name, content, or content_hash replaces " +
			"the value mapping (Terraform deletes the artifact and uploads it again) rather than " +
			"updating it in place, since SAP's Value Mapping API does not have a confirmed " +
			"in-place update path — see docs/sap-api-references.md.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier in the form \"<package_id>/<mapping_id>\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package this value mapping belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mapping_id": schema.StringAttribute{
				Required:    true,
				Description: "The value mapping's technical ID. Immutable: changing it replaces the value mapping.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "The value mapping's display name. Changing it replaces the value " +
					"mapping: SAP's Value Mapping API does not have a confirmed in-place update " +
					"path (see docs/sap-api-references.md), so Terraform deletes the " +
					"artifact and uploads it again rather than retaining an unverified update " +
					"call.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Path to the local content file for the value mapping project, for " +
					"example \"${path.module}/value-mappings/company-codes.zip\". Required to " +
					"manage the mapping's content; left as-is on import until a matching " +
					"configuration is applied, since SAP does not return a local file path for an " +
					"existing design-time artifact. Changing it replaces the value mapping — see " +
					"the \"name\" attribute above for why.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"content_hash": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "SHA-256 hash of the content file, for example " +
					"filesha256(\"${path.module}/value-mappings/company-codes.zip\"). Terraform " +
					"replaces the value mapping when this hash changes — see the \"name\" " +
					"attribute above for why.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"version": schema.StringAttribute{
				Computed: true,
				Description: "The design-time version SAP holds: the Bundle-Version of the uploaded content, " +
					"or save_as_version once that is set.",
			},
			"save_as_version": schema.StringAttribute{
				Optional: true,
				Description: "A version number to give the value mapping, for example \"1.0.3\", " +
					"through ValueMappingDesigntimeArtifactSaveAsVersion of the Integration Content API. " +
					"SAP keeps one version of a value mapping, so this relabels that version; it does not keep " +
					"the previous one and does not change the content. A changed value relabels in place, " +
					"without replacing the value mapping, and SAP accepts a lower number too. On a replacement " +
					"(new content or name) the new upload is labeled with it again. Without it, the version is " +
					"the Bundle-Version in the content's manifest.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(designtimeVersionPattern, "must be a version of the form major.minor.patch, for example 1.0.3"),
				},
			},
		},
	}
}

func (r *valueMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *valueMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan valueMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxValueMappingContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read value mapping content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Value mapping content hash mismatch", err.Error())
		return
	}

	mapping, err := r.client.CreateValueMapping(ctx, plan.PackageID.ValueString(), plan.MappingID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite value mapping", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveValueMappingAsVersion(ctx, plan.MappingID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, valueMappingToModel(plan.PackageID.ValueString(), mapping, unsaved))...)
			resp.Diagnostics.AddError("Value mapping created, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		mapping.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, valueMappingToModel(plan.PackageID.ValueString(), mapping, plan))...)
}

func (r *valueMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state valueMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mapping, err := r.client.GetValueMapping(ctx, state.PackageID.ValueString(), state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite value mapping", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, valueMappingToModel(state.PackageID.ValueString(), mapping, state))...)
}

// Update only relabels: name, content and content_hash are RequiresReplace
// (see Schema), because a tenant answered every PUT on a value mapping with
// 501 Not Implemented (October 2026). What remains in place is
// save_as_version: a changed value is saved through SaveAsVersion, a removed
// one changes nothing on SAP's side.
func (r *valueMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior valueMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var mapping *cloudintegration.ValueMapping
	var err error
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		mapping, err = r.client.SaveValueMappingAsVersion(ctx, plan.MappingID.ValueString(), v)
		if err != nil {
			resp.Diagnostics.AddError("Failed to save value mapping as version "+v, diagnosticDetail(err))
			return
		}
	} else {
		mapping, err = r.client.GetValueMapping(ctx, plan.PackageID.ValueString(), plan.MappingID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read SAP Integration Suite value mapping", diagnosticDetail(err))
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, valueMappingToModel(plan.PackageID.ValueString(), mapping, plan))...)
}

func (r *valueMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state valueMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteValueMapping(ctx, state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite value mapping", diagnosticDetail(err))
	}
}

func (r *valueMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, mappingID, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("mapping_id"), mappingID)...)
}

func valueMappingToModel(packageID string, mapping *cloudintegration.ValueMapping, previous valueMappingModel) valueMappingModel {
	return valueMappingModel{
		ID:          types.StringValue(packageID + "/" + mapping.ID),
		PackageID:   types.StringValue(packageID),
		MappingID:   types.StringValue(mapping.ID),
		Name:        types.StringValue(mapping.Name),
		Content:     previous.Content,
		ContentHash: previous.ContentHash,
		Version:     types.StringValue(mapping.Version),

		SaveAsVersion: previous.SaveAsVersion,
	}
}
