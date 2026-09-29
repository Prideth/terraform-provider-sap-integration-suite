package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/integrationassessment"
)

// A technology's profile is three kinds of association: the domains and
// styles it serves and how well it meets key characteristic values. Each
// association only links a technology to taxonomy entries (a key
// characteristic also has a description). The service has no update for
// them: a tenant answered PATCH on a key characteristic with 400 V101, so
// every attribute forces a new association.

// iaProfileLink is one required link attribute of an association.
type iaProfileLink struct {
	attr, description string
}

// iaProfileValues is an association as the resource sees it: the Id of every
// linked object by attribute name, and the optional description.
type iaProfileValues struct {
	links       map[string]string
	description *string
}

// iaProfileResource is one technology profile association resource.
type iaProfileResource struct {
	suffix         string // after "sapintegrationsuite_integration_assessment_"
	object         string // for messages, for example "technology domain"
	description    string
	links          []iaProfileLink
	hasDescription bool
	create         func(ctx context.Context, c *integrationassessment.Client, v iaProfileValues) (string, error)
	read           func(ctx context.Context, c *integrationassessment.Client, id string) (iaProfileValues, error)
	remove         func(ctx context.Context, c *integrationassessment.Client, id string) error
	client         *integrationassessment.Client
}

var technologyIDLink = iaProfileLink{"technology_id", "Id of the technology, from " +
	"sapintegrationsuite_integration_assessment_technology (resource or data source)."}

// NewIntegrationAssessmentTechnologyDomainResource returns the resource for
// sapintegrationsuite_integration_assessment_technology_domain.
func NewIntegrationAssessmentTechnologyDomainResource() resource.Resource {
	return &iaProfileResource{
		suffix: "technology_domain", object: "technology domain",
		description: "Links a technology to an integration domain it serves. Backed by the TechnologyDomain " +
			"entity set of the Entities API.",
		links: []iaProfileLink{technologyIDLink, {"domain_id", "Id of the domain, from the " +
			"sapintegrationsuite_integration_assessment_domain data source."}},
		create: func(ctx context.Context, c *integrationassessment.Client, v iaProfileValues) (string, error) {
			d, err := c.CreateTechnologyDomain(ctx, v.links["technology_id"], v.links["domain_id"])
			return d.ID, err
		},
		read: func(ctx context.Context, c *integrationassessment.Client, id string) (iaProfileValues, error) {
			d, err := c.GetTechnologyDomain(ctx, id)
			if err != nil {
				return iaProfileValues{}, err
			}
			return iaProfileValues{links: map[string]string{"technology_id": d.TechnologyID(), "domain_id": d.DomainID()}}, nil
		},
		remove: func(ctx context.Context, c *integrationassessment.Client, id string) error {
			return c.DeleteTechnologyDomain(ctx, id)
		},
	}
}

// NewIntegrationAssessmentTechnologyStyleResource returns the resource for
// sapintegrationsuite_integration_assessment_technology_style.
func NewIntegrationAssessmentTechnologyStyleResource() resource.Resource {
	return &iaProfileResource{
		suffix: "technology_style", object: "technology style",
		description: "Links a technology to an integration style it supports. Backed by the TechnologyStyle " +
			"entity set of the Entities API.",
		links: []iaProfileLink{technologyIDLink, {"style_id", "Id of the style, from the " +
			"sapintegrationsuite_integration_assessment_style data source."}},
		create: func(ctx context.Context, c *integrationassessment.Client, v iaProfileValues) (string, error) {
			s, err := c.CreateTechnologyStyle(ctx, v.links["technology_id"], v.links["style_id"])
			return s.ID, err
		},
		read: func(ctx context.Context, c *integrationassessment.Client, id string) (iaProfileValues, error) {
			s, err := c.GetTechnologyStyle(ctx, id)
			if err != nil {
				return iaProfileValues{}, err
			}
			return iaProfileValues{links: map[string]string{"technology_id": s.TechnologyID(), "style_id": s.StyleID()}}, nil
		},
		remove: func(ctx context.Context, c *integrationassessment.Client, id string) error {
			return c.DeleteTechnologyStyle(ctx, id)
		},
	}
}

