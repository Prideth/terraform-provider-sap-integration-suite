// Package provider implements the sapintegrationsuite Terraform provider.
package provider

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/auth"
	sapthttp "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/http"
)

// Version is set at build time via -ldflags by GoReleaser; it defaults to
// "dev" for local builds.
var Version = "dev"

// New returns a provider.Provider factory for the terraform-plugin-framework
// server, capturing the build version for the User-Agent header.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &sapIntegrationSuiteProvider{version: version}
	}
}

type sapIntegrationSuiteProvider struct {
	version string
}

// providerModel mirrors the provider block's schema.
type providerModel struct {
	Host               types.String         `tfsdk:"host"`
	EnableExperimental types.Bool           `tfsdk:"enable_experimental"`
	EnableUnofficial   types.Bool           `tfsdk:"enable_unofficial"`
	OAuth              *oauthModel          `tfsdk:"oauth"`
	APIManagement      *apiManagementModel  `tfsdk:"api_management"`
	APIComposition     *apiCompositionModel `tfsdk:"api_composition"`
}

type oauthModel struct {
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// apiManagementModel mirrors the provider block's optional api_management
// nested block: Classic API Management (the API Portal) authenticates
// against its own application URL and OAuth client (the apiportal-apiaccess
// service plan), entirely independent of the oauth block above, which only
// ever authenticates Cloud Integration and current API Management calls.
type apiManagementModel struct {
	Host         types.String `tfsdk:"host"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// apiCompositionModel mirrors the optional api_composition block. API
// Composition's Configuration API has its own region-specific host and its
// own OAuth client, which come from an instance of the API Composition
// service with plan "configuration". SAP does not document the field names
// of that instance's service key, so the four values are taken as they are
// rather than parsed from a key file. See docs/guides/api-composition.md.
type apiCompositionModel struct {
	Host         types.String `tfsdk:"host"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// Data is the fully resolved provider configuration made available to every
// resource and data source via Configure. Capability-specific clients (Cloud
// Integration, API Management, ...) are built lazily by each resource from
// these shared ingredients, rather than eagerly during provider Configure,
// so that a tenant missing one capability does not block using another.
type Data struct {
	Host       string
	HTTPClient *sapthttp.Client
	Version    string

	// APIManagementClassicHost and APIManagementClassicHTTPClient are the
	// Classic API Management (API Portal) equivalents of Host/HTTPClient
	// above, populated only when provider.api_management (or the
	// corresponding SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment
	// variables) is fully configured. They are never derived from, or
	// substituted with, the Cloud Integration Host/HTTPClient: SAP does not
	// document the two credential sets as interchangeable.
	APIManagementClassicHost       string
	APIManagementClassicHTTPClient *sapthttp.Client

	// APICompositionHost and APICompositionHTTPClient are the API
	// Composition Configuration API counterparts of Host and HTTPClient,
	// set only when provider.api_composition (or its environment
	// variables) is complete. They are never derived from the other two
	// credential sets.
	APICompositionHost       string
	APICompositionHTTPClient *sapthttp.Client

	// EnableExperimental and EnableUnofficial are the provider's opt-ins
	// for resources and data sources whose catalog status is experimental
	// or unofficial; see requireOptIn.
	EnableExperimental bool
	EnableUnofficial   bool
}

func (p *sapIntegrationSuiteProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "sapintegrationsuite"
	resp.Version = p.version
}

func (p *sapIntegrationSuiteProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Configures the SAP Integration Suite provider. This provider administers " +
			"content and capabilities inside an already-provisioned SAP Integration Suite tenant; " +
			"it does not create BTP subaccounts, entitlements, or the Integration Suite subscription " +
			"itself. Use the official SAP BTP provider for those.",
		Attributes: map[string]schema.Attribute{
			"enable_experimental": schema.BoolAttribute{
				Optional: true,
				Description: "Allows resources and data sources whose support status is " +
					"\"experimental\": implemented on a documented API, but their lifecycle has not " +
					"yet passed an acceptance test on a tenant, so behavior or schema may still " +
					"change. Off by default; a configuration that uses one fails until this is true. " +
					"Can also be set via the SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL environment " +
					"variable. See docs/feature-support.md for which ones they are.",
			},
			"enable_unofficial": schema.BoolAttribute{
				Optional: true,
				Description: "Allows resources and data sources whose support status is " +
					"\"unofficial\": they work and were verified on a tenant, but SAP does not " +
					"document the API behind them (it is known only from the service's $metadata), " +
					"so SAP may change it without notice. It also allows the unofficial operations of " +
					"otherwise documented resources, for example updating a message mapping's content " +
					"in place or deleting a number range. Off by default; a configuration or plan that " +
					"needs one fails until this is true. Can also be set via the " +
					"SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL environment variable.",
			},
			"host": schema.StringAttribute{
				Optional: true,
				Description: "Base URL of the SAP Integration Suite tenant used for Cloud Integration " +
					"APIs, for example https://<tenant>.it-cpi<...>.cfapps.<region>.hana.ondemand.com. " +
					"Can also be set via the SAP_INTEGRATION_SUITE_HOST environment variable.",
			},
		},
		Blocks: map[string]schema.Block{
			"oauth": schema.SingleNestedBlock{
				Description: "OAuth 2.0 client credentials used to authenticate against the SAP " +
					"Integration Suite APIs.",
				Attributes: map[string]schema.Attribute{
					"token_url": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 token endpoint URL. Can also be set via the " +
							"SAP_INTEGRATION_SUITE_TOKEN_URL environment variable.",
					},
					"client_id": schema.StringAttribute{
						Optional:    true,
						Description: "OAuth 2.0 client ID. Can also be set via the SAP_INTEGRATION_SUITE_CLIENT_ID environment variable.",
					},
					"client_secret": schema.StringAttribute{
						Optional:    true,
						Sensitive:   true,
						Description: "OAuth 2.0 client secret. Can also be set via the SAP_INTEGRATION_SUITE_CLIENT_SECRET environment variable.",
					},
				},
			},
			"api_management": schema.SingleNestedBlock{
				Description: "Optional, and independent of the oauth block above. Classic API Management " +
					"(API Providers, API Proxies, API Products, Key Value Maps) authenticates against its " +
					"own API Portal application URL and its own OAuth 2.0 client, generated from the " +
					"apiportal-apiaccess service plan — never the Cloud Integration credentials configured " +
					"above. Leave this entire block out if you do not use any sapintegrationsuite_api_provider, " +
					"sapintegrationsuite_api_product, sapintegrationsuite_api_key_value_map, or " +
					"sapintegrationsuite_api_management_certificate_store_reference resource or data source. " +
					"All four values (or their SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment variable " +
					"equivalents) must be supplied together, or all left unset.",
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Optional: true,
						Description: "Base URL of the API Portal application, for example " +
							"https://<tenant>.prod-eu10.apiportal.cfapps.eu10.hana.ondemand.com, as returned " +
							"by the apiportal-apiaccess service key's \"url\" field. Can also be set via the " +
							"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST environment variable.",
					},
					"token_url": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 token endpoint URL from the apiportal-apiaccess service key's " +
							"\"tokenUrl\" field. Can also be set via the " +
							"SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL environment variable.",
					},
					"client_id": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 client ID from the apiportal-apiaccess service key. Can also " +
							"be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID environment variable.",
					},
					"client_secret": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
						Description: "OAuth 2.0 client secret from the apiportal-apiaccess service key. Can " +
							"also be set via the SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET " +
							"environment variable.",
					},
				},
			},
			"api_composition": schema.SingleNestedBlock{
				Description: "Credentials for API Composition's Configuration API, used only by " +
					"sapintegrationsuite_business_data_graph. The API has its own region-specific host " +
					"and OAuth client, from a service key of an API Composition service instance with " +
					"plan \"configuration\"; the oauth and api_management credentials do not work " +
					"there. Set all four values, or none. Each can also come from a " +
					"SAP_INTEGRATION_SUITE_API_COMPOSITION_* environment variable.",
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Optional: true,
						Description: "Region-specific API Composition host from the service key, for " +
							"example https://eu10.graph.sap. The provider appends " +
							"/configuration/v1/sap.graph. Environment variable: " +
							"SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST.",
					},
					"token_url": schema.StringAttribute{
						Optional: true,
						Description: "Full OAuth 2.0 token endpoint URL, ending in /oauth/token. If the " +
							"service key only has the authentication server URL, append /oauth/token. " +
							"Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_TOKEN_URL.",
					},
					"client_id": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 client ID from the service key. Environment variable: " +
							"SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_ID.",
					},
					"client_secret": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
						Description: "OAuth 2.0 client secret from the service key. Environment variable: " +
							"SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_SECRET.",
					},
				},
			},
		},
	}
}

