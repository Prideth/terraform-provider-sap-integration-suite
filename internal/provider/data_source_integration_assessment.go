package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

// iaNamedEntry is what a lookup by name returns: the Id, the description
// (when the lookup has one) and the values of its extra attributes.
type iaNamedEntry struct {
	id, description string
	extra           map[string]string
}

// iaLookupExtra is a computed attribute a lookup returns besides the Id,
// usually the Id of a linked taxonomy entry.
type iaLookupExtra struct {
	attr, description string
}

// iaLookup is one Integration Assessment data source that finds an entry
// of an entity set by its exact name: deployment models, vendors and
// technologies, whose Ids differ between tenants while the names do not.
type iaLookup struct {
	suffix      string // after "sapintegrationsuite_integration_assessment_"
	object      string // for messages, for example "deployment model"
	description string
	withDesc    bool
	extras      []iaLookupExtra
	list        func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error)
	client      *integrationassessment.Client
}

// NewIntegrationAssessmentDeploymentModelDataSource returns the data source
// for sapintegrationsuite_integration_assessment_deployment_model.
func NewIntegrationAssessmentDeploymentModelDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "deployment_model", object: "deployment model", withDesc: true,
		description: "Finds one of SAP's deployment models (for example cloud or on-premise) by name. " +
			"Application and technology instances link to a deployment model by its Id, which differs " +
			"between tenants; the name does not.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListDeploymentModels(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, d := range all {
				desc := ""
				if d.Description != nil {
					desc = *d.Description
				}
				byName[d.Name] = append(byName[d.Name], iaNamedEntry{id: d.ID, description: desc})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentVendorDataSource returns the data source for
// sapintegrationsuite_integration_assessment_vendor.
func NewIntegrationAssessmentVendorDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "vendor", object: "vendor",
		description: "Finds a vendor by name, for example SAP or a vendor maintained in the UI, to link " +
			"applications and technologies to it without managing it.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListVendors(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, v := range all {
				byName[v.Name] = append(byName[v.Name], iaNamedEntry{id: v.ID})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentTechnologyDataSource returns the data source for
// sapintegrationsuite_integration_assessment_technology.
func NewIntegrationAssessmentTechnologyDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "technology", object: "technology",
		description: "Finds a technology by name, SAP's own (for example Cloud Integration) or one " +
			"maintained in the UI, to create technology instances of it.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListTechnologies(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, t := range all {
				byName[t.Name] = append(byName[t.Name], iaNamedEntry{id: t.ID})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentDomainDataSource returns the data source for
// sapintegrationsuite_integration_assessment_domain.
func NewIntegrationAssessmentDomainDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "domain", object: "domain", withDesc: true,
		description: "Finds an integration domain of SAP's Integration Solution Advisory Methodology taxonomy " +
			"by name, to link a technology to it with " +
			"sapintegrationsuite_integration_assessment_technology_domain.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListDomains(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, d := range all {
				byName[d.Name] = append(byName[d.Name], iaNamedEntry{id: d.ID, description: stringOrEmptyPtr(d.Description)})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentStyleDataSource returns the data source for
// sapintegrationsuite_integration_assessment_style.
func NewIntegrationAssessmentStyleDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "style", object: "style", withDesc: true,
		description: "Finds an integration style of SAP's Integration Solution Advisory Methodology taxonomy " +
			"by name, to link a technology to it with " +
			"sapintegrationsuite_integration_assessment_technology_style.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListStyles(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, s := range all {
				byName[s.Name] = append(byName[s.Name], iaNamedEntry{id: s.ID, description: stringOrEmptyPtr(s.Description)})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentRecommendationDegreeDataSource returns the data
// source for sapintegrationsuite_integration_assessment_recommendation_degree.
func NewIntegrationAssessmentRecommendationDegreeDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "recommendation_degree", object: "recommendation degree",
		description: "Finds a recommendation degree by name: how strongly a technology meets a key " +
			"characteristic value, used by sapintegrationsuite_integration_assessment_technology_key_characteristic.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListRecommendationDegrees(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, r := range all {
				byName[r.Name] = append(byName[r.Name], iaNamedEntry{id: r.ID})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentUseCasePatternDataSource returns the data source