// NewIntegrationAssessmentTechnologyKeyCharacteristicResource returns the
// resource for sapintegrationsuite_integration_assessment_technology_key_characteristic.
func NewIntegrationAssessmentTechnologyKeyCharacteristicResource() resource.Resource {
	return &iaProfileResource{
		suffix: "technology_key_characteristic", object: "technology key characteristic",
		description: "Rates a technology on one key characteristic value with a recommendation degree, which " +
			"the assessment uses to compare technologies. Backed by the TechnologyKeyCharacteristic entity set " +
			"of the Entities API.",
		links: []iaProfileLink{technologyIDLink,
			{"key_characteristic_value_id", "Id of the key characteristic value, from the " +
				"sapintegrationsuite_integration_assessment_key_characteristic_value data source."},
			{"recommendation_degree_id", "Id of the recommendation degree, from the " +
				"sapintegrationsuite_integration_assessment_recommendation_degree data source."}},
		hasDescription: true,
		create: func(ctx context.Context, c *integrationassessment.Client, v iaProfileValues) (string, error) {
			k, err := c.CreateTechnologyKeyCharacteristic(ctx, v.links["technology_id"],
				v.links["key_characteristic_value_id"], v.links["recommendation_degree_id"], v.description)
			return k.ID, err
		},
		read: func(ctx context.Context, c *integrationassessment.Client, id string) (iaProfileValues, error) {
			k, err := c.GetTechnologyKeyCharacteristic(ctx, id)
			if err != nil {
				return iaProfileValues{}, err
			}
			return iaProfileValues{links: map[string]string{
				"technology_id":               k.TechnologyID(),
				"key_characteristic_value_id": k.KeyCharacteristicValueID(),
				"recommendation_degree_id":    k.RecommendationDegreeID(),
			}, description: k.Description}, nil
		},
		remove: func(ctx context.Context, c *integrationassessment.Client, id string) error {
			return c.DeleteTechnologyKeyCharacteristic(ctx, id)
		},
	}
}

func (r *iaProfileResource) typeName() string {
	return "sapintegrationsuite_integration_assessment_" + r.suffix
}

func (r *iaProfileResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_" + r.suffix
}

func (r *iaProfileResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := map[string]schema.Attribute{"id": iaIDAttribute(r.object)}
	for _, l := range r.links {
		attrs[l.attr] = iaReplacingLinkAttribute(l.description + " Changing it creates a new " + r.object +
			": the service has no update for it.")
	}
	if r.hasDescription {
		attrs["description"] = schema.StringAttribute{
			Optional: true,
			Description: "A note on the rating. Changing it creates a new " + r.object + ": the service " +
				"refuses to update it.",
			PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
		}
	}
	resp.Schema = schema.Schema{Description: iaUnofficialNote + r.description, Attributes: attrs}
}

func (r *iaProfileResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, r.typeName(), "resource", &resp.Diagnostics); c != nil {
		r.client = c
	}
}

// setState writes an association into state attribute by attribute, since
// the three resources have different attributes.
func (r *iaProfileResource) setState(ctx context.Context, state *tfsdk.State, id string, v iaProfileValues) diag.Diagnostics {
	diags := state.SetAttribute(ctx, pathRoot("id"), types.StringValue(id))
	for _, l := range r.links {
		diags.Append(state.SetAttribute(ctx, pathRoot(l.attr), stringOrNull(v.links[l.attr]))...)
	}
	if r.hasDescription {
		diags.Append(state.SetAttribute(ctx, pathRoot("description"), stringPtrOrNull(v.description))...)
	}
	return diags
}

func (r *iaProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	v := iaProfileValues{links: map[string]string{}}
	for _, l := range r.links {
		var s types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, pathRoot(l.attr), &s)...)
		v.links[l.attr] = s.ValueString()
	}
	if r.hasDescription {
		var d types.String
		resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, pathRoot("description"), &d)...)
		v.description = optionalString(d)
	}
	if resp.Diagnostics.HasError() {
		return
	}
	id, err := r.create(ctx, r.client, v)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Integration Assessment "+r.object, diagnosticDetail(err))
		return
	}
	read, err := r.read(ctx, r.client, id)
	if err != nil {
		resp.Diagnostics.Append(r.setState(ctx, &resp.State, id, v)...)
		resp.Diagnostics.AddError("Integration Assessment "+r.object+" created, but reading it back failed", diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, id, read)...)
}

func (r *iaProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, pathRoot("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	read, err := r.read(ctx, r.client, id.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read Integration Assessment "+r.object, diagnosticDetail(err))
		return
	}
	resp.Diagnostics.Append(r.setState(ctx, &resp.State, id.ValueString(), read)...)
}

// Update is never called: every attribute forces a new association.
func (r *iaProfileResource) Update(_ context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.State.Raw = req.Plan.Raw
}

func (r *iaProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, pathRoot("id"), &id)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.remove(ctx, r.client, id.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete Integration Assessment "+r.object, diagnosticDetail(err))
	}
}

func (r *iaProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
