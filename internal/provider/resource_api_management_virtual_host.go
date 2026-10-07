package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

const apiManagementVirtualHostTypeName = "sapintegrationsuite_api_management_virtual_host"

// virtualHostAlias is SAP's rule for an alias, from the tenant's answer to
// a wrong one (400 VHR_VIRTUALHOST_ALIAS_FORMAT_INVALID): "only
// Alphanumerics and hyphens", no leading or trailing hyphen.
var virtualHostAlias = regexp.MustCompile(`^[A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?$`)

var _ resource.ResourceWithValidateConfig = &apiManagementVirtualHostResource{}

// virtualHostWait bounds the wait for a request to show in the host list.
const virtualHostWait = 2 * time.Minute

// NewAPIManagementVirtualHostResource returns the resource for
// sapintegrationsuite_api_management_virtual_host.
func NewAPIManagementVirtualHostResource() resource.Resource {
	return &apiManagementVirtualHostResource{}
}

type apiManagementVirtualHostResource struct {
	client    *apimanagementclassic.Client
	subdomain string
}

type apiManagementVirtualHostModel struct {
	ID       types.String `tfsdk:"id"`
	Alias    types.String `tfsdk:"alias"`
	HostName types.String `tfsdk:"host_name"`
	Port     types.Int64  `tfsdk:"port"`
	Default  types.Bool   `tfsdk:"default"`
	SSL      types.Bool   `tfsdk:"ssl"`

	ClientAuthEnabled types.Bool   `tfsdk:"client_auth_enabled"`
	TrustStore        types.String `tfsdk:"trust_store"`
}

func (r *apiManagementVirtualHostResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_management_virtual_host"
}

func (r *apiManagementVirtualHostResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a virtual host of Classic API Management on the tenant's default domain: an " +
			"additional host name, <alias>.<tenant domain>, under which the API portal exposes API proxies. " +
			"SAP Help documents the requests (Configuring a Default Domain for a Virtual Host): create, " +
			"update and delete are POST requests to Configuration.svc/VirtualHostRequests, and the hosts are " +
			"read from Management.svc/VirtualHosts.\n\n" +
			"With client_auth_enabled and trust_store, the host asks every client for a certificate and " +
			"checks it against a truststore of the API portal (mutual TLS, SAP Help: Configuring Mutual TLS " +
			"for Default Domain Virtual Host).\n\n" +
			"Needs provider.api_management_self_service, a key with the role " +
			"APIManagement.SelfService.Administrator; the api_management key is refused. Virtual hosts with " +
			"a custom domain or a keystore are not managed by this resource yet. SAP refuses to delete a " +
			"virtual host while it is the default one or while API proxies, drafts or revisions refer to it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The virtual host's ID (virtualHostId), assigned by SAP.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"alias": schema.StringAttribute{
				Required: true,
				Description: "The alias, the first label of the host name, for example prod-apis. Letters, digits " +
					"and hyphens, not starting or ending with a hyphen, at most 63 characters, and unique on the " +
					"tenant. Changing it renames the host in place; SAP then asks to redeploy and republish the " +
					"API proxies of products that use it.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, apimanagementclassic.VirtualHostAliasMaxLength),
					stringvalidator.RegexMatches(virtualHostAlias,
						"may only contain letters, digits and hyphens, and may not start or end with a hyphen"),
				},
			},
			"host_name": schema.StringAttribute{
				Computed:    true,
				Description: "The full host name SAP assigned, <alias>.<tenant domain>.",
			},
			"port": schema.Int64Attribute{
				Computed:      true,
				Description:   "The port of the virtual host, 443 on the tenant this was tested with.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"default": schema.BoolAttribute{
				Computed: true,
				Description: "Whether this is the API portal's default virtual host. The provider creates " +
					"additional hosts only; an imported default host keeps this flag on update.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"ssl": schema.BoolAttribute{
				Computed:      true,
				Description:   "Whether the virtual host serves HTTPS.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"client_auth_enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Mutual TLS: whether the host asks every client for a certificate and accepts only " +
					"those trust_store vouches for. Needs trust_store. Switching it on or off changes the host in " +
					"place; SAP then asks to redeploy and republish the API proxies of products that use it. " +
					"Default false.",
			},
			"trust_store": schema.StringAttribute{
				Optional: true,
				Description: "The truststore that holds the client certificates, or the certificates of the " +
					"CAs that issued them, for client_auth_enabled: the name of a truststore of the API portal, " +
					"or ref://<name> for a certificate store reference that points at one (see " +
					"sapintegrationsuite_api_management_certificate_store_reference). SAP asks for the whole " +
					"chain (client, intermediate and root certificates) in the truststore, and for client " +
					"certificates with the Client Authentication extended key usage. The truststore itself is " +
					"created in the SAP Integration Suite UI. Only allowed with client_auth_enabled.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
		},
	}
}

