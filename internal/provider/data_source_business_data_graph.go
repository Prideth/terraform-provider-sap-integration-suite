package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apicomposition"
)

// NewBusinessDataGraphDataSource returns a fresh datasource.DataSource
// implementation for sapintegrationsuite_business_data_graph.
func NewBusinessDataGraphDataSource() datasource.DataSource {
	return &businessDataGraphDataSource{}
}

type businessDataGraphDataSource struct {
	client *apicomposition.Client
}

// businessDataGraphDataSourceModel is businessDataGraphModel without the
// resource's id and timeouts.
type businessDataGraphDataSourceModel struct {
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
}

func businessDataGraphToDataSourceModel(cfg *apicomposition.GraphConfiguration) businessDataGraphDataSourceModel {
	full := businessDataGraphFromClient(cfg)
	return businessDataGraphDataSourceModel{
		BusinessDataGraphIdentifier: full.BusinessDataGraphIdentifier,
		SchemaVersion:               full.SchemaVersion,
		Description:                 full.Description,
		GraphModelVersion:           full.GraphModelVersion,
		EffectiveGraphModelVersion:  full.EffectiveGraphModelVersion,
		ODataContainment:            full.ODataContainment,
		Exclude:                     full.Exclude,
		DataSources:                 full.DataSources,
		LocatingPolicy:              full.LocatingPolicy,
		Extensions:                  full.Extensions,
		Status:                      full.Status,
		StatusDetails:               full.StatusDetails,
		LogMessages:                 full.LogMessages,
	}
}

func (d *businessDataGraphDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_business_data_graph"
}

func computedString(description string) schema.StringAttribute {
	return schema.StringAttribute{Computed: true, Description: description}
}

func computedStringList(description string) schema.ListAttribute {
	return schema.ListAttribute{Computed: true, ElementType: types.StringType, Description: description}
}

func keyMappingSideDataSourceAttribute(description string) schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed:    true,
		Description: description,
		Attributes: map[string]schema.Attribute{
			"data_source": computedString("Name of the data source."),
			"entity_name": computedString("Fully qualified entity name."),
			"attributes":  computedStringList("The key attribute."),
			"strategy": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Key format translation, if any.",
				Attributes: map[string]schema.Attribute{
					"name":    computedString("Strategy name, \"format\"."),
					"match":   computedString("RE2 expression with named capturing groups."),
					"replace": computedString("Pattern of the referenced value."),
				},
			},
		},
	}
}

func (d *businessDataGraphDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads the configuration and processing status of a business data graph from API " +
			"Composition's Configuration API. Needs provider.api_composition. The attributes match " +
			"sapintegrationsuite_business_data_graph.",
		Attributes: map[string]schema.Attribute{
			"business_data_graph_identifier": schema.StringAttribute{
				Required:    true,
				Description: "Identifier of the graph to read.",
			},
			"schema_version": computedString("Version of the configuration schema."),
			"description":    computedString("Description of the graph."),
			"odata_containment": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether contained entities are reached only through their parent entity.",
			},
			"graph_model_version":           computedString("Requested version of the unified entity model."),
			"effective_graph_model_version": computedString("Model version SAP actually applied."),
			"exclude":                       computedStringList("Mirrored entities removed from the graph's API."),
			"data_sources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The business systems of the landscape.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":      computedString("Name of the data source."),
						"namespace": computedString("Namespace of a custom service's entities."),
						"services": schema.ListNestedAttribute{
							Computed:    true,
							Description: "The destinations this data source reads from.",
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"destination_name": computedString("Name of the BTP destination."),
									"path":             computedString("Path appended to the destination URL."),
								},
							},
						},
					},
				},
			},
			"locating_policy": schema.SingleNestedAttribute{
				Computed:    true,
				Description: "Where each entity is read from and how keys are translated.",
				Attributes: map[string]schema.Attribute{
					"description": computedString("Description of the locating policy."),
					"cues": schema.ListNestedAttribute{
						Computed:    true,
						Description: "Declared cues.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":        computedString("Name of the cue."),
								"description": computedString("What the cue selects."),
							},
						},
					},
					"key_mapping": schema.ListNestedAttribute{
						Computed:    true,
						Description: "Foreign key mappings between systems.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"cues":        computedStringList("Cues that select this key mapping."),
								"foreign_key": keyMappingSideDataSourceAttribute("The referencing side."),
								"references":  keyMappingSideDataSourceAttribute("The referenced side."),
							},
						},
					},
					"rules": schema.ListNestedAttribute{
						Computed:    true,
						Description: "Locating rules.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name":          computedString("Entity name or namespace wildcard."),
								"leading":       computedString("Leading data source."),
								"local":         computedStringList("Data sources whose references stay local."),
								"cues":          computedStringList("Cues that select this rule."),
								"source_entity": computedString("Source entity of a composite custom entity."),
							},
						},
					},
				},
			},
			"extensions":     computedStringList("Model extensions of the graph."),
			"status":         computedString("Processing status: PROCESSING, DEPLOYMENT_INITIATED or FAILED."),
			"status_details": computedString("SAP's explanation of the status."),
			"log_messages":   computedStringList("Processing log, one JSON document per entry."),
		},
	}
}

func (d *businessDataGraphDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	if !requireAPICompositionHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = apicomposition.New(data.APICompositionHTTPClient, data.APICompositionHost)
}

func (d *businessDataGraphDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config businessDataGraphDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	found, err := d.client.GetGraphConfiguration(ctx, config.BusinessDataGraphIdentifier.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read business data graph", businessDataGraphErrorDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, businessDataGraphToDataSourceModel(found))...)
}
