package provider

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

const integrationPackageCustomTagType = "sapintegrationsuite_integration_package_custom_tag"

// packageCustomTagLocks serializes the tag writes to one package: Terraform
// applies the tags of a package in parallel.
var packageCustomTagLocks sync.Map

func lockPackageCustomTags(packageID string) func() {
	m, _ := packageCustomTagLocks.LoadOrStore(packageID, &sync.Mutex{})
	mu := m.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// NewIntegrationPackageCustomTagResource returns the resource for
// sapintegrationsuite_integration_package_custom_tag.
func NewIntegrationPackageCustomTagResource() resource.Resource {
	return &integrationPackageCustomTagResource{}
}

type integrationPackageCustomTagResource struct {
	client *cloudintegration.Client
}

type integrationPackageCustomTagModel struct {
	ID        types.String `tfsdk:"id"`
	PackageID types.String `tfsdk:"package_id"`
	Name      types.String `tfsdk:"name"`
	Values    types.List   `tfsdk:"values"`
}

func (r *integrationPackageCustomTagResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_package_custom_tag"
}

func (r *integrationPackageCustomTagResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The values of one custom tag on an integration package, set through " +
			"IntegrationPackages('<id>')/$links/CustomTags('<name>') of the Integration Content API. The tag " +
			"must exist in the tenant's custom tag configuration. Destroying the resource clears the tag.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "<package_id>/<name>, the form the import takes.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"package_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the integration package. Changing it moves the values to another package.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of the custom tag, exactly as in the tenant's custom tag configuration (SAP " +
					"compares names case-sensitively). Changing it clears the old tag and sets the new one.",
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"values": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
				Description: "The tag's values, in order. SAP stores them in one string separated by \"|\", " +
					"so a value cannot contain \"|\"; it also trims values and drops duplicates, which the " +
					"provider therefore refuses at plan time.",
				Validators: []validator.List{
					listvalidator.SizeAtLeast(1),
					listvalidator.UniqueValues(),
					listvalidator.ValueStringsAre(customTagValueValidator{}),
				},
			},
		},
	}
}

// customTagValueValidator refuses what SAP would change on the way in: the
// separator, surrounding blanks and empty values.
type customTagValueValidator struct{}

func (customTagValueValidator) Description(context.Context) string {
	return `must not be empty, contain "|" or start or end with a blank`
}

func (v customTagValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (customTagValueValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	switch {
	case strings.TrimSpace(value) == "":
		resp.Diagnostics.AddAttributeError(req.Path, "Empty custom tag value", "SAP ignores empty values; leave it out.")
	case strings.Contains(value, "|"):
		resp.Diagnostics.AddAttributeError(req.Path, "Custom tag value contains \"|\"",
			fmt.Sprintf("SAP separates the values of a tag with \"|\", so %q would become several values. "+
				"List them as separate elements.", value))
	case strings.TrimSpace(value) != value:
		resp.Diagnostics.AddAttributeError(req.Path, "Custom tag value starts or ends with a blank",
			fmt.Sprintf("SAP trims the values of a tag, so %q would be stored as %q.", value, strings.TrimSpace(value)))
	}
}

func (r *integrationPackageCustomTagResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, integrationPackageCustomTagType, &resp.Diagnostics) || !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
}

// findTag reads the package's tags and returns the named one. found is false
// when the package is gone or the configuration no longer has the tag.
func (r *integrationPackageCustomTagResource) findTag(ctx context.Context, packageID, name string) (tag *cloudintegration.PackageCustomTag, all []cloudintegration.PackageCustomTag, err error) {
	all, err = r.client.ListPackageCustomTags(ctx, packageID)
	if err != nil {
		if isNotFound(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	for i := range all {
		if all[i].Name == name {
			return &all[i], all, nil
		}
	}
	return nil, all, nil
}

// write sets the planned values. On create it first checks that the package
// lists the tag, to explain a tag the configuration does not have better
// than SAP's 400 does.
func (r *integrationPackageCustomTagResource) write(ctx context.Context, m *integrationPackageCustomTagModel, create bool) (diags diag.Diagnostics) {
	packageID, name := m.PackageID.ValueString(), m.Name.ValueString()
	var values []string
	if diags.Append(m.Values.ElementsAs(ctx, &values, false)...); diags.HasError() {
		return diags
	}
	unlock := lockPackageCustomTags(packageID)
	defer unlock()

	if create {
		tag, all, err := r.findTag(ctx, packageID, name)
		if err != nil {
			diags.AddError("Failed to read the custom tags of package "+packageID, diagnosticDetail(err))
			return diags
		}
		if all == nil && tag == nil {
			diags.AddError("Integration package "+packageID+" not found", "The package does not exist, or the key cannot read it.")
			return diags
		}
		if tag == nil {
			names := make([]string, 0, len(all))
			for _, t := range all {
				names = append(names, t.Name)
			}
			diags.AddError("Custom tag "+name+" is not configured",
				fmt.Sprintf("The tenant's custom tag configuration has no tag %q (names are case-sensitive). Configured "+
					"tags: %s. Add it with sapintegrationsuite_custom_tag_configuration, or in the settings of SAP's UI.",
					name, strings.Join(names, ", ")))
			return diags
		}
	}
	if err := r.client.SetPackageCustomTag(ctx, packageID, name, values); err != nil {
		diags.AddError("Failed to set custom tag "+name+" of package "+packageID, diagnosticDetail(err))
		return diags
	}
	m.ID = types.StringValue(packageID + "/" + name)
	return diags
}

func (r *integrationPackageCustomTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan integrationPackageCustomTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.Diagnostics.Append(r.write(ctx, &plan, true)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *integrationPackageCustomTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state integrationPackageCustomTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	packageID, name := state.PackageID.ValueString(), state.Name.ValueString()
	tag, _, err := r.findTag(ctx, packageID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the custom tags of package "+packageID, diagnosticDetail(err))
		return
	}
	// A cleared tag is gone as far as Terraform is concerned: the next apply
	// sets it again.
	if tag == nil || len(tag.Values) == 0 {
		resp.State.RemoveResource(ctx)
		return
	}
	values, d := types.ListValueFrom(ctx, types.StringType, tag.Values)
	resp.Diagnostics.Append(d...)
	state.Values = values
	state.ID = types.StringValue(packageID + "/" + name)
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *integrationPackageCustomTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan integrationPackageCustomTagModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if resp.Diagnostics.Append(r.write(ctx, &plan, false)...); resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

// Delete clears the tag (an empty value, probe #76). A package that is gone,
// or a tag the configuration no longer has, leaves nothing to clear.
func (r *integrationPackageCustomTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state integrationPackageCustomTagModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	packageID, name := state.PackageID.ValueString(), state.Name.ValueString()
	unlock := lockPackageCustomTags(packageID)
	defer unlock()
	tag, _, err := r.findTag(ctx, packageID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the custom tags of package "+packageID, diagnosticDetail(err))
		return
	}
	if tag == nil || len(tag.Values) == 0 {
		return
	}
	if err := r.client.SetPackageCustomTag(ctx, packageID, name, nil); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to clear custom tag "+name+" of package "+packageID, diagnosticDetail(err))
	}
}

func (r *integrationPackageCustomTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, name, ok := strings.Cut(req.ID, "/")
	if !ok || packageID == "" || name == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Expected <package_id>/<name>, got "+req.ID)
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRootID(), req.ID)...)
}
