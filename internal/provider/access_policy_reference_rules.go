package provider

import (
	"context"
	"errors"
	"fmt"
	"regexp/syntax"
	"strings"
	"unicode"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// The values an access policy reference may be created with. Every value is
// one that SAP's Security Content API stores in the ArtifactReferences
// entity; none is derived from a UI label:
//
//   - SAP Help documents the UI choices (artifact types, the attributes Name
//     and ID, the operators Equals and Matches) and their restrictions:
//     Matches is not available for Integration Package, and message queues,
//     global variables and global data stores are matched only by name.
//   - SAP's audit log documentation shows the stored values Type
//     INTEGRATION_FLOW, ConditionAttribute Name and ConditionType exactString
//     and regularExpression; SAP's access policy CI/CD actions use the same.
//   - The tenant names the rest itself: a create with an unknown value
//     answers 400 with the list of allowed artifact types, "Only ID and Name
//     allowed" and "Only 'exactString' and 'regularExpression' are
//     permitted". A tenant test (2026-10-01) created every combination below
//     and read it back unchanged, and SAP rejected exactly the combinations
//     the rules below exclude, with the restrictions SAP Help states.
//
// SAP's list also names eight types that SAP Help does not offer for access
// policies (credentials, secure parameters, adapters, service interfaces,
// fault message types). The tenant accepts them with every attribute and
// operator, but only the tenant vouches for them, so they are unofficial:
// a plan that creates such a reference needs enable_unofficial.
const (
	referenceAttributeName = "Name"
	referenceAttributeID   = "ID"
	referenceOperatorExact = "exactString"
	referenceOperatorRegex = "regularExpression"
)

var (
	referenceAttributes = []string{referenceAttributeName, referenceAttributeID}
	referenceOperators  = []string{referenceOperatorExact, referenceOperatorRegex}
)

// referenceArtifactType is one artifact type an access policy reference can
// protect, with the attributes and operators SAP allows for it. Unofficial
// marks a type SAP Help does not document for access policies.
type referenceArtifactType struct {
	WireValue  string
	Attributes []string
	Operators  []string
	Unofficial bool
}

// referenceArtifactTypes is the compatibility matrix: the documented types,
// then the unofficial ones, each group sorted by WireValue.
var referenceArtifactTypes = []referenceArtifactType{
	{"API_ARTIFACT", referenceAttributes, referenceOperators, false},
	{"DATA_TYPE", referenceAttributes, referenceOperators, false},
	{"GLOBAL_DATA_STORE", []string{referenceAttributeName}, referenceOperators, false},
	{"GLOBAL_VARIABLE", []string{referenceAttributeName}, referenceOperators, false},
	{"INTEGRATION_FLOW", referenceAttributes, referenceOperators, false},
	{"INTEGRATION_PACKAGE", referenceAttributes, []string{referenceOperatorExact}, false},
	{"MESSAGE_MAPPING", referenceAttributes, referenceOperators, false},
	{"MESSAGE_QUEUE", []string{referenceAttributeName}, referenceOperators, false},
	{"MESSAGE_TYPE", referenceAttributes, referenceOperators, false},
	{"ODATA_SERVICE", referenceAttributes, referenceOperators, false},
	{"REST_API_PROVIDER", referenceAttributes, referenceOperators, false},
	{"SCRIPT_COLLECTION", referenceAttributes, referenceOperators, false},
	{"SOAP_API_PROVIDER", referenceAttributes, referenceOperators, false},
	{"VALUE_MAPPING", referenceAttributes, referenceOperators, false},

	// Accepted by the tenant, not documented for access policies.
	{"AUTH2_AUTHORIZATION_CODE", referenceAttributes, referenceOperators, true},
	{"AUTH2_SAML_BEARER_ASSERTION", referenceAttributes, referenceOperators, true},
	{"FAULT_MESSAGE_TYPE", referenceAttributes, referenceOperators, true},
	{"INTEGRATION_ADAPTER", referenceAttributes, referenceOperators, true},
	{"OAUTH2_CLIENT_CRED", referenceAttributes, referenceOperators, true},
	{"SECURE_PARAMETER", referenceAttributes, referenceOperators, true},
	{"SERVICE_INTERFACE", referenceAttributes, referenceOperators, true},
	{"USER_CREDENTIAL", referenceAttributes, referenceOperators, true},
}

func referenceArtifactTypeByWireValue(value string) (referenceArtifactType, bool) {
	for _, t := range referenceArtifactTypes {
		if t.WireValue == value {
			return t, true
		}
	}
	return referenceArtifactType{}, false
}

// referenceArtifactTypeWireValues returns the wire values of the documented
// or of the unofficial artifact types.
func referenceArtifactTypeWireValues(unofficial bool) []string {
	var values []string
	for _, t := range referenceArtifactTypes {
		if t.Unofficial == unofficial {
			values = append(values, t.WireValue)
		}
	}
	return values
}

// Suggestions for values that are not SAP wire values but whose meaning is
// clear: UI labels, spellings of earlier provider releases and common
// abbreviations. They only point to the right value in the diagnostic; the
// provider never converts a value.
var (
	referenceOperatorHints = map[string]string{
		"equals": referenceOperatorExact, "eq": referenceOperatorExact, "exact": referenceOperatorExact,
		"matches": referenceOperatorRegex, "match": referenceOperatorRegex, "regex": referenceOperatorRegex,
		"regexp": referenceOperatorRegex,
	}
	referenceArtifactTypeHints = map[string]string{
		"odataapi": "ODATA_SERVICE", "restapi": "REST_API_PROVIDER", "soapapi": "SOAP_API_PROVIDER",
		"jmsqueue": "MESSAGE_QUEUE",
	}
)

// hintKey reduces a value to lower-case letters and digits, so that
// "IntegrationPackage", "Integration Package" and "integration_package" all
// find INTEGRATION_PACKAGE.
func hintKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
}

