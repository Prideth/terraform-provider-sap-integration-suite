package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

var (
	_ resource.Resource                = &messageTypeResource{}
	_ resource.ResourceWithConfigure   = &messageTypeResource{}
	_ resource.ResourceWithImportState = &messageTypeResource{}
)

// NewMessageTypeResource manages MessageTypeDesigntimeArtifacts.
func NewMessageTypeResource() resource.Resource {
	return &messageTypeResource{kind: cloudintegration.KindMessageType, typeSuffix: "message_type", idAttr: "message_type_id"}
}

// NewFaultMessageTypeResource manages FaultMessageTypeDesigntimeArtifacts,
// which share the message type's shape.
func NewFaultMessageTypeResource() resource.Resource {
	return &messageTypeResource{kind: cloudintegration.KindFaultMessageType, typeSuffix: "fault_message_type", idAttr: "fault_message_type_id"}
}

type messageTypeResource struct {
	client     *cloudintegration.Client
	kind       cloudintegration.MessageTypeKind
	typeSuffix string
	idAttr     string
}

// messageTypeFields are the attributes both resources share. Each resource
// embeds them in a model with its own ID attribute, because a model must
// match the schema exactly.
type messageTypeFields struct {
	ID            types.String `tfsdk:"id"`
	PackageID     types.String `tfsdk:"package_id"`
	Name          types.String `tfsdk:"name"`
	Namespace     types.String `tfsdk:"namespace"`
	Description   types.String `tfsdk:"description"`
	DataTypeID    types.String `tfsdk:"data_type_id"`
	Version       types.String `tfsdk:"version"`
	SaveAsVersion types.String `tfsdk:"save_as_version"`
}

type messageTypeModel struct {
	messageTypeFields
	MessageTypeID types.String `tfsdk:"message_type_id"`
}

type faultMessageTypeModel struct {
	messageTypeFields
	FaultMessageTypeID types.String `tfsdk:"fault_message_type_id"`
}

// getter is the Get method of tfsdk.Plan and tfsdk.State.
type getter func(context.Context, interface{}) diag.Diagnostics

// read decodes the resource's model and returns its shared fields and ID.
func (r *messageTypeResource) read(ctx context.Context, get getter) (messageTypeFields, string, diag.Diagnostics) {
	if r.kind == cloudintegration.KindFaultMessageType {
		var m faultMessageTypeModel
		diags := get(ctx, &m)
		return m.messageTypeFields, m.FaultMessageTypeID.ValueString(), diags
	}
	var m messageTypeModel
	diags := get(ctx, &m)
	return m.messageTypeFields, m.MessageTypeID.ValueString(), diags
}

// setState writes the shared fields and the ID into the resource's model.
func (r *messageTypeResource) setState(ctx context.Context, state *tfsdk.State, f messageTypeFields, id string, diags *diag.Diagnostics) {
	if r.kind == cloudintegration.KindFaultMessageType {
		diags.Append(state.Set(ctx, faultMessageTypeModel{messageTypeFields: f, FaultMessageTypeID: types.StringValue(id)})...)
		return
	}
	diags.Append(state.Set(ctx, messageTypeModel{messageTypeFields: f, MessageTypeID: types.StringValue(id)})...)
}

func (r *messageTypeResource) typeName() string { return "sapintegrationsuite_" + r.typeSuffix }

func (r *messageTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_" + r.typeSuffix
}

