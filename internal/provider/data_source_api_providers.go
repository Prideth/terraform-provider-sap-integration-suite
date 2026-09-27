package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIProvidersDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_api_providers.
func NewAPIProvidersDataSource() datasource.DataSource {
	return &apiProvidersDataSource{}
}

type apiProvidersDataSource struct {
	client *apimanagementclassic.Client
}

type apiProvidersDataSourceModel struct {
	Providers []apiProviderDataSourceModel `tfsdk:"providers"`
}

func (d *apiProvidersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_providers"
}

func (d *apiProvidersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists every Classic API Management API provider (APIProviders) visible to the configured credentials.",
		Attributes: map[string]schema.Attribute{
			"providers": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Every API provider, with the same attributes as the sapintegrationsuite_api_provider data source.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: apiProviderDataSourceSchema(),
				},
			},
		},
	}
}

func (d *apiProvidersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiProvidersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	found, err := d.client.ListAPIProviders(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Classic API Management API providers", diagnosticDetail(err))
		return
	}

	model := apiProvidersDataSourceModel{Providers: make([]apiProviderDataSourceModel, 0, len(found))}
	for _, p := range found {
		model.Providers = append(model.Providers, apiProviderToDataSourceModel(p))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}
