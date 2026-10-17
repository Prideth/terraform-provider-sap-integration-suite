package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

const integrationPackageCustomTagsType = "sapintegrationsuite_integration_package_custom_tags"

// NewIntegrationPackageCustomTagsDataSource returns the data source for
// sapintegrationsuite_integration_package_custom_tags.
func NewIntegrationPackageCustomTagsDataSource() datasource.DataSource {
	return &integrationPackageCustomTagsDataSource{}
}

type integrationPackageCustomTagsDataSource struct {
	client *cloudintegration.Client
}

type integrationPackageCustomTagsModel struct {
	ID        types.String                     `tfsdk:"id"`
	PackageID types.String                     `tfsdk:"package_id"`
	Tags      []integrationPackageCustomTagRow `tfsdk:"tags"`
}

type integrationPackageCustomTagRow struct {
	Name   types.String `tfsdk:"name"`
	Values types.List   `tfsdk:"values"`
}

func (d *integrationPackageCustomTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_package_custom_tags"
}

func (d *integrationPackageCustomTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Every custom tag of an integration package with its values, read through " +
			"IntegrationPackages('<id>')/CustomTags of the Integration Content API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "The package ID."},
			"package_id": schema.StringAttribute{
				Required:    true,
				Description: "ID of the integration package.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"tags": schema.ListNestedAttribute{
				Computed: true,
				Description: "One entry per tag of the tenant's custom tag configuration, in the order SAP returns " +
					"them; a tag without a value has an empty list.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"name":   schema.StringAttribute{Computed: true, Description: "Name of the custom tag."},
					"values": schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "The tag's values, in order."},
				}},
			},
		},
	}
}

func (d *integrationPackageCustomTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, integrationPackageCustomTagsType, &resp.Diagnostics) || !requireHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = cloudintegration.New(data.HTTPClient, data.Host)
}

func (d *integrationPackageCustomTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m integrationPackageCustomTagsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	packageID := m.PackageID.ValueString()
	tags, err := d.client.ListPackageCustomTags(ctx, packageID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the custom tags of package "+packageID, diagnosticDetail(err))
		return
	}
	m.Tags = make([]integrationPackageCustomTagRow, 0, len(tags))
	for _, t := range tags {
		values := t.Values
		if values == nil {
			values = []string{}
		}
		list, diags := types.ListValueFrom(ctx, types.StringType, values)
		resp.Diagnostics.Append(diags...)
		m.Tags = append(m.Tags, integrationPackageCustomTagRow{Name: types.StringValue(t.Name), Values: list})
	}
	m.ID = types.StringValue(packageID)
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}
