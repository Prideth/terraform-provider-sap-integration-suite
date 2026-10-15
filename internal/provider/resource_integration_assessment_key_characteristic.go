package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

// Key characteristics a tenant maintains itself: a group, a key
// characteristic in a group, and a value of a key characteristic. SAP's
// Entities API specification documents create, update and delete for all
// three; none of them has passed an acceptance test yet, so they are
// experimental.

const (
	iaKeyCharacteristicGroupType = "sapintegrationsuite_integration_assessment_key_characteristic_group"
	iaKeyCharacteristicType      = "sapintegrationsuite_integration_assessment_key_characteristic"
)

// iaNameAttribute is a required name that changes in place.
func iaNameAttribute(object string) schema.StringAttribute {
	return schema.StringAttribute{
		Required:    true,
		Description: "The " + object + "'s name. Changes in place.",
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
	}
}

// iaDescriptionAttribute is an optional description that changes in place;
// removing it clears it.
func iaDescriptionAttribute(object string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: "A description of the " + object + ". Changes in place; removing it clears it.",
		Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
	}
}

// iaDescriptionValue maps a description read from the service to the
// state: an empty one is the same as none.
func iaDescriptionValue(s *string) types.String {
	if s == nil || *s == "" {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// --- group ---

// NewIntegrationAssessmentKeyCharacteristicGroupResource returns the
// resource for sapintegrationsuite_integration_assessment_key_characteristic_group.
func NewIntegrationAssessmentKeyCharacteristicGroupResource() resource.Resource {
	return &iaKeyCharacteristicGroupResource{}
}

type iaKeyCharacteristicGroupResource struct {
	client *integrationassessment.Client
}

type iaKeyCharacteristicGroupModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *iaKeyCharacteristicGroupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_key_characteristic_group"
}

func (r *iaKeyCharacteristicGroupResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A group of key characteristics a tenant maintains itself, next to SAP's groups such as " +
			"Connectivity or Transformations. Backed by the KeyCharacteristicGroup entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id":          iaIDAttribute("key characteristic group"),
			"name":        iaNameAttribute("group"),
			"description": iaDescriptionAttribute("group"),
		},
	}
}

func (r *iaKeyCharacteristicGroupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaKeyCharacteristicGroupType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func iaKeyCharacteristicGroupToModel(g *integrationassessment.KeyCharacteristicGroup) iaKeyCharacteristicGroupModel {
	return iaKeyCharacteristicGroupModel{ID: types.StringValue(g.ID), Name: types.StringValue(g.Name), Description: iaDescriptionValue(g.Description)}
}

func (r *iaKeyCharacteristicGroupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaKeyCharacteristicGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := r.client.CreateKeyCharacteristicGroup(ctx, plan.Name.ValueString(), optionalString(plan.Description))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment key characteristic group", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicGroupToModel(g))...)
}

func (r *iaKeyCharacteristicGroupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaKeyCharacteristicGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	g, err := r.client.GetKeyCharacteristicGroup(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic group", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicGroupToModel(g))...)
}

func (r *iaKeyCharacteristicGroupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaKeyCharacteristicGroupModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateKeyCharacteristicGroup(ctx, id, plan.Name.ValueString(), optionalString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment key characteristic group", diagnosticDetail(err))
		return
	}
	g, err := r.client.GetKeyCharacteristicGroup(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic group after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicGroupToModel(g))...)
}

func (r *iaKeyCharacteristicGroupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaKeyCharacteristicGroupModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteKeyCharacteristicGroup(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment key characteristic group", diagnosticDetail(err))
	}
}

func (r *iaKeyCharacteristicGroupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- key characteristic ---

// NewIntegrationAssessmentKeyCharacteristicResource returns the resource for
// sapintegrationsuite_integration_assessment_key_characteristic.
func NewIntegrationAssessmentKeyCharacteristicResource() resource.Resource {
	return &iaKeyCharacteristicResource{}
}

type iaKeyCharacteristicResource struct {
	client *integrationassessment.Client
}

type iaKeyCharacteristicModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsSingle    types.Bool   `tfsdk:"is_single"`
	GroupID     types.String `tfsdk:"key_characteristic_group_id"`
}

func (r *iaKeyCharacteristicResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_key_characteristic"
}

func (r *iaKeyCharacteristicResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A key characteristic a tenant maintains itself: a criterion the assessment rates " +
			"technologies on and asks about in questionnaires. Its values are " +
			"sapintegrationsuite_integration_assessment_key_characteristic_value resources. Backed by the " +
			"KeyCharacteristic entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id":          iaIDAttribute("key characteristic"),
			"name":        iaNameAttribute("key characteristic"),
			"description": iaDescriptionAttribute("key characteristic"),
			"is_single": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Default:       booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
				Description: "The type the UI shows: true for Single, a question answered with yes or no; false " +
					"(the default) for Multi, a question with several answers. Changing it creates a new key " +
					"characteristic: SAP refuses to update the type.",
			},
			"key_characteristic_group_id": iaReplacingLinkAttribute("Id of the group, from " +
				"sapintegrationsuite_integration_assessment_key_characteristic_group (resource or data source). " +
				"Changing it creates a new key characteristic: the specification's update body has no group."),
		},
	}
}