// ValidateConfig checks that trust_store and client_auth_enabled come
// together: SAP Help documents mutual TLS as client authentication against
// a truststore, and a truststore alone as nothing.
func (r *apiManagementVirtualHostResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config apiManagementVirtualHostModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.ClientAuthEnabled.IsUnknown() || config.TrustStore.IsUnknown() {
		return
	}
	enabled := config.ClientAuthEnabled.ValueBool()
	switch {
	case enabled && config.TrustStore.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("trust_store"), "Missing truststore",
			"client_auth_enabled = true needs trust_store: the truststore (or ref://<certificate store "+
				"reference>) the host checks client certificates against.")
	case !enabled && !config.TrustStore.IsNull():
		resp.Diagnostics.AddAttributeError(path.Root("trust_store"), "Truststore without client authentication",
			"trust_store is used only for mutual TLS; set client_auth_enabled = true or remove trust_store.")
	}
}

// planTLS is the mutual TLS setting of a plan, nil without client
// authentication.
func planTLS(plan apiManagementVirtualHostModel) *apimanagementclassic.VirtualHostTLS {
	if !plan.ClientAuthEnabled.ValueBool() {
		return nil
	}
	return &apimanagementclassic.VirtualHostTLS{ClientAuthEnabled: true, TrustStore: plan.TrustStore.ValueString()}
}

// tlsApplied reports whether the host shows the client authentication of
// tls (nil: off). The truststore is not compared: SAP lists it as sent, a
// ref:// reference included (tenant test, October 2026), and a difference
// would rather show as drift than as a timeout.
func tlsApplied(h *apimanagementclassic.VirtualHost, tls *apimanagementclassic.VirtualHostTLS) bool {
	return h.IsClientAuthEnabled == (tls != nil && tls.ClientAuthEnabled)
}

func (r *apiManagementVirtualHostResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, apiManagementVirtualHostTypeName, &resp.Diagnostics) {
		return
	}
	if !requireAPIManagementSelfServiceHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = apimanagementclassic.New(data.APIManagementSelfServiceHTTPClient, data.APIManagementSelfServiceHost)
	r.subdomain = data.APIManagementSelfServiceSubdomain
}

// requireSubdomain reports whether the accountId of SAP's requests is known.
func (r *apiManagementVirtualHostResource) requireSubdomain(diags *diag.Diagnostics) bool {
	if r.subdomain != "" {
		return true
	}
	diags.AddError("Subaccount subdomain unknown",
		"SAP's virtual host requests carry the subdomain of the subaccount as accountId. It could not be "+
			"taken from provider.api_management_self_service.token_url; set subaccount_subdomain in that block.")
	return false
}

func (r *apiManagementVirtualHostResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiManagementVirtualHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !r.requireSubdomain(&resp.Diagnostics) {
		return
	}

	tls := planTLS(plan)
	created, err := r.client.CreateVirtualHost(ctx, r.subdomain, plan.Alias.ValueString(), tls)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Classic API Management virtual host", diagnosticDetail(err))
		return
	}
	if created.VirtualHostID == "" {
		resp.Diagnostics.AddError("Failed to create Classic API Management virtual host",
			"SAP answered the create request without a virtualHostId.")
		return
	}
	// Saved from SAP's answer before the wait, so that a host SAP created is
	// never lost from state; the read below replaces it.
	resp.Diagnostics.Append(resp.State.Set(ctx, apiManagementVirtualHostModel{
		ID:                types.StringValue(created.VirtualHostID),
		Alias:             plan.Alias,
		HostName:          stringOrNull(created.VirtualHostURL),
		Port:              types.Int64Value(int64(created.AllocatedPort)),
		Default:           types.BoolValue(false),
		SSL:               types.BoolNull(),
		ClientAuthEnabled: plan.ClientAuthEnabled,
		TrustStore:        plan.TrustStore,
	})...)

	host := r.wait(ctx, created.VirtualHostID, func(h *apimanagementclassic.VirtualHost) bool {
		return h != nil && tlsApplied(h, tls)
	}, &resp.Diagnostics)
	if host == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, virtualHostModel(host, plan.Alias))...)
}

// wait reads the host list until ready accepts the host, and reports a
// timeout as an error.
func (r *apiManagementVirtualHostResource) wait(ctx context.Context, id string, ready func(*apimanagementclassic.VirtualHost) bool, diags *diag.Diagnostics) *apimanagementclassic.VirtualHost {
	waitCtx, cancel := context.WithTimeout(ctx, virtualHostWait)
	defer cancel()
	host, err := r.client.WaitForVirtualHost(waitCtx, id, ready)
	if err != nil {
		diags.AddError("Classic API Management virtual host not visible",
			fmt.Sprintf("SAP accepted the request for virtual host %s, but its list of virtual hosts did not show "+
				"the change within %s: %s", id, virtualHostWait, diagnosticDetail(err)))
		return nil
	}
	return host
}

