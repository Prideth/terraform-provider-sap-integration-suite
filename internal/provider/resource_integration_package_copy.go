package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

const integrationPackageCopyType = "sapintegrationsuite_integration_package_copy"

// NewIntegrationPackageCopyResource returns the resource for
// sapintegrationsuite_integration_package_copy.
func NewIntegrationPackageCopyResource() resource.Resource {
	return &integrationPackageCopyResource{}
}

type integrationPackageCopyResource struct {
	client *cloudintegration.Client
}

type integrationPackageCopyModel struct {
	ID                types.String `tfsdk:"id"`
	SourcePackageID   types.String `tfsdk:"source_package_id"`
	Suffix            types.String `tfsdk:"suffix"`
	UpgradeTrigger    types.String `tfsdk:"upgrade_trigger"`
	KeepConfiguration types.Bool   `tfsdk:"keep_configuration"`
	PackageID         types.String `tfsdk:"package_id"`
	Name              types.String `tfsdk:"name"`
	Version           types.String `tfsdk:"version"`
	Vendor            types.String `tfsdk:"vendor"`
	Mode              types.String `tfsdk:"mode"`
	IntegrationFlows  types.List   `tfsdk:"integration_flows"`
}

var integrationPackageCopyFlowType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id": types.StringType, "name": types.StringType, "version": types.StringType,
}}

// copySuffixPattern keeps a suffix to what SAP appends cleanly to package and
// flow IDs: letters, digits and underscores (a provider choice; SAP's
// specification names no limit).
var copySuffixPattern = regexp.MustCompile(`^[A-Za-z0-9_]+$`)

func (r *integrationPackageCopyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_package_copy"
}

func (r *integrationPackageCopyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Computed: true, Description: description}
	}
	resp.Schema = schema.Schema{
		Description: "A copy of an SAP standard integration package from the Discover catalog in Design, made " +
			"through CopyIntegrationPackage of the Integration Content API. The copy is configure-only: its " +
			"flows can be configured and deployed, its content and metadata come from SAP. Destroying the " +
			"resource deletes the copy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the copy in Design, the same as package_id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"source_package_id": schema.StringAttribute{
				Required: true,
				Description: "ID of the package in the Discover catalog, as in its URL, for example " +
					"\"ERPtoSuccessFactorsEmployeeCentralCostCenter\". Changing it replaces the copy.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"suffix": schema.StringAttribute{
				Optional: true,
				Description: "Makes an additional copy (ImportMode CREATE_COPY): SAP appends \".<suffix>\" to the " +
					"package ID, the name and the IDs of the flows, so that the copy can live next to the " +
					"original. Letters, digits and underscores. Changing it replaces the copy.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(copySuffixPattern, "must contain only letters, digits and underscores"),
				},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"upgrade_trigger": schema.StringAttribute{
				Optional: true,
				Description: "Any value, for example the package version from the Discover catalog. Changing it " +
					"copies the package again over the existing copy, which brings it to SAP's current version " +
					"(ImportMode OVERWRITE_MERGE, or OVERWRITE with keep_configuration = false). A copy with a " +
					"suffix cannot be overwritten (SAP overwrites the original instead), so for it a change " +
					"replaces the copy and its configuration is lost.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(
					func(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						var suffix types.String
						resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, pathRoot("suffix"), &suffix)...)
						resp.RequiresReplace = !suffix.IsNull() && suffix.ValueString() != ""
					},
					"Replaces a copy with a suffix, which SAP cannot overwrite.",
					"Replaces a copy with a suffix, which SAP cannot overwrite.",
				)},
			},
			"keep_configuration": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Whether an upgrade keeps the externalized parameters set on the copy's flows " +
					"(OVERWRITE_MERGE, the default) or resets them to SAP's defaults (OVERWRITE).",
			},
			"package_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The ID of the copy in Design: source_package_id, followed by \".<suffix>\" with a suffix.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":    computed("The package's name, as SAP gives it (with \".<suffix>\" for an additional copy)."),
			"version": computed("The package version SAP copied."),
			"vendor": schema.StringAttribute{
				Computed: true, Description: "The vendor, SAP for standard packages.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"mode": schema.StringAttribute{
				Computed: true, Description: "The package mode; READ_ONLY (configure-only) for a copy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"integration_flows": schema.ListNestedAttribute{
				Computed: true,
				Description: "The integration flows of the copy, in the order SAP returns them, for the " +
					"configuration and deployment resources.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":      schema.StringAttribute{Computed: true, Description: "ID of the flow."},
					"name":    schema.StringAttribute{Computed: true, Description: "Name of the flow."},
					"version": schema.StringAttribute{Computed: true, Description: "Design-time version of the flow."},
				}},
			},
		},
	}
}

func (r *integrationPackageCopyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, integrationPackageCopyType, &resp.Diagnostics) || !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
}

