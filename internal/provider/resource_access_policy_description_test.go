package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// SAP keeps only the first 200 characters of an access policy description.
// Earlier releases accepted longer values: the POST succeeded, SAP truncated
// the description, and the read-back failed the apply with "Provider produced
// inconsistent result after apply" after the policy already existed. These
// tests pin the replacement behavior: a description SAP cannot keep is
// rejected while Terraform validates the configuration, before any request.

// validateAccessPolicyConfig runs the configuration through the provider's
// ValidateResourceConfig RPC, the call Terraform makes during validate and
// plan. description nil means the attribute is omitted.
func validateAccessPolicyConfig(t *testing.T, description *string) []*tfprotov6.Diagnostic {
	t.Helper()
	server, err := providerserver.NewProtocol6WithError(New("test")())()
	if err != nil {
		t.Fatal(err)
	}

	objectType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":          tftypes.String,
		"role_name":   tftypes.String,
		"description": tftypes.String,
	}}
	descriptionValue := tftypes.NewValue(tftypes.String, nil)
	if description != nil {
		descriptionValue = tftypes.NewValue(tftypes.String, *description)
	}
	config, err := tfprotov6.NewDynamicValue(objectType, tftypes.NewValue(objectType, map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, nil),
		"role_name":   tftypes.NewValue(tftypes.String, "TFACC_DESCRIPTION_LIMIT"),
		"description": descriptionValue,
	}))
	if err != nil {
		t.Fatal(err)
	}

	resp, err := server.ValidateResourceConfig(context.Background(), &tfprotov6.ValidateResourceConfigRequest{
		TypeName: "sapintegrationsuite_access_policy",
		Config:   &config,
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Diagnostics
}

func TestAccessPolicyResource_DescriptionLength(t *testing.T) {
	ptr := func(s string) *string { return &s }

	cases := []struct {
		name        string
		description *string
		wantSummary string // empty: the configuration is valid
	}{
		{"omitted", nil, ""},
		{"1 character", ptr("x"), ""},
		{"199 characters", ptr(strings.Repeat("a", 199)), ""},
		{"exactly 200 characters", ptr(strings.Repeat("a", 200)), ""},
		{"200 non-ASCII characters", ptr(strings.Repeat("ä", 200)), ""},
		{"empty string", ptr(""), "Access policy description is empty"},
		{"201 characters", ptr(strings.Repeat("a", 201)), "Access policy description is too long"},
		{"242 characters from the tenant report", ptr(strings.Repeat("b", 242)), "Access policy description is too long"},
		{"1200 characters", ptr(strings.Repeat("c", 1200)), "Access policy description is too long"},
		{"emoji count as two UTF-16 units", ptr(strings.Repeat("a", 199) + "😀"), "Access policy description is too long"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			diags := validateAccessPolicyConfig(t, c.description)
			if c.wantSummary == "" {
				for _, d := range diags {
					if d.Severity == tfprotov6.DiagnosticSeverityError {
						t.Fatalf("unexpected error diagnostic: %s: %s", d.Summary, d.Detail)
					}
				}
				return
			}
			if len(diags) != 1 || diags[0].Severity != tfprotov6.DiagnosticSeverityError {
				t.Fatalf("diagnostics = %v, want exactly one error", diags)
			}
			if diags[0].Summary != c.wantSummary {
				t.Errorf("summary = %q, want %q", diags[0].Summary, c.wantSummary)
			}
			if diags[0].Attribute == nil || !diags[0].Attribute.Equal(tftypes.NewAttributePath().WithAttributeName("description")) {
				t.Errorf("diagnostic attribute = %v, want description", diags[0].Attribute)
			}
		})
	}
}

func TestAccessPolicyResource_DescriptionTooLongDiagnosticIsActionable(t *testing.T) {
	var resp validator.StringResponse
	accessPolicyDescriptionValidator{}.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("description"),
		ConfigValue: types.StringValue(strings.Repeat("a", 264)),
	}, &resp)

	if resp.Diagnostics.ErrorsCount() != 1 {
		t.Fatalf("diagnostics = %v, want one error", resp.Diagnostics)
	}
	detail := resp.Diagnostics.Errors()[0].Detail()
	for _, want := range []string{"at most 200 characters", "this one has 264", "sapintegrationsuite_access_policy_reference"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail %q does not mention %q", detail, want)
		}
	}
}

// An unknown description (interpolated from another resource) is checked
// once its value is known; it must not fail validation on its own.
func TestAccessPolicyResource_DescriptionUnknownIsDeferred(t *testing.T) {
	var resp validator.StringResponse
	accessPolicyDescriptionValidator{}.ValidateString(context.Background(), validator.StringRequest{
		Path:        path.Root("description"),
		ConfigValue: types.StringUnknown(),
	}, &resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unknown description produced diagnostics: %v", resp.Diagnostics)
	}
}
