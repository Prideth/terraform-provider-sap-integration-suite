package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apimanagementclassic"
)

// NewAPIProductResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_api_product.
func NewAPIProductResource() resource.Resource {
	return &apiProductResource{}
}

type apiProductResource struct {
	client *apimanagementclassic.Client
}

type apiProductModel struct {
	ID                   types.String                        `tfsdk:"id"`
	Name                 types.String                        `tfsdk:"name"`
	Version              types.String                        `tfsdk:"version"`
	Title                types.String                        `tfsdk:"title"`
	Description          types.String                        `tfsdk:"description"`
	Scope                types.String                        `tfsdk:"scope"`
	StatusCode           types.String                        `tfsdk:"status_code"`
	IsPublished          types.Bool                          `tfsdk:"is_published"`
	IsRestricted         types.Bool                          `tfsdk:"is_restricted"`
	QuotaCount           types.Int64                         `tfsdk:"quota_count"`
	QuotaInterval        types.Int64                         `tfsdk:"quota_interval"`
	QuotaTimeUnit        types.String                        `tfsdk:"quota_time_unit"`
	APIProxyNames        []string                            `tfsdk:"api_proxy_names"`
	AdditionalProperties []apiProductAdditionalPropertyModel `tfsdk:"additional_properties"`
}

type apiProductAdditionalPropertyModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

func (r *apiProductResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_product"
}

// replaceString, replaceComputedString and the other helpers below keep the
// schema readable: SAP answers every update of an API product with 405, so
// every attribute forces a new product.
func replaceString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

func replaceComputedString() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()}
}

func replaceComputedBool() []planmodifier.Bool {
	return []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()}
}

func replaceInt64() []planmodifier.Int64 {
	return []planmodifier.Int64{int64planmodifier.RequiresReplace()}
}

