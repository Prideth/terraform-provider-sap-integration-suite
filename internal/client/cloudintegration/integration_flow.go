package cloudintegration

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const integrationDesigntimeArtifactsEntitySet = "IntegrationDesigntimeArtifacts"

// IntegrationFlow is the wire representation of an
// IntegrationDesigntimeArtifacts entity.
type IntegrationFlow struct {
	ID        string `json:"Id"`
	Name      string `json:"Name"`
	PackageID string `json:"PackageId"`
	Version   string `json:"Version,omitempty"`

	// Content is the base64-encoded ZIP of the integration flow project. It
	// is only populated on Create/Update requests; Read does not return it
	// (SAP does not include binary content in the metadata response).
	Content string `json:"ArtifactContent,omitempty"`
}

// GetIntegrationFlow reads the active design-time version's metadata for
// the flow identified by flowID within packageID.
func (c *Client) GetIntegrationFlow(ctx context.Context, packageID, flowID string) (*IntegrationFlow, error) {
	key, err := designtimeArtifactKey(flowID, activeVersion)
	if err != nil {
		return nil, err
	}

	body, err := c.odata.Get(ctx, v2.BuildPath(integrationDesigntimeArtifactsEntitySet, key, ""))
	if err != nil {
		return nil, err
	}

	var flow IntegrationFlow
	if err := v2.DecodeEntity(body, &flow); err != nil {
		return nil, err
	}
	return &flow, nil
}

// CreateIntegrationFlow creates a new integration flow design-time artifact
// from a ZIP project archive. content must already be the raw (not yet
// base64-encoded) ZIP bytes.
func (c *Client) CreateIntegrationFlow(ctx context.Context, packageID, flowID, name string, content []byte) (*IntegrationFlow, error) {
	payload, err := json.Marshal(IntegrationFlow{
		ID:        flowID,
		Name:      name,
		PackageID: packageID,
		Content:   base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding integration flow: %w", err)
	}

	body, err := c.odata.Post(ctx, integrationDesigntimeArtifactsEntitySet, payload)
	if err != nil {
		return nil, err
	}

	var flow IntegrationFlow
	if err := v2.DecodeEntity(body, &flow); err != nil {
		return nil, err
	}
	return &flow, nil
}

// UpdateIntegrationFlow uploads new content for an existing integration
// flow, creating a new design-time version under the same flow ID (SAP's
// design-time API is version-based, not in-place, which is why the flow's
// identity — packageID/flowID — never changes on update). Unlike creating a
// brand new flow, this targets the existing (Id, Version) entity with PUT
// rather than POSTing to the collection again, which is how OData V2
// distinguishes "create a new entity" from "update this one".
func (c *Client) UpdateIntegrationFlow(ctx context.Context, flowID, name string, content []byte) (*IntegrationFlow, error) {
	payload, err := json.Marshal(IntegrationFlow{
		Name:    name,
		Content: base64.StdEncoding.EncodeToString(content),
	})
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: encoding integration flow: %w", err)
	}

	key, err := designtimeArtifactKey(flowID, activeVersion)
	if err != nil {
		return nil, err
	}

	body, err := c.odata.Put(ctx, v2.BuildPath(integrationDesigntimeArtifactsEntitySet, key, ""), payload)
	if err != nil {
		return nil, err
	}

	var flow IntegrationFlow
	if err := v2.DecodeEntity(body, &flow); err != nil {
		return nil, err
	}
	return &flow, nil
}

// DeleteIntegrationFlow deletes an integration flow design-time artifact
// (all versions).
func (c *Client) DeleteIntegrationFlow(ctx context.Context, flowID string) error {
	key, err := designtimeArtifactKey(flowID, activeVersion)
	if err != nil {
		return err
	}
	return c.odata.Delete(ctx, v2.BuildPath(integrationDesigntimeArtifactsEntitySet, key, ""))
}

// DeployIntegrationFlow triggers deployment of the given design-time
// version of an integration flow. It returns immediately once SAP has
// accepted the deployment request; callers must poll GetRuntimeArtifact for
// completion, since deployment is asynchronous.
//
// version identifies which design-time version to deploy. Passing the
// literal "active" defers to whichever version is currently active, but the
// Terraform resource always passes the concrete version it read from the
// design-time artifact, so that changing the deployed version is a visible,
// plannable change rather than an implicit side effect of "whatever is
// active right now".
func (c *Client) DeployIntegrationFlow(ctx context.Context, flowID, version string) (taskID string, err error) {
	path := fmt.Sprintf("DeployIntegrationDesigntimeArtifact?Id='%s'&Version='%s'",
		v2.EscapeLiteral(flowID), v2.EscapeLiteral(version))
	body, err := c.odata.Post(ctx, path, nil)
	if err != nil {
		return "", err
	}
	return deployTaskID(body), nil
}

// deployTaskID reads the task ID from a deploy response. The $metadata
// declares Edm.String as the result; a tenant answered 202 with the bare ID
// as text ("2f93475d-9c44-4175-6b13-3f0b2c26e246"), so both that and the
// OData JSON form {"d": {"DeployIntegrationDesigntimeArtifact": "..."}} are
// read. An empty result means no task to follow.
func deployTaskID(body []byte) string {
	text := strings.TrimSpace(string(body))
	var wrapped struct {
		D map[string]json.RawMessage `json:"d"`
	}
	if json.Unmarshal(body, &wrapped) == nil && len(wrapped.D) > 0 {
		for _, raw := range wrapped.D {
			var id string
			if json.Unmarshal(raw, &id) == nil {
				return id
			}
		}
		return ""
	}
	if strings.ContainsAny(text, " {}<>\n") {
		return ""
	}
	return strings.Trim(text, "\"")
}
