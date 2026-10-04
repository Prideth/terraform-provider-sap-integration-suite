package provider

import (
	"context"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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

var (
	_ resource.Resource                = &serviceInterfaceResource{}
	_ resource.ResourceWithConfigure   = &serviceInterfaceResource{}
	_ resource.ResourceWithImportState = &serviceInterfaceResource{}
)

// NewServiceInterfaceResource manages ServiceInterfaceDesigntimeArtifacts.
func NewServiceInterfaceResource() resource.Resource {
	return &serviceInterfaceResource{}
}

type serviceInterfaceResource struct {
	client *cloudintegration.Client
}

type serviceInterfaceModel struct {
	ID                 types.String                     `tfsdk:"id"`
	PackageID          types.String                     `tfsdk:"package_id"`
	ServiceInterfaceID types.String                     `tfsdk:"service_interface_id"`
	Name               types.String                     `tfsdk:"name"`
	Namespace          types.String                     `tfsdk:"namespace"`
	Description        types.String                     `tfsdk:"description"`
	Operations         []serviceInterfaceOperationModel `tfsdk:"operation"`
	Category           types.String                     `tfsdk:"category"`
	InterfacePattern   types.String                     `tfsdk:"interface_pattern"`
	Version            types.String                     `tfsdk:"version"`
	SaveAsVersion      types.String                     `tfsdk:"save_as_version"`
}

type serviceInterfaceOperationModel struct {
	Name                  types.String   `tfsdk:"name"`
	RequestMessageTypeID  types.String   `tfsdk:"request_message_type_id"`
	ResponseMessageTypeID types.String   `tfsdk:"response_message_type_id"`
	FaultMessageTypeIDs   []types.String `tfsdk:"fault_message_type_ids"`
}

const serviceInterfaceTypeName = "sapintegrationsuite_service_interface"

func (r *serviceInterfaceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_interface"
}

func (r *serviceInterfaceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a service interface in an integration package: its operations and the " +
			"message types, response message types and fault message types they use. SAP documents no " +
			"request for service interfaces; the Integration Content API's $metadata declares them, and " +
			"the operation model follows the bundles SAP's editor writes, so this resource needs the " +
			"provider's opt-in switch shown in the status.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Composite identifier in the form \"<package_id>/<service_interface_id>\".",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"package_id": schema.StringAttribute{
				Required:      true,
				Description:   "ID of the integration package the service interface belongs to. Changing it replaces the service interface.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"service_interface_id": schema.StringAttribute{
				Required:      true,
				Description:   "Technical ID of the service interface. Changing it replaces the service interface.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of the service interface. Changing it replaces the service interface, " +
					"because an in-place rename was not tested.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"namespace": schema.StringAttribute{
				Required: true,
				Description: "XML namespace of the service interface. Changing it replaces the service " +
					"interface, because an in-place namespace change was not tested.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Description: "Description. Changes in place; removing it replaces the service interface, " +
					"because clearing it in place was not tested.",
				PlanModifiers: []planmodifier.String{requiresReplaceWhenRemoved()},
				Validators:    []validator.String{stringvalidator.LengthAtLeast(1)},
			},
			"operation": schema.ListNestedAttribute{
				Required: true,
				Description: "The operations, in order. An operation with a response message type is " +
					"synchronous, one without is asynchronous. Changes in place: the provider writes the " +
					"operations into the service interface's bundle and uploads it.",
				Validators: []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:    true,
							Description: "Name of the operation.",
							Validators:  []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"request_message_type_id": schema.StringAttribute{
							Optional: true,
							Description: "ID of the message type of the request, usually " +
								"sapintegrationsuite_message_type.<name>.message_type_id.",
							Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
						},
						"response_message_type_id": schema.StringAttribute{
							Optional: true,
							Description: "ID of the message type of the response. Setting it makes the " +
								"operation synchronous; it needs a request message type.",
							Validators: []validator.String{
								stringvalidator.LengthAtLeast(1),
								stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("request_message_type_id")),
							},
						},
						"fault_message_type_ids": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
							Description: "IDs of the fault message types the operation can raise, usually " +
								"sapintegrationsuite_fault_message_type.<name>.fault_message_type_id.",
							Validators: []validator.List{listvalidator.SizeAtLeast(1), listvalidator.ValueStringsAre(stringvalidator.LengthAtLeast(1))},
						},
					},
				},
			},
			"category": schema.StringAttribute{
				Computed:      true,
				Description:   "Direction SAP records for the interface, OUTBOUND for an interface created through the API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"interface_pattern": schema.StringAttribute{
				Computed:      true,
				Description:   "Interface pattern SAP records, STATELESS for an interface created through the API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"version": schema.StringAttribute{
				Computed:    true,
				Description: "The design-time version SAP reports.",
			},
			"save_as_version": saveAsVersionAttribute("service interface"),
		},
	}
}

