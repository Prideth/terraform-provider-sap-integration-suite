package provider

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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

const xmlSchemaNamespace = "http://www.w3.org/2001/XMLSchema"

var (
	_ resource.Resource                   = &dataTypeResource{}
	_ resource.ResourceWithConfigure      = &dataTypeResource{}
	_ resource.ResourceWithImportState    = &dataTypeResource{}
	_ resource.ResourceWithValidateConfig = &dataTypeResource{}
)

func NewDataTypeResource() resource.Resource {
	return &dataTypeResource{}
}

type dataTypeResource struct {
	client *cloudintegration.Client
}

type dataTypeModel struct {
	ID            types.String `tfsdk:"id"`
	PackageID     types.String `tfsdk:"package_id"`
	DataTypeID    types.String `tfsdk:"data_type_id"`
	Name          types.String `tfsdk:"name"`
	Namespace     types.String `tfsdk:"namespace"`
	Description   types.String `tfsdk:"description"`
	XSD           types.String `tfsdk:"xsd"`
	Version       types.String `tfsdk:"version"`
	SaveAsVersion types.String `tfsdk:"save_as_version"`
}

func (r *dataTypeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_data_type"
}

func (r *dataTypeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a complex data type in an integration package: an XML schema that message " +
			"types and service interfaces build on, through the DataTypeDesigntimeArtifacts of SAP's " +
			"Integration Content API.\n\n" +
			"The provider builds the artifact bundle from xsd, namespace and description the way SAP " +
			"stores a data type, because SAP rejects a bundle without its additional attribute files. " +
			"SAP stores the complex type under the data type's name and normalizes the schema, so the " +
			"schema read from SAP is never compared with xsd: a change outside Terraform is not shown " +
			"as a diff.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Composite identifier in the form \"<package_id>/<data_type_id>\".",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"package_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the integration package the data type belongs to. Changing it replaces the data type.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"data_type_id": schema.StringAttribute{
				Required:      true,
				Description:   "Technical ID of the data type. Changing it replaces the data type.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Display name of the data type. SAP stores the schema's complex type under " +
					"this name.",
				Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"namespace": schema.StringAttribute{
				Required: true,
				Description: "XML namespace of the data type, for example \"urn:example:orders\". It must " +
					"equal the targetNamespace of xsd. Changing it replaces the data type, because an " +
					"in-place namespace change was not tested.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Description of the data type. Omit it rather than setting an empty string.",
				Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"xsd": schema.StringAttribute{
				Required: true,
				Description: "The XML schema of the data type, for example file(\"${path.module}/order.xsd\"): " +
					"an xsd:schema with exactly one top-level complexType and the targetNamespace " +
					"given in namespace. A change uploads the new schema in place.",
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP reports for the data type.",
			},
			"save_as_version": saveAsVersionAttribute("data type"),
		},
	}
}

func (r *dataTypeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, "sapintegrationsuite_data_type", &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
}

// xsdSummary is what the plan-time check needs to know about a schema.
type xsdSummary struct {
	targetNamespace string
	complexTypes    []string
}

// summarizeXSD parses an XML schema far enough to check its root element,
// its targetNamespace and its top-level complex types.
func summarizeXSD(text string) (xsdSummary, error) {
	dec := xml.NewDecoder(strings.NewReader(text))
	var summary xsdSummary
	depth := 0
	sawRoot := false
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return summary, fmt.Errorf("the schema is not well-formed XML: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 1 {
				if t.Name.Space != xmlSchemaNamespace || t.Name.Local != "schema" {
					return summary, fmt.Errorf("the root element must be xsd:schema in namespace %s, got %s", xmlSchemaNamespace, t.Name.Local)
				}
				sawRoot = true
				for _, a := range t.Attr {
					if a.Name.Local == "targetNamespace" && a.Name.Space == "" {
						summary.targetNamespace = a.Value
					}
				}
			}
			if depth == 2 && t.Name.Space == xmlSchemaNamespace && t.Name.Local == "complexType" {
				name := ""
				for _, a := range t.Attr {
					if a.Name.Local == "name" && a.Name.Space == "" {
						name = a.Value
					}
				}
				summary.complexTypes = append(summary.complexTypes, name)
			}
		case xml.EndElement:
			depth--
		}
	}
	if !sawRoot {
		return summary, errors.New("the schema is empty")
	}
	return summary, nil
}

