package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIKeyValueMapDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_api_key_value_map.
func NewAPIKeyValueMapDataSource() datasource.DataSource {
	return &apiKeyValueMapDataSource{}
}

type apiKeyValueMapDataSource struct {
	client *apimanagementclassic.Client
}

type apiKeyValueMapEntryDataSourceModel struct {
	Key   types.String `tfsdk:"key"`
	Value types.String `tfsdk:"value"`
}

type apiKeyValueMapDataSourceModel struct {
	Name    types.String                         `tfsdk:"name"`
	Scope   types.String                         `tfsdk:"scope"`
	ScopeID types.String                         `tfsdk:"scope_id"`
	Entries []apiKeyValueMapEntryDataSourceModel `tfsdk:"entries"`
}

func (d *apiKeyValueMapDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key_value_map"
}

func (d *apiKeyValueMapDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Classic API Management Key Value Map (GenericKeyMapEntries) by its " +
			"composite (name, scope, scope_id) key. See sapintegrationsuite_api_key_value_map for " +
			"the corresponding resource and its documented scope (unencrypted maps only). Entry " +
			"values are marked sensitive since a Key Value Map can carry runtime credentials, even " +
			"though this data source only supports unencrypted maps.",
		Attributes: map[string]schema.Attribute{
			"name":     schema.StringAttribute{Required: true, Description: "The map's name."},
			"scope":    schema.StringAttribute{Required: true, Description: "The map's scope, for example \"APIPROXY\"."},
			"scope_id": schema.StringAttribute{Required: true, Description: "The object the scope refers to; for \"APIPROXY\", the proxy's name."},
			"entries": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The map's entries.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Computed: true, Description: "The entry's key."},
						"value": schema.StringAttribute{Computed: true, Sensitive: true, Description: "The entry's value. Sensitive, because maps can carry runtime credentials."},
					},
				},
			},
		},
	}
}

func (d *apiKeyValueMapDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiKeyValueMapDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiKeyValueMapDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetKeyValueMap(ctx, config.Name.ValueString(), config.Scope.ValueString(), config.ScopeID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management key value map", diagnosticDetail(err))
		return
	}

	entries := make([]apiKeyValueMapEntryDataSourceModel, 0, len(found.Entries))
	for _, e := range found.Entries {
		entries = append(entries, apiKeyValueMapEntryDataSourceModel{Key: types.StringValue(e.Name), Value: types.StringValue(e.Value)})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiKeyValueMapDataSourceModel{
		Name:    types.StringValue(found.Name),
		Scope:   types.StringValue(found.Scope),
		ScopeID: types.StringValue(found.ScopeID),
		Entries: entries,
	})...)
}
