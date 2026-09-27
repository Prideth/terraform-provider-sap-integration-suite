package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIProductDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_api_product.
func NewAPIProductDataSource() datasource.DataSource {
	return &apiProductDataSource{}
}

type apiProductDataSource struct {
	client *apimanagementclassic.Client
}

type apiProductDataSourceModel struct {
	Name          types.String `tfsdk:"name"`
	Version       types.String `tfsdk:"version"`
	Title         types.String `tfsdk:"title"`
	Description   types.String `tfsdk:"description"`
	Scope         types.String `tfsdk:"scope"`
	StatusCode    types.String `tfsdk:"status_code"`
	IsPublished   types.Bool   `tfsdk:"is_published"`
	IsRestricted  types.Bool   `tfsdk:"is_restricted"`
	APIProxyNames []string     `tfsdk:"api_proxy_names"`
}

func (d *apiProductDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_product"
}

func (d *apiProductDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Classic API Management API product (APIProducts) by name.",
		Attributes: map[string]schema.Attribute{
			"name":          schema.StringAttribute{Required: true, Description: "The API product's name to look up."},
			"version":       schema.StringAttribute{Computed: true, Description: "The product's version."},
			"title":         schema.StringAttribute{Computed: true, Description: "The title shown in the API business hub enterprise."},
			"description":   schema.StringAttribute{Computed: true, Description: "A longer description of the product."},
			"scope":         schema.StringAttribute{Computed: true, Description: "OAuth scopes the product grants."},
			"status_code":   schema.StringAttribute{Computed: true, Description: "The product's status, for example \"PUBLISHED\" or \"DRAFT\"."},
			"is_published":  schema.BoolAttribute{Computed: true, Description: "Whether the product is published for subscription."},
			"is_restricted": schema.BoolAttribute{Computed: true, Description: "Whether a subscription needs approval."},
			"api_proxy_names": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Names of the API proxies bundled in this product.",
			},
		},
	}
}

func (d *apiProductDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireAPIManagementClassicHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = apimanagementclassic.New(data.APIManagementClassicHTTPClient, data.APIManagementClassicHost)
}

func (d *apiProductDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiProductDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetAPIProduct(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management API product", diagnosticDetail(err))
		return
	}
	proxyNames, err := d.client.GetAPIProductProxyNames(ctx, found.Name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the API proxies linked to a Classic API Management API product", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiProductDataSourceModel{
		Name:          types.StringValue(found.Name),
		Version:       stringOrNull(found.Version),
		Title:         stringOrNull(found.Title),
		Description:   stringOrNull(found.Description),
		Scope:         stringOrNull(found.Scope),
		StatusCode:    stringOrNull(found.StatusCode),
		IsPublished:   types.BoolValue(found.IsPublished),
		IsRestricted:  types.BoolValue(found.IsRestricted),
		APIProxyNames: proxyNames,
	})...)
}