func (r *dataTypeResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config dataTypeModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.XSD.IsNull() || config.XSD.IsUnknown() {
		return
	}
	summary, err := summarizeXSD(config.XSD.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("xsd"), "Invalid data type schema", err.Error())
		return
	}
	if len(summary.complexTypes) != 1 {
		resp.Diagnostics.AddAttributeError(path.Root("xsd"), "Invalid data type schema",
			fmt.Sprintf("A data type is one complex type: the schema needs exactly one top-level complexType, it has %d. "+
				"Model further types as separate data types.", len(summary.complexTypes)))
		return
	}
	if !config.Namespace.IsNull() && !config.Namespace.IsUnknown() && summary.targetNamespace != config.Namespace.ValueString() {
		resp.Diagnostics.AddAttributeError(path.Root("xsd"), "Schema namespace does not match",
			fmt.Sprintf("The schema's targetNamespace is %q, but namespace is %q. Use the same value in both.",
				summary.targetNamespace, config.Namespace.ValueString()))
	}
	if !config.Name.IsNull() && !config.Name.IsUnknown() && summary.complexTypes[0] != config.Name.ValueString() {
		resp.Diagnostics.AddAttributeWarning(path.Root("xsd"), "SAP renames the complex type",
			fmt.Sprintf("The schema's complex type is named %q. SAP stores it under the data type's name %q, so "+
				"references to it must use that name.", summary.complexTypes[0], config.Name.ValueString()))
	}
}

func dataTypeBundle(m dataTypeModel) (cloudintegration.DataTypeBundle, []byte, error) {
	b := cloudintegration.DataTypeBundle{
		ID:          m.DataTypeID.ValueString(),
		Name:        m.Name.ValueString(),
		Namespace:   m.Namespace.ValueString(),
		Description: m.Description.ValueString(),
		XSD:         m.XSD.ValueString(),
	}
	content, err := b.Build(time.Now())
	return b, content, err
}

func dataTypeToModel(dt *cloudintegration.DataType, packageID string, previous dataTypeModel) dataTypeModel {
	return dataTypeModel{
		ID:            types.StringValue(packageID + "/" + dt.ID),
		PackageID:     types.StringValue(packageID),
		DataTypeID:    types.StringValue(dt.ID),
		Name:          types.StringValue(dt.Name),
		Namespace:     stringOrNull(dt.Namespace),
		Description:   stringOrNull(dt.Description),
		XSD:           previous.XSD,
		Version:       types.StringValue(dt.Version),
		SaveAsVersion: previous.SaveAsVersion,
	}
}

func (r *dataTypeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan dataTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, bundle, err := dataTypeBundle(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build the data type bundle", err.Error())
		return
	}
	dt, err := r.client.CreateDataType(ctx, plan.PackageID.ValueString(), b, bundle)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite data type", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveDataTypeAsVersion(ctx, plan.DataTypeID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = types.StringNull()
			resp.Diagnostics.Append(resp.State.Set(ctx, dataTypeToModel(dt, plan.PackageID.ValueString(), unsaved))...)
			resp.Diagnostics.AddError("Data type created, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		dt = saved
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, dataTypeToModel(dt, plan.PackageID.ValueString(), plan))...)
}

func (r *dataTypeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dataTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dt, err := r.client.GetDataType(ctx, state.DataTypeID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite data type", diagnosticDetail(err))
		return
	}
	packageID := state.PackageID.ValueString()
	if dt.PackageID != "" {
		packageID = dt.PackageID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, dataTypeToModel(dt, packageID, state))...)
}

func (r *dataTypeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior dataTypeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, bundle, err := dataTypeBundle(plan)
	if err != nil {
		resp.Diagnostics.AddError("Failed to build the data type bundle", err.Error())
		return
	}
	dt, err := r.client.UpdateDataType(ctx, b, bundle)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite data type", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveDataTypeAsVersion(ctx, plan.DataTypeID.ValueString(), v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			resp.Diagnostics.Append(resp.State.Set(ctx, dataTypeToModel(dt, plan.PackageID.ValueString(), unsaved))...)
			resp.Diagnostics.AddError("Data type updated, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		dt = saved
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, dataTypeToModel(dt, plan.PackageID.ValueString(), plan))...)
}

func (r *dataTypeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state dataTypeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDataType(ctx, state.DataTypeID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite data type", diagnosticDetail(err))
	}
}

func (r *dataTypeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, dataTypeID, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			"Expected \"<package_id>/<data_type_id>\", got "+fmt.Sprintf("%q", req.ID)+".")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("data_type_id"), dataTypeID)...)
}
