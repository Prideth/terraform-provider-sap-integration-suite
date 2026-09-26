package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const scriptCollectionDesigntimeArtifactsEntitySet = "ScriptCollectionDesigntimeArtifacts"

// ScriptCollection is the wire representation of a
// ScriptCollectionDesigntimeArtifacts entity: a reusable bundle of
// Groovy/JavaScript scripts, created within a package so it can be shared
// across any number of integration flows.
type ScriptCollection struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	PackageID string `json:"PackageId"`
	Version   string `json:"Version,omitempty"`

	// Content is the base64-encoded script collection project content: a
	// ZIP archive of Groovy/JavaScript script files. This client
	// transports it opaquely; it is only populated on Create/Update
	// requests, since Read does not return it.
	Content string `json:"ArtifactContent,omitempty"`
}

// GetScriptCollection reads the active design-time version's metadata for
// the script collection identified by scriptCollectionID within packageID.
func (c *Client) GetScriptCollection(ctx context.Context, packageID, scriptCollectionID string) (*ScriptCollection, error) {
	key, err := designtimeArtifactKey(scriptCollectionID, activeVersion)
	if err != nil {
		return nil, err
	}

	body, err := c.odata.Get(ctx, v2.BuildPath(scriptCollectionDesigntimeArtifactsEntitySet, key, ""))
	if err != nil {
		return nil, err
	}

	var sc ScriptCollection
	if err := v2.DecodeEntity(body, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

// CreateScriptCollection creates a new script collection design-time
// artifact from uploaded content. content must already be the raw (not
// yet base64-encoded) ZIP bytes. No SAP-documented minimum-content
// precondition was found for script collections, so none is enforced
// here, the same as message mapping.
func (c *Client) CreateScriptCollection(ctx context.Context, packageID, scriptCollectionID, name string, content []byte) (*ScriptCollection, error) {
	payload, err := json.Marshal(ScriptCollection{
		ID:        scriptCollectionID,
		Name:      name,
		PackageID: packageID,
		Content:   base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding script collection: %w", err)
	}

	body, err := c.odata.Post(ctx, scriptCollectionDesigntimeArtifactsEntitySet, payload)
	if err != nil {
		return nil, err
	}

	var sc ScriptCollection
	if err := v2.DecodeEntity(body, &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

// UpdateScriptCollection uploads new content for an existing script
// collection, creating a new design-time version under the same ID via
// PUT against the keyed (Id, Version) entity.
//
// PUT here rests on the same entity-specific evidence that decided
// UpdateMessageMapping: ScriptCollectionDesigntimeArtifacts shares
// IntegrationDesigntimeArtifacts' (Id, Version) key shape and confirmed
// version-creating PUT behavior, and the same independent third-party
// OData client that explicitly disables generic update for
// ValueMappingDesigntimeArtifacts explicitly enables it here, matching
// IntegrationDesigntimeArtifacts and MessageMappingDesigntimeArtifacts. No
// SAP Knowledge Base Article or other evidence of a documented PUT problem
// for this entity set was found. See docs/sap-api-references.md.
func (c *Client) UpdateScriptCollection(ctx context.Context, scriptCollectionID, name string, content []byte) (*ScriptCollection, error) {
	return updateDesigntimeArtifact(ctx, c, scriptCollectionDesigntimeArtifactsEntitySet, scriptCollectionID, name, content,
		func() (*ScriptCollection, error) { return c.GetScriptCollection(ctx, "", scriptCollectionID) })
}

// DeleteScriptCollection deletes the script collection design-time
// artifact identified by scriptCollectionID, addressed via the "active"
// version key. Whether this removes only the active/current design-time
// version or every version of the artifact has not been confirmed against
// a primary source, the same open question already flagged for every
// other design-time artifact type in this provider.
func (c *Client) DeleteScriptCollection(ctx context.Context, scriptCollectionID string) error {
	key, err := designtimeArtifactKey(scriptCollectionID, activeVersion)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, v2.BuildPath(scriptCollectionDesigntimeArtifactsEntitySet, key, ""))
}

// DeployScriptCollection triggers deployment of the given design-time
// version of a script collection. It returns immediately once SAP has
// accepted the deployment request; callers must poll GetRuntimeArtifact
// for completion, since deployment is asynchronous.
//
// The action name is singular ("...Artifact", not "...Artifacts"),
// confirmed via SAP's own documentation, matching every other design-time
// artifact type's deploy action in this API family.
func (c *Client) DeployScriptCollection(ctx context.Context, scriptCollectionID, version string) error {
	path := fmt.Sprintf("DeployScriptCollectionDesigntimeArtifact?Id='%s'&Version='%s'",
		v2.EscapeLiteral(scriptCollectionID), v2.EscapeLiteral(version))
	_, err := c.odata.Post(ctx, path, nil)
	return err
}
