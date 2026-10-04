package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
)

// UI labels of access policy references. SAP's UI shows labels ("Matches",
// "Integration Flow") where the API stores constants ("regularExpression",
// "INTEGRATION_FLOW"). With convert_ui_labels and enable_experimental, the
// provider accepts the labels whose constant is proven and sends the
// constant; without them it rejects every label and names the constant.
//
// Proven are only:
//   - the operators Equals and Matches, which SAP Help and SAP's audit log
//     documentation pair with exactString and regularExpression;
//   - the attributes Name and ID in any letter case;
//   - artifact type labels that spell their constant exactly once case,
//     spaces and underscores are ignored ("Integration Flow" and
//     INTEGRATION_FLOW).
//
// Labels whose constant has another name (the UI's "API", "OData API",
// "REST API" and "SOAP API") are not converted until a tenant shows which
// constant the UI stores for them.

type uiLabelKind int

const (
	uiLabelArtifactType uiLabelKind = iota
	uiLabelAttribute
	uiLabelOperator
)

func (k uiLabelKind) attribute() string {
	switch k {
	case uiLabelArtifactType:
		return "artifact_type"
	case uiLabelAttribute:
		return "attribute"
	default:
		return "operator"
	}
}

// uiLabelOperators are the operator labels, by hintKey.
var uiLabelOperators = map[string]string{"equals": referenceOperatorExact, "matches": referenceOperatorRegex}

// convertUILabel returns the constant a proven UI label stands for. It
// returns false for a value that already is a constant and for anything
// that is not a proven label.
func convertUILabel(kind uiLabelKind, value string) (string, bool) {
	var constants []string
	switch kind {
	case uiLabelOperator:
		if contains(referenceOperators, value) {
			return "", false
		}
		c, ok := uiLabelOperators[hintKey(value)]
		return c, ok
	case uiLabelAttribute:
		constants = referenceAttributes
	default:
		for _, t := range referenceArtifactTypes {
			constants = append(constants, t.WireValue)
		}
	}
	if contains(constants, value) {
		return "", false
	}
	for _, c := range constants {
		if hintKey(c) == hintKey(value) {
			return c, true
		}
	}
	return "", false
}

// canonicalUILabel is the constant a value stands for: the value itself, or
// the constant of a proven label.
func canonicalUILabel(kind uiLabelKind, value string) string {
	if c, ok := convertUILabel(kind, value); ok {
		return c
	}
	return value
}

// uiLabelEquivalent keeps the state's value when the configuration says the
// same thing in another spelling, for example "Matches" for a stored
// "regularExpression" after an import. It must run before RequiresReplace.
type uiLabelEquivalent struct{ kind uiLabelKind }

func (m uiLabelEquivalent) Description(context.Context) string {
	return "A UI label and the constant it stands for are the same value."
}

func (m uiLabelEquivalent) MarkdownDescription(ctx context.Context) string { return m.Description(ctx) }

func (m uiLabelEquivalent) PlanModifyString(_ context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() || req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}
	if req.PlanValue.ValueString() != req.StateValue.ValueString() &&
		canonicalUILabel(m.kind, req.PlanValue.ValueString()) == canonicalUILabel(m.kind, req.StateValue.ValueString()) {
		resp.PlanValue = req.StateValue
	}
}

// uiLabelConversion decides what happens to the UI labels of a reference
// that is about to be created: with the switches on, each conversion is a
// warning; without them, each label is an error that names the constant.
// It returns false when the plan must stop.
func uiLabelConversion(enabled, experimental bool, values map[uiLabelKind]string, diags *diag.Diagnostics) bool {
	ok := true
	for _, kind := range []uiLabelKind{uiLabelArtifactType, uiLabelAttribute, uiLabelOperator} {
		value, set := values[kind]
		if !set {
			continue
		}
		constant, isLabel := convertUILabel(kind, value)
		if !isLabel {
			continue
		}
		attr := path.Root(kind.attribute())
		if enabled && experimental {
			diags.AddAttributeWarning(attr, "UI label converted",
				fmt.Sprintf("%q is the label SAP's UI shows; the provider sends SAP's constant %q "+
					"(convert_ui_labels). The state keeps %q.", value, constant, value))
			continue
		}
		var missing []string
		if !enabled {
			missing = append(missing, "convert_ui_labels = true")
		}
		if !experimental {
			missing = append(missing, "enable_experimental = true")
		}
		diags.AddAttributeError(attr, "UI label instead of SAP's constant",
			fmt.Sprintf("%q is the label SAP's UI shows; SAP's API stores %q. Use %q, or set %s in the "+
				"provider block to let the provider convert the label.", value, constant, constant,
				strings.Join(missing, " and ")))
		ok = false
	}
	return ok
}