func (m integrationPackageCopyModel) copyID() string {
	return cloudintegration.CopiedPackageID(m.SourcePackageID.ValueString(), m.Suffix.ValueString())
}

// importID is the import ID of a copy: the source ID, and "/<suffix>" for an
// additional copy.
func (m integrationPackageCopyModel) importID() string {
	if s := m.Suffix.ValueString(); s != "" {
		return m.SourcePackageID.ValueString() + "/" + s
	}
	return m.SourcePackageID.ValueString()
}

// read fills the computed attributes from the copy in Design; found is false
// when it is gone.
func (r *integrationPackageCopyResource) read(ctx context.Context, m *integrationPackageCopyModel) (found bool, diags diag.Diagnostics) {
	id := m.copyID()
	pkg, err := r.client.GetPackage(ctx, id)
	if err != nil {
		if isNotFound(err) {
			return false, diags
		}
		diags.AddError("Failed to read integration package "+id, diagnosticDetail(err))
		return false, diags
	}
	flows, err := r.client.ListPackageIntegrationFlows(ctx, id)
	if err != nil {
		diags.AddError("Failed to read the integration flows of package "+id, diagnosticDetail(err))
		return false, diags
	}
	values := make([]attr.Value, 0, len(flows))
	for _, f := range flows {
		values = append(values, types.ObjectValueMust(integrationPackageCopyFlowType.AttrTypes, map[string]attr.Value{
			"id": types.StringValue(f.ID), "name": types.StringValue(f.Name), "version": types.StringValue(f.Version),
		}))
	}
	list, d := types.ListValue(integrationPackageCopyFlowType, values)
	diags.Append(d...)
	m.ID, m.PackageID = types.StringValue(pkg.ID), types.StringValue(pkg.ID)
	m.Name, m.Version = types.StringValue(pkg.Name), types.StringValue(pkg.Version)
	m.Vendor, m.Mode = types.StringValue(pkg.Vendor), types.StringValue(pkg.Mode)
	m.IntegrationFlows = list
	return true, diags
}

func (r *integrationPackageCopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationPackageCopyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.copyID()
	if _, err := r.client.GetPackage(ctx, id); err == nil {
		resp.Diagnostics.AddError("Integration package "+id+" is in Design already",
			fmt.Sprintf("Design already holds %s. Import it instead of copying it again: terraform import <address> %s",
				id, plan.importID()))
		return
	} else if !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to read integration package "+id, diagnosticDetail(err))
		return
	}
	mode := cloudintegration.CopyAsNew
	if plan.Suffix.ValueString() != "" {
		mode = cloudintegration.CopyCreateCopy
	}
	if _, err := r.client.CopyPackage(ctx, plan.SourcePackageID.ValueString(), mode, plan.Suffix.ValueString()); err != nil {
		detail := diagnosticDetail(err)
		if isNotFound(err) {
			detail = "The Discover catalog has no package " + plan.SourcePackageID.ValueString() +
				". Take the ID from the package's URL in Discover.\n" + detail
		}
		resp.Diagnostics.AddError("Failed to copy integration package "+plan.SourcePackageID.ValueString(), detail)
		return
	}
	found, diags := r.read(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Copied integration package "+id+" is missing",
			"SAP accepted the copy, but Design does not list "+id+".")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationPackageCopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationPackageCopyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, diags := r.read(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

// Update copies the package again over the copy when upgrade_trigger
// changed; a change of keep_configuration alone sends nothing. A copy with
// a suffix never gets here with a changed trigger: it is replaced.
func (r *integrationPackageCopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state integrationPackageCopyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !plan.UpgradeTrigger.Equal(state.UpgradeTrigger) {
		mode := cloudintegration.CopyOverwriteMerge
		if !plan.KeepConfiguration.ValueBool() {
			mode = cloudintegration.CopyOverwrite
		}
		if _, err := r.client.CopyPackage(ctx, plan.SourcePackageID.ValueString(), mode, ""); err != nil {
			resp.Diagnostics.AddError("Failed to upgrade integration package "+plan.copyID(), diagnosticDetail(err))
			return
		}
	}
	found, diags := r.read(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if !found && !resp.Diagnostics.HasError() {
		resp.Diagnostics.AddError("Integration package "+plan.copyID()+" is missing", "Design no longer lists the copy.")
	}
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationPackageCopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationPackageCopyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePackage(ctx, state.copyID()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete integration package "+state.copyID(), diagnosticDetail(err))
	}
}

// ImportState takes <source_package_id> or <source_package_id>/<suffix>.
func (r *integrationPackageCopyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	source, suffix, hasSuffix := strings.Cut(req.ID, "/")
	if source == "" || (hasSuffix && !copySuffixPattern.MatchString(suffix)) {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <source_package_id> or <source_package_id>/<suffix>, got "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("source_package_id"), source)...)
	if hasSuffix {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("suffix"), suffix)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("keep_configuration"), true)...)
}
