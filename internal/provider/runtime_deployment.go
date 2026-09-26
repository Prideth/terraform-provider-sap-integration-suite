package provider

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/apierror"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// Poll timing for deployments; variables so that tests can shorten them.
var (
	deploymentPollInterval = 3 * time.Second
	deploymentPollMax      = 20 * time.Second
)

// waitForRuntimeArtifact polls the shared runtime-artifacts entity for id
// until it reaches STARTED (success) or ERROR (failure), using exponential
// backoff with full jitter, bounded by ctx's deadline. It never sleeps a
// fixed duration. Shared by every *_deployment resource (integration flow,
// value mapping, ...) since they all deploy through the same runtime status
// model.
func waitForRuntimeArtifact(ctx context.Context, client *cloudintegration.Client, id string) (*cloudintegration.RuntimeArtifact, error) {
	return waitForDeployment(ctx, client, id, "")
}

// waitForDeployment is waitForRuntimeArtifact that also follows the task a
// deploy request returned (empty: no task). While the runtime artifact does
// not exist yet, it reads the task in BuildAndDeployStatus: a failed task
// ends the wait at once, and a task that succeeded while the artifact still
// does not appear means SAP deployed it to another runtime (see
// deployedElsewhere), so waiting for the timeout would be pointless.
func waitForDeployment(ctx context.Context, client *cloudintegration.Client, id, taskID string) (*cloudintegration.RuntimeArtifact, error) {
	attempt, missingAfterSuccess := 0, 0
	for {
		artifact, err := client.GetRuntimeArtifact(ctx, id)
		if err == nil {
			switch artifact.Status {
			case cloudintegration.StatusStarted:
				return artifact, nil
			case cloudintegration.StatusError:
				return nil, deploymentFailure(ctx, client, id)
			}
		} else {
			var apiErr *apierror.Error
			if !errors.As(err, &apiErr) || !apiErr.IsNotFound() {
				return nil, err
			}
			// Not found yet: the runtime artifact has not been created by
			// SAP's asynchronous deployment pipeline. Keep polling, unless the
			// deploy task says there is nothing to wait for.
			if err := checkDeployTask(ctx, client, id, taskID, &missingAfterSuccess); err != nil {
				return nil, err
			}
		}

		delay := pollBackoff(attempt)
		attempt++

		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, fmt.Errorf("timed out waiting for the deployment to become ready: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

// deploymentFailure builds the error for a deployment SAP reports as ERROR.
// The reason is a separate media entity; reading it is best effort, since
// the deployment failed either way.
func deploymentFailure(ctx context.Context, client *cloudintegration.Client, id string) error {
	errInfo, err := client.GetRuntimeArtifactErrorInformation(ctx, id)
	if err != nil || errInfo == "" {
		errInfo = "SAP reported status ERROR without further detail on the runtime artifact"
	}
	return fmt.Errorf("deployment failed: %s", errInfo)
}

// pollBackoff computes an exponential backoff with full jitter for status
// polling, capped at deploymentPollMax.
func pollBackoff(attempt int) time.Duration {
	capped := math.Min(float64(deploymentPollMax), float64(deploymentPollInterval)*math.Pow(1.5, float64(attempt)))
	//nolint:gosec // G404: jitter timing does not need a cryptographically secure random source
	return time.Duration(rand.Int63n(int64(capped))) + deploymentPollInterval/2
}

// missingPollsAfterSuccess is how many polls in a row may still find no
// runtime artifact after SAP reported the deploy task as SUCCESS. On a
// tenant, a flow deployed to the Cloud Integration runtime was STARTED on
// the first poll after SUCCESS.
const missingPollsAfterSuccess = 3

// checkDeployTask reads the deploy task while the runtime artifact is still
// missing. It returns an error when the task failed, or when it succeeded
// and the artifact has not appeared for missingPollsAfterSuccess polls. A
// failing status read is ignored: the runtime status stays the source of
// truth and the timeout still applies.
func checkDeployTask(ctx context.Context, client *cloudintegration.Client, id, taskID string, missingAfterSuccess *int) error {
	if taskID == "" {
		return nil
	}
	task, err := client.GetBuildAndDeployStatus(ctx, taskID)
	if err != nil {
		return nil //nolint:nilerr // best effort; the runtime status and the timeout still decide
	}
	status := strings.ToUpper(task.Status)
	switch {
	case status == "SUCCESS":
		*missingAfterSuccess++
		if *missingAfterSuccess >= missingPollsAfterSuccess {
			return deployedElsewhere(id, taskID)
		}
	case strings.Contains(status, "FAIL") || strings.Contains(status, "ERROR"):
		return fmt.Errorf("deployment failed: SAP reports the deployment task %s of %s as %s", taskID, id, task.Status)
	}
	return nil
}

// deployedElsewhere explains the case a tenant showed in September 2026: a
// flow whose externalized parameter SAP_ProfileId is "integrationcell" is
// deployed successfully, but to the Integration Cell runtime, so it never
// appears among the Cloud Integration runtime artifacts. With
// SAP_ProfileId "iflmap" the same flow was STARTED at once.
func deployedElsewhere(id, taskID string) error {
	return fmt.Errorf("SAP reports the deployment task %s as SUCCESS, but %s does not appear in the Cloud "+
		"Integration runtime. It was most likely deployed to another runtime profile: check the flow's "+
		"SAP_ProfileId configuration parameter. \"integrationcell\" deploys to Edge/Integration Cell, which this "+
		"provider does not support; set it to \"iflmap\" with sapintegrationsuite_integration_flow_configuration "+
		"to deploy to Cloud Integration", taskID, id)
}
