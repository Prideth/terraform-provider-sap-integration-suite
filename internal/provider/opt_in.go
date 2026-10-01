package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/features"
)

// catalogStatus returns the support status the feature catalog records for a
// resource or data source type, or "" when no feature lists it.
func catalogStatus(typeName string) features.SupportStatus {
	for _, f := range features.Catalog {
		for _, t := range f.ResourceTypes {
			if t == typeName {
				return f.SupportStatus
			}
		}
		for _, t := range f.DataSourceTypes {
			if t == typeName {
				return f.SupportStatus
			}
		}
	}
	return ""
}

// requireOptIn refuses a resource or data source whose catalog status is
// experimental or unofficial unless the provider enables that status. It is
// called from Configure, so a configuration fails only when it actually uses
// such a type. It reports whether the type may be used.
func requireOptIn(data *Data, typeName string, diags *diag.Diagnostics) bool {
	switch catalogStatus(typeName) {
	case features.StatusExperimental:
		if data.EnableExperimental {
			return true
		}
		diags.AddError(typeName+" is experimental",
			fmt.Sprintf("%s is implemented on a documented SAP API, but its lifecycle has not yet passed an "+
				"acceptance test on a tenant, so its behavior or schema may still change. To use it anyway, "+
				"set enable_experimental = true in the provider block (or "+
				"SAP_INTEGRATION_SUITE_ENABLE_EXPERIMENTAL=true). See docs/feature-support.md.", typeName))
		return false
	case features.StatusUnofficial:
		if data.EnableUnofficial {
			return true
		}
		diags.AddError(typeName+" is unofficial",
			fmt.Sprintf("%s works and was verified on a tenant, but SAP does not document the API behind it; "+
				"it is known only from the service's $metadata, so SAP may change it without notice. To use it "+
				"anyway, set enable_unofficial = true in the provider block (or "+
				"SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL=true). See docs/feature-support.md.", typeName))
		return false
	}
	return true
}

// requireUnofficialOperation refuses a single undocumented operation of an
// otherwise documented resource unless the provider enables unofficial
// features. operation names it the way the feature catalog's
// UndocumentedOperations does. It reports whether the operation may run.
func requireUnofficialOperation(allowed bool, typeName, operation string, diags *diag.Diagnostics) bool {
	if allowed {
		return true
	}
	diags.AddError(fmt.Sprintf("%s: %s is unofficial", typeName, operation),
		fmt.Sprintf("The API behind %s is official, but this operation is not: SAP neither documents it nor "+
			"uses it in its own tooling. It is known only from the service's $metadata or the tenant's own "+
			"answers and was verified on a tenant, so SAP may change it without notice. To allow it, set "+
			"enable_unofficial = true in the provider block (or SAP_INTEGRATION_SUITE_ENABLE_UNOFFICIAL=true). "+
			"The documented operations of %s work without it. See docs/feature-support.md, \"Contract sources\".",
			typeName, typeName))
	return false
}

// isInPlaceUpdate reports whether a plan updates an existing resource in
// place: prior state and plan both exist, they differ, and no attribute
// forces a replacement. Resource-level ModifyPlan runs after the attribute
// plan modifiers, so resp.RequiresReplace is already filled.
func isInPlaceUpdate(req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) bool {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return false
	}
	return !req.Plan.Raw.Equal(req.State.Raw) && len(resp.RequiresReplace) == 0
}

// isPlannedDelete reports whether a plan deletes an existing object: a
// destroy, or a replacement, which deletes before or after it creates.
func isPlannedDelete(req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) bool {
	if req.State.Raw.IsNull() {
		return false
	}
	return req.Plan.Raw.IsNull() || len(resp.RequiresReplace) > 0
}
