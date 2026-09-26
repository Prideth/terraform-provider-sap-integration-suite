package provider

import (
	"context"
	"fmt"
	"strings"

	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const runtimeLocationDescription = "NOT SUPPORTED YET. Edge Integration Cell targeting has not " +
	"passed its acceptance test on a tenant with an Edge Integration Cell; leave this unset until " +
	"it has. If set, requests go to /location/<id>/api/v1 on the same " +
	"tenant host, the service root SAP Help documents for Edge Integration Cells, and the value " +
	"is the runtime location ID SAP shows in the monitoring URL after selecting the Edge " +
	"Integration Cell as runtime ({\"edge\":{\"runtimeLocationId\":\"myedge\"}}). Use it only at " +
	"your own risk."

// runtimeLocationResourceAttribute is the shared runtime_location_id attribute
// of resources that exist once per runtime. Moving an object to another
// runtime means creating it there, so the attribute forces replacement.
func runtimeLocationResourceAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Optional:    true,
		Description: runtimeLocationDescription + " Changing it replaces the resource.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
		Validators: []validator.String{runtimeLocationValidator{}},
	}
}

// runtimeLocationDataSourceAttribute is the data source counterpart.
func runtimeLocationDataSourceAttribute() dsschema.StringAttribute {
	return dsschema.StringAttribute{
		Optional:    true,
		Description: runtimeLocationDescription,
		Validators:  []validator.String{runtimeLocationValidator{}},
	}
}

// locatedClient returns the client for the runtime named by loc: the client
// itself for the cloud runtime, or one routed through /location/<id>.
func locatedClient[T interface{ AtLocation(string) (T, error) }](c T, loc types.String, diags *diag.Diagnostics) (T, bool) {
	located, err := c.AtLocation(loc.ValueString())
	if err != nil {
		diags.AddAttributeError(path.Root("runtime_location_id"), "Invalid runtime location ID", err.Error())
		return located, false
	}
	return located, true
}

// importLocationPrefix marks an import ID that addresses an Edge Integration
// Cell: "location:<runtime location id>/<regular import ID>". An explicit
// prefix keeps the form unambiguous even for identifiers that may themselves
// contain "/", such as keystore aliases.
const importLocationPrefix = "location:"

// splitLocatedImportID splits an import ID into an optional runtime location
// ("" for the cloud runtime) and the regular import ID's segments. With parts
// == 1 the regular ID is returned whole, so it may contain "/".
func splitLocatedImportID(id string, parts int) (string, []string, error) {
	var loc string
	if strings.HasPrefix(id, importLocationPrefix) {
		prefixed, rest, found := strings.Cut(strings.TrimPrefix(id, importLocationPrefix), "/")
		if !found {
			return "", nil, fmt.Errorf("expected %q followed by \"/\" and the regular import ID, got %q", importLocationPrefix+"<runtime location id>", id)
		}
		if err := v2.ValidateRuntimeLocationID(prefixed); err != nil {
			return "", nil, err
		}
		loc, id = prefixed, rest
	}

	segments := []string{id}
	if parts > 1 {
		segments = strings.Split(id, "/")
		if len(segments) != parts {
			return "", nil, fmt.Errorf("expected %d segments separated by \"/\" (optionally preceded by %q), got %q", parts, importLocationPrefix+"<runtime location id>/", id)
		}
	}
	for _, s := range segments {
		if s == "" {
			return "", nil, fmt.Errorf("import ID %q contains an empty segment", id)
		}
	}
	return loc, segments, nil
}

func setImportedRuntimeLocation(ctx context.Context, loc string, set func(context.Context, path.Path, interface{}) diag.Diagnostics, diags *diag.Diagnostics) {
	if loc != "" {
		diags.Append(set(ctx, path.Root("runtime_location_id"), loc)...)
	}
}

type runtimeLocationValidator struct{}

func (runtimeLocationValidator) Description(context.Context) string {
	return "value must be a runtime location ID made of letters, digits, '.', '_' and '-'"
}

func (v runtimeLocationValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (runtimeLocationValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if err := v2.ValidateRuntimeLocationID(req.ConfigValue.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid runtime location ID", err.Error())
	}
}
