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

// maxMessageMappingContentBytes bounds how large a local message mapping
// content file this provider will read, for the same reasons documented on
// maxIntegrationFlowContentBytes.
const maxMessageMappingContentBytes = 32 * 1024 * 1024 // 32 MiB

// NewMessageMappingResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_message_mapping.
func NewMessageMappingResource() resource.Resource {
	return &messageMappingResource{}
}

type messageMappingResource struct {
	client *cloudintegration.Client
}

type messageMappingModel struct {
	ID            types.String `tfsdk:"id"`
	PackageID     types.String `tfsdk:"package_id"`
	MappingID     types.String `tfsdk:"mapping_id"`
	Name          types.String `tfsdk:"name"`
	Content       types.String `tfsdk:"content"`
	ContentHash   types.String `tfsdk:"content_hash"`
	Version       types.String `tfsdk:"version"`
	SaveAsVersion types.String `tfsdk:"save_as_version"`
}

func (r *messageMappingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_message_mapping"
}

func (r *messageMappingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the design-time content of a reusable Cloud Integration message " +
			"mapping artifact, uploaded from a local content file (a ZIP archive containing a " +
			"mapping definition file and any schema files it references). Backed by the public " +
			"Integration Content OData V2 API (MessageMappingDesigntimeArtifacts). This is the " +
			"reusable, package-level message mapping artifact that an integration flow can " +
			"reference from a message mapping step — not the inline/local message mapping " +
			"configuration an integration flow can also define directly inside its own content; " +
			"see docs/resource-design.md for that distinction. Deploying to a runtime is handled " +
			"by the separate sapintegrationsuite_message_mapping_deployment resource, and is " +
			"never triggered automatically by deploying an integration flow that references this " +
			"mapping — SAP does not do that either.",
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
				Description: "ID of the integration package this message mapping belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"mapping_id": schema.StringAttribute{
				Required:    true,
				Description: "The message mapping's technical ID. Immutable: changing it replaces the message mapping.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The message mapping's display name.",
			},
			"content": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Path to the local content file for the message mapping project, for " +
					"example \"${path.module}/message-mappings/customer-mapping.zip\". Required to " +
					"manage the mapping's content; left as-is on import until a matching " +
					"configuration is applied, since SAP does not return a local file path for an " +
					"existing design-time artifact. The copy that is uploaded carries mapping_id as " +
					"Bundle-SymbolicName and in Provide-Capability (the file itself is not changed, and a " +
					"warning says when this happened): SAP rejects a content update whose bundle ID differs " +
					"from the mapping ID.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_hash": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "SHA-256 hash of the content file, for example " +
					"filesha256(\"${path.module}/message-mappings/customer-mapping.zip\"). " +
					"Terraform only re-uploads the file when this hash changes.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP assigned to the most recent upload.",
			},
			"save_as_version": saveAsVersionAttribute("message mapping"),
		},
	}
}

func (r *messageMappingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *messageMappingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan messageMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxMessageMappingContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read message mapping content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Message mapping content hash mismatch", err.Error())
		return
	}
	content = alignUploadBundleID(content, plan.MappingID.ValueString(), plan.Content.ValueString(), &resp.Diagnostics)

	mapping, err := r.client.CreateMessageMapping(ctx, plan.PackageID.ValueString(), plan.MappingID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite message mapping", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveMessageMappingAsVersion(ctx, plan.MappingID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, messageMappingToModel(plan.PackageID.ValueString(), mapping, unsaved))...)
			resp.Diagnostics.AddError("Message mapping created, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		mapping.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, messageMappingToModel(plan.PackageID.ValueString(), mapping, plan))...)
}

func (r *messageMappingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state messageMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	mapping, err := r.client.GetMessageMapping(ctx, state.PackageID.ValueString(), state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite message mapping", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, messageMappingToModel(state.PackageID.ValueString(), mapping, state))...)
}

func (r *messageMappingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan messageMappingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var prior messageMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}

	content, err := readBoundedFile(plan.Content.ValueString(), maxMessageMappingContentBytes)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read message mapping content file", err.Error())
		return
	}
	if err := verifyContentHash(content, plan.ContentHash.ValueString()); err != nil {
		resp.Diagnostics.AddError("Message mapping content hash mismatch", err.Error())
		return
	}
	content = alignUploadBundleID(content, plan.MappingID.ValueString(), plan.Content.ValueString(), &resp.Diagnostics)

	mapping, err := r.client.UpdateMessageMapping(ctx, plan.MappingID.ValueString(), plan.Name.ValueString(), content)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite message mapping", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveMessageMappingAsVersion(ctx, plan.MappingID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			resp.Diagnostics.Append(resp.State.Set(ctx, messageMappingToModel(plan.PackageID.ValueString(), mapping, unsaved))...)
			resp.Diagnostics.AddError("Message mapping updated, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		mapping.Version = saved.Version
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, messageMappingToModel(plan.PackageID.ValueString(), mapping, plan))...)
}

func (r *messageMappingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state messageMappingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteMessageMapping(ctx, state.MappingID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite message mapping", diagnosticDetail(err))
	}
}

func (r *messageMappingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, mappingID, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("mapping_id"), mappingID)...)
}

func messageMappingToModel(packageID string, mapping *cloudintegration.MessageMapping, previous messageMappingModel) messageMappingModel {
	return messageMappingModel{
		ID:            types.StringValue(packageID + "/" + mapping.ID),
		PackageID:     types.StringValue(packageID),
		MappingID:     types.StringValue(mapping.ID),
		Name:          types.StringValue(mapping.Name),
		Content:       previous.Content,
		ContentHash:   previous.ContentHash,
		Version:       types.StringValue(mapping.Version),
		SaveAsVersion: previous.SaveAsVersion,
	}
}