func (r *messageTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	what := r.kind.DisplayName
	generated := "SAP generates its schema: an element of the data type given in data_type_id."
	if r.kind == cloudintegration.KindFaultMessageType {
		generated = "SAP generates its schema: SAP's standard fault data (ExchangeFaultData) plus, as " +
			"additional detail, the data type given in data_type_id."
	}
	resp.Schema = schema.Schema{
		Description: fmt.Sprintf("Manages a %s in an integration package through SAP's Integration "+
			"Content API. %s", what, generated),
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   fmt.Sprintf("Composite identifier in the form \"<package_id>/<%s>\".", r.idAttr),
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"package_id": schema.StringAttribute{
				Required:      true,
				Description:   fmt.Sprintf("ID of the integration package the %s belongs to. Changing it replaces the %s.", what, what),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			r.idAttr: schema.StringAttribute{
				Required:      true,
				Description:   fmt.Sprintf("Technical ID of the %s. Changing it replaces the %s.", what, what),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("Name of the %s, also the name of its root element. Changing it "+
					"replaces the %s: SAP refuses to update the name.", what, what),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"namespace": schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("XML namespace of the %s. Changing it replaces the %s, because an "+
					"in-place namespace change was not tested.", what, what),
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Description: "Description. Changes in place; removing it replaces the " + what +
					", because clearing it in place was not tested.",
				PlanModifiers: []planmodifier.String{requiresReplaceWhenRemoved()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"data_type_id": schema.StringAttribute{
				Optional: true,
				Description: "ID of the data type the " + what + " is built on, usually " +
					"sapintegrationsuite_data_type.<name>.data_type_id. Changes in place; removing it " +
					"replaces the " + what + ". SAP does not check that the data type exists or stays: " +
					"it lets a data type in use be deleted.",
				PlanModifiers: []planmodifier.String{requiresReplaceWhenRemoved()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP reports.",
			},
			"save_as_version": saveAsVersionAttribute(what),
		},
	}
}

// requiresReplaceWhenRemoved replaces the resource when an optional value
// that was set is removed: clearing it in place was not tested.
func requiresReplaceWhenRemoved() planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
		},
		"Removing the value replaces the resource.",
		"Removing the value replaces the resource.",
	)
}

func (r *messageTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, r.typeName(), &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
}

func (r *messageTypeResource) toModel(mt *cloudintegration.MessageType, packageID string, previous messageTypeFields) messageTypeFields {
	return messageTypeFields{
		ID:            types.StringValue(packageID + "/" + mt.ID),
		PackageID:     types.StringValue(packageID),
		Name:          types.StringValue(mt.Name),
		Namespace:     stringOrNull(mt.Namespace),
		Description:   stringOrNull(mt.Description),
		DataTypeID:    stringOrNull(mt.DataTypeUsed),
		Version:       types.StringValue(mt.Version),
		SaveAsVersion: previous.SaveAsVersion,
	}
}

func (r *messageTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	plan, id, diags := r.read(ctx, req.Plan.Get)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mt, err := r.client.CreateMessageType(ctx, r.kind, cloudintegration.MessageType{
		ID: id, Name: plan.Name.ValueString(), PackageID: plan.PackageID.ValueString(), Namespace: plan.Namespace.ValueString(),
		Description: plan.Description.ValueString(), DataTypeUsed: plan.DataTypeID.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite "+r.kind.DisplayName, diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveMessageTypeAsVersion(ctx, r.kind, id, v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			r.setState(ctx, &resp.State, r.toModel(mt, plan.PackageID.ValueString(), unsaved), id, &resp.Diagnostics)
			resp.Diagnostics.AddError("Created the "+r.kind.DisplayName+", but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		mt = saved
	}
	r.setState(ctx, &resp.State, r.toModel(mt, plan.PackageID.ValueString(), plan), id, &resp.Diagnostics)
}

func (r *messageTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	state, id, diags := r.read(ctx, req.State.Get)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mt, err := r.client.GetMessageType(ctx, r.kind, id)
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite "+r.kind.DisplayName, diagnosticDetail(err))
		return
	}
	packageID := state.PackageID.ValueString()
	if mt.PackageID != "" {
		packageID = mt.PackageID
	}
	r.setState(ctx, &resp.State, r.toModel(mt, packageID, state), id, &resp.Diagnostics)
}

func (r *messageTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	plan, id, diags := r.read(ctx, req.Plan.Get)
	resp.Diagnostics.Append(diags...)
	prior, _, diags := r.read(ctx, req.State.Get)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	mt, err := r.client.UpdateMessageType(ctx, r.kind, id, plan.Description.ValueString(), plan.DataTypeID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite "+r.kind.DisplayName, diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveMessageTypeAsVersion(ctx, r.kind, id, v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			r.setState(ctx, &resp.State, r.toModel(mt, plan.PackageID.ValueString(), unsaved), id, &resp.Diagnostics)
			resp.Diagnostics.AddError("Updated the "+r.kind.DisplayName+", but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		mt = saved
	}
	r.setState(ctx, &resp.State, r.toModel(mt, plan.PackageID.ValueString(), plan), id, &resp.Diagnostics)
}

func (r *messageTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	_, id, diags := r.read(ctx, req.State.Get)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteMessageType(ctx, r.kind, id); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite "+r.kind.DisplayName, diagnosticDetail(err))
	}
}

func (r *messageTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, id, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected \"<package_id>/<%s>\", got %q.", r.idAttr, req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.idAttr), id)...)
}
