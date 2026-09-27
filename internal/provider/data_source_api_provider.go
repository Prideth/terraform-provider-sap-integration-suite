package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIProviderDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_api_provider.
func NewAPIProviderDataSource() datasource.DataSource {
	return &apiProviderDataSource{}
}

type apiProviderDataSource struct {
	client *apimanagementclassic.Client
}

type apiProviderDataSourceModel struct {
	Name                 types.String `tfsdk:"name"`
	Title                types.String `tfsdk:"title"`
	Description          types.String `tfsdk:"description"`
	DestType             types.String `tfsdk:"dest_type"`
	Host                 types.String `tfsdk:"host"`
	Port                 types.Int64  `tfsdk:"port"`
	UseSSL               types.Bool   `tfsdk:"use_ssl"`
	TrustAll             types.Bool   `tfsdk:"trust_all"`
	PathPrefix           types.String `tfsdk:"path_prefix"`
	ServiceCollectionURL types.String `tfsdk:"service_collection_url"`
	AuthType             types.String `tfsdk:"auth_type"`
	UserName             types.String `tfsdk:"user_name"`
}

func apiProviderDataSourceSchema() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"name":        schema.StringAttribute{Computed: true, Description: "The API provider's name, its key."},
		"title":       schema.StringAttribute{Computed: true, Description: "Display title of the provider."},
		"description": schema.StringAttribute{Computed: true, Description: "Free-text description of the provider."},
		"dest_type": schema.StringAttribute{Computed: true, Description: "Connection type as SAP stores it in destType, " +
			"for example \"INTERNET\". The data sources also return providers of the connection types the resource " +
			"does not manage."},
		"host":      schema.StringAttribute{Computed: true, Description: "Backend host, without scheme."},
		"port":      schema.Int64Attribute{Computed: true, Description: "Backend port."},
		"use_ssl":   schema.BoolAttribute{Computed: true, Description: "Whether the connection to the backend uses TLS."},
		"trust_all": schema.BoolAttribute{Computed: true, Description: "Whether API Management accepts any TLS certificate the backend presents."},
		"path_prefix": schema.StringAttribute{Computed: true, Description: "Path prefix for SAP Gateway catalog service " +
			"discovery, if set."},
		"service_collection_url": schema.StringAttribute{Computed: true, Description: "Relative path of the SAP Gateway " +
			"catalog service, if set."},
		"auth_type": schema.StringAttribute{Computed: true, Description: "Authentication method for the backend, for example \"BASIC\"."},
		"user_name": schema.StringAttribute{Computed: true, Description: "User for basic authentication. The password is never returned."},
	}
}

func apiProviderToDataSourceModel(p apimanagementclassic.APIProvider) apiProviderDataSourceModel {
	return apiProviderDataSourceModel{
		Name:                 types.StringValue(p.Name),
		Title:                stringOrNull(p.Title),
		Description:          stringOrNull(p.Description),
		DestType:             stringOrNull(p.DestType),
		Host:                 stringOrNull(p.Host),
		Port:                 int64OrNull(p.Port),
		UseSSL:               types.BoolValue(p.UseSSL),
		TrustAll:             types.BoolValue(p.TrustAll),
		PathPrefix:           stringOrNull(p.PathPrefix),
		ServiceCollectionURL: stringOrNull(p.URL),
		AuthType:             stringOrNull(p.AuthType),
		UserName:             stringOrNull(p.UserName),
	}
}

func (d *apiProviderDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_provider"
}

func (d *apiProviderDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := apiProviderDataSourceSchema()
	attrs["name"] = schema.StringAttribute{
		Required:    true,
		Description: "The API provider's name to look up.",
	}
	resp.Schema = schema.Schema{
		Description: "Reads a Classic API Management API provider (APIProviders) by name. See " +
			"sapintegrationsuite_api_provider for the corresponding resource and its documented " +
			"scope (Internet connection type only).",
		Attributes: attrs,
	}
}

func (d *apiProviderDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiProviderDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiProviderDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetAPIProvider(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management API provider", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiProviderToDataSourceModel(*found))...)
}