// for sapintegrationsuite_integration_assessment_use_case_pattern.
func NewIntegrationAssessmentUseCasePatternDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "use_case_pattern", object: "use case pattern", withDesc: true,
		description: "Finds a use case pattern of SAP's Integration Solution Advisory Methodology taxonomy by " +
			"name. A use case pattern refines an integration style; the data source returns that style's Id " +
			"as well, for sapintegrationsuite_integration_assessment_technology_style.",
		extras: []iaLookupExtra{
			{"example", "SAP's example of the pattern."},
			{"style_id", "Id of the integration style the pattern refines."},
		},
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListUseCasePatterns(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, u := range all {
				byName[u.Name] = append(byName[u.Name], iaNamedEntry{id: u.ID, description: stringOrEmptyPtr(u.Description),
					extra: map[string]string{"example": stringOrEmptyPtr(u.Example), "style_id": u.StyleID()}})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentIntegrationPatternDataSource returns the data
// source for sapintegrationsuite_integration_assessment_integration_pattern.
func NewIntegrationAssessmentIntegrationPatternDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "integration_pattern", object: "integration pattern",
		description: "Finds an integration pattern of SAP's Integration Solution Advisory Methodology taxonomy " +
			"by name. An integration pattern combines an integration domain and an integration style; the data " +
			"source returns both Ids, for the technology profile resources.",
		extras: []iaLookupExtra{
			{"domain_id", "Id of the pattern's integration domain."},
			{"style_id", "Id of the pattern's integration style."},
		},
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListIntegrationPatterns(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, p := range all {
				byName[p.Name] = append(byName[p.Name], iaNamedEntry{id: p.ID,
					extra: map[string]string{"domain_id": p.DomainID(), "style_id": p.StyleID()}})
			}
			return byName, err
		},
	}
}

// NewIntegrationAssessmentKeyCharacteristicGroupDataSource returns the data
// source for sapintegrationsuite_integration_assessment_key_characteristic_group.
func NewIntegrationAssessmentKeyCharacteristicGroupDataSource() datasource.DataSource {
	return &iaLookup{
		suffix: "key_characteristic_group", object: "key characteristic group", withDesc: true,
		description: "Finds a key characteristic group of SAP's Integration Solution Advisory Methodology " +
			"taxonomy by name. Groups structure the key characteristics that technologies are rated on.",
		list: func(ctx context.Context, c *integrationassessment.Client) (map[string][]iaNamedEntry, error) {
			all, err := c.ListKeyCharacteristicGroups(ctx)
			byName := map[string][]iaNamedEntry{}
			for _, g := range all {
				byName[g.Name] = append(byName[g.Name], iaNamedEntry{id: g.ID, description: stringOrEmptyPtr(g.Description)})
			}
			return byName, err
		},
	}
}

// stringOrEmptyPtr returns the value, or "" for nil.
func stringOrEmptyPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (d *iaLookup) typeName() string {
	return "sapintegrationsuite_integration_assessment_" + d.suffix
}

func (d *iaLookup) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_" + d.suffix
}

func (d *iaLookup) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := map[string]schema.Attribute{
		"name": schema.StringAttribute{
			Required:    true,
			Description: "The exact name of the " + d.object + ".",
			Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
		},
		"id": schema.StringAttribute{
			Computed:    true,
			Description: "Id of the " + d.object + ", to link other objects to it.",
		},
	}
	if d.withDesc {
		attrs["description"] = schema.StringAttribute{Computed: true, Description: "SAP's description of the " + d.object + "."}
	}
	for _, e := range d.extras {
		attrs[e.attr] = schema.StringAttribute{Computed: true, Description: e.description}
	}
	resp.Schema = schema.Schema{Description: iaUnofficialNote + d.description, Attributes: attrs}
}

func (d *iaLookup) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, d.typeName(), "data source", &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *iaLookup) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var name types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("name"), &name)...)
	if resp.Diagnostics.HasError() {
		return
	}
	byName, err := d.list(ctx, d.client)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Integration Assessment "+d.object+"s", diagnosticDetail(err))
		return
	}
	matches := byName[name.ValueString()]
	switch len(matches) {
	case 0:
		resp.Diagnostics.AddError("Integration Assessment "+d.object+" not found",
			fmt.Sprintf("No %s is named %q. Names are compared exactly, including case.", d.object, name.ValueString()))
		return
	case 1:
	default:
		resp.Diagnostics.AddError("Integration Assessment "+d.object+" name is ambiguous",
			fmt.Sprintf("%d entries are named %q; the name does not identify one.", len(matches), name.ValueString()))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("name"), name)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("id"), types.StringValue(matches[0].id))...)
	if d.withDesc {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("description"), stringOrNull(matches[0].description))...)
	}
	for _, e := range d.extras {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot(e.attr), stringOrNull(matches[0].extra[e.attr]))...)
	}
}
