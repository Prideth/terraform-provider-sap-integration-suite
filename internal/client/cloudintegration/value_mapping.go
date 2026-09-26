package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const valueMappingDesigntimeArtifactsEntitySet = "ValueMappingDesigntimeArtifacts"

// ValueMapping is the wire representation of a ValueMappingDesigntimeArtifacts
// entity.
type ValueMapping struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	PackageID string `json:"PackageId"`
	Version   string `json:"Version,omitempty"`

	// Content is the base64-encoded value mapping project content (SAP's
	// own design-time export/import format for this artifact type). It is
	// only populated on Create/Update requests; Read does not return it.
	Content string `json:"ArtifactContent,omitempty"`
}

// GetValueMapping reads the active design-time version's metadata for the
// value mapping identified by mappingID within packageID.
func (c *Client) GetValueMapping(ctx context.Context, packageID, mappingID string) (*ValueMapping, error) {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return nil, err
	}

	body, err := c.odata.Get(ctx, v2.BuildPath(valueMappingDesigntimeArtifactsEntitySet, key, ""))
	if err != nil {
		return nil, err
	}

	var mapping ValueMapping
	if err := v2.DecodeEntity(body, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// CreateValueMapping creates a new value mapping design-time artifact from
// uploaded content. content must already be the raw (not yet
// base64-encoded) bytes, and per SAP's own documented constraint must
// contain at least one mapping entry — a value mapping cannot be saved
// empty.
func (c *Client) CreateValueMapping(ctx context.Context, packageID, mappingID, name string, content []byte) (*ValueMapping, error) {
	payload, err := json.Marshal(ValueMapping{
		ID:        mappingID,
		Name:      name,
		PackageID: packageID,
		Content:   base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding value mapping: %w", err)
	}

	body, err := c.odata.Post(ctx, valueMappingDesigntimeArtifactsEntitySet, payload)
	if err != nil {
		return nil, err
	}

	var mapping ValueMapping
	if err := v2.DecodeEntity(body, &mapping); err != nil {
		return nil, err
	}
	return &mapping, nil
}

// There is deliberately no UpdateValueMapping. A prior version of this
// client called PUT against the keyed (Id, Version) entity, by analogy with
// IntegrationDesigntimeArtifacts' confirmed update behavior. That analogy
// does not hold up under closer scrutiny and was removed rather than kept
// as a shipped guess:
//
//   - SAP documents a distinct ValueMappingDesigntimeArtifactSaveAsVersion
//     action (POST) that takes the artifact's technical ID plus a
//     caller-supplied new version identifier. Multiple independent secondary
//     sources describe it specifically as the way to "change the version of
//     ValueMapping" — a materially different contract from
//     IntegrationDesigntimeArtifacts' PUT, which lets SAP assign the new
//     version implicitly. A dedicated SAP Knowledge Base Article (3502529,
//     "HTTP/403 Forbidden response while trying to change Version of the
//     ValueMapping") independently confirms that changing a value mapping's
//     version is treated as its own distinct, separately gated operation.
//   - An independent third-party tool built directly against this same SAP
//     OData API (github.com/lemaiwo/ci-mcp-server) explicitly disables its
//     generic "update" operation for ValueMappingDesigntimeArtifacts, while
//     leaving it enabled for IntegrationDesigntimeArtifacts,
//     MessageMappingDesigntimeArtifacts, and ScriptCollectionDesigntimeArtifacts
//     — the same family of design-time artifact entity sets. That is a
//     deliberate difference, not an oversight, and it points the same
//     direction as the two findings above.
//
// None of this is a byte-for-byte primary-source confirmation: help.sap.com,
// api.sap.com, community.sap.com, blogs.sap.com, and every mirror/proxy this
// environment could reach for them were blocked by network egress policy
// during this investigation, so the exact SaveAsVersion request/response
// shape could not be fetched and verified directly. Given that, retaining an
// unverified PUT — which the evidence above suggests does not work the way
// sapintegrationsuite_integration_flow's does — would be shipping a guess
// dressed up as a feature. sapintegrationsuite_value_mapping instead treats
// name/content/content_hash changes as replacing the resource (see
// resource_value_mapping.go), which only relies on Create and Delete, both
// independently confirmed. Implementing update-in-place via
// ValueMappingDesigntimeArtifactSaveAsVersion is a P3 item in ROADMAP.md
// once its request/response contract can be confirmed against a live
// tenant or a reachable primary source; see docs/sap-api-references.md and
// docs/resource-design.md.

// DeleteValueMapping deletes the value mapping design-time artifact
// identified by mappingID, addressed the same way GetValueMapping and the
// (removed) update path address it: via the "active" version key. Whether
// this removes only the active/current design-time version or every version
// of the artifact has not been confirmed against a primary source (the same
// network restrictions described above applied to this question too). This
// provider does not expose multiple versions of a value mapping
// concurrently, so the distinction does not change this resource's
// behavior today, but it means a value mapping deleted through Terraform
// could in principle leave older, non-active versions behind on SAP's side.
// Treat this as an open item for verification against a live tenant before
// relying on "destroy always removes everything SAP stored" as a guarantee.
func (c *Client) DeleteValueMapping(ctx context.Context, mappingID string) error {
	key, err := designtimeArtifactKey(mappingID, activeVersion)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, v2.BuildPath(valueMappingDesigntimeArtifactsEntitySet, key, ""))
}

// DeployValueMapping triggers deployment of the given design-time version
// of a value mapping. It returns immediately once SAP has accepted the
// deployment request; callers must poll GetRuntimeArtifact for completion,
// since deployment is asynchronous.
//
// The action name is singular ("...Artifact", not "...Artifacts"), matching
// the equivalent action for every other design-time artifact type in this
// API family (DeployIntegrationDesigntimeArtifact,
// DeployMessageMappingDesigntimeArtifact, ...). This was re-checked this
// phase after a report that current SAP documentation uses the plural form:
// every reachable secondary source (multiple independent search results
// summarizing SAP community/blog content) consistently gives the singular
// form and the Id='...'&Version='...' query-parameter shape already
// implemented below, and no source found gave the plural form. The
// documentation pages that would settle this conclusively
// (help.sap.com/api.sap.com) were not reachable from this environment to
// fetch and read directly — see the note on DeleteValueMapping above for
// why.
func (c *Client) DeployValueMapping(ctx context.Context, mappingID, version string) error {
	path := fmt.Sprintf("DeployValueMappingDesigntimeArtifact?Id='%s'&Version='%s'",
		v2.EscapeLiteral(mappingID), v2.EscapeLiteral(version))
	_, err := c.odata.Post(ctx, path, nil)
	return err
}