// Configure resolves whatever SAP connectivity configuration is available,
// but never fails Configure itself just because it is incomplete or absent.
// The provider feature catalog data sources (sapintegrationsuite_provider_features,
// sapintegrationsuite_provider_feature) describe this provider binary itself
// and need no SAP tenant connection at all; requiring SAP credentials here
// would make them unusable exactly where they are most useful (a first
// look at what this provider can do, offline, before any tenant exists).
// Every SAP-backed resource and data source instead checks
// Data.HTTPClient itself in its own Configure method and returns a clear,
// specific configuration error if it is nil — see requireHTTPClient in
// helpers.go.
func (p *sapIntegrationSuiteProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := stringOrEnv(config.Host, "SAP_INTEGRATION_SUITE_HOST")

	var tokenURL, clientID, clientSecret types.String
	if config.OAuth != nil {
		tokenURL = config.OAuth.TokenURL
		clientID = config.OAuth.ClientID
		clientSecret = config.OAuth.ClientSecret
	}

	oauthCfg := auth.Config{
		TokenURL:     stringOrEnv(tokenURL, "SAP_INTEGRATION_SUITE_TOKEN_URL"),
		ClientID:     stringOrEnv(clientID, "SAP_INTEGRATION_SUITE_CLIENT_ID"),
		ClientSecret: stringOrEnv(clientSecret, "SAP_INTEGRATION_SUITE_CLIENT_SECRET"),
	}

	data := &Data{
		Version:            p.version,
		EnableExperimental: boolOrEnv(config.EnableExperimental, "SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL"),
		EnableUnofficial:   boolOrEnv(config.EnableUnofficial, "SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL"),
	}

	if host != "" && oauthCfg.TokenURL != "" && oauthCfg.ClientID != "" && oauthCfg.ClientSecret != "" {
		authenticatedClient, invalidateToken, err := oauthCfg.HTTPClient(ctx, http.DefaultClient)
		if err != nil {
			resp.Diagnostics.AddError("Unable to configure SAP Integration Suite authentication", err.Error())
			return
		}

		data.Host = host
		data.HTTPClient = sapthttp.New(sapthttp.Config{
			Transport:       authenticatedClient,
			UserAgent:       sapthttp.UserAgent(p.version),
			InvalidateToken: invalidateToken,
		})
	}

	if !p.configureAPIManagementClassic(ctx, config.APIManagement, data, &resp.Diagnostics) {
		return
	}

	if !p.configureAPIComposition(ctx, config.APIComposition, data, &resp.Diagnostics) {
		return
	}

	resp.DataSourceData = data
	resp.ResourceData = data
}