// wireValueValidator accepts only the listed SAP wire values, compared
// case-sensitively, and names the intended value when the input is a known
// label or spelling of one.
type wireValueValidator struct {
	what       string // for example "artifact type"
	allowed    []string
	unofficial []string          // also accepted; the plan then needs enable_unofficial
	hints      map[string]string // hintKey(input) -> wire value
}

func (v wireValueValidator) Description(context.Context) string {
	return fmt.Sprintf("must be one of the %s values SAP's API stores: %s", v.what, strings.Join(v.all(), ", "))
}

func (v wireValueValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// all returns the documented and the unofficial values in a new slice.
func (v wireValueValidator) all() []string {
	return append(append([]string{}, v.allowed...), v.unofficial...)
}

func (v wireValueValidator) suggestion(value string) string {
	key := hintKey(value)
	if hint, ok := v.hints[key]; ok {
		return hint
	}
	for _, allowed := range v.all() {
		if hintKey(allowed) == key {
			return allowed
		}
	}
	return ""
}

func (v wireValueValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	value := req.ConfigValue.ValueString()
	if contains(v.allowed, value) || contains(v.unofficial, value) {
		return
	}
	detail := fmt.Sprintf("%q is not an access policy %s value SAP's API accepts. The provider expects SAP's "+
		"wire values, not UI labels, and does not convert them. ", value, v.what)
	if s := v.suggestion(value); s != "" {
		detail += fmt.Sprintf("Use %q. ", s)
	}
	detail += fmt.Sprintf("Supported values: %s.", strings.Join(v.allowed, ", "))
	if len(v.unofficial) > 0 {
		detail += fmt.Sprintf(" With enable_unofficial = true also: %s.", strings.Join(v.unofficial, ", "))
	}
	detail += "\n\nReferences that already exist in SAP with another value can still be read and imported, " +
		"but this provider version does not create them."
	resp.Diagnostics.AddAttributeError(req.Path, "Unsupported access policy "+v.what, detail)
}

var (
	referenceArtifactTypeValidator = wireValueValidator{what: "artifact type", allowed: referenceArtifactTypeWireValues(false),
		unofficial: referenceArtifactTypeWireValues(true), hints: referenceArtifactTypeHints}
	referenceAttributeValidator = wireValueValidator{what: "attribute", allowed: referenceAttributes}
	referenceOperatorValidator  = wireValueValidator{what: "operator", allowed: referenceOperators, hints: referenceOperatorHints}
)

// validateReferenceCombination checks attribute and operator against what
// SAP allows for the artifact type. Values the field validators reject, and
// unknown values, are skipped here.
func validateReferenceCombination(artifactType, attribute, operator string, report func(attr path.Path, summary, detail string)) {
	t, ok := referenceArtifactTypeByWireValue(artifactType)
	if !ok {
		return
	}
	if contains(referenceAttributes, attribute) && !contains(t.Attributes, attribute) {
		report(path.Root("attribute"), "Attribute not supported for this artifact type",
			fmt.Sprintf("SAP matches %s references only by %s; %q is not allowed for this type.",
				t.WireValue, strings.Join(t.Attributes, " or "), attribute))
	}
	if contains(referenceOperators, operator) && !contains(t.Operators, operator) {
		report(path.Root("operator"), "Operator not supported for this artifact type",
			fmt.Sprintf("SAP allows only %s for %s references; regular expressions (%s, Matches in the UI) are not "+
				"available for this type.", strings.Join(t.Operators, " or "), t.WireValue, referenceOperatorRegex))
	}
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// javaRegexError reports syntax errors that java.util.regex.Pattern rejects
// as well: unbalanced parentheses or brackets, a quantifier without anything
// to repeat (glob syntax such as "*_ORDERS"), a reversed character range and
// a trailing backslash. Go's regexp package is not Java's: it lacks
// lookaround, backreferences and possessive quantifiers, so any other parse
// error is not reported, and a pattern that passes here can still be
// rejected by SAP.
func javaRegexError(pattern string) error {
	_, err := syntax.Parse(pattern, syntax.Perl)
	var syntaxErr *syntax.Error
	if !errors.As(err, &syntaxErr) {
		return nil
	}
	switch syntaxErr.Code {
	case syntax.ErrMissingParen, syntax.ErrUnexpectedParen, syntax.ErrMissingBracket,
		syntax.ErrMissingRepeatArgument, syntax.ErrInvalidCharRange, syntax.ErrTrailingBackslash:
		return syntaxErr
	}
	return nil
}

// globStarSuggestion returns the pattern with ".*" in place of every "*"
// that directly follows a letter, digit, underscore or hyphen outside a
// character class, or "" when there is none. "IFL_CORE_*" is valid, but it
// matches IFL_CORE followed by underscores, not every name that starts with
// IFL_CORE_; the glob-style star is almost always a mistake there.
func globStarSuggestion(pattern string) string {
	var b strings.Builder
	found, inClass, escaped := false, false, false
	var prev rune // the previous literal character, 0 after an escape sequence
	for _, r := range pattern {
		literal := r
		switch {
		case escaped:
			escaped = false
			literal = 0
		case r == '\\':
			escaped = true
		case inClass:
			inClass = r != ']'
		case r == '[':
			inClass = true
		case r == '*' && (unicode.IsLetter(prev) || unicode.IsDigit(prev) || prev == '_' || prev == '-'):
			found = true
			b.WriteRune('.')
		}
		b.WriteRune(r)
		prev = literal
	}
	if !found {
		return ""
	}
	return b.String()
}
