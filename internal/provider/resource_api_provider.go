package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIProviderResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_api_provider.
func NewAPIProviderResource() resource.Resource {
	return &apiProviderResource{}
}

type apiProviderResource struct {
	client *apimanagementclassic.Client
}

type apiProviderModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Title                types.String `tfsdk:"title"`
	Description          types.String `tfsdk:"description"`
	Host                 types.String `tfsdk:"host"`
	Port                 types.Int64  `tfsdk:"port"`
	UseSSL               types.Bool   `tfsdk:"use_ssl"`
	TrustAll             types.Bool   `tfsdk:"trust_all"`
	PathPrefix           types.String `tfsdk:"path_prefix"`
	ServiceCollectionURL types.String `tfsdk:"service_collection_url"`
	AuthType             types.String `tfsdk:"auth_type"`
	UserName             types.String `tfsdk:"user_name"`
	PasswordWO           types.String `tfsdk:"password_wo"`
}

func (r *apiProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_provider"
}

func (r *apiProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		Description: "Manages a Classic API Management API Provider (APIProviders), the backend " +
			"connection definition an API Proxy targets. Backed by the API Portal's " +
			"Management.svc OData API — see provider.api_management for the separate " +
			"apiportal-apiaccess credentials this resource requires.\n\n" +
			"IMPORTANT — this resource has no Update: SAP's own tooling (the official Piper " +
			"apiProviderUpload pipeline step) documents that only Create is supported for API " +
			"Providers through this API; an existing provider must be deleted and recreated to " +
			"change anything. Every attribute is therefore RequiresReplace, and this provider does " +
			"not guess at an unconfirmed PUT/PATCH.\n\n" +
			"IMPORTANT — this resource only supports the \"Internet\" connection type (a direct " +
			"host/port connection, optionally over SSL). SAP documents three further connection " +
			"types (On Premise via Cloud Connector, Open Connectors, Cloud Integration) with their " +
			"own distinct field sets, none of which this provider could confirm a field-level JSON " +
			"mapping for from a reachable primary source — see docs/sap-api-references.md. Configure " +
			"those connection types through the SAP Integration Suite UI instead.\n\n" +
			"IMPORTANT — eventual consistency: SAP's own documentation states that changes made " +
			"through this API may take up to approximately 20 seconds to be reflected in subsequent " +
			"GET requests, due to response caching in front of the destination service. This " +
			"resource's Create waits (with bounded, jittered backoff, never a fixed sleep) for the " +
			"new provider to become visible before returning, so a chained apply referencing it does " +
			"not race that window; Read shortly after Create or Import may still occasionally need a " +
			"retry.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Always equal to name — SAP's confirmed OData key for this entity (APIProviders('<name>')).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "The API provider's name. SAP documents a short list of reserved names " +
					"that cannot be used (DEST_CI, APIMGMT_PORTAL, ContentCatalog, ON_PREM, Apimgmt_rt, " +
					"Apimgmt_rt*, TC, TC_*) — this provider does not validate that list itself, since " +
					"SAP may extend it; a reserved name simply fails at apply time with SAP's own error.",
				PlanModifiers: replace,
			},
			"title": schema.StringAttribute{
				Optional:      true,
				Description:   "Display title of the provider. Changing it replaces the provider.",
				PlanModifiers: replace,
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Description:   "Free-text description of the provider. Changing it replaces the provider.",
				PlanModifiers: replace,
			},
			"host": schema.StringAttribute{
				Required:      true,
				Description:   "The backend host to connect to, for example \"example.com\" (no scheme).",
				PlanModifiers: replace,
			},
			"port": schema.Int64Attribute{
				Optional:      true,
				Description:   "The backend port. SAP's UI documents 443 for SSL connections and 8080 otherwise as typical values, not enforced defaults.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"use_ssl": schema.BoolAttribute{
				Optional:      true,
				Description:   "Whether to secure the connection using SSL/TLS (HTTPS).",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"trust_all": schema.BoolAttribute{
				Optional: true,
				Description: "Whether API Management trusts all TLS certificates presented by the " +
					"backend without validating them against a truststore. SAP's UI documents this as " +
					"the default behavior when unset, but this provider does not assume that as a wire " +
					"default and always sends the value explicitly.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"path_prefix": schema.StringAttribute{
				Optional: true,
				Description: "Relative path prefix for SAP Gateway catalog service discovery (Catalog " +
					"Service Settings). Only relevant for SAP Gateway-enabled SAP NetWeaver backends.",
				PlanModifiers: replace,
			},
			"service_collection_url": schema.StringAttribute{
				Optional: true,
				Description: "Relative path from which the SAP Gateway catalog service should be " +
					"fetched, for example \"/CATALOGSERVICE/ServiceCollection\". Only relevant for SAP " +
					"Gateway-enabled SAP NetWeaver backends.",
				PlanModifiers: replace,
			},
			"auth_type": schema.StringAttribute{
				Optional: true,
				Description: "Authentication method for connection requests to the backend. The only " +
					"value this provider found confirmed in a worked example is \"BASIC\"; this " +
					"attribute is not validated against a closed enum, since SAP's full set of accepted " +
					"values is not confirmed.",
				PlanModifiers: replace,
			},
			"user_name": schema.StringAttribute{
				Optional:      true,
				Description:   "Username for auth_type = \"BASIC\" connections.",
				PlanModifiers: replace,
			},
			"password_wo": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				WriteOnly: true,
				Description: "Password for auth_type = \"BASIC\" connections. Write-only: Terraform " +
					"never stores this in plan or state. Since this resource has no Update, there is no " +
					"companion _wo_version attribute — any change to this value already requires " +
					"replacing the whole resource, the same as every other attribute here.",
			},
		},
	}
}

