package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIManagementCertificateStoreReferenceDataSource returns a fresh
// datasource.DataSource implementation for
// sapintegrationsuite_api_management_certificate_store_reference.
func NewAPIManagementCertificateStoreReferenceDataSource() datasource.DataSource {
	return &apiManagementCertificateStoreReferenceDataSource{}
}

type apiManagementCertificateStoreReferenceDataSource struct {
	client *apimanagementclassic.Client
}

func (d *apiManagementCertificateStoreReferenceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_management_certificate_store_reference"
}

func (d *apiManagementCertificateStoreReferenceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Classic API Management certificate store reference (CertificateStoreReferences) by name.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true, Description: "Always equal to name."},
			"name":                   schema.StringAttribute{Required: true, Description: "The certificate store reference's name to look up."},
			"certificate_store_name": schema.StringAttribute{Computed: true, Description: "The keystore or truststore the reference points to."},
			"store_type":             schema.StringAttribute{Computed: true, Description: "The referenced store's type as SAP reports it, for example \"TRUSTSTORE\"."},
		},
	}
}

func (d *apiManagementCertificateStoreReferenceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *apiManagementCertificateStoreReferenceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config apiManagementCertificateStoreReferenceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetCertificateStoreReference(ctx, config.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management certificate store reference", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, apiManagementCertificateStoreReferenceModel{
		ID:                   types.StringValue(found.Name),
		Name:                 types.StringValue(found.Name),
		CertificateStoreName: types.StringValue(found.CertificateStoreName),
		StoreType:            stringOrNull(found.StoreType),
	})...)
}
