package provider

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apicomposition"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
)

// businessDataGraphIDPattern follows SAP's rule for the identifier: up to
// 20 lowercase alphanumeric characters with hyphen separators.
var businessDataGraphIDPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const businessDataGraphDefaultTimeout = 20 * time.Minute

// NewBusinessDataGraphResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_business_data_graph.
func NewBusinessDataGraphResource() resource.Resource {
	return &businessDataGraphResource{}
}

type businessDataGraphResource struct {
	client *apicomposition.Client
	// allowUnofficial is the provider's enable_unofficial: SAP documents
	// creating and reading graphs, not updating (PATCH) or deleting them.
	allowUnofficial bool
}

type businessDataGraphModel struct {
	ID                          types.String              `tfsdk:"id"`
	BusinessDataGraphIdentifier types.String              `tfsdk:"business_data_graph_identifier"`
	SchemaVersion               types.String              `tfsdk:"schema_version"`
	Description                 types.String              `tfsdk:"description"`
	GraphModelVersion           types.String              `tfsdk:"graph_model_version"`
	EffectiveGraphModelVersion  types.String              `tfsdk:"effective_graph_model_version"`
	ODataContainment            types.Bool                `tfsdk:"odata_containment"`
	Exclude                     []string                  `tfsdk:"exclude"`
	DataSources                 []businessDataSourceModel `tfsdk:"data_sources"`
	LocatingPolicy              *locatingPolicyModel      `tfsdk:"locating_policy"`
	Extensions                  types.List                `tfsdk:"extensions"`
	Status                      types.String              `tfsdk:"status"`
	StatusDetails               types.String              `tfsdk:"status_details"`
	LogMessages                 types.List                `tfsdk:"log_messages"`
	Timeouts                    timeouts.Value            `tfsdk:"timeouts"`
}

type businessDataSourceModel struct {
	Name      types.String                     `tfsdk:"name"`
	Namespace types.String                     `tfsdk:"namespace"`
	Services  []businessDataSourceServiceModel `tfsdk:"services"`
}

type businessDataSourceServiceModel struct {
	DestinationName types.String `tfsdk:"destination_name"`
	Path            types.String `tfsdk:"path"`
}

type locatingPolicyModel struct {
	Description types.String        `tfsdk:"description"`
	Cues        []locatingCueModel  `tfsdk:"cues"`
	KeyMapping  []keyMappingModel   `tfsdk:"key_mapping"`
	Rules       []locatingRuleModel `tfsdk:"rules"`
}

type locatingCueModel struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type keyMappingModel struct {
	Cues       []string             `tfsdk:"cues"`
	ForeignKey *keyMappingSideModel `tfsdk:"foreign_key"`
	References *keyMappingSideModel `tfsdk:"references"`
}

type keyMappingSideModel struct {
	DataSource types.String            `tfsdk:"data_source"`
	EntityName types.String            `tfsdk:"entity_name"`
	Attributes []string                `tfsdk:"attributes"`
	Strategy   *keyFormatStrategyModel `tfsdk:"strategy"`
}

type keyFormatStrategyModel struct {
	Name    types.String `tfsdk:"name"`
	Match   types.String `tfsdk:"match"`
	Replace types.String `tfsdk:"replace"`
}

type locatingRuleModel struct {
	Name         types.String `tfsdk:"name"`
	Leading      types.String `tfsdk:"leading"`
	Local        []string     `tfsdk:"local"`
	Cues         []string     `tfsdk:"cues"`
	SourceEntity types.String `tfsdk:"source_entity"`
}

func (r *businessDataGraphResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_business_data_graph"
}

// nonEmptyList rejects an explicitly empty list. SAP reports an empty list
// and an absent one the same way, so only the absent form round-trips.
func nonEmptyList() []validator.List {
	return []validator.List{listvalidator.SizeAtLeast(1)}
}

func nonEmptyString() []validator.String {
	return []validator.String{stringvalidator.LengthAtLeast(1)}
}

func keyMappingSideAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Required:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"data_source": schema.StringAttribute{
				Required:    true,
				Description: "Name of a data source from data_sources.",
			},
			"entity_name": schema.StringAttribute{
				Required:    true,
				Description: "Fully qualified entity name including its namespace, for example \"sap.s4.A_Product\".",
			},
			"attributes": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Validators:  nonEmptyList(),
				Description: "The key attribute. SAP currently supports exactly one attribute here.",
			},
			"strategy": schema.SingleNestedAttribute{
				Optional: true,
				Description: "Key format translation, for keys that are written differently in the two " +
					"systems.",
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:    true,
						Description: "Strategy name. SAP supports only \"format\".",
						Validators:  []validator.String{stringvalidator.OneOf("format")},
					},
					"match": schema.StringAttribute{
						Required: true,
						Description: "RE2 regular expression that splits the attribute value into named " +
							"capturing groups.",
					},
					"replace": schema.StringAttribute{
						Required: true,
						Description: "Pattern of the referenced value. Reference the groups from match " +
							"with $name; every name used here must occur in match.",
					},
				},
			},
		},
	}
}

func (r *businessDataGraphResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a business data graph through API Composition's Configuration API. A " +
			"business data graph exposes the business systems of a landscape as one connected API. " +
			"This resource needs provider.api_composition, a credential set separate from " +
			"provider.oauth and provider.api_management.\n\n" +
			"SAP processes a new or changed graph asynchronously. Create and Update wait until the " +
			"status leaves PROCESSING. When SAP reports FAILED, the graph is still stored in state, " +
			"with status_details and log_messages, and Terraform marks it tainted.\n\n" +
			"SAP documents the Create body and the GET and PATCH URLs. It gives no example body for " +
			"PATCH and no request for delete; this resource sends the writable properties as the " +
			"PATCH body and DELETE to the graph's URL. Both are therefore unofficial and also need " +
			"enable_unofficial = true: without it a graph can be created and read, but not updated " +
			"in place or destroyed. Extensions cannot be managed through this " +
			"API and are only reported. See the API Composition guide.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Same as business_data_graph_identifier.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"business_data_graph_identifier": schema.StringAttribute{
				Required: true,
				Description: "Identifier of the graph. Client applications use it in the API " +
					"Composition URL. Up to 20 lowercase alphanumeric characters separated by hyphens, " +
					"for example \"my-bdg\". Changing it replaces the graph.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(20),
					stringvalidator.RegexMatches(businessDataGraphIDPattern,
						"must be lowercase alphanumeric characters separated by hyphens"),
				},
			},
			"schema_version": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Description:   "Version of the configuration schema. SAP fills it when left out.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"description": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Validators: nonEmptyString(),
				Description: "Description of the graph. Unofficial: the property is known only from the " +
					"API's $metadata, so setting it needs enable_unofficial = true. When left out, SAP's " +
					"value is kept and shown.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"odata_containment": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Whether contained entities are reached only through their parent entity " +
					"(OData containment). SAP enables it by default. Unofficial: SAP describes the setting " +
					"but names the property (odataContainment) only in the API's $metadata, so setting it " +
					"needs enable_unofficial = true. When left out, SAP's value is kept and shown.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"graph_model_version": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Version of API Composition's unified entity model to build the graph " +
					"against, for example \"1.0.0\". SAP fills it when left out.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"effective_graph_model_version": schema.StringAttribute{
				Computed:    true,
				Description: "Model version SAP actually applied.",
			},
			"exclude": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators:  nonEmptyList(),
				Description: "Mirrored entities to remove from the graph's API, by full name, with an " +
					"optional trailing wildcard, for example [\"sap.s4.A_Product\", \"sap.c4c.*\"]. " +
					"Associations to them disappear as well; custom entities can still use them as a " +
					"source.",
			},
			"data_sources": schema.ListNestedAttribute{
				Required:   true,
				Validators: nonEmptyList(),
				Description: "The business systems of the landscape. Each one reads from one or more BTP " +
					"destinations of the subaccount.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required: true,
							Description: "Name of the data source, used by locating rules and key " +
								"mappings. It appears in response payloads as the key qualifier, so SAP " +
								"recommends short names.",
						},
						"namespace": schema.StringAttribute{
							Optional:   true,
							Validators: nonEmptyString(),
							Description: "Namespace of the entities of a custom OData service, for " +
								"example \"company.custom\". Not needed for SAP systems API Composition " +
								"knows.",
						},
						"services": schema.ListNestedAttribute{
							Required:    true,
							Validators:  nonEmptyList(),
							Description: "The destinations this data source reads from.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"destination_name": schema.StringAttribute{
										Required:    true,
										Description: "Name of the BTP destination, exactly as defined in the subaccount.",
									},
									"path": schema.StringAttribute{
										Optional:   true,
										Validators: nonEmptyString(),
										Description: "Path appended to the destination URL, for example " +
											"\"/odata/sap/API_BUSINESS_PARTNER\". Lets one root " +
											"destination serve several services of the same system.",
									},
								},
							},
						},
					},
				},
			},
			"locating_policy": schema.SingleNestedAttribute{
				Required: true,
				Description: "Tells API Composition which data source to read each entity from, and " +
					"how to translate keys between systems.",
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{
						Optional:   true,
						Validators: nonEmptyString(),
						Description: "Description of the locating policy. Unofficial: known only from the API's " +
							"$metadata, so setting it needs enable_unofficial = true.",
					},
					"cues": schema.ListNestedAttribute{
						Optional:    true,
						Validators:  nonEmptyList(),
						Description: "Cues that rules can reference. Every cue used in a rule must be declared here.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{Required: true, Description: "Name of the cue."},
								"description": schema.StringAttribute{
									Required:   true,
									Validators: nonEmptyString(),
									Description: "What the cue selects. Required: SAP rejects a cue without a " +
										"description (HTTP 400, \"must have required property 'description'\").",
								},
							},
						},
					},
					"key_mapping": schema.ListNestedAttribute{
						Optional:   true,
						Validators: nonEmptyList(),
						Description: "Foreign key mappings for systems that identify the same entity " +
							"with different keys.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"cues": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Validators:  nonEmptyList(),
									Description: "Cues that select this key mapping. SAP describes cue-scoped key " +
										"mappings but names the property only in the API's $metadata, so setting it " +
										"needs enable_unofficial = true.",
								},
								"foreign_key": keyMappingSideAttribute("The referencing side: the entity and attribute holding the foreign key."),
								"references":  keyMappingSideAttribute("The referenced side: the entity and key attribute in the other system."),
							},
						},
					},
					"rules": schema.ListNestedAttribute{
						Required:   true,
						Validators: nonEmptyList(),
						Description: "Locating rules. A specific entity name overrides a wildcard, and a " +
							"rule with cues overrides the default rule, so order does not matter. A " +
							"well-formed policy has one rule without cues for every entity.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required: true,
									Description: "Entity name including its namespace, or a namespace " +
										"with a trailing wildcard, for example \"sap.s4.*\".",
								},
								"leading": schema.StringAttribute{
									Required:    true,
									Description: "Data source that is the leading system for the entity.",
								},
								"local": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Validators:  nonEmptyList(),
									Description: "Data sources whose key-based references to the entity " +
										"stay in their own system instead of going to the leading one.",
								},
								"cues": schema.ListAttribute{
									Optional:    true,
									ElementType: types.StringType,
									Validators:  nonEmptyList(),
									Description: "Cues that select this rule. A cue may appear in only one rule per entity.",
								},
								"source_entity": schema.StringAttribute{
									Optional:   true,
									Validators: nonEmptyString(),
									Description: "For a composite custom entity, the mirrored source entity " +
										"this rule applies to. Needed for source entities from a data " +
										"source other than the main one.",
								},
							},
						},
					},
				},
			},
			"extensions": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Model extensions of the graph. The Configuration API does not manage " +
					"extensions, so this is read-only and updates leave them alone.",
				PlanModifiers: []planmodifier.List{listplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Processing status: PROCESSING, DEPLOYMENT_INITIATED or FAILED.",
			},
			"status_details": schema.StringAttribute{
				Computed:    true,
				Description: "SAP's explanation of the status, mainly for FAILED.",
			},
			"log_messages": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Processing log, one JSON document per entry, as SAP returns it.",
			},
		},
		Blocks: map[string]schema.Block{
			"timeouts": timeouts.Block(ctx, timeouts.Opts{
				Create: true,
				Update: true,
			}),
		},
	}
}

