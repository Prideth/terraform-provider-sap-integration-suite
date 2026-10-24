package cloudintegration

import (
	"context"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// Copies of SAP's standard packages. SAP's Integration Content API
// specification documents POST CopyIntegrationPackage?Id='<id>', which
// copies a package from the Discover catalog into Design, with the optional
// ImportMode (CREATE_COPY, OVERWRITE, OVERWRITE_MERGE) and Suffix. A tenant
// probe (2026-10-24, PENDING #77) showed:
//
//   - A copy answers 201 with the package: SAP's ID, name and version,
//     Vendor SAP and Mode READ_ONLY. Its metadata cannot be changed (PUT 403
//     "Configure-Only"); its flows can be configured and deployed.
//   - Copying again without ImportMode answers 409, an unknown ID 404.
//   - OVERWRITE_MERGE replaces the content with the catalog's and keeps the
//     externalized parameters set on the copy; OVERWRITE resets them.
//   - CREATE_COPY needs a Suffix (409 without one) and appends ".<suffix>"
//     to the package ID, the name and the IDs of the flows.

// CopyImportMode is the ImportMode of CopyIntegrationPackage.
type CopyImportMode string

// The import modes SAP's specification lists. An empty mode copies a package
// Design does not have yet.
const (
	CopyAsNew          CopyImportMode = ""
	CopyCreateCopy     CopyImportMode = "CREATE_COPY"
	CopyOverwrite      CopyImportMode = "OVERWRITE"
	CopyOverwriteMerge CopyImportMode = "OVERWRITE_MERGE"
)

const copyIntegrationPackageFunction = "CopyIntegrationPackage"

// CopyPackage copies the package sourceID from the Discover catalog into
// Design and returns the package SAP created or overwrote. suffix is sent
// only with CREATE_COPY.
func (c *Client) CopyPackage(ctx context.Context, sourceID string, mode CopyImportMode, suffix string) (*Package, error) {
	path := copyIntegrationPackageFunction + "?Id=" + quotedParam(sourceID)
	if mode != CopyAsNew {
		path += "&ImportMode=" + quotedParam(string(mode))
	}
	if mode == CopyCreateCopy {
		path += "&Suffix=" + quotedParam(suffix)
	}
	body, err := c.odata.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	var pkg Package
	if err := v2.DecodeEntity(body, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

// CopiedPackageID is the ID SAP gives a copy: the source ID, and with a
// suffix (CREATE_COPY) the source ID followed by "." and the suffix.
func CopiedPackageID(sourceID, suffix string) string {
	if suffix == "" {
		return sourceID
	}
	return sourceID + "." + suffix
}

// ListPackageIntegrationFlows lists the integration flows of a package.
func (c *Client) ListPackageIntegrationFlows(ctx context.Context, packageID string) ([]IntegrationFlow, error) {
	return v2.GetAllPages[IntegrationFlow](ctx, c.odata,
		v2.BuildPath(integrationPackagesEntitySet, v2.KeyPredicate(packageID), "")+"/"+integrationDesigntimeArtifactsEntitySet)
}