func (r *serviceInterfaceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, serviceInterfaceTypeName, &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = cloudintegration.New(data.HTTPClient, data.Host)
}

// operationSpecs resolves the configured message types to the references
// an operation records (name, namespace, version, package).
func (r *serviceInterfaceResource) operationSpecs(ctx context.Context, ops []serviceInterfaceOperationModel) ([]cloudintegration.ServiceInterfaceOperationSpec, error) {
	resolve := func(kind cloudintegration.MessageTypeKind, id types.String) (*cloudintegration.ServiceInterfaceMessageRef, error) {
		if id.IsNull() || id.IsUnknown() || id.ValueString() == "" {
			return nil, nil
		}
		ref, err := r.client.ServiceInterfaceMessage(ctx, kind, id.ValueString())
		if err != nil {
			return nil, err
		}
		return &ref, nil
	}
	specs := make([]cloudintegration.ServiceInterfaceOperationSpec, 0, len(ops))
	for _, op := range ops {
		spec := cloudintegration.ServiceInterfaceOperationSpec{Name: op.Name.ValueString()}
		var err error
		if spec.Request, err = resolve(cloudintegration.KindMessageType, op.RequestMessageTypeID); err != nil {
			return nil, err
		}
		if spec.Response, err = resolve(cloudintegration.KindMessageType, op.ResponseMessageTypeID); err != nil {
			return nil, err
		}
		for _, id := range op.FaultMessageTypeIDs {
			fault, err := resolve(cloudintegration.KindFaultMessageType, id)
			if err != nil {
				return nil, err
			}
			if fault != nil {
				spec.Faults = append(spec.Faults, *fault)
			}
		}
		specs = append(specs, spec)
	}
	return specs, nil
}

func (r *serviceInterfaceResource) toModel(si *cloudintegration.ServiceInterface, packageID string, previous serviceInterfaceModel) serviceInterfaceModel {
	m := serviceInterfaceModel{
		ID:                 types.StringValue(packageID + "/" + si.ID),
		PackageID:          types.StringValue(packageID),
		ServiceInterfaceID: types.StringValue(si.ID),
		Name:               types.StringValue(si.Name),
		Namespace:          stringOrNull(si.Namespace),
		Description:        stringOrNull(si.Description),
		Category:           types.StringNull(),
		InterfacePattern:   types.StringNull(),
		Version:            types.StringValue(si.Version),
		SaveAsVersion:      previous.SaveAsVersion,
	}
	if si.Model == nil {
		return m
	}
	m.Category = stringOrNull(si.Model.Category)
	m.InterfacePattern = stringOrNull(si.Model.InterfacePattern)
	for _, op := range si.Model.Operations {
		om := serviceInterfaceOperationModel{
			Name:                  types.StringValue(op.Name),
			RequestMessageTypeID:  types.StringNull(),
			ResponseMessageTypeID: types.StringNull(),
		}
		if op.Request != nil && op.Request.BundleSymbolicName != "" {
			om.RequestMessageTypeID = types.StringValue(op.Request.BundleSymbolicName)
		}
		if op.Response != nil && op.Response.BundleSymbolicName != "" {
			om.ResponseMessageTypeID = types.StringValue(op.Response.BundleSymbolicName)
		}
		for _, f := range op.Faults {
			om.FaultMessageTypeIDs = append(om.FaultMessageTypeIDs, types.StringValue(f.BundleSymbolicName))
		}
		m.Operations = append(m.Operations, om)
	}
	return m
}

