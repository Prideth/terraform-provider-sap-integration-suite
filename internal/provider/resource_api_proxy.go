package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// maxAPIProxyBundleBytes bounds the bundle read from disk. Proxy bundles are
// XML descriptors and small resources; SAP's samples are a few kilobytes.
const maxAPIProxyBundleBytes = 20 << 20

const apiProxyDefaultCreateTimeout = 5 * time.Minute

// NewAPIProxyResource returns a fresh resource.Resource implementation for
// sapintegrationsuite_api_proxy.
func NewAPIProxyResource() resource.Resource {
	return &apiProxyResource{}
}

type apiProxyResource struct {
	client *apimanagementclassic.Client
}

type apiProxyModel struct {
	ID           types.String   `tfsdk:"id"`
	Name         types.String   `tfsdk:"name"`
	Content      types.String   `tfsdk:"content"`
	ContentHash  types.String   `tfsdk:"content_hash"`
	Title        types.String   `tfsdk:"title"`
	Description  types.String   `tfsdk:"description"`
	Version      types.String   `tfsdk:"version"`
	ServiceCode  types.String   `tfsdk:"service_code"`
	ProviderName types.String   `tfsdk:"provider_name"`
	State        types.String   `tfsdk:"state"`
	StatusCode   types.String   `tfsdk:"status_code"`
	IsPublished  types.Bool     `tfsdk:"is_published"`
	Timeouts     timeouts.Value `tfsdk:"timeouts"`
}

func (r *apiProxyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_proxy"
}

// replaceUnlessImported forces a new proxy when a content attribute changes,
// except when the state holds no value yet: after terraform import, the first
// apply only records the configured file and hash.
func replaceUnlessImported(description string) planmodifier.String {
	return stringplanmodifier.RequiresReplaceIf(
		func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
			resp.RequiresReplace = !req.StateValue.IsNull()
		},
		description, description,
	)
}

func (r *apiProxyResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Computed:      true,
			Description:   description,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		Description: "EXPERIMENTAL. Manages a Classic API Management API proxy (API portal) from a " +
			"proxy bundle ZIP, the format the API portal exports.\n\n" +
			"The provider uploads the bundle the way SAP's API Management Client SDK does (POST " +
			"/apiportal/api/1.0/Transport.svc/APIProxies with the ZIP as application/octet-stream), " +
			"reads the proxy from Management.svc/APIProxies and deletes it there. SAP's documentation " +
			"says a proxy imported this way is deployed by default. Whether importing a changed bundle " +
			"over an existing proxy replaces it cleanly is not documented, so any change of the bundle " +
			"deletes the proxy and imports it again: the proxy is briefly unavailable, and products " +
			"that include it lose the link. The lifecycle has not yet passed an acceptance test on a " +
			"tenant; until it has, the resource stays experimental.\n\n" +
			"The bundle must name the proxy in its APIProxy/<name>.xml descriptor, and that name must " +
			"equal name. API providers the bundle's target endpoint references must already exist.",
		Attributes: map[string]schema.Attribute{
			"id": computed("Always equal to name, SAP's key for this entity (APIProxies('<name>'))."),
			"name": schema.StringAttribute{
				Required: true,
				Description: "The proxy's name, as declared in the bundle's APIProxy/<name>.xml " +
					"descriptor. It is the proxy's key; changing it replaces the proxy.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"content": schema.StringAttribute{
				Required: true,
				Description: "Path to the API proxy bundle ZIP, for example " +
					"\"${path.module}/proxies/orders.zip\". After terraform import, the first apply " +
					"only records the path; the provider assumes the file matches the proxy on the " +
					"tenant.",
				PlanModifiers: []planmodifier.String{replaceUnlessImported("Changing the bundle file replaces the proxy.")},
			},
			"content_hash": schema.StringAttribute{
				Required: true,
				Description: "SHA-256 hash of the bundle, for example " +
					"filesha256(\"${path.module}/proxies/orders.zip\"). A different hash replaces the proxy.",
				PlanModifiers: []planmodifier.String{replaceUnlessImported("Changing the bundle hash replaces the proxy.")},
			},
			"title":         computed("The proxy's title, from the bundle."),
			"description":   computed("The proxy's description, from the bundle."),
			"version":       computed("The proxy's version, from the bundle."),
			"service_code":  computed("The API type, for example REST, SOAP or ODATA."),
			"provider_name": computed("The API provider the target endpoint uses, or NONE for a target URL."),
			"state":         computed("The proxy's deployment state as SAP reports it, for example DEPLOYED."),
			"status_code":   computed("The proxy's status code as SAP reports it."),
			"is_published": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the proxy is part of a published product.",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{Create: true}),
		},
	}
}

func (r *apiProxyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func applyAPIProxy(m *apiProxyModel, p *apimanagementclassic.APIProxy) {
	m.ID = types.StringValue(p.Name)
	m.Name = types.StringValue(p.Name)
	m.Title = types.StringValue(p.Title)
	m.Description = types.StringValue(p.Description)
	m.Version = types.StringValue(p.Version)
	m.ServiceCode = types.StringValue(p.ServiceCode)
	m.ProviderName = types.StringValue(p.ProviderName)
	m.State = types.StringValue(p.State)
	m.StatusCode = types.StringValue(p.StatusCode)
	m.IsPublished = types.BoolValue(p.IsPublished)
}

func (r *apiProxyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	timeout, diags := plan.Timeouts.Create(ctx, apiProxyDefaultCreateTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bundle, err := readBoundedFile(plan.Content.ValueString(), maxAPIProxyBundleBytes)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("content"), "Cannot read the API proxy bundle", err.Error())
		return
	}
	bundleName, err := apimanagementclassic.APIProxyBundleName(bundle)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("content"), "Invalid API proxy bundle", err.Error())
		return
	}
	if bundleName != plan.Name.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Proxy name does not match the bundle",
			fmt.Sprintf("The bundle declares the proxy %q, but name is %q. SAP names the proxy after the bundle, "+
				"so the two must be equal.", bundleName, plan.Name.ValueString()))
		return
	}

	if err := r.client.ImportAPIProxy(ctx, bundle); err != nil {
		resp.Diagnostics.AddError("Failed to import Classic API Management API proxy", diagnosticDetail(err))
		return
	}
	proxy, err := r.client.WaitForAPIProxy(ctx, bundleName)
	if err != nil {
		// The import may have succeeded; keep the proxy in state so that
		// Terraform deletes it (tainted) instead of leaving it untracked.
		plan.ID = types.StringValue(bundleName)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
		resp.Diagnostics.AddError("Classic API Management API proxy not readable after import", diagnosticDetail(err))
		return
	}
	applyAPIProxy(&plan, proxy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiProxyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	proxy, err := r.client.GetAPIProxy(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Classic API Management API proxy", diagnosticDetail(err))
		return
	}
	applyAPIProxy(&state, proxy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only runs for the first apply after an import, when content and
// content_hash are recorded; every other change replaces the proxy.
func (r *apiProxyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiProxyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Content = plan.Content
	state.ContentHash = plan.ContentHash
	state.Timeouts = plan.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *apiProxyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiProxyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteAPIProxy(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Classic API Management API proxy", diagnosticDetail(err))
	}
}

func (r *apiProxyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), req.ID)...)
}