func (r *apiProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireAPIManagementClassicHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = apimanagementclassic.New(data.APIManagementClassicHTTPClient, data.APIManagementClassicHost)
}

func apiProviderToClient(plan apiProviderModel, password string) apimanagementclassic.APIProvider {
	return apimanagementclassic.APIProvider{
		Name:        plan.Name.ValueString(),
		Title:       plan.Title.ValueString(),
		Description: plan.Description.ValueString(),
		DestType:    "INTERNET",
		Host:        plan.Host.ValueString(),
		Port:        int(plan.Port.ValueInt64()),
		UseSSL:      plan.UseSSL.ValueBool(),
		TrustAll:    plan.TrustAll.ValueBool(),
		PathPrefix:  plan.PathPrefix.ValueString(),
		URL:         plan.ServiceCollectionURL.ValueString(),
		AuthType:    plan.AuthType.ValueString(),
		UserName:    plan.UserName.ValueString(),
		Password:    password,
	}
}

func (r *apiProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var password types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("password_wo"), &password)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAPIProvider(ctx, apiProviderToClient(plan, password.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Classic API Management API provider", diagnosticDetail(err))
		return
	}

	plan.ID = types.StringValue(created.Name)
	plan.PasswordWO = types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.GetAPIProvider(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Classic API Management API provider", diagnosticDetail(err))
		return
	}

	state.Name = types.StringValue(found.Name)
	state.Title = stringOrNull(found.Title)
	state.Description = stringOrNull(found.Description)
	state.Host = stringOrNull(found.Host)
	state.Port = int64OrNull(found.Port)
	state.UseSSL = types.BoolValue(found.UseSSL)
	state.TrustAll = types.BoolValue(found.TrustAll)
	state.PathPrefix = stringOrNull(found.PathPrefix)
	state.ServiceCollectionURL = stringOrNull(found.URL)
	state.AuthType = stringOrNull(found.AuthType)
	state.UserName = stringOrNull(found.UserName)
	// password_wo is write-only and never returned by GET; state.PasswordWO
	// is left as whatever Terraform already has (always null post-Create).

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update is never called: every attribute in Schema is RequiresReplace, so
// the Terraform Plugin Framework always routes a change through
// Delete+Create instead. This method exists only to satisfy the
// resource.Resource interface.
func (r *apiProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan apiProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIProvider(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Classic API Management API provider", diagnosticDetail(err))
	}
}

func (r *apiProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
