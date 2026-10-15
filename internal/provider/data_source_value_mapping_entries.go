package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

const valueMappingEntriesType = "sapintegrationsuite_value_mapping_entries"

// NewValueMappingEntriesDataSource returns the data source for
// sapintegrationsuite_value_mapping_entries.
func NewValueMappingEntriesDataSource() datasource.DataSource {
	return &valueMappingEntriesDataSource{}
}

type valueMappingEntriesDataSource struct {
	client *cloudintegration.Client
}

type valueMappingEntriesModel struct {
	ID               types.String             `tfsdk:"id"`
	ValueMappingID   types.String             `tfsdk:"value_mapping_id"`
	SourceAgency     types.String             `tfsdk:"source_agency"`
	SourceIdentifier types.String             `tfsdk:"source_identifier"`
	TargetAgency     types.String             `tfsdk:"target_agency"`
	TargetIdentifier types.String             `tfsdk:"target_identifier"`
	Entries          []valueMappingEntryValue `tfsdk:"entries"`
}

type valueMappingEntryValue struct {
	EntryID     types.String `tfsdk:"entry_id"`
	SourceValue types.String `tfsdk:"source_value"`
	TargetValue types.String `tfsdk:"target_value"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (d *valueMappingEntriesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_value_mapping_entries"
}

func (d *valueMappingEntriesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	required := func(description string) schema.StringAttribute {
		return schema.StringAttribute{Required: true, Description: description, Validators: []validator.String{stringvalidator.LengthAtLeast(1)}}
	}
	resp.Schema = schema.Schema{
		Description: "The source and target value pairs of one agency pair of a value mapping, read through " +
			"ValMapSchema(...)/ValMaps of the Integration Content API.",
		Attributes: map[string]schema.Attribute{
			"id":                schema.StringAttribute{Computed: true, Description: "<value_mapping_id>/<source_agency>/<source_identifier>/<target_agency>/<target_identifier>."},
			"value_mapping_id":  required("ID of the value mapping."),
			"source_agency":     required("Source agency of the agency pair."),
			"source_identifier": required("Source identifier (schema) of the agency pair."),
			"target_agency":     required("Target agency of the agency pair."),
			"target_identifier": required("Target identifier (schema) of the agency pair."),
			"entries": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The entries in the order SAP returns them. Reading the pair the other way round swaps source and target values.",
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"entry_id":     schema.StringAttribute{Computed: true, Description: "The ID SAP gives the entry."},
					"source_value": schema.StringAttribute{Computed: true, Description: "The source value."},
					"target_value": schema.StringAttribute{Computed: true, Description: "The target value."},
					"is_default": schema.BoolAttribute{Computed: true,
						Description: "Whether SAP uses this entry by default for its source value."},
				}},
			},
		},
	}
}

func (d *valueMappingEntriesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, valueMappingEntriesType, &resp.Diagnostics) || !requireHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = cloudintegration.New(data.HTTPClient, data.Host)
}

func (d *valueMappingEntriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var m valueMappingEntriesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &m)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappingID := m.ValueMappingID.ValueString()
	pair := cloudintegration.ValueMappingAgencyPair{
		SourceAgency: m.SourceAgency.ValueString(), SourceIdentifier: m.SourceIdentifier.ValueString(),
		TargetAgency: m.TargetAgency.ValueString(), TargetIdentifier: m.TargetIdentifier.ValueString(),
	}
	entries, err := d.client.ListValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the entries of value mapping "+mappingID, diagnosticDetail(err))
		return
	}
	defaults, err := d.client.ListDefaultValueMappingEntries(ctx, mappingID, pair)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the default entries of value mapping "+mappingID, diagnosticDetail(err))
		return
	}
	isDefault := map[string]bool{}
	for _, e := range defaults {
		isDefault[e.ID] = true
	}
	m.Entries = make([]valueMappingEntryValue, 0, len(entries))
	for _, e := range entries {
		m.Entries = append(m.Entries, valueMappingEntryValue{
			EntryID: types.StringValue(e.ID), SourceValue: types.StringValue(e.SourceValue),
			TargetValue: types.StringValue(e.TargetValue), IsDefault: types.BoolValue(isDefault[e.ID]),
		})
	}
	m.ID = types.StringValue(strings.Join([]string{mappingID, pair.SourceAgency, pair.SourceIdentifier, pair.TargetAgency, pair.TargetIdentifier}, "/"))
	resp.Diagnostics.Append(resp.State.Set(ctx, m)...)
}
