// Package provider implements the sapintegrationsuite Terraform provider.
package provider

import (
	"context"
	"fmt"
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
	Host                  types.String                `tfsdk:"host"`
	EnableExperimental    types.Bool                  `tfsdk:"enable_experimental"`
	EnableUnofficial      types.Bool                  `tfsdk:"enable_unofficial"`
	ConvertUILabels       types.Bool                  `tfsdk:"convert_ui_labels"`
	OAuth                 *oauthModel                 `tfsdk:"oauth"`
	APIManagement         *apiManagementModel         `tfsdk:"api_management"`
	APIComposition        *apiCompositionModel        `tfsdk:"api_composition"`
	IntegrationAssessment *integrationAssessmentModel `tfsdk:"integration_assessment"`
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

type integrationAssessmentModel struct {
	EntitiesURL  types.String `tfsdk:"entities_url"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// apiCompositionModel mirrors the optional api_composition block. API
// Composition's Configuration API has its own region-specific host and its
// own OAuth client, which come from an instance of the API Composition
// service with plan "configuration". SAP does not document the field names
// of that instance's service key, so the four values are taken as they are
// rather than parsed from a key file. Username, Password and Origin log a
// key user in through that client (password grant): on a tenant only such
// a token carried the API's scope. See docs/guides/api-composition.md.
type apiCompositionModel struct {
	Host         types.String `tfsdk:"host"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
	Username     types.String `tfsdk:"username"`
	Password     types.String `tfsdk:"password"`
	Origin       types.String `tfsdk:"origin"`
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

	// IntegrationAssessmentURL and IntegrationAssessmentHTTPClient are the
	// Entities API service root and client of Integration Assessment, set
	// only when provider.integration_assessment (or its environment
	// variables) is complete.
	IntegrationAssessmentURL        string
	IntegrationAssessmentHTTPClient *sapthttp.Client

	// EnableExperimental and EnableUnofficial are the provider's opt-ins
	// for resources and data sources whose catalog status is experimental
	// or unofficial; see requireOptIn.
	EnableExperimental bool
	EnableUnofficial   bool

	// ConvertUILabels is the provider's convert_ui_labels. It takes effect
	// only together with EnableExperimental; see uiLabelConversion.
	ConvertUILabels bool
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
					"otherwise documented resources, for example changing an access policy's description " +
					"in place or deleting a number range. Off by default; a configuration or plan that " +
					"needs one fails until this is true. Can also be set via the " +
					"SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL environment variable.",
			},
			"convert_ui_labels": schema.BoolAttribute{
				Optional: true,
				Description: "Lets sapintegrationsuite_access_policy_reference accept the labels SAP's UI " +
					"shows where SAP's API stores constants: \"Matches\" and \"Equals\" as operator, " +
					"\"name\" or \"Id\" as attribute, and an artifact type label that spells exactly its " +
					"constant, such as \"Integration Flow\" for INTEGRATION_FLOW. The provider sends the " +
					"constant and shows each conversion as a plan warning; the state keeps your spelling. " +
					"Only these proven conversions are made; other labels are still rejected with the " +
					"value to use. Experimental: takes effect only together with enable_experimental = " +
					"true. Off by default. Can also be set via the SAP_INTEGRATION_SUITE_CONVERT_UI_LABELS " +
					"environment variable.",
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
					"(API Providers, API Products, Key Value Maps, Certificate Store References) authenticates against its " +
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
			"integration_assessment": schema.SingleNestedBlock{
				Description: "Credentials for the Integration Assessment Entities API, used only by the " +
					"sapintegrationsuite_integration_assessment_* resources and data sources. They come from a " +
					"service key of a service instance of \"Integration Assessment APIs\" (plan default); the " +
					"oauth, api_management and api_composition credentials do not work there. Set all four " +
					"values, or none. Each can also come from a SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_* " +
					"environment variable.",
				Attributes: map[string]schema.Attribute{
					"entities_url": schema.StringAttribute{
						Optional: true,
						Description: "Service root of the Entities API, the service key's \"entities\" value, for " +
							"example https://intas-api.cfapps.eu10.hana.ondemand.com/intas/entities/v1. Environment " +
							"variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_ENTITIES_URL.",
					},
					"token_url": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 token endpoint: the service key's \"url\" followed by /oauth/token. " +
							"Environment variable: SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_TOKEN_URL.",
					},
					"client_id": schema.StringAttribute{
						Optional: true,
						Description: "OAuth 2.0 client ID from the service key. Environment variable: " +
							"SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_ID.",
					},
					"client_secret": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
						Description: "OAuth 2.0 client secret from the service key. Environment variable: " +
							"SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_CLIENT_SECRET.",
					},
				},
			},
			"api_composition": schema.SingleNestedBlock{
				Description: "Credentials for API Composition's Configuration API, used only by " +
					"sapintegrationsuite_business_data_graph. The API has its own region-specific host " +
					"and OAuth client, from a service key of an API Composition service instance with " +
					"plan \"configuration\"; the oauth and api_management credentials do not work " +
					"there. Set host, token_url, client_id and client_secret together, or none. " +
					"username and password add a key user's login through that client, which the " +
					"Configuration API needed on a tenant. Each value can also come from a " +
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
					"username": schema.StringAttribute{
						Optional: true,
						Description: "User with the role collection Graph.KeyUser. With username and password the " +
							"provider requests the token with the password grant through the service key's client, " +
							"so it carries the user's roles. The identity provider must accept passwords " +
							"without a second factor. Set together with password. Environment variable: " +
							"SAP_INTEGRATION_SUITE_API_COMPOSITION_USERNAME.",
					},
					"password": schema.StringAttribute{
						Optional:  true,
						Sensitive: true,
						Description: "Password of username. Environment variable: " +
							"SAP_INTEGRATION_SUITE_API_COMPOSITION_PASSWORD.",
					},
					"origin": schema.StringAttribute{
						Optional: true,
						Description: "Origin key of the user's identity provider in the subaccount (Security, " +
							"Trust Configuration), sent as login_hint. Leave it out for the default identity " +
							"provider. Environment variable: SAP_INTEGRATION_SUITE_API_COMPOSITION_ORIGIN.",
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
		ConvertUILabels:    boolOrEnv(config.ConvertUILabels, "SAP_INTEGRATION_SUITE_CONVERT_UI_LABELS"),
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

	if !p.configureIntegrationAssessment(ctx, config.IntegrationAssessment, data, &resp.Diagnostics) {
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

// credentialBlock describes one optional provider block with its own OAuth
// client: api_composition and integration_assessment.
type credentialBlock struct {
	title     string // for messages, for example "API Composition"
	block     string // the provider block, for example "api_composition"
	urlAttr   string // the block's URL attribute, for example "host"
	envPrefix string // for example "SAP_INTEGRATION_SUITE_API_COMPOSITION_"
	usedBy    string // what stays unconfigured without it
}

// userLogin is the optional key user of a credential block. With it, the
// block's client requests tokens with the password grant.
type userLogin struct {
	username, password, origin types.String
}

// resolve reads the block's four values, each falling back to its
// environment variable, and builds the authenticated HTTP client when all
// four are present. It returns a nil client when none is set, and fails only
// when some but not all are. user, if not nil, adds the block's optional
// username, password and origin.
func (b credentialBlock) resolve(ctx context.Context, version string, url, tokenURL, clientID, clientSecret types.String, user *userLogin, diags *diag.Diagnostics) (string, *sapthttp.Client, bool) {
	values := []string{
		stringOrEnv(url, b.envPrefix+strings.ToUpper(b.urlAttr)),
		stringOrEnv(tokenURL, b.envPrefix+"TOKEN_URL"),
		stringOrEnv(clientID, b.envPrefix+"CLIENT_ID"),
		stringOrEnv(clientSecret, b.envPrefix+"CLIENT_SECRET"),
	}
	present := 0
	for _, v := range values {
		if v != "" {
			present++
		}
	}
	if present == 0 {
		return "", nil, true
	}
	if present < len(values) {
		diags.AddError(
			"Incomplete "+b.title+" configuration",
			fmt.Sprintf("provider.%s requires %s, token_url, client_id, and client_secret (or their "+
				"%s* environment variable equivalents) to be supplied together. Supply all four, or omit "+
				"the block entirely to leave %s unconfigured.", b.block, b.urlAttr, b.envPrefix, b.usedBy),
		)
		return "", nil, false
	}
	authConfig := auth.Config{
		TokenURL:     values[1],
		ClientID:     values[2],
		ClientSecret: values[3],
	}
	if user != nil {
		authConfig.Username = stringOrEnv(user.username, b.envPrefix+"USERNAME")
		authConfig.Password = stringOrEnv(user.password, b.envPrefix+"PASSWORD")
		authConfig.Origin = stringOrEnv(user.origin, b.envPrefix+"ORIGIN")
		if (authConfig.Username == "") != (authConfig.Password == "") || (authConfig.Origin != "" && authConfig.Username == "") {
			diags.AddError(
				"Incomplete "+b.title+" user login",
				fmt.Sprintf("provider.%s requires username and password together (or %sUSERNAME and "+
					"%sPASSWORD), and origin only with them. Supply both to log in as a user, or neither "+
					"to use the client's own credentials.", b.block, b.envPrefix, b.envPrefix),
			)
			return "", nil, false
		}
	}
	authenticatedClient, invalidateToken, err := authConfig.HTTPClient(ctx, http.DefaultClient)
	if err != nil {
		diags.AddError("Unable to configure "+b.title+" authentication", err.Error())
		return "", nil, false
	}
	return values[0], sapthttp.New(sapthttp.Config{
		Transport:       authenticatedClient,
		UserAgent:       sapthttp.UserAgent(version),
		InvalidateToken: invalidateToken,
	}), true
}

// configureAPIComposition resolves the optional api_composition block for
// API Composition's Configuration API.
func (p *sapIntegrationSuiteProvider) configureAPIComposition(ctx context.Context, cfg *apiCompositionModel, data *Data, diags *diag.Diagnostics) bool {
	var host, tokenURL, clientID, clientSecret types.String
	user := &userLogin{}
	if cfg != nil {
		host, tokenURL, clientID, clientSecret = cfg.Host, cfg.TokenURL, cfg.ClientID, cfg.ClientSecret
		user = &userLogin{username: cfg.Username, password: cfg.Password, origin: cfg.Origin}
	}
	block := credentialBlock{
		title: "API Composition", block: "api_composition", urlAttr: "host",
		envPrefix: "SAP_INTEGRATION_SUITE_API_COMPOSITION_", usedBy: "sapintegrationsuite_business_data_graph",
	}
	resolved, client, ok := block.resolve(ctx, p.version, host, tokenURL, clientID, clientSecret, user, diags)
	data.APICompositionHost, data.APICompositionHTTPClient = resolved, client
	return ok
}

// configureIntegrationAssessment resolves the optional
// integration_assessment block for the Integration Assessment Entities API.
func (p *sapIntegrationSuiteProvider) configureIntegrationAssessment(ctx context.Context, cfg *integrationAssessmentModel, data *Data, diags *diag.Diagnostics) bool {
	var entitiesURL, tokenURL, clientID, clientSecret types.String
	if cfg != nil {
		entitiesURL, tokenURL, clientID, clientSecret = cfg.EntitiesURL, cfg.TokenURL, cfg.ClientID, cfg.ClientSecret
	}
	block := credentialBlock{
		title: "Integration Assessment", block: "integration_assessment", urlAttr: "entities_url",
		envPrefix: "SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_", usedBy: "the integration_assessment resources and data sources",
	}
	resolved, client, ok := block.resolve(ctx, p.version, entitiesURL, tokenURL, clientID, clientSecret, nil, diags)
	data.IntegrationAssessmentURL, data.IntegrationAssessmentHTTPClient = resolved, client
	return ok
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
		NewDataTypeResource,
		NewMessageTypeResource,
		NewServiceInterfaceResource,
		NewFaultMessageTypeResource,
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
		NewKeyPairCertificateChainResource,
		NewPGPPublicKeyResource,
		NewPGPSecretKeyResource,
		NewAPIProviderResource,
		NewAPIProductResource,
		NewAPIManagementCertificateStoreReferenceResource,
		NewAPIKeyValueMapResource,
		NewBusinessDataGraphResource,
		NewIntegrationAssessmentVendorResource,
		NewIntegrationAssessmentApplicationResource,
		NewIntegrationAssessmentApplicationInstanceResource,
		NewIntegrationAssessmentTechnologyResource,
		NewIntegrationAssessmentTechnologyInstanceResource,
		NewIntegrationAssessmentTechnologyDomainResource,
		NewIntegrationAssessmentTechnologyStyleResource,
		NewIntegrationAssessmentTechnologyKeyCharacteristicResource,
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
		NewIntegrationAssessmentDeploymentModelDataSource,
		NewIntegrationAssessmentVendorDataSource,
		NewIntegrationAssessmentTechnologyDataSource,
		NewIntegrationAssessmentDomainDataSource,
		NewIntegrationAssessmentStyleDataSource,
		NewIntegrationAssessmentKeyCharacteristicValueDataSource,
		NewIntegrationAssessmentRecommendationDegreeDataSource,
		NewIntegrationAssessmentUseCasePatternDataSource,
		NewIntegrationAssessmentIntegrationPatternDataSource,
		NewIntegrationAssessmentKeyCharacteristicGroupDataSource,
		NewIntegrationAssessmentDomainDeterminationDataSource,
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