func (r *businessDataGraphResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, "sapintegrationsuite_business_data_graph", &resp.Diagnostics) {
		return
	}
	if !requireAPICompositionHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = apicomposition.New(data.APICompositionHTTPClient, data.APICompositionHost)
	r.allowUnofficial = data.EnableUnofficial
}

func keyMappingSideToClient(m *keyMappingSideModel) apicomposition.KeyMappingSide {
	side := apicomposition.KeyMappingSide{
		DataSource: m.DataSource.ValueString(),
		EntityName: m.EntityName.ValueString(),
		Attributes: m.Attributes,
	}
	if m.Strategy != nil {
		side.Strategy = &apicomposition.KeyFormatStrategy{
			Name:    m.Strategy.Name.ValueString(),
			Match:   m.Strategy.Match.ValueString(),
			Replace: m.Strategy.Replace.ValueString(),
		}
	}
	return side
}

func businessDataGraphToClient(m businessDataGraphModel) apicomposition.GraphConfigurationInput {
	dataSources := make([]apicomposition.DataSource, 0, len(m.DataSources))
	for _, ds := range m.DataSources {
		services := make([]apicomposition.DataSourceService, 0, len(ds.Services))
		for _, svc := range ds.Services {
			services = append(services, apicomposition.DataSourceService{
				DestinationName: svc.DestinationName.ValueString(),
				Path:            svc.Path.ValueString(),
			})
		}
		dataSources = append(dataSources, apicomposition.DataSource{
			Name:      ds.Name.ValueString(),
			Namespace: ds.Namespace.ValueString(),
			Services:  services,
		})
	}

	var policy apicomposition.LocatingPolicy
	if p := m.LocatingPolicy; p != nil {
		for _, cue := range p.Cues {
			policy.Cues = append(policy.Cues, apicomposition.LocatingCue{
				Name:        cue.Name.ValueString(),
				Description: cue.Description.ValueString(),
			})
		}
		policy.Description = p.Description.ValueString()
		for _, km := range p.KeyMapping {
			policy.KeyMapping = append(policy.KeyMapping, apicomposition.KeyMapping{
				Cues:       km.Cues,
				ForeignKey: keyMappingSideToClient(km.ForeignKey),
				References: keyMappingSideToClient(km.References),
			})
		}
		for _, rule := range p.Rules {
			policy.Rules = append(policy.Rules, apicomposition.LocatingRule{
				Name:         rule.Name.ValueString(),
				Leading:      rule.Leading.ValueString(),
				Local:        rule.Local,
				Cues:         rule.Cues,
				SourceEntity: rule.SourceEntity.ValueString(),
			})
		}
	}

	schemaVersion, graphModelVersion := "", ""
	if !m.SchemaVersion.IsUnknown() {
		schemaVersion = m.SchemaVersion.ValueString()
	}
	if !m.GraphModelVersion.IsUnknown() {
		graphModelVersion = m.GraphModelVersion.ValueString()
	}

	var description string
	if !m.Description.IsUnknown() {
		description = m.Description.ValueString()
	}
	var containment *bool
	if !m.ODataContainment.IsUnknown() && !m.ODataContainment.IsNull() {
		containment = m.ODataContainment.ValueBoolPointer()
	}

	return apicomposition.GraphConfigurationInput{
		BusinessDataGraphIdentifier: m.BusinessDataGraphIdentifier.ValueString(),
		SchemaVersion:               schemaVersion,
		Description:                 description,
		GraphModelVersion:           graphModelVersion,
		ODataContainment:            containment,
		Exclude:                     m.Exclude,
		DataSources:                 dataSources,
		LocatingPolicy:              policy,
	}
}