// configureAPIManagementClassic resolves the optional api_management block
// (or its environment variable equivalents) and, when fully supplied,
// builds a dedicated authenticated HTTP client for Classic API Management —
// entirely independent of the Cloud Integration client above. It returns
// false (after recording a diagnostic) only when some but not all of the
// four required values are present; an empty configuration is not an
// error, since api_management is optional and every existing resource and
// data source must keep working unaffected by its absence.
func (p *sapIntegrationSuiteProvider) configureAPIManagementClassic(ctx context.Context, cfg *apiManagementModel, data *Data, diags *diag.Diagnostics) bool {
	var host, tokenURL, clientID, clientSecret types.String
	if cfg != nil {
		host = cfg.Host
		tokenURL = cfg.TokenURL
		clientID = cfg.ClientID
		clientSecret = cfg.ClientSecret
	}

	resolvedHost := stringOrEnv(host, "SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST")
	resolvedTokenURL := stringOrEnv(tokenURL, "SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL")
	resolvedClientID := stringOrEnv(clientID, "SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID")
	resolvedClientSecret := stringOrEnv(clientSecret, "SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET")

	present := 0
	for _, v := range []string{resolvedHost, resolvedTokenURL, resolvedClientID, resolvedClientSecret} {
		if v != "" {
			present++
		}
	}
	if present == 0 {
		return true
	}
	if present < 4 {
		diags.AddError(
			"Incomplete Classic API Management configuration",
			"provider.api_management requires host, token_url, client_id, and client_secret (or their "+
				"SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment variable equivalents) to be supplied "+
				"together. Supply all four, or omit the block entirely to leave Classic API Management "+
				"resources and data sources unconfigured.",
		)
		return false
	}

	authenticatedClient, invalidateToken, err := auth.Config{
		TokenURL:     resolvedTokenURL,
		ClientID:     resolvedClientID,
		ClientSecret: resolvedClientSecret,
	}.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		diags.AddError("Unable to configure Classic API Management authentication", err.Error())
		return false
	}

	data.APIManagementClassicHost = resolvedHost
	data.APIManagementClassicHTTPClient = sapthttp.New(sapthttp.Config{
		Transport:       authenticatedClient,
		UserAgent:       sapthttp.UserAgent(p.version),
		InvalidateToken: invalidateToken,
	})
	return true
}

