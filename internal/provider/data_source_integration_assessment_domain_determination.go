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

const iaDomainDeterminationType = "sapintegrationsuite_integration_assessment_domain_determination"

// NewIntegrationAssessmentDomainDeterminationDataSource returns the data
// source for sapintegrationsuite_integration_assessment_domain_determination.
func NewIntegrationAssessmentDomainDeterminationDataSource() datasource.DataSource {
	return &iaDomainDeterminationDataSource{}
}

// A domain determination has no name: it is the rule that an integration
// between a source and a target deployment model belongs to a domain. It is
// therefore found by the pair of deployment models.
type iaDomainDeterminationDataSource struct {
	client *integrationassessment.Client
}

type iaDomainDeterminationModel struct {
	SourceDeploymentModelID types.String `tfsdk:"source_deployment_model_id"`
	TargetDeploymentModelID types.String `tfsdk:"target_deployment_model_id"`
	ID                      types.String `tfsdk:"id"`
	DomainID                types.String `tfsdk:"domain_id"`
}

func (d *iaDomainDeterminationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_domain_determination"
}

func (d *iaDomainDeterminationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "Finds the integration domain that SAP's Integration Solution Advisory " +
			"Methodology assigns to an integration between two deployment models, for example between a public " +
			"cloud application and an on-premise one. A domain determination has no name, so it is found by the " +
			"Ids of its source and target deployment models.",
		Attributes: map[string]schema.Attribute{
			"source_deployment_model_id": schema.StringAttribute{
				Required: true,
				Description: "Id of the deployment model the integration starts from, from the " +
					"sapintegrationsuite_integration_assessment_deployment_model data source.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"target_deployment_model_id": schema.StringAttribute{
				Required:    true,
				Description: "Id of the deployment model the integration goes to.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"id":        schema.StringAttribute{Computed: true, Description: "Id of the domain determination."},
			"domain_id": schema.StringAttribute{Computed: true, Description: "Id of the integration domain that applies."},
		},
	}
}

func (d *iaDomainDeterminationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaDomainDeterminationType, "data source", &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *iaDomainDeterminationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg iaDomainDeterminationModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	all, err := d.client.ListDomainDeterminations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Integration Assessment domain determinations", diagnosticDetail(err))
		return
	}
	var matches []integrationassessment.DomainDetermination
	for _, dd := range all {
		if dd.SourceDeploymentModelID() == cfg.SourceDeploymentModelID.ValueString() &&
			dd.TargetDeploymentModelID() == cfg.TargetDeploymentModelID.ValueString() {
			matches = append(matches, dd)
		}
	}
	if len(matches) != 1 {
		resp.Diagnostics.AddError("Integration Assessment domain determination not found or ambiguous",
			fmt.Sprintf("%d domain determinations go from deployment model %q to %q; exactly one is needed. The order "+
				"matters: source and target are not interchangeable.", len(matches),
				cfg.SourceDeploymentModelID.ValueString(), cfg.TargetDeploymentModelID.ValueString()))
		return
	}
	cfg.ID = types.StringValue(matches[0].ID)
	cfg.DomainID = stringOrNull(matches[0].DomainID())
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