// stringListValue turns values into a list attribute. Computed lists must be
// types.List in the model: Terraform plans them as unknown on create, which a
// []string field cannot hold (the first tenant run failed with "Value
// Conversion Error" on log_messages). nullIfEmpty keeps an empty list null,
// as for extensions; log_messages stays an empty list.
func stringListValue(values []string, nullIfEmpty bool) types.List {
	if len(values) == 0 && nullIfEmpty {
		return types.ListNull(types.StringType)
	}
	elems := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elems = append(elems, types.StringValue(v))
	}
	return types.ListValueMust(types.StringType, elems)
}

// listOrNull maps an empty list from SAP to null, the form an omitted
// optional list has in the configuration.
func listOrNull[T any](v []T) []T {
	if len(v) == 0 {
		return nil
	}
	return v
}

func keyMappingSideFromClient(s apicomposition.KeyMappingSide) *keyMappingSideModel {
	side := &keyMappingSideModel{
		DataSource: types.StringValue(s.DataSource),
		EntityName: types.StringValue(s.EntityName),
		Attributes: s.Attributes,
	}
	if s.Strategy != nil {
		side.Strategy = &keyFormatStrategyModel{
			Name:    types.StringValue(s.Strategy.Name),
			Match:   types.StringValue(s.Strategy.Match),
			Replace: types.StringValue(s.Strategy.Replace),
		}
	}
	return side
}