// configureAPIComposition resolves the optional api_composition block and
// its environment variables and, when all four values are present, builds
// the HTTP client for API Composition's Configuration API. It fails only
// when some but not all values are set; leaving all of them out is fine.
func (p *sapIntegrationSuiteProvider) configureAPIComposition(ctx context.Context, cfg *apiCompositionModel, data *Data, diags *diag.Diagnostics) bool {
	var host, tokenURL, clientID, clientSecret types.String
	if cfg != nil {
		host = cfg.Host
		tokenURL = cfg.TokenURL
		clientID = cfg.ClientID
		clientSecret = cfg.ClientSecret
	}

	resolvedHost := stringOrEnv(host, "SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST")
	resolvedTokenURL := stringOrEnv(tokenURL, "SAP_INTEGRATION_SUITE_API_COMPOSITION_TOKEN_URL")
	resolvedClientID := stringOrEnv(clientID, "SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_ID")
	resolvedClientSecret := stringOrEnv(clientSecret, "SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_SECRET")

	present := 0
	for _, v := range []string{resolvedHost, resolvedTokenURL, resolvedClientID, resolvedClientSecret} {
		if v != "" {
			present++
		}
	}
	if present == 0 {
		return true
	}
	if present < 4 {
		diags.AddError(
			"Incomplete API Composition configuration",
			"provider.api_composition requires host, token_url, client_id, and client_secret (or their "+
				"SAP_INTEGRATION_SUITE_API_COMPOSITION_* environment variable equivalents) to be supplied "+
				"together. Supply all four, or omit the block entirely to leave "+
				"sapintegrationsuite_business_data_graph unconfigured.",
		)
		return false
	}

	authenticatedClient, invalidateToken, err := auth.Config{
		TokenURL:     resolvedTokenURL,
		ClientID:     resolvedClientID,
		ClientSecret: resolvedClientSecret,
	}.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		diags.AddError("Unable to configure API Composition authentication", err.Error())
		return false
	}

	data.APICompositionHost = resolvedHost
	data.APICompositionHTTPClient = sapthttp.New(sapthttp.Config{
		Transport:       authenticatedClient,
		UserAgent:       sapthttp.UserAgent(p.version),
		InvalidateToken: invalidateToken,
	})
	return true
}

