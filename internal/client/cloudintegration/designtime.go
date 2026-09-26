package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
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

// designtimeUpdateRequest is the body of a content update (PUT on
// <Artifact>(Id,Version='active')). It carries only the name and the content:
// a tenant answered a message mapping update that also sent empty Id and
// PackageId with 500 "Update of PackageId and Id are not allowed", while the
// same update with just these two fields answered 200 (September 2026).
type designtimeUpdateRequest struct {
	Name    string `json:"Name"`
	Content string `json:"ArtifactContent"`
}

// updateDesigntimeArtifact uploads new content for an existing artifact.
// The tenant answered these updates with 200 and an empty body, so without
// a body the active version is read back.
func updateDesigntimeArtifact[T any](ctx context.Context, c *Client, entitySet, id, name string, content []byte, readActive func() (*T, error)) (*T, error) {
	payload, err := json.Marshal(designtimeUpdateRequest{Name: name, Content: base64.StdEncoding.EncodeToString(content)})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding %s update: %w", entitySet, err)
	}
	key, err := designtimeArtifactKey(id, activeVersion)
	if err != nil {
		return nil, err
	}
	body, err := c.odata.Put(ctx, v2.BuildPath(entitySet, key, ""), payload)
	if err != nil {
		return nil, err
	}
	if v2.EmptyBody(body) {
		return readActive()
	}
	var updated T
	if err := v2.DecodeEntity(body, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}
