package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// NewAccessPolicyRuntimeAssignmentsDataSource returns a fresh
// datasource.DataSource implementation for
// sapintegrationsuite_access_policy_runtime_assignments.
func NewAccessPolicyRuntimeAssignmentsDataSource() datasource.DataSource {
	return &accessPolicyRuntimeAssignmentsDataSource{}
}

type accessPolicyRuntimeAssignmentsDataSource struct {
	client *cloudintegration.Client
}

type accessPolicyRuntimeAssignmentsModel struct {
	AccessPolicyID types.String                         `tfsdk:"access_policy_id"`
	Assignments    []accessPolicyRuntimeAssignmentEntry `tfsdk:"assignments"`
}

type accessPolicyRuntimeAssignmentEntry struct {
	ID                types.String `tfsdk:"id"`
	RuntimeLocationID types.String `tfsdk:"runtime_location_id"`
	TransferStatus    types.String `tfsdk:"transfer_status"`
	TransferErrors    types.String `tfsdk:"transfer_errors"`
	StatusUpdatedAt   types.String `tfsdk:"status_updated_at"`
}

func (d *accessPolicyRuntimeAssignmentsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy_runtime_assignments"
}

func (d *accessPolicyRuntimeAssignmentsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "UNOFFICIAL: needs enable_unofficial = true in the provider block. Lists the runtimes an access policy is replicated to (Cloud Integration runtime, " +
			"Integration Cell, Edge Integration Cells) and the replication state SAP reports for " +
			"each. This is the data behind the Runtimes column of the Access Policies screen. " +
			"Read-only: which runtimes a policy is assigned to is still chosen in the UI. UNOFFICIAL: the " +
			"AccessPolicyRuntimeAssignments navigation is known from the service's $metadata and was " +
			"read on a tenant, but SAP does not document it.",
		Attributes: map[string]schema.Attribute{
			"access_policy_id": schema.StringAttribute{
				Required:    true,
				Description: "Numeric ID of the access policy.",
				Validators:  []validator.String{int64StringValidator{}},
			},
			"assignments": schema.ListNestedAttribute{
				Computed:    true,
				Description: "One entry per runtime the policy is assigned to, sorted by runtime_location_id.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:    true,
							Description: "Numeric ID of the assignment.",
						},
						"runtime_location_id": schema.StringAttribute{
							Computed:    true,
							Description: "Identifier of the runtime location, as SAP stores it.",
						},
						"transfer_status": schema.StringAttribute{
							Computed: true,
							Description: "Replication state as SAP returns it. The UI shows Fail, Success " +
								"or Pending; the exact API values are not documented, so they are " +
								"passed through unchanged.",
						},
						"transfer_errors": schema.StringAttribute{
							Computed:    true,
							Description: "Error details of a failed replication, or null.",
						},
						"status_updated_at": schema.StringAttribute{
							Computed:    true,
							Description: "When the status last changed, as an RFC 3339 timestamp in UTC, or null.",
						},
					},
				},
			},
		},
	}
}

func (d *accessPolicyRuntimeAssignmentsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, "sapintegrationsuite_access_policy_runtime_assignments", &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "data source", &resp.Diagnostics) {
		return
	}
	d.client = cloudintegration.New(data.HTTPClient, data.Host)
}

func (d *accessPolicyRuntimeAssignmentsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config accessPolicyRuntimeAssignmentsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	assignments, err := d.client.ListAccessPolicyRuntimeAssignments(ctx, config.AccessPolicyID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite access policy runtime assignments", diagnosticDetail(err))
		return
	}

	sort.Slice(assignments, func(i, j int) bool {
		if assignments[i].RuntimeLocationID != assignments[j].RuntimeLocationID {
			return assignments[i].RuntimeLocationID < assignments[j].RuntimeLocationID
		}
		return assignments[i].ID < assignments[j].ID
	})

	entries := make([]accessPolicyRuntimeAssignmentEntry, 0, len(assignments))
	for _, a := range assignments {
		entries = append(entries, accessPolicyRuntimeAssignmentEntry{
			ID:                types.StringValue(a.ID),
			RuntimeLocationID: stringOrNull(a.RuntimeLocationID),
			TransferStatus:    stringOrNull(a.TransferStatus),
			TransferErrors:    stringOrNull(a.TransferErrors),
			StatusUpdatedAt:   odataDateToRFC3339(a.StatusUpdatedAt),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, accessPolicyRuntimeAssignmentsModel{
		AccessPolicyID: config.AccessPolicyID,
		Assignments:    entries,
	})...)
}