func (p *sapIntegrationSuiteProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewIntegrationPackageResource,
		NewIntegrationFlowResource,
		NewIntegrationFlowDeploymentResource,
		NewIntegrationFlowConfigurationResource,
		NewAccessPolicyResource,
		NewAccessPolicyReferenceResource,
		NewValueMappingResource,
		NewValueMappingDeploymentResource,
		NewMessageMappingResource,
		NewMessageMappingDeploymentResource,
		NewScriptCollectionResource,
		NewScriptCollectionDeploymentResource,
		NewPartnerStringParameterResource,
		NewPartnerBinaryParameterResource,
		NewAlternativePartnerResource,
		NewPartnerAuthorizedUserResource,
		NewPartnerUserCredentialParameterResource,
		NewUserCredentialResource,
		NewSecureParameterResource,
		NewOAuth2ClientCredentialResource,
		NewIntegrationAdapterResource,
		NewIntegrationAdapterDeploymentResource,
		NewCustomTagConfigurationResource,
		NewNumberRangeResource,
		NewCertificateResource,
		NewKeyPairResource,
		NewAPIProviderResource,
		NewAPIProductResource,
		NewAPIManagementCertificateStoreReferenceResource,
		NewAPIKeyValueMapResource,
		NewBusinessDataGraphResource,
	}
}

func (p *sapIntegrationSuiteProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewIntegrationPackageDataSource,
		NewValueMappingDataSource,
		NewMessageMappingDataSource,
		NewScriptCollectionDataSource,
		NewAccessPolicyDataSource,
		NewAccessPolicyReferenceDataSource,
		NewAccessPolicyRuntimeAssignmentsDataSource,
		NewPartnerDataSource,
		NewPartnersDataSource,
		NewPartnerStringParameterDataSource,
		NewPartnerStringParametersDataSource,
		NewPartnerBinaryParameterDataSource,
		NewAlternativePartnerDataSource,
		NewPartnerAuthorizedUserDataSource,
		NewProviderFeaturesDataSource,
		NewProviderFeatureDataSource,
		NewUserCredentialDataSource,
		NewOAuth2ClientCredentialDataSource,
		NewServiceEndpointsDataSource,
		NewIntegrationAdapterDataSource,
		NewCustomTagConfigurationDataSource,
		NewKeystoreEntryDataSource,
		NewKeystoreEntriesDataSource,
		NewAPIProviderDataSource,
		NewAPIProvidersDataSource,
		NewAPIProductDataSource,
		NewAPIManagementCertificateStoreReferenceDataSource,
		NewAPIKeyValueMapDataSource,
		NewBusinessDataGraphDataSource,
	}
}

// stringOrEnv returns value's string contents if it is known and non-empty,
// otherwise falls back to the named environment variable.
// boolOrEnv returns the configured value, or else the environment variable
// read as a boolean ("true", "1", ...); anything unset or unparsable is false.
func boolOrEnv(value types.Bool, envVar string) bool {
	if !value.IsNull() && !value.IsUnknown() {
		return value.ValueBool()
	}
	b, err := strconv.ParseBool(strings.TrimSpace(os.Getenv(envVar)))
	return err == nil && b
}

func stringOrEnv(value types.String, envVar string) string {
	if !value.IsNull() && !value.IsUnknown() && value.ValueString() != "" {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}
