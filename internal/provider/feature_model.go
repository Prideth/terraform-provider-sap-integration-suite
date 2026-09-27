package provider

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// operationsModel mirrors features.Operations for Terraform state.
type operationsModel struct {
	Create   types.Bool `tfsdk:"create"`
	Read     types.Bool `tfsdk:"read"`
	Update   types.Bool `tfsdk:"update"`
	Delete   types.Bool `tfsdk:"delete"`
	Import   types.Bool `tfsdk:"import"`
	Deploy   types.Bool `tfsdk:"deploy"`
	Undeploy types.Bool `tfsdk:"undeploy"`
}

// featureModel mirrors features.Feature for Terraform state. Shared between
// sapintegrationsuite_provider_features (a list of these) and
// sapintegrationsuite_provider_feature (exactly one, flattened into the
// data source's own top-level attributes by featureAttributes(true)).
type featureModel struct {
	Key             types.String    `tfsdk:"key"`
	Domain          types.String    `tfsdk:"domain"`
	Name            types.String    `tfsdk:"name"`
	Description     types.String    `tfsdk:"description"`
	SupportStatus   types.String    `tfsdk:"support_status"`
	SupportReason   types.String    `tfsdk:"support_reason"`
	ResourceTypes   []types.String  `tfsdk:"resource_types"`
	DataSourceTypes []types.String  `tfsdk:"data_source_types"`
	PublicAPI       types.Bool      `tfsdk:"public_api"`
	APIProtocol     types.String    `tfsdk:"api_protocol"`
	Planned         types.Bool      `tfsdk:"planned"`
	Limitations     []types.String  `tfsdk:"limitations"`
	Operations      operationsModel `tfsdk:"operations"`
	ContractSource  types.String    `tfsdk:"contract_source"`
	Undocumented    []types.String  `tfsdk:"undocumented_operations"`
}

// featureAttributes returns the schema attributes shared by both feature
// data sources, with "key" as a computed output. sapintegrationsuite_provider_feature
// overrides "key" to be a required input instead, since there the caller
// supplies which feature to look up — see its own Schema method.
func featureAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"key": schema.StringAttribute{
			Computed:    true,
			Description: "Stable, hierarchical, machine-readable feature identifier, for example \"cloud_integration.value_mapping\".",
		},
		"domain": schema.StringAttribute{
			Computed:    true,
			Description: "Groups related features, for example \"cloud_integration\" or \"security\".",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Short, human-readable feature name.",
		},
		"description": schema.StringAttribute{
			Computed:    true,
			Description: "What the feature is and, briefly, its support status.",
		},
		"support_status": schema.StringAttribute{
			Computed: true,
			Description: "One of \"supported\", \"partial\", \"read_only\", \"unofficial\" (works, but " +
				"the contract is known only from the service's $metadata), \"experimental\", " +
				"\"unsupported\" or \"separate_provider\". See docs/feature-support.md for exact meanings.",
		},
		"support_reason": schema.StringAttribute{
			Computed: true,
			Description: "Why support_status is not \"supported\": one of \"not_implemented\", " +
				"\"public_api_incomplete\", \"no_public_api\", \"research_required\", " +
				"\"out_of_scope\", or \"unsafe_terraform_lifecycle\". Empty when support_status is " +
				"\"supported\".",
		},
		"resource_types": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Full Terraform resource type names this provider registers for this feature, if any.",
		},
		"data_source_types": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Full Terraform data source type names this provider registers for this feature, if any.",
		},
		"public_api": schema.BoolAttribute{
			Computed: true,
			Description: "Whether SAP publishes a public, supported API for this feature at all — " +
				"independent of whether this provider implements it.",
		},
		"api_protocol": schema.StringAttribute{
			Computed:    true,
			Description: "The public API's wire protocol, for example \"OData V2\", when public_api is true and a single protocol applies.",
		},
		"planned": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether this feature has a concrete place on this provider's roadmap.",
		},
		"limitations": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Specific, concrete caveats to know before relying on this feature.",
		},
		"contract_source": schema.StringAttribute{
			Computed: true,
			Description: "Where the contract of the implemented part comes from: \"sap_documentation\", " +
				"\"api_specification\", \"sap_tooling\" or \"metadata_only\". Empty when nothing is " +
				"implemented.",
		},
		"undocumented_operations": schema.ListAttribute{
			Computed:    true,
			ElementType: types.StringType,
			Description: "Implemented operations that work but that SAP does not document, although the rest of the feature is documented.",
		},
		"operations": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Which lifecycle operations this provider implements for this feature.",
			Attributes: map[string]schema.Attribute{
				"create":   schema.BoolAttribute{Computed: true},
				"read":     schema.BoolAttribute{Computed: true},
				"update":   schema.BoolAttribute{Computed: true},
				"delete":   schema.BoolAttribute{Computed: true},
				"import":   schema.BoolAttribute{Computed: true},
				"deploy":   schema.BoolAttribute{Computed: true},
				"undeploy": schema.BoolAttribute{Computed: true},
			},
		},
	}
}

// stringList converts a []string to the []types.String shape Terraform
// list attributes expect, returning an empty (not nil) slice for a nil or
// empty input so the resulting Terraform value is an empty list rather
// than null — every Feature.ResourceTypes/DataSourceTypes/Limitations is a
// "the answer is zero items" case, not an "unknown/not applicable" one.
func stringList(values []string) []types.String {
	out := make([]types.String, 0, len(values))
	for _, v := range values {
		out = append(out, types.StringValue(v))
	}
	return out
}

// featureToModel converts a features.Feature into Terraform state.
func featureToModel(f features.Feature) featureModel {
	return featureModel{
		Key:             types.StringValue(f.Key),
		Domain:          types.StringValue(f.Domain),
		Name:            types.StringValue(f.Name),
		Description:     types.StringValue(f.Description),
		SupportStatus:   types.StringValue(string(f.SupportStatus)),
		SupportReason:   types.StringValue(string(f.SupportReason)),
		ResourceTypes:   stringList(f.ResourceTypes),
		DataSourceTypes: stringList(f.DataSourceTypes),
		PublicAPI:       types.BoolValue(f.PublicAPI),
		APIProtocol:     types.StringValue(f.APIProtocol),
		Planned:         types.BoolValue(f.Planned),
		Limitations:     stringList(f.Limitations),
		ContractSource:  types.StringValue(string(f.ContractSource)),
		Undocumented:    stringList(f.UndocumentedOperations),
		Operations: operationsModel{
			Create:   types.BoolValue(f.Operations.Create),
			Read:     types.BoolValue(f.Operations.Read),
			Update:   types.BoolValue(f.Operations.Update),
			Delete:   types.BoolValue(f.Operations.Delete),
			Import:   types.BoolValue(f.Operations.Import),
			Deploy:   types.BoolValue(f.Operations.Deploy),
			Undeploy: types.BoolValue(f.Operations.Undeploy),
		},
	}
}
