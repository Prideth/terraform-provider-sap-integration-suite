package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const messageMappingDesigntimeArtifactsEntitySet = "MessageMappingDesigntimeArtifacts"

// MessageMapping is the wire representation of a
// MessageMappingDesigntimeArtifacts entity: a reusable, package-level
// message mapping artifact, not the inline/local message mapping step an
// integration flow can define directly inside its own content. See
// docs/resource-design.md for that distinction.
type MessageMapping struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	PackageID string `json:"PackageId"`
	Version   string `json:"Version,omitempty"`

	// Content is the base64-encoded message mapping project content: a ZIP
	// archive containing the mapping definition (a .mmap file) and any
	// dependent schema files (XSD, WSDL, EDMX, Swagger/OpenAPI JSON) it
	// references for its source/target message structures. This client
	// transports it opaquely; it is only populated on Create/Update
	// requests, since Read does not return it.
	Content string `json:"ArtifactContent,omitempty"`
}

// GetMessageMapping reads the active design-time version's metadata for the
// message mapping identified by mappingID within packageID.
func (c *Client) GetMessageMapping(ctx context.Context, packageID, mappingID string) (*MessageMapping, error) {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return nil, err
	}

	body, err := c.odata.Get(ctx, v2.BuildPath(messageMappingDesigntimeArtifactsEntitySet, key, ""))
	if err != nil {
		return nil, err
	}

	var mapping MessageMapping
	if err := v2.DecodeEntity(body, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// CreateMessageMapping creates a new message mapping design-time artifact
// from uploaded content. content must already be the raw (not yet
// base64-encoded) ZIP bytes. Unlike CreateValueMapping, no SAP-documented
// minimum-content precondition was found for message mapping, so none is
// enforced here.
func (c *Client) CreateMessageMapping(ctx context.Context, packageID, mappingID, name string, content []byte) (*MessageMapping, error) {
	payload, err := json.Marshal(MessageMapping{
		ID:        mappingID,
		Name:      name,
		PackageID: packageID,
		Content:   base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding message mapping: %w", err)
	}

	body, err := c.odata.Post(ctx, messageMappingDesigntimeArtifactsEntitySet, payload)
	if err != nil {
		return nil, err
	}

	var mapping MessageMapping
	if err := v2.DecodeEntity(body, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// UpdateMessageMapping uploads new content for an existing message mapping,
// creating a new design-time version under the same mapping ID via PUT
// against the keyed (Id, Version) entity.
//
// Unlike UpdateValueMapping (removed after this project found evidence
// against a working generic PUT for that specific entity set), PUT here
// rests on positive, entity-specific evidence: MessageMappingDesigntimeArtifacts
// shares IntegrationDesigntimeArtifacts' exact (Id, Version) key shape and
// confirmed version-creating PUT behavior; an independent third-party OData
// client built against this same API (github.com/lemaiwo/ci-mcp-server)
// explicitly enables generic update for this entity set, the same as it
// does for IntegrationDesigntimeArtifacts and ScriptCollectionDesigntimeArtifacts
// and unlike ValueMappingDesigntimeArtifacts, where it explicitly disables
// it; and no SAP Knowledge Base Article or other evidence of a documented
// PUT problem for this entity set was found. See docs/sap-api-references.md
// for the full reasoning, including why MessageMappingDesigntimeArtifactSaveAsVersion
// existing alongside PUT here is not evidence against PUT.
func (c *Client) UpdateMessageMapping(ctx context.Context, mappingID, name string, content []byte) (*MessageMapping, error) {
	return updateDesigntimeArtifact(ctx, c, messageMappingDesigntimeArtifactsEntitySet, mappingID, name, content,
		func() (*MessageMapping, error) { return c.GetMessageMapping(ctx, "", mappingID) })
}

// DeleteMessageMapping deletes the message mapping design-time artifact
// identified by mappingID, addressed via the "active" version key, the same
// way GetMessageMapping and UpdateMessageMapping address it. Whether this
// removes only the active/current design-time version or every version of
// the artifact has not been confirmed against a primary source (the same
// open question already flagged for DeleteValueMapping). This provider
// does not create more than one version of a message mapping concurrently,
// so the distinction does not change this resource's behavior today, but a
// value mapping deleted through Terraform could in principle leave older,
// non-active versions behind on SAP's side.
func (c *Client) DeleteMessageMapping(ctx context.Context, mappingID string) error {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, v2.BuildPath(messageMappingDesigntimeArtifactsEntitySet, key, ""))
}

// DeployMessageMapping triggers deployment of the given design-time version
// of a message mapping. It returns immediately once SAP has accepted the
// deployment request; callers must poll GetRuntimeArtifact for completion,
// since deployment is asynchronous.
//
// The action name is singular ("...Artifact", not "...Artifacts"), matching
// DeployIntegrationDesigntimeArtifact and DeployValueMappingDesigntimeArtifact
// — confirmed via SAP's own documentation (read through the SAP-docs GitHub
// mirror) rather than copied blindly from either sibling action.
func (c *Client) DeployMessageMapping(ctx context.Context, mappingID, version string) error {
	path := fmt.Sprintf("DeployMessageMappingDesigntimeArtifact?Id='%s'&Version='%s'",
		v2.EscapeLiteral(mappingID), v2.EscapeLiteral(version))
	_, err := c.odata.Post(ctx, path, nil)
	return err
}