func (r *serviceInterfaceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, packageID := plan.ServiceInterfaceID.ValueString(), plan.PackageID.ValueString()
	specs, err := r.operationSpecs(ctx, plan.Operations)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the message types of the service interface", diagnosticDetail(err))
		return
	}
	// SAP creates a service interface without content (it generates one
	// operation without messages); the operations follow in an update.
	si, err := r.client.CreateServiceInterface(ctx, cloudintegration.ServiceInterface{
		ID: id, Name: plan.Name.ValueString(), PackageID: packageID,
		Namespace: plan.Namespace.ValueString(), Description: plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create SAP Integration Suite service interface", diagnosticDetail(err))
		return
	}
	unsaved := plan
	unsaved.SaveAsVersion = types.StringNull()
	updated, err := r.client.UpdateServiceInterface(ctx, id, plan.Name.ValueString(), plan.Description.ValueString(), specs)
	if err != nil {
		r.setState(ctx, resp, r.toModel(si, packageID, unsaved), &resp.Diagnostics)
		resp.Diagnostics.AddError("Created the service interface, but writing its operations failed", diagnosticDetail(err))
		return
	}
	si = updated
	if v, due := versionToSave(plan.SaveAsVersion, types.StringNull()); due {
		saved, err := r.client.SaveServiceInterfaceAsVersion(ctx, id, v)
		if err != nil {
			r.setState(ctx, resp, r.toModel(si, packageID, unsaved), &resp.Diagnostics)
			resp.Diagnostics.AddError("Created the service interface, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		si = saved
	}
	r.setState(ctx, resp, r.toModel(si, packageID, plan), &resp.Diagnostics)
}

func (r *serviceInterfaceResource) setState(ctx context.Context, resp *resource.CreateResponse, m serviceInterfaceModel, diags *diag.Diagnostics) {
	diags.Append(resp.State.Set(ctx, m)...)
}

func (r *serviceInterfaceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	si, err := r.client.GetServiceInterface(ctx, state.ServiceInterfaceID.ValueString())
	if err != nil {
		var apiErr *apierror.Error
		if errors.As(err, &apiErr) && apiErr.IsNotFound() {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite service interface", diagnosticDetail(err))
		return
	}
	packageID := state.PackageID.ValueString()
	if si.PackageID != "" {
		packageID = si.PackageID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(si, packageID, state))...)
}

func (r *serviceInterfaceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, prior serviceInterfaceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id, packageID := plan.ServiceInterfaceID.ValueString(), plan.PackageID.ValueString()
	specs, err := r.operationSpecs(ctx, plan.Operations)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the message types of the service interface", diagnosticDetail(err))
		return
	}
	si, err := r.client.UpdateServiceInterface(ctx, id, plan.Name.ValueString(), plan.Description.ValueString(), specs)
	if err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite service interface", diagnosticDetail(err))
		return
	}
	if v, due := versionToSave(plan.SaveAsVersion, prior.SaveAsVersion); due {
		saved, err := r.client.SaveServiceInterfaceAsVersion(ctx, id, v)
		if err != nil {
			unsaved := plan
			unsaved.SaveAsVersion = prior.SaveAsVersion
			resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(si, packageID, unsaved))...)
			resp.Diagnostics.AddError("Updated the service interface, but saving it as version "+v+" failed", diagnosticDetail(err))
			return
		}
		si = saved
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, r.toModel(si, packageID, plan))...)
}

func (r *serviceInterfaceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceInterfaceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteServiceInterface(ctx, state.ServiceInterfaceID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite service interface", diagnosticDetail(err))
	}
}

func (r *serviceInterfaceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	packageID, id, err := splitCompositeID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID",
			fmt.Sprintf("Expected \"<package_id>/<service_interface_id>\", got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("package_id"), packageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_interface_id"), id)...)
}
