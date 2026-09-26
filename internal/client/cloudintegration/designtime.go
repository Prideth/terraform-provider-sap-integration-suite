package cloudintegration

import (
	"context"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// activeVersion is the OData V2 key literal SAP's Integration Content API
// accepts to mean "the current active design-time version" without the
// caller needing to track version numbers itself. It is shared by every
// versioned design-time artifact type in this API family (integration
// flows, value mappings, ...).
const activeVersion = "active"

// designtimeArtifactKey builds the composite (Id, Version) key predicate
// shared by every versioned design-time artifact entity set in this API.
func designtimeArtifactKey(id, version string) (string, error) {
	return v2.CompositeKeyPredicate("Id", id, "Version", version)
}

// saveAsVersion calls one of the <Artifact>SaveAsVersion function imports
// (parameters Id and SaveAsVersion, POST), as SAP documents for
// IntegrationDesigntimeArtifactSaveAsVersion: upload the content with PUT
// first, then save it under an explicit version. The $metadata declares the
// saved artifact as the return type, but a tenant answered
// IntegrationDesigntimeArtifactSaveAsVersion with 200 and an empty body
// (September 2026), so without a body the active version is read back.
func saveAsVersion[T any](ctx context.Context, c *Client, functionImport, id, version string, readActive func() (*T, error)) (*T, error) {
	path := fmt.Sprintf("%s?Id='%s'&SaveAsVersion='%s'", functionImport, v2.EscapeLiteral(id), v2.EscapeLiteral(version))
	body, err := c.odata.Post(ctx, path, nil)
	if err != nil {
		return nil, err
	}
	if v2.EmptyBody(body) {
		return readActive()
	}
	var saved T
	if err := v2.DecodeEntity(body, &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

// SaveIntegrationFlowAsVersion saves the current content of an integration
// flow under an explicit version such as "1.0.3".
func (c *Client) SaveIntegrationFlowAsVersion(ctx context.Context, flowID, version string) (*IntegrationFlow, error) {
	return saveAsVersion(ctx, c, "IntegrationDesigntimeArtifactSaveAsVersion", flowID, version,
		func() (*IntegrationFlow, error) { return c.GetIntegrationFlow(ctx, "", flowID) })
}

// SaveMessageMappingAsVersion saves the current content of a message mapping
// under an explicit version.
func (c *Client) SaveMessageMappingAsVersion(ctx context.Context, mappingID, version string) (*MessageMapping, error) {
	return saveAsVersion(ctx, c, "MessageMappingDesigntimeArtifactSaveAsVersion", mappingID, version,
		func() (*MessageMapping, error) { return c.GetMessageMapping(ctx, "", mappingID) })
}

// SaveScriptCollectionAsVersion saves the current content of a script
// collection under an explicit version.
func (c *Client) SaveScriptCollectionAsVersion(ctx context.Context, scriptCollectionID, version string) (*ScriptCollection, error) {
	return saveAsVersion(ctx, c, "ScriptCollectionDesigntimeArtifactSaveAsVersion", scriptCollectionID, version,
		func() (*ScriptCollection, error) { return c.GetScriptCollection(ctx, "", scriptCollectionID) })
}
