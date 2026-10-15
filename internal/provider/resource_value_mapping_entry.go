package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

const (
	valueMappingEntryType = "sapintegrationsuite_value_mapping_entry"
	// valueMappingEntryChangeOp is the in-place change of an entry:
	// UpsertValMaps with ValMapId, which only the $metadata declares.
	valueMappingEntryChangeOp = "entry change in place (UpsertValMaps with ValMapId)"
)

// valueMappingLocks serializes the writes to one value mapping: every entry
// of it changes the same design-time artifact, and Terraform applies
// resources in parallel.
var valueMappingLocks sync.Map

func lockValueMapping(mappingID string) func() {
	m, _ := valueMappingLocks.LoadOrStore(mappingID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// NewValueMappingEntryResource returns the resource for
// sapintegrationsuite_value_mapping_entry.
func NewValueMappingEntryResource() resource.Resource {
	return &valueMappingEntryResource{}
}

type valueMappingEntryResource struct {
	client *cloudintegration.Client
	// allowUnofficial is the provider's enable_unofficial: it allows the
	// in-place change of an entry, which the specification does not document.
	allowUnofficial bool
}

type valueMappingEntryModel struct {
	ID               types.String `tfsdk:"id"`
	ValueMappingID   types.String `tfsdk:"value_mapping_id"`
	SourceAgency     types.String `tfsdk:"source_agency"`
	SourceIdentifier types.String `tfsdk:"source_identifier"`
	TargetAgency     types.String `tfsdk:"target_agency"`
	TargetIdentifier types.String `tfsdk:"target_identifier"`
	SourceValue      types.String `tfsdk:"source_value"`
	TargetValue      types.String `tfsdk:"target_value"`
	EntryID          types.String `tfsdk:"entry_id"`
	IsDefault        types.Bool   `tfsdk:"is_default"`
}

func (m valueMappingEntryModel) pair() cloudintegration.ValueMappingAgencyPair {
	return cloudintegration.ValueMappingAgencyPair{
		SourceAgency: m.SourceAgency.ValueString(), SourceIdentifier: m.SourceIdentifier.ValueString(),
		TargetAgency: m.TargetAgency.ValueString(), TargetIdentifier: m.TargetIdentifier.ValueString(),
	}
}

// valueMappingEntryImportID joins what locates an entry; the import ID has
// the same form.
func valueMappingEntryImportID(mappingID string, p cloudintegration.ValueMappingAgencyPair, entryID string) string {
	return strings.Join([]string{mappingID, p.SourceAgency, p.SourceIdentifier, p.TargetAgency, p.TargetIdentifier, entryID}, "/")
}

func (r *valueMappingEntryResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_mapping_entry"
}

func (r *valueMappingEntryResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	locator := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Required:      true,
			Description:   description + " Changing it creates a new entry and leaves the old one in place.",
			Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
	}
	value := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Required: true,
			Description: description + " Changing it changes the entry in place, which needs " +
				"enable_unofficial = true: the request (UpsertValMaps with ValMapId) is known only from the $metadata.",
			Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
		}
	}
	resp.Schema = schema.Schema{
		Description: "One source and target value pair of a value mapping, in an agency pair the value mapping's " +
			"content already has. Added through UpsertValMaps of the Integration Content API. SAP has no request " +
			"that deletes a single entry, so destroying the resource leaves the entry in the value mapping.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				Description: "<value_mapping_id>/<source_agency>/<source_identifier>/<target_agency>/<target_identifier>/<entry_id>, " +
					"the form the import takes.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"value_mapping_id":  locator("ID of the value mapping, for example sapintegrationsuite_value_mapping.<name>.mapping_id."),
			"source_agency":     locator("Source agency of the agency pair, as in the value mapping's content."),
			"source_identifier": locator("Source identifier (schema) of the agency pair."),
			"target_agency":     locator("Target agency of the agency pair."),
			"target_identifier": locator("Target identifier (schema) of the agency pair."),
			"source_value":      value("The source value."),
			"target_value":      value("The target value."),
			"entry_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID SAP gives the entry (a GUID; the group ID in the value mapping's content).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"is_default": schema.BoolAttribute{
				Computed: true,
				Description: "Whether SAP uses this entry by default for its source value, which matters when the " +
					"source value has more than one target value. SAP makes the newest such entry the default.",
			},
		},
	}
}

func (r *valueMappingEntryResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, valueMappingEntryType, &resp.Diagnostics) || !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
	r.allowUnofficial = data.EnableUnofficial
}

// ModifyPlan stops an in-place change without enable_unofficial and warns
// that a destroy leaves the entry in the value mapping.
func (r *valueMappingEntryResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return
	}
	if isPlannedDelete(req, resp) {
		resp.Diagnostics.AddWarning("Value mapping entry stays in the value mapping",
			"SAP has no request that deletes a single value mapping entry, so Terraform only forgets it. "+
				"Delete it in SAP's UI, or replace the value mapping's content.")
	}
	if isInPlaceUpdate(req, resp) {
		requireUnofficialOperation(r.allowUnofficial, valueMappingEntryType, valueMappingEntryChangeOp, &resp.Diagnostics)
	}
}

