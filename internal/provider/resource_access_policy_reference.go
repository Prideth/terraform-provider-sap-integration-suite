package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// NewAccessPolicyReferenceResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_access_policy_reference.
func NewAccessPolicyReferenceResource() resource.Resource {
	return &accessPolicyReferenceResource{}
}

type accessPolicyReferenceResource struct {
	client *cloudintegration.Client
	// allowUnofficial is the provider's enable_unofficial: references to
	// artifact types SAP does not document for access policies need it.
	allowUnofficial bool
}

type accessPolicyReferenceModel struct {
	ID             types.String `tfsdk:"id"`
	AccessPolicyID types.String `tfsdk:"access_policy_id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	ArtifactType   types.String `tfsdk:"artifact_type"`
	Attribute      types.String `tfsdk:"attribute"`
	Operator       types.String `tfsdk:"operator"`
	Value          types.String `tfsdk:"value"`
}

func (r *accessPolicyReferenceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_policy_reference"
}

func (r *accessPolicyReferenceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "One artifact reference of an access policy: a rule that says which artifacts " +
			"(by type, and by name or ID) the policy protects. A policy usually has several. Each " +
			"reference is its own ArtifactReferences entity with a server-assigned ID, which is " +
			"why it is a separate resource and not a block inside sapintegrationsuite_access_policy.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Composite identifier \"<access_policy_id>/<reference_id>\", both numeric SAP IDs.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"access_policy_id": schema.StringAttribute{
				Required:    true,
				Description: "Numeric ID of the access policy this reference belongs to.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					int64StringValidator{},
				},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of the reference as shown in the policy's References table. Mandatory in SAP. " +
					"At most 50 characters: SAP stores only the first 50, so the provider rejects a longer " +
					"name during planning.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					keptLengthValidator{max: accessPolicyReferenceNameMaxLength, summary: "Access policy reference name is too long",
						what: "access policy reference names", advice: "Use a short label and put details into description."},
				},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Description: "Optional description, for example what a regular expression is meant to match. " +
					"At most 200 characters: SAP stores only the first 200, so the provider rejects a longer " +
					"description during planning.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					keptLengthValidator{max: accessPolicyReferenceDescriptionLength, summary: "Access policy reference description is too long",
						what: "access policy reference descriptions", advice: "Shorten the description."},
				},
			},
			"artifact_type": schema.StringAttribute{
				Required: true,
				Description: "Artifact type constant as SAP's API stores it in the Type property, for " +
					"example \"INTEGRATION_FLOW\" or \"INTEGRATION_PACKAGE\", not the UI label. Only the " +
					"types listed on this page are accepted; the plan fails for any other value.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					referenceArtifactTypeValidator,
				},
			},
			"attribute": schema.StringAttribute{
				Required: true,
				Description: "Artifact attribute the condition is evaluated against, as stored in " +
					"ConditionAttribute: \"Name\" or \"ID\". Message queues, global variables and global " +
					"data stores can only be matched by \"Name\".",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					referenceAttributeValidator,
				},
			},
			"operator": schema.StringAttribute{
				Required: true,
				Description: "Condition type as stored in ConditionType: \"exactString\" (Equals in the " +
					"UI) or \"regularExpression\" (Matches in the UI). Integration packages only allow " +
					"\"exactString\". UI labels such as EQUALS or MATCHES are rejected.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					referenceOperatorValidator,
				},
			},
			"value": schema.StringAttribute{
				Required: true,
				Description: "Stored in ConditionValue. With \"exactString\" the exact name or ID, taken " +
					"literally. With \"regularExpression\" a Java regular expression, for example " +
					"\"SALES_.*\" for every name that starts with SALES_ (not the glob \"SALES_*\"). " +
					"At most 150 characters: SAP stores only the first 150, so the provider rejects a " +
					"longer value during planning.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					keptLengthValidator{max: accessPolicyReferenceValueMaxLength, summary: "Access policy reference value is too long",
						what: "access policy reference values", advice: "Match several artifacts with a shorter regularExpression, or split the condition into several references."},
				},
			},
		},
	}
}

// ValidateConfig checks what the single attribute validators cannot: whether
// SAP allows the attribute and operator for the artifact type, and whether a
// regular expression is well-formed. It runs before the plan, so a reference
// SAP would reject is never sent while its policy is being created.
func (r *accessPolicyReferenceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config accessPolicyReferenceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if known(config.ArtifactType, config.Attribute, config.Operator) {
		validateReferenceCombination(config.ArtifactType.ValueString(), config.Attribute.ValueString(),
			config.Operator.ValueString(), func(attr path.Path, summary, detail string) {
				resp.Diagnostics.AddAttributeError(attr, summary, detail)
			})
	}

	if !known(config.Operator, config.Value) || config.Operator.ValueString() != referenceOperatorRegex {
		return
	}
	value := config.Value.ValueString()
	if err := javaRegexError(value); err != nil {
		detail := fmt.Sprintf("SAP expects a Java regular expression for the regularExpression operator, and %q is "+
			"not one: %s.", value, err)
		if strings.HasPrefix(value, "*") {
			detail += " A leading * is glob syntax; in a regular expression, .* stands for any characters."
		}
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Invalid regular expression", detail)
		return
	}
	if suggestion := globStarSuggestion(value); suggestion != "" {
		resp.Diagnostics.AddAttributeWarning(path.Root("value"), "Regular expression looks like a glob pattern",
			fmt.Sprintf("In a regular expression, * repeats only the character before it, so %q does not match "+
				"every name that starts with the text before the *. If that is what you mean, write %q. "+
				"The value is sent to SAP unchanged.", value, suggestion))
	}
}

// ModifyPlan stops a plan that creates a reference to an artifact type SAP
// does not document for access policies, unless enable_unofficial is set.
// Every attribute forces replacement, so creating is the only write.
func (r *accessPolicyReferenceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if r.client == nil || req.Plan.Raw.IsNull() {
		return // provider not configured yet (Create checks again), or a destroy
	}
	if !req.State.Raw.IsNull() && len(resp.RequiresReplace) == 0 {
		return // nothing is created
	}
	var artifactType types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("artifact_type"), &artifactType)...)
	if resp.Diagnostics.HasError() || artifactType.IsUnknown() {
		return
	}
	r.allowArtifactType(artifactType.ValueString(), &resp.Diagnostics)
}

// allowArtifactType reports whether a reference to the artifact type may be
// created, and explains the refusal of an unofficial one.
func (r *accessPolicyReferenceResource) allowArtifactType(artifactType string, diags *diag.Diagnostics) bool {
	t, ok := referenceArtifactTypeByWireValue(artifactType)
	if !ok || !t.Unofficial {
		return true
	}
	return requireUnofficialOperation(r.allowUnofficial, "sapintegrationsuite_access_policy_reference",
		referenceUnofficialTypeOperation+" ("+artifactType+")", diags)
}

// referenceUnofficialTypeOperation names the gated operation the way the
// feature catalog lists it.
const referenceUnofficialTypeOperation = "create a reference to an artifact type SAP does not document for access policies"

// known reports whether every value is set and known.
func known(values ...types.String) bool {
	for _, v := range values {
		if v.IsNull() || v.IsUnknown() {
			return false
		}
	}
	return true
}

func (r *accessPolicyReferenceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
	r.allowUnofficial = data.EnableUnofficial
}

func (r *accessPolicyReferenceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan accessPolicyReferenceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.allowArtifactType(plan.ArtifactType.ValueString(), &resp.Diagnostics) {
		return
	}

	created, err := r.client.CreateAccessPolicyReference(ctx, plan.AccessPolicyID.ValueString(), cloudintegration.AccessPolicyReference{
		Name:               plan.Name.ValueString(),
		Description:        plan.Description.ValueString(),
		Type:               plan.ArtifactType.ValueString(),
		ConditionAttribute: plan.Attribute.ValueString(),
		ConditionType:      plan.Operator.ValueString(),
		ConditionValue:     plan.Value.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite access policy reference", diagnosticDetail(err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, referenceToModel(plan.AccessPolicyID.ValueString(), created))...)
}

func (r *accessPolicyReferenceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state accessPolicyReferenceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ref, err := r.client.FindAccessPolicyReference(ctx, state.AccessPolicyID.ValueString(), referenceIDFrom(state.ID.ValueString()))
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite access policy reference", diagnosticDetail(err))
		return
	}
	if ref == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, referenceToModel(state.AccessPolicyID.ValueString(), ref))...)
}

// Update is unreachable: every attribute forces replacement, because SAP's
// public tooling only creates and deletes references and no in-place update
// contract for ArtifactReferences has been confirmed.
func (r *accessPolicyReferenceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Update not supported",
		"sapintegrationsuite_access_policy_reference does not support in-place updates; "+
			"Terraform should have replaced this resource instead of updating it.",
	)
}

func (r *accessPolicyReferenceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state accessPolicyReferenceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.DeleteAccessPolicyReference(ctx, referenceIDFrom(state.ID.ValueString()))
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite access policy reference", diagnosticDetail(err))
	}
}

func (r *accessPolicyReferenceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	policyID, referenceID, err := splitCompositeID(req.ID)
	if err == nil {
		for _, id := range []string{policyID, referenceID} {
			if _, perr := strconv.ParseInt(id, 10, 64); perr != nil {
				err = fmt.Errorf("both parts of %q must be numeric SAP IDs", req.ID)
				break
			}
		}
	}
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected \"<access_policy_id>/<reference_id>\" with numeric IDs: "+err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("access_policy_id"), policyID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRootID(), policyID+"/"+referenceID)...)
}

func referenceToModel(policyID string, ref *cloudintegration.AccessPolicyReference) accessPolicyReferenceModel {
	return accessPolicyReferenceModel{
		ID:             types.StringValue(policyID + "/" + ref.ID),
		AccessPolicyID: types.StringValue(policyID),
		Name:           types.StringValue(ref.Name),
		Description:    stringOrNull(ref.Description),
		ArtifactType:   types.StringValue(ref.Type),
		Attribute:      types.StringValue(ref.ConditionAttribute),
		Operator:       types.StringValue(ref.ConditionType),
		Value:          types.StringValue(ref.ConditionValue),
	}
}

// referenceIDFrom extracts the reference ID segment from a composite
// "<access_policy_id>/<reference_id>" Terraform ID.
func referenceIDFrom(id string) string {
	_, referenceID, err := splitCompositeID(id)
	if err != nil {
		return id
	}
	return referenceID
}

// int64StringValidator accepts only base-10 integers that fit Edm.Int64.
type int64StringValidator struct{}

func (int64StringValidator) Description(context.Context) string {
	return "value must be a numeric SAP ID"
}

func (v int64StringValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (int64StringValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if _, err := strconv.ParseInt(req.ConfigValue.ValueString(), 10, 64); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid SAP ID",
			fmt.Sprintf("%q is not a numeric SAP ID.", req.ConfigValue.ValueString()))
	}
}