func (r *iaKeyCharacteristicResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaKeyCharacteristicType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func iaKeyCharacteristicToModel(k *integrationassessment.KeyCharacteristic) iaKeyCharacteristicModel {
	return iaKeyCharacteristicModel{
		ID: types.StringValue(k.ID), Name: types.StringValue(k.Name), Description: iaDescriptionValue(k.Description),
		IsSingle: types.BoolValue(k.IsSingle), GroupID: stringOrNull(k.KeyCharacteristicGroupID()),
	}
}

func (r *iaKeyCharacteristicResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaKeyCharacteristicModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.client.CreateKeyCharacteristic(ctx, plan.Name.ValueString(), optionalString(plan.Description),
		plan.IsSingle.ValueBool(), plan.GroupID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment key characteristic", diagnosticDetail(err))
		return
	}
	// The create response carries the group only as a deferred link.
	read, err := r.client.GetKeyCharacteristic(ctx, k.ID)
	if err != nil {
		plan.ID = types.StringValue(k.ID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Integration Assessment key characteristic created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicToModel(read))...)
}

func (r *iaKeyCharacteristicResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaKeyCharacteristicModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	k, err := r.client.GetKeyCharacteristic(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicToModel(k))...)
}

func (r *iaKeyCharacteristicResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaKeyCharacteristicModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateKeyCharacteristic(ctx, id, plan.Name.ValueString(), optionalString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment key characteristic", diagnosticDetail(err))
		return
	}
	k, err := r.client.GetKeyCharacteristic(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicToModel(k))...)
}

func (r *iaKeyCharacteristicResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaKeyCharacteristicModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteKeyCharacteristic(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment key characteristic", diagnosticDetail(err))
	}
}

func (r *iaKeyCharacteristicResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// --- value ---

// NewIntegrationAssessmentKeyCharacteristicValueResource returns the
// resource for sapintegrationsuite_integration_assessment_key_characteristic_value.
func NewIntegrationAssessmentKeyCharacteristicValueResource() resource.Resource {
	return &iaKeyCharacteristicValueResource{}
}

type iaKeyCharacteristicValueResource struct {
	client *integrationassessment.Client
}

type iaKeyCharacteristicValueResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	KeyCharacteristicID types.String `tfsdk:"key_characteristic_id"`
}

func (r *iaKeyCharacteristicValueResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_key_characteristic_value"
}

func (r *iaKeyCharacteristicValueResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A value of a key characteristic: one of the predefined answers of its question in a " +
			"questionnaire, and what a technology key characteristic rates. Backed by the KeyCharacteristicValue " +
			"entity set of the Entities API.",
		Attributes: map[string]schema.Attribute{
			"id":          iaIDAttribute("key characteristic value"),
			"name":        iaNameAttribute("value"),
			"description": iaDescriptionAttribute("value"),
			"key_characteristic_id": iaReplacingLinkAttribute("Id of the key characteristic, from " +
				"sapintegrationsuite_integration_assessment_key_characteristic. Changing it creates a new value: the " +
				"specification's update body has no key characteristic."),
		},
	}
}

func (r *iaKeyCharacteristicValueResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaKeyCharacteristicValueType, "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

func iaKeyCharacteristicValueToModel(v *integrationassessment.KeyCharacteristicValue) iaKeyCharacteristicValueResourceModel {
	return iaKeyCharacteristicValueResourceModel{
		ID: types.StringValue(v.ID), Name: types.StringValue(v.Name), Description: iaDescriptionValue(v.Description),
		KeyCharacteristicID: stringOrNull(v.KeyCharacteristicID()),
	}
}

func (r *iaKeyCharacteristicValueResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan iaKeyCharacteristicValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, err := r.client.CreateKeyCharacteristicValue(ctx, plan.Name.ValueString(), optionalString(plan.Description), plan.KeyCharacteristicID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment key characteristic value", diagnosticDetail(err))
		return
	}
	// The create response carries the key characteristic only as a deferred link.
	read, err := r.client.GetKeyCharacteristicValue(ctx, v.ID)
	if err != nil {
		plan.ID = types.StringValue(v.ID)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Integration Assessment key characteristic value created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicValueToModel(read))...)
}

func (r *iaKeyCharacteristicValueResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state iaKeyCharacteristicValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	v, err := r.client.GetKeyCharacteristicValue(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic value", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicValueToModel(v))...)
}

func (r *iaKeyCharacteristicValueResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan iaKeyCharacteristicValueResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()
	if err := r.client.UpdateKeyCharacteristicValue(ctx, id, plan.Name.ValueString(), optionalString(plan.Description)); err != nil {
		resp.Diagnostics.AddError("Failed to update Integration Assessment key characteristic value", diagnosticDetail(err))
		return
	}
	v, err := r.client.GetKeyCharacteristicValue(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Integration Assessment key characteristic value after update", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, iaKeyCharacteristicValueToModel(v))...)
}

func (r *iaKeyCharacteristicValueResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state iaKeyCharacteristicValueResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteKeyCharacteristicValue(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment key characteristic value", diagnosticDetail(err))
	}
}

func (r *iaKeyCharacteristicValueResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
