package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// NewUserCredentialDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_user_credential.
func NewUserCredentialDataSource() datasource.DataSource {
	return &userCredentialDataSource{}
}

type userCredentialDataSource struct {
	client *securitycontent.Client
}

type userCredentialDataSourceModel struct {
	ID                types.String `tfsdk:"id"`
	Kind              types.String `tfsdk:"kind"`
	Description       types.String `tfsdk:"description"`
	User              types.String `tfsdk:"user"`
	CompanyID         types.String `tfsdk:"company_id"`
	RuntimeLocationID types.String `tfsdk:"runtime_location_id"`
}

func (d *userCredentialDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user_credential"
}

func (d *userCredentialDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an existing Security Content \"User Credentials\" artifact by its name. " +
			"Never returns the password: SAP's Security Content API does not document returning a " +
			"stored credential's password, and this data source has no field for one even if it did.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationDataSourceAttribute(),
			"id": schema.StringAttribute{
				Required:    true,
				Description: "The credential artifact's name (its OData key and adapter alias).",
			},
			"kind": schema.StringAttribute{
				Computed:    true,
				Description: "The credential's system-specific type as SAP reports it: \"default\" for a generic credential, \"SuccessFactors\" or \"OpenConnectors\".",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "The credential artifact's free-text description.",
			},
			"user": schema.StringAttribute{
				Computed:    true,
				Description: "The username that authenticates to the receiver system.",
			},
			"company_id": schema.StringAttribute{
				Computed:    true,
				Description: "The SuccessFactors company ID, when kind is \"SuccessFactors\".",
			},
		},
	}
}

func (d *userCredentialDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = securitycontent.New(data.HTTPClient, data.Host)
}

func (d *userCredentialDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userCredentialDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(d.client, config.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	cred, err := client.GetUserCredential(ctx, config.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite user credential", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, userCredentialDataSourceModel{
		ID:                types.StringValue(cred.Name),
		RuntimeLocationID: config.RuntimeLocationID,
		Kind:              stringOrNull(cred.Kind),
		Description:       stringOrNull(cred.Description),
		User:              types.StringValue(cred.User),
		CompanyID:         stringOrNull(cred.CompanyID),
	})...)
}