func businessDataGraphFromClient(cfg *apicomposition.GraphConfiguration) businessDataGraphModel {
	dataSources := make([]businessDataSourceModel, 0, len(cfg.DataSources))
	for _, ds := range cfg.DataSources {
		services := make([]businessDataSourceServiceModel, 0, len(ds.Services))
		for _, svc := range ds.Services {
			services = append(services, businessDataSourceServiceModel{
				DestinationName: types.StringValue(svc.DestinationName),
				Path:            stringOrNull(svc.Path),
			})
		}
		dataSources = append(dataSources, businessDataSourceModel{
			Name:      types.StringValue(ds.Name),
			Namespace: stringOrNull(ds.Namespace),
			Services:  services,
		})
	}

	policy := &locatingPolicyModel{Description: stringOrNull(cfg.LocatingPolicy.Description)}
	for _, cue := range cfg.LocatingPolicy.Cues {
		policy.Cues = append(policy.Cues, locatingCueModel{
			Name:        types.StringValue(cue.Name),
			Description: stringOrNull(cue.Description),
		})
	}
	for _, km := range cfg.LocatingPolicy.KeyMapping {
		policy.KeyMapping = append(policy.KeyMapping, keyMappingModel{
			Cues:       listOrNull(km.Cues),
			ForeignKey: keyMappingSideFromClient(km.ForeignKey),
			References: keyMappingSideFromClient(km.References),
		})
	}
	for _, rule := range cfg.LocatingPolicy.Rules {
		policy.Rules = append(policy.Rules, locatingRuleModel{
			Name:         types.StringValue(rule.Name),
			Leading:      types.StringValue(rule.Leading),
			Local:        listOrNull(rule.Local),
			Cues:         listOrNull(rule.Cues),
			SourceEntity: stringOrNull(rule.SourceEntity),
		})
	}

	logMessages := make([]string, 0, len(cfg.LogMessages))
	for _, raw := range cfg.LogMessages {
		logMessages = append(logMessages, string(raw))
	}

	return businessDataGraphModel{
		ID:                          types.StringValue(cfg.BusinessDataGraphIdentifier),
		BusinessDataGraphIdentifier: types.StringValue(cfg.BusinessDataGraphIdentifier),
		SchemaVersion:               stringOrNull(cfg.SchemaVersion),
		Description:                 stringOrNull(cfg.Description),
		GraphModelVersion:           stringOrNull(cfg.GraphModelVersion),
		EffectiveGraphModelVersion:  stringOrNull(cfg.EffectiveGraphModelVersion),
		ODataContainment:            types.BoolPointerValue(cfg.ODataContainment),
		Exclude:                     listOrNull(cfg.Exclude),
		DataSources:                 dataSources,
		LocatingPolicy:              policy,
		Extensions:                  stringListValue(cfg.ExtensionNames(), true),
		Status:                      stringOrNull(cfg.Status),
		StatusDetails:               stringOrNull(cfg.StatusDetails),
		LogMessages:                 stringListValue(logMessages, false),
	}
}

// saveProcessedGraph records the graph SAP reported after a Create or
// Update. When processing failed or did not finish in time, the graph
// still exists on SAP's side, so it is stored and the error is reported;
// Terraform then marks the resource tainted.
func saveProcessedGraph(ctx context.Context, state *tfsdk.State, tf timeouts.Value, cfg *apicomposition.GraphConfiguration, err error, summary string, diags *diag.Diagnostics) {
	if cfg != nil {
		model := businessDataGraphFromClient(cfg)
		model.Timeouts = tf
		diags.Append(state.Set(ctx, model)...)
	}
	if err != nil {
		var failed *apicomposition.ProcessingFailedError
		if errors.As(err, &failed) {
			summary = "SAP could not process the business data graph"
		}
		diags.AddError(summary, businessDataGraphErrorDetail(err))
	}
}

// businessDataGraphErrorDetail is diagnosticDetail plus what helps with the
// Configuration API's errors: SAP's trace ID, which SAP support asks for,
// and on HTTP 403 the way to a token that carries the API's scope. SAP's
// 403 text only says to check the assigned roles. On a test tenant a
// client-credentials token of a `configuration` service key carried no
// scope for the API and was refused with code 2707 on every request,
// including $metadata; a key user's token through the same client
// (password grant) carried the scope `config` and was accepted (2026-10-03).
func businessDataGraphErrorDetail(err error) string {
	detail := diagnosticDetail(err)
	var apiErr *apierror.Error
	if !errors.As(err, &apiErr) {
		return detail
	}
	if apiErr.RequestID != "" {
		detail += "\nSAP trace ID: " + apiErr.RequestID
	}
	if apiErr.StatusCode == http.StatusForbidden {
		detail += "\n\nThe access token of provider.api_composition lacks the Configuration API's " +
			"scope (\"config\"), which SAP grants with the role Graph_Key_User. A token that a " +
			"service key of plan \"configuration\" issues for client credentials can lack it. Set " +
			"username and password (and origin for an identity provider other than the default) " +
			"to log in as a user with the role collection Graph.KeyUser; see \"Credentials\" in " +
			"the API Composition guide."
	}
	return detail
}

