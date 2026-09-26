package provider

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// isNotFound reports whether err represents an HTTP 404 from a SAP API
// call, the shared condition every resource's Read/Delete checks to decide
// whether to remove a resource from state (Read) or treat a delete against
// an already-gone object as a success (Delete) rather than an error.
func isNotFound(err error) bool {
	var apiErr *apierror.Error
	return errors.As(err, &apiErr) && apiErr.IsNotFound()
}

// requireHTTPClient adds a clear, actionable diagnostic and reports failure
// when data has no SAP HTTP client configured. Every SAP-backed resource
// and data source calls this from its own Configure method: the provider's
// own Configure never fails just because SAP connectivity is missing (see
// provider.go), since the feature catalog data sources need none, so each
// SAP-backed type is responsible for surfacing this itself, in place of
// silently building a client that can never authenticate. noun should be
// "resource" or "data source", matching what the diagnostic is attached to.
func requireHTTPClient(data *Data, noun string, diags *diag.Diagnostics) bool {
	if data.HTTPClient != nil {
		return true
	}
	diags.AddError(
		"SAP Integration Suite API configuration is required for this "+noun+".",
		"Configure provider.host and OAuth credentials or the corresponding SAP_INTEGRATION_SUITE_* environment variables.",
	)
	return false
}

// requireAPIManagementClassicHTTPClient is the Classic API Management
// (API Portal) counterpart of requireHTTPClient: every Classic API
// Management resource and data source calls this from its own Configure
// method, since provider.api_management is optional and the provider's own
// Configure never fails just because it is absent.
func requireAPIManagementClassicHTTPClient(data *Data, noun string, diags *diag.Diagnostics) bool {
	if data.APIManagementClassicHTTPClient != nil {
		return true
	}
	diags.AddError(
		"Classic API Management configuration is required for this "+noun+".",
		"Configure provider.api_management (host, token_url, client_id, client_secret) or the "+
			"corresponding SAP_INTEGRATION_SUITE_API_MANAGEMENT_* environment variables.",
	)
	return false
}

// requireAPICompositionHTTPClient is the API Composition counterpart of
// requireHTTPClient. provider.api_composition is optional, so the check
// happens when a resource or data source that needs it is configured.
func requireAPICompositionHTTPClient(data *Data, noun string, diags *diag.Diagnostics) bool {
	if data.APICompositionHTTPClient != nil {
		return true
	}
	diags.AddError(
		"API Composition configuration is required for this "+noun+".",
		"Configure provider.api_composition (host, token_url, client_id, client_secret) or the "+
			"corresponding SAP_INTEGRATION_SUITE_API_COMPOSITION_* environment variables.",
	)
	return false
}

// diagnosticDetail renders an error for a Terraform diagnostic detail
// string. For a SAP API error it surfaces the status code, SAP error code,
// and message so the user sees what SAP actually reported rather than a bare
// Go error string; for any other error it falls back to err.Error().
//
// Credential material never reaches this path: the HTTP client never logs
// or returns Authorization headers, tokens, or client secrets as part of an
// error.
func diagnosticDetail(err error) string {
	var apiErr *apierror.Error
	if errors.As(err, &apiErr) {
		detail := apiErr.Error()
		for _, d := range apiErr.Details {
			detail += "\n  - " + d.Code + ": " + d.Message
		}
		return detail
	}

	return err.Error()
}

// pathRootID returns path.Root("id"), used by ImportStatePassthroughID
// implementations for resources whose Terraform ID is a single SAP-assigned
// or user-assigned key.
func pathRootID() path.Path {
	return path.Root("id")
}

// pathRoot is a short alias for path.Root, used when setting individual
// attributes during ImportState for resources with a composite ID.
func pathRoot(name string) path.Path {
	return path.Root(name)
}

// splitCompositeID splits a "<a>/<b>" import ID into its two parts.
func splitCompositeID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected an import ID in the form \"<package_id>/<flow_id>\", got %q", id)
	}
	return parts[0], parts[1], nil
}

func errFileTooLarge(path string, size, max int64) error {
	return fmt.Errorf("file %q is %d bytes, which exceeds the %d byte limit", path, size, max)
}

func errHashMismatch(expected, actual string) error {
	return fmt.Errorf("content_hash %q does not match the actual file content hash %q", expected, actual)
}

// stringOrNull converts an empty string (SAP's way of saying "no value") to
// a null Terraform string, so optional attributes do not show perpetual
// drift against an unset configuration value.
func stringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// int64OrNull converts a zero int (SAP's Go zero value for "no value was
// returned/decoded", never a real key size or similar) to a null Terraform
// number, the numeric counterpart to stringOrNull.
func int64OrNull(n int) types.Int64 {
	if n == 0 {
		return types.Int64Null()
	}
	return types.Int64Value(int64(n))
}

// alignUploadBundleID gives the copy of an integration flow or message
// mapping that is uploaded the artifact's ID as Bundle-SymbolicName, and
// warns when that changed anything. SAP writes the ID into the MANIFEST on
// create and rejects a later content update whose Bundle-SymbolicName
// differs (tenant test, September 2026), so a ZIP exported under another ID
// could be created but never updated. The local file is not changed.
func alignUploadBundleID(content []byte, id, file string, diags *diag.Diagnostics) []byte {
	aligned, previous, err := cloudintegration.AlignBundleID(content, id)
	if err != nil {
		diags.AddWarning("Could not align the bundle ID of "+file,
			fmt.Sprintf("The content is uploaded unchanged. SAP rejects content updates whose Bundle-SymbolicName differs from %q. Details: %v", id, err))
		return content
	}
	if previous != id {
		diags.AddWarning("Bundle-SymbolicName adjusted for upload",
			fmt.Sprintf("%s has Bundle-SymbolicName %q in META-INF/MANIFEST.MF; the provider uploads it as %q, because SAP "+
				"rejects content updates whose bundle ID differs from the artifact ID. Your file is not changed. "+
				"Set Bundle-SymbolicName to %q in the file to remove this warning.", file, previous, id, id))
	}
	return aligned
}
