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

const iaKeyCharacteristicValueType = "sapintegrationsuite_integration_assessment_key_characteristic_value"

// NewIntegrationAssessmentKeyCharacteristicValueDataSource returns the data
// source for sapintegrationsuite_integration_assessment_key_characteristic_value.
func NewIntegrationAssessmentKeyCharacteristicValueDataSource() datasource.DataSource {
	return &iaKeyCharacteristicValueDataSource{}
}

// The same value name can belong to several key characteristics, so a value
// is looked up by the name of its key characteristic and its own name.
type iaKeyCharacteristicValueDataSource struct {
	client *integrationassessment.Client
}

type iaKeyCharacteristicValueModel struct {
	KeyCharacteristic   types.String `tfsdk:"key_characteristic"`
	Name                types.String `tfsdk:"name"`
	ID                  types.String `tfsdk:"id"`
	Description         types.String `tfsdk:"description"`
	KeyCharacteristicID types.String `tfsdk:"key_characteristic_id"`
}

func (d *iaKeyCharacteristicValueDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_assessment_key_characteristic_value"
}

func (d *iaKeyCharacteristicValueDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: iaUnofficialNote + "Finds a value of one of the key characteristics that SAP's Integration " +
			"Solution Advisory Methodology rates technologies on, to rate a technology with " +
			"sapintegrationsuite_integration_assessment_technology_key_characteristic.",
		Attributes: map[string]schema.Attribute{
			"key_characteristic": schema.StringAttribute{
				Required:    true,
				Description: "The exact name of the key characteristic the value belongs to.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "The exact name of the value.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"id":                    schema.StringAttribute{Computed: true, Description: "Id of the value, to rate a technology on it."},
			"description":           schema.StringAttribute{Computed: true, Description: "SAP's description of the value."},
			"key_characteristic_id": schema.StringAttribute{Computed: true, Description: "Id of the key characteristic."},
		},
	}
}

func (d *iaKeyCharacteristicValueDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if c := integrationAssessmentClient(req.ProviderData, iaKeyCharacteristicValueType, "data source", &resp.Diagnostics); c != nil {
		d.client = c
	}
}

func (d *iaKeyCharacteristicValueDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg iaKeyCharacteristicValueModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	characteristics, err := d.client.ListKeyCharacteristics(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Integration Assessment key characteristics", diagnosticDetail(err))
		return
	}
	var characteristicIDs []string
	for _, k := range characteristics {
		if k.Name == cfg.KeyCharacteristic.ValueString() {
			characteristicIDs = append(characteristicIDs, k.ID)
		}
	}
	if len(characteristicIDs) != 1 {
		resp.Diagnostics.AddError("Integration Assessment key characteristic not found or ambiguous",
			fmt.Sprintf("%d key characteristics are named %q; exactly one is needed. Names are compared exactly, including case.",
				len(characteristicIDs), cfg.KeyCharacteristic.ValueString()))
		return
	}
	values, err := d.client.ListKeyCharacteristicValues(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list Integration Assessment key characteristic values", diagnosticDetail(err))
		return
	}
	var matches []integrationassessment.KeyCharacteristicValue
	for _, v := range values {
		if v.Name == cfg.Name.ValueString() && v.KeyCharacteristicID() == characteristicIDs[0] {
			matches = append(matches, v)
		}
	}
	if len(matches) != 1 {
		resp.Diagnostics.AddError("Integration Assessment key characteristic value not found or ambiguous",
			fmt.Sprintf("%d values of key characteristic %q are named %q; exactly one is needed.",
				len(matches), cfg.KeyCharacteristic.ValueString(), cfg.Name.ValueString()))
		return
	}
	cfg.ID = types.StringValue(matches[0].ID)
	cfg.Description = stringPtrOrNull(matches[0].Description)
	cfg.KeyCharacteristicID = types.StringValue(characteristicIDs[0])
	resp.Diagnostics.Append(resp.State.Set(ctx, cfg)...)
}