func (r *businessDataGraphResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if !r.requireMetadataSettings(ctx, req.Config, &resp.Diagnostics) {
		return
	}
	var plan businessDataGraphModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Create(ctx, businessDataGraphDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	created, err := r.client.CreateGraphConfiguration(ctx, businessDataGraphToClient(plan))
	saveProcessedGraph(ctx, &resp.State, plan.Timeouts, created, err, "Failed to create business data graph", &resp.Diagnostics)
}

func (r *businessDataGraphResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state businessDataGraphModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := r.client.GetGraphConfiguration(ctx, state.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read business data graph", businessDataGraphErrorDetail(err))
		return
	}

	model := businessDataGraphFromClient(found)
	model.Timeouts = state.Timeouts
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

const (
	businessDataGraphUpdateOp = "update (PATCH with the writable properties)"
	businessDataGraphDeleteOp = "delete (DELETE on the graph)"
	// The settings SAP's pages do not name; the $metadata does.
	businessDataGraphSettingsOp = "settings named only in the $metadata (description, odata_containment, locating_policy.description, key_mapping cues)"
)

// ModifyPlan stops, without enable_unofficial, a plan that updates a graph in
// place or deletes it (destroy or replacement). Both work on a tenant but SAP
// does not document them.
// metadataOnlySettings lists the settings of m that SAP's documentation does
// not name, so that setting them can be gated behind enable_unofficial. m is
// the configuration: an attribute left out is null there.
func metadataOnlySettings(m businessDataGraphModel) []string {
	var set []string
	if !m.Description.IsNull() {
		set = append(set, "description")
	}
	if !m.ODataContainment.IsNull() {
		set = append(set, "odata_containment")
	}
	if p := m.LocatingPolicy; p != nil {
		if !p.Description.IsNull() {
			set = append(set, "locating_policy.description")
		}
		for _, km := range p.KeyMapping {
			if km.Cues != nil {
				set = append(set, "locating_policy.key_mapping.cues")
				break
			}
		}
	}
	return set
}

// requireMetadataSettings refuses a configuration that sets a $metadata-only
// setting without enable_unofficial.
func (r *businessDataGraphResource) requireMetadataSettings(ctx context.Context, config tfsdk.Config, diags *diag.Diagnostics) bool {
	if r.allowUnofficial || config.Raw.IsNull() {
		return true
	}
	var m businessDataGraphModel
	diags.Append(config.Get(ctx, &m)...)
	if diags.HasError() || len(metadataOnlySettings(m)) == 0 {
		return !diags.HasError()
	}
	return requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_business_data_graph", businessDataGraphSettingsOp, diags)
}

func (r *businessDataGraphResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil {
		return // provider not configured yet; Create, Update and Delete check again
	}
	if !req.Plan.Raw.IsNull() {
		r.requireMetadataSettings(ctx, req.Config, &resp.Diagnostics)
	}
	switch {
	case isPlannedDelete(req, resp):
		requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_business_data_graph", businessDataGraphDeleteOp, &resp.Diagnostics)
	case isInPlaceUpdate(req, resp):
		requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_business_data_graph", businessDataGraphUpdateOp, &resp.Diagnostics)
	}
}

func (r *businessDataGraphResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if !requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_business_data_graph", businessDataGraphUpdateOp, &resp.Diagnostics) {
		return
	}
	var plan businessDataGraphModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	timeout, diags := plan.Timeouts.Update(ctx, businessDataGraphDefaultTimeout)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	updated, err := r.client.UpdateGraphConfiguration(ctx, plan.ID.ValueString(), businessDataGraphToClient(plan))
	saveProcessedGraph(ctx, &resp.State, plan.Timeouts, updated, err, "Failed to update business data graph", &resp.Diagnostics)
}

func (r *businessDataGraphResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if !requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_business_data_graph", businessDataGraphDeleteOp, &resp.Diagnostics) {
		return
	}
	var state businessDataGraphModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := r.client.DeleteGraphConfiguration(ctx, state.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete business data graph", businessDataGraphErrorDetail(err))
	}
}

func (r *businessDataGraphResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, pathRootID(), req, resp)
}