// read fills the entry's ID and default flag from SAP; found is false when
// the value mapping, the agency pair or the entry is gone.
func (r *valueMappingEntryResource) read(ctx context.Context, m *valueMappingEntryModel) (found bool, err error) {
	mappingID, pair := m.ValueMappingID.ValueString(), m.pair()
	entries, err := r.client.ListValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	var entry *cloudintegration.ValueMappingEntry
	for i := range entries {
		if entries[i].ID == m.EntryID.ValueString() {
			entry = &entries[i]
		}
	}
	if entry == nil {
		return false, nil
	}
	defaults, err := r.client.ListDefaultValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		return false, err
	}
	isDefault := false
	for _, d := range defaults {
		isDefault = isDefault || d.ID == entry.ID
	}
	m.SourceValue, m.TargetValue = types.StringValue(entry.SourceValue), types.StringValue(entry.TargetValue)
	m.IsDefault = types.BoolValue(isDefault)
	m.ID = types.StringValue(valueMappingEntryImportID(mappingID, pair, entry.ID))
	return true, nil
}

func (r *valueMappingEntryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan valueMappingEntryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappingID, pair := plan.ValueMappingID.ValueString(), plan.pair()
	unlock := lockValueMapping(mappingID)
	defer unlock()

	has, err := r.client.HasValueMappingAgencyPair(ctx, mappingID, pair)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the agency pairs of value mapping "+mappingID, diagnosticDetail(err))
		return
	}
	if !has {
		resp.Diagnostics.AddError("Value mapping "+mappingID+" has no such agency pair",
			fmt.Sprintf("The value mapping has no agency pair %s/%s -> %s/%s. SAP adds entries only to agency pairs "+
				"the value mapping's content already has, and cannot create a pair through the API: add it to the "+
				"content of sapintegrationsuite_value_mapping, or in SAP's UI.",
				pair.SourceAgency, pair.SourceIdentifier, pair.TargetAgency, pair.TargetIdentifier))
		return
	}
	entries, err := r.client.ListValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the entries of value mapping "+mappingID, diagnosticDetail(err))
		return
	}
	for _, e := range entries {
		if e.SourceValue == plan.SourceValue.ValueString() && e.TargetValue == plan.TargetValue.ValueString() {
			resp.Diagnostics.AddError("Value mapping entry exists already",
				fmt.Sprintf("Value mapping %s already maps %q to %q. Import it instead: terraform import <address> %s",
					mappingID, e.SourceValue, e.TargetValue, valueMappingEntryImportID(mappingID, pair, e.ID)))
			return
		}
	}
	entry, err := r.client.AddValueMappingEntry(ctx, mappingID, pair, plan.SourceValue.ValueString(), plan.TargetValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to add the value mapping entry", diagnosticDetail(err))
		return
	}
	plan.EntryID = types.StringValue(entry.ID)
	plan.ID = types.StringValue(valueMappingEntryImportID(mappingID, pair, entry.ID))
	plan.IsDefault = types.BoolValue(false)
	if _, err := r.read(ctx, &plan); err != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
		resp.Diagnostics.AddError("Value mapping entry added, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *valueMappingEntryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state valueMappingEntryModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := r.read(ctx, &state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the value mapping entry", diagnosticDetail(err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *valueMappingEntryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requireUnofficialOperation(r.allowUnofficial, valueMappingEntryType, valueMappingEntryChangeOp, &resp.Diagnostics) {
		return
	}
	var plan valueMappingEntryModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappingID := plan.ValueMappingID.ValueString()
	unlock := lockValueMapping(mappingID)
	defer unlock()
	if err := r.client.ChangeValueMappingEntry(ctx, mappingID, plan.pair(), plan.EntryID.ValueString(),
		plan.SourceValue.ValueString(), plan.TargetValue.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to change the value mapping entry", diagnosticDetail(err))
		return
	}
	found, err := r.read(ctx, &plan)
	if err != nil || !found {
		resp.Diagnostics.AddError("Failed to read the value mapping entry after the change", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete leaves the entry in the value mapping: SAP has no request that
// deletes a single entry (probe #75), and DeleteValMaps would remove the
// whole agency pair for good.
func (r *valueMappingEntryResource) Delete(_ context.Context, _ resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Value mapping entry left in the value mapping",
		"SAP has no request that deletes a single value mapping entry, so the entry stays. Delete it in SAP's UI, "+
			"or replace the value mapping's content.")
}

func (r *valueMappingEntryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, "/")
	if len(parts) != 6 {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected <value_mapping_id>/<source_agency>/<source_identifier>/<target_agency>/<target_identifier>/<entry_id>, got "+req.ID)
		return
	}
	for i, name := range []string{"value_mapping_id", "source_agency", "source_identifier", "target_agency", "target_identifier", "entry_id"} {
		if parts[i] == "" {
			resp.Diagnostics.AddError("Invalid import ID", name+" is empty in "+req.ID)
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot(name), parts[i])...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("id"), req.ID)...)
}
