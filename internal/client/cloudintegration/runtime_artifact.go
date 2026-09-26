package cloudintegration

import (
	"context"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// runtimeArtifactsEntitySet is the shared runtime-artifacts entity set: SAP
// documents its GET as the general "Runtime Status API" for "currently
// deployed integration artifacts", not one entity set per design-time
// artifact type. Integration flows, value mappings, and (per that same
// documentation) other deployable design-time artifact types are all read
// and undeployed through this single entity set.
const runtimeArtifactsEntitySet = "IntegrationRuntimeArtifacts"

// RuntimeArtifact is the wire representation of an IntegrationRuntimeArtifacts
// entity: the deployed state of a design-time artifact (integration flow,
// value mapping, ...).
//
// The tenant $metadata declares ErrorInformation on this entity only as a
// navigation property to RuntimeArtifactErrorInformation, a media entity
// (m:HasStream). A read returns it as a {"__deferred": ...} link, never as
// text, so it is not part of this struct; GetRuntimeArtifactErrorInformation
// reads the error text.
type RuntimeArtifact struct {
	ID         string `json:"Id"`
	Version    string `json:"Version"`
	Name       string `json:"Name,omitempty"`
	Type       string `json:"Type,omitempty"`
	Status     string `json:"Status"`
	DeployedBy string `json:"DeployedBy,omitempty"`
	DeployedOn string `json:"DeployedOn,omitempty"`
}

// Runtime deployment status values reported by IntegrationRuntimeArtifacts.
const (
	StatusStarted  = "STARTED"
	StatusStarting = "STARTING"
	StatusStopped  = "STOPPED"
	StatusError    = "ERROR"
)

// GetRuntimeArtifact reads the current runtime deployment status of a
// deployed design-time artifact (identified by its design-time ID, which is
// also the runtime artifact's ID). A missing deployment surfaces as a 404
// *apierror.Error.
func (c *Client) GetRuntimeArtifact(ctx context.Context, id string) (*RuntimeArtifact, error) {
	path := v2.BuildPath(runtimeArtifactsEntitySet, v2.KeyPredicate(id), "")

	body, err := c.odata.Get(ctx, path)
	if err != nil {
		return nil, err
	}

	var artifact RuntimeArtifact
	if err := v2.DecodeEntity(body, &artifact); err != nil {
		return nil, err
	}
	return &artifact, nil
}

// GetRuntimeArtifactErrorInformation reads why a deployment failed: the
// content of the ErrorInformation media entity, addressed with the standard
// OData $value path. SAP returns the text as it is, so the body is returned
// trimmed and without decoding.
func (c *Client) GetRuntimeArtifactErrorInformation(ctx context.Context, id string) (string, error) {
	path := v2.BuildPath(runtimeArtifactsEntitySet, v2.KeyPredicate(id), "") + "/ErrorInformation/$value"

	body, err := c.odata.Get(ctx, path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(body)), nil
}

// UndeployRuntimeArtifact removes the runtime deployment of a design-time
// artifact. A 404 means it was already undeployed.
func (c *Client) UndeployRuntimeArtifact(ctx context.Context, id string) error {
	path := v2.BuildPath(runtimeArtifactsEntitySet, v2.KeyPredicate(id), "")
	return c.odata.Delete(ctx, path)
}

const buildAndDeployStatusEntitySet = "BuildAndDeployStatus"

// BuildAndDeployStatus is the state of the task a deploy request started,
// keyed by the task ID the deploy returns. A tenant reported "SUCCESS" once
// the task was done (September 2026); other values are not documented.
type BuildAndDeployStatus struct {
	TaskID string `json:"TaskId"`
	Status string `json:"Status"`
}

// GetBuildAndDeployStatus reads the status of a deployment task.
func (c *Client) GetBuildAndDeployStatus(ctx context.Context, taskID string) (*BuildAndDeployStatus, error) {
	body, err := c.odata.Get(ctx, v2.BuildPath(buildAndDeployStatusEntitySet, v2.KeyPredicate(taskID), ""))
	if err != nil {
		return nil, err
	}
	var status BuildAndDeployStatus
	if err := v2.DecodeEntity(body, &status); err != nil {
		return nil, err
	}
	return &status, nil
}