func (r *apiManagementVirtualHostResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiManagementVirtualHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host, err := r.client.FindVirtualHost(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management virtual host", diagnosticDetail(err))
		return
	}
	if host == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, virtualHostModel(host, state.Alias))...)
}

func (r *apiManagementVirtualHostResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiManagementVirtualHostModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() || !r.requireSubdomain(&resp.Diagnostics) {
		return
	}

	current, err := r.client.FindVirtualHost(ctx, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management virtual host", diagnosticDetail(err))
		return
	}
	if current == nil {
		resp.Diagnostics.AddError("Classic API Management virtual host not found",
			fmt.Sprintf("Virtual host %s no longer exists. Run terraform apply again to create it.", state.ID.ValueString()))
		return
	}
	if unsupported := unsupportedVirtualHost(current); unsupported != "" {
		resp.Diagnostics.AddError("Classic API Management virtual host not supported", unsupported)
		return
	}

	alias := plan.Alias.ValueString()
	// The TLS fields are sent only when mutual TLS is on or is switched
	// off; a host without it gets the default domain body. On a tenant an
	// update without them kept a host's mutual TLS, and isClientAuthEnabled
	// false without a truststore switched it off and cleared the truststore.
	tls := planTLS(plan)
	if tls == nil && current.IsClientAuthEnabled {
		tls = &apimanagementclassic.VirtualHostTLS{}
	}
	if _, err := r.client.UpdateVirtualHost(ctx, r.subdomain, current.ID, alias, current.IsDefault, tls); err != nil {
		resp.Diagnostics.AddError("Failed to update Classic API Management virtual host", diagnosticDetail(err))
		return
	}
	host := r.wait(ctx, current.ID, func(h *apimanagementclassic.VirtualHost) bool {
		return h != nil && strings.EqualFold(h.Alias(), alias) && tlsApplied(h, tls)
	}, &resp.Diagnostics)
	if host == nil {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, virtualHostModel(host, plan.Alias))...)
}

func (r *apiManagementVirtualHostResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiManagementVirtualHostModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteVirtualHost(ctx, state.ID.ValueString())
	if err != nil && !apimanagementclassic.IsUnknownVirtualHost(err) {
		resp.Diagnostics.AddError("Failed to delete Classic API Management virtual host", diagnosticDetail(err))
	}
}

// ImportState takes the virtual host's ID, its alias or its full host name.
func (r *apiManagementVirtualHostResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	hosts, err := r.client.ListVirtualHosts(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read Classic API Management virtual hosts", diagnosticDetail(err))
		return
	}
	for i := range hosts {
		h := &hosts[i]
		if h.ID != req.ID && !strings.EqualFold(h.HostName, req.ID) && !strings.EqualFold(h.Alias(), req.ID) {
			continue
		}
		if unsupported := unsupportedVirtualHost(h); unsupported != "" {
			resp.Diagnostics.AddError("Classic API Management virtual host not supported", unsupported)
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, virtualHostModel(h, types.StringNull()))...)
		return
	}
	resp.Diagnostics.AddError("Classic API Management virtual host not found",
		fmt.Sprintf("No virtual host has the ID, alias or host name %q.", req.ID))
}

// unsupportedVirtualHost explains why the resource cannot manage a host,
// or returns "". Its update sends the default domain body, which would
// drop a custom domain or a keystore.
func unsupportedVirtualHost(h *apimanagementclassic.VirtualHost) string {
	if h.IsForCustomDomain || h.KeyStoreName != "" || h.KeyStoreAlias != "" {
		return fmt.Sprintf("Virtual host %s (%s) uses a custom domain or a keystore. This resource manages "+
			"only virtual hosts on the default domain, with or without mutual TLS; change this host in SAP "+
			"instead.", h.ID, h.HostName)
	}
	return ""
}

// virtualHostModel is the state of a host. configured is the alias in the
// configuration; it is kept when the host's alias differs only in case.
// The truststore counts only while client authentication is on; on a tenant
// SAP cleared it when client authentication was switched off.
func virtualHostModel(h *apimanagementclassic.VirtualHost, configured types.String) apiManagementVirtualHostModel {
	alias := types.StringValue(h.Alias())
	if !configured.IsNull() && !configured.IsUnknown() && strings.EqualFold(configured.ValueString(), h.Alias()) {
		alias = configured
	}
	trustStore := types.StringNull()
	if h.IsClientAuthEnabled {
		trustStore = stringOrNull(h.TrustStore)
	}
	return apiManagementVirtualHostModel{
		ID:                types.StringValue(h.ID),
		Alias:             alias,
		HostName:          types.StringValue(h.HostName),
		Port:              types.Int64Value(int64(h.Port)),
		Default:           types.BoolValue(h.IsDefault),
		SSL:               types.BoolValue(h.IsSSL),
		ClientAuthEnabled: types.BoolValue(h.IsClientAuthEnabled),
		TrustStore:        trustStore,
	}
}