func (r *apiProductResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Classic API Management API product (APIProducts): a bundle of one " +
			"or more API proxies that application developers subscribe to.\n\n" +
			"An API product cannot be changed after it is created. A tenant test in September " +
			"2026 answered PUT, PATCH and MERGE on an existing product with 405 \"UPDATE operation " +
			"not supported on APIProduct entity\". Every attribute therefore forces a new product: " +
			"Terraform deletes the product and creates it again. Applications subscribed to the old " +
			"product lose that subscription, so review any plan that replaces a product.\n\n" +
			"SAP requires at least one linked API proxy. The proxies must already exist, for " +
			"example managed with sapintegrationsuite_api_proxy (experimental).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Always equal to name, SAP's key for this entity (APIProducts('<name>')).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "The API product's name. It is the product's key and cannot be changed.",
				PlanModifiers: replaceString(),
			},
			"version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "The product's version, for example \"1\". When left out, SAP sets one " +
					"itself (\"1\" on the tested tenant).",
				PlanModifiers: replaceComputedString(),
			},
			"title": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "The title shown in the API business hub enterprise.",
				PlanModifiers: replaceComputedString(),
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Description:   "A longer description of the product.",
				PlanModifiers: replaceString(),
			},
			"scope": schema.StringAttribute{
				Optional: true,
				Description: "OAuth scopes the product grants, as SAP expects them. Sent as an empty " +
					"string when left out.",
				PlanModifiers: replaceString(),
			},
			"status_code": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("PUBLISHED"),
				Description: "The product's status. SAP requires one on create; without it the create " +
					"fails inside SAP. Defaults to \"PUBLISHED\", the value confirmed on a tenant.",
				PlanModifiers: replaceString(),
			},
			"is_published": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether the product is published for subscription. When left out, SAP decides.",
				PlanModifiers: replaceComputedBool(),
			},
			"is_restricted": schema.BoolAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Whether a subscription to this product needs approval. When left out, SAP decides.",
				PlanModifiers: replaceComputedBool(),
			},
			"quota_count": schema.Int64Attribute{
				Optional: true,
				Description: "How many requests the quota allows per interval. SAP's own examples send " +
					"-99 for \"no quota\"; the provider sends the value as configured and null when unset.",
				PlanModifiers: replaceInt64(),
			},
			"quota_interval": schema.Int64Attribute{
				Optional:      true,
				Description:   "The quota interval, counted in quota_time_unit.",
				PlanModifiers: replaceInt64(),
			},
			"quota_time_unit": schema.StringAttribute{
				Optional:      true,
				Description:   "The unit of quota_interval, as SAP expects it (for example \"minute\").",
				PlanModifiers: replaceString(),
			},
			"api_proxy_names": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Names of existing API proxies bundled in this product. SAP rejects a " +
					"product without one (\"At least one API Proxy should be linked to an API Product\").",
				Validators:    []validator.List{listvalidator.SizeAtLeast(1)},
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"additional_properties": schema.SetNestedAttribute{
				Optional: true,
				Description: "Custom name/value attributes sent with the product when it is created. " +
					"Order carries no meaning, hence a set.",
				PlanModifiers: []planmodifier.Set{setplanmodifier.RequiresReplace()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":  schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *apiProductResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func apiProductToClient(plan apiProductModel) apimanagementclassic.APIProduct {
	product := apimanagementclassic.APIProduct{
		Name:          plan.Name.ValueString(),
		Version:       plan.Version.ValueString(),
		Title:         plan.Title.ValueString(),
		Description:   plan.Description.ValueString(),
		Scope:         plan.Scope.ValueString(),
		StatusCode:    plan.StatusCode.ValueString(),
		IsPublished:   plan.IsPublished.ValueBool(),
		IsRestricted:  plan.IsRestricted.ValueBool(),
		ApiProxyNames: plan.APIProxyNames,
	}
	if !plan.QuotaCount.IsNull() {
		v := plan.QuotaCount.ValueInt64()
		product.QuotaCount = &v
	}
	if !plan.QuotaInterval.IsNull() {
		v := plan.QuotaInterval.ValueInt64()
		product.QuotaInterval = &v
	}
	if !plan.QuotaTimeUnit.IsNull() {
		v := plan.QuotaTimeUnit.ValueString()
		product.QuotaTimeUnit = &v
	}
	for _, p := range plan.AdditionalProperties {
		product.AdditionalProperties = append(product.AdditionalProperties, apimanagementclassic.APIProductAdditionalProperty{
			EntityID: product.Name,
			Name:     p.Name.ValueString(),
			Value:    p.Value.ValueString(),
		})
	}
	return product
}

// applyAPIProduct copies what SAP returned into the model. The linked proxy
// names and additional properties are not part of SAP's product response;
// Read fetches them separately, and Create keeps the planned values.
func applyAPIProduct(m *apiProductModel, found *apimanagementclassic.APIProduct) {
	m.ID = types.StringValue(found.Name)
	m.Name = types.StringValue(found.Name)
	m.Version = stringOrNull(found.Version)
	m.Title = stringOrNull(found.Title)
	m.Description = stringOrNull(found.Description)
	m.Scope = stringOrNull(found.Scope)
	m.StatusCode = stringOrNull(found.StatusCode)
	m.IsPublished = types.BoolValue(found.IsPublished)
	m.IsRestricted = types.BoolValue(found.IsRestricted)
	if found.QuotaCount != nil {
		m.QuotaCount = types.Int64Value(*found.QuotaCount)
	} else {
		m.QuotaCount = types.Int64Null()
	}
	if found.QuotaInterval != nil {
		m.QuotaInterval = types.Int64Value(*found.QuotaInterval)
	} else {
		m.QuotaInterval = types.Int64Null()
	}
	if found.QuotaTimeUnit != nil {
		m.QuotaTimeUnit = types.StringValue(*found.QuotaTimeUnit)
	} else {
		m.QuotaTimeUnit = types.StringNull()
	}
}

func (r *apiProductResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiProductModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	created, err := r.client.CreateAPIProduct(ctx, apiProductToClient(plan))
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Classic API Management API product", diagnosticDetail(err))
		return
	}

	applyAPIProduct(&plan, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *apiProductResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiProductModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.GetAPIProduct(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Classic API Management API product", diagnosticDetail(err))
		return
	}

	applyAPIProduct(&state, found)

	proxyNames, err := r.client.GetAPIProductProxyNames(ctx, found.Name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the API proxies linked to a Classic API Management API product", diagnosticDetail(err))
		return
	}
	state.APIProxyNames = keepListOrder(state.APIProxyNames, proxyNames)

	props, err := r.client.GetAPIProductAdditionalProperties(ctx, found.Name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the additional properties of a Classic API Management API product", diagnosticDetail(err))
		return
	}
	state.AdditionalProperties = apiProductPropertiesToModel(props)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// keepListOrder returns the prior list when it holds the same names as
// SAP's answer, so that a different order from SAP does not show up as a
// change (and with it a replacement). Otherwise SAP's answer wins.
func keepListOrder(prior, current []string) []string {
	if len(prior) != len(current) {
		return current
	}
	seen := make(map[string]int, len(current))
	for _, name := range current {
		seen[name]++
	}
	for _, name := range prior {
		if seen[name] == 0 {
			return current
		}
		seen[name]--
	}
	return prior
}

// apiProductPropertiesToModel maps SAP's properties to the set attribute;
// no properties map to null, matching a configuration that leaves the
// attribute out.
func apiProductPropertiesToModel(props []apimanagementclassic.APIProductAdditionalProperty) []apiProductAdditionalPropertyModel {
	if len(props) == 0 {
		return nil
	}
	result := make([]apiProductAdditionalPropertyModel, 0, len(props))
	for _, p := range props {
		result = append(result, apiProductAdditionalPropertyModel{
			Name:  types.StringValue(p.Name),
			Value: types.StringValue(p.Value),
		})
	}
	return result
}

// Update is unreachable: every attribute forces replacement, because SAP
// answers every update of an API product with 405.
func (r *apiProductResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"sapintegrationsuite_api_product does not support in-place updates; "+
			"Terraform should have replaced this resource instead of updating it.",
	)
}

func (r *apiProductResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiProductModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteAPIProduct(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Classic API Management API product", diagnosticDetail(err))
	}
}

func (r *apiProductResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
