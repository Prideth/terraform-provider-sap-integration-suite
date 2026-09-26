package provider

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/cloudintegration"
)

// TestWaitForRuntimeArtifact_Started covers the ordinary success path used
// by both sapintegrationsuite_integration_flow_deployment and
// sapintegrationsuite_value_mapping_deployment: SAP eventually reports
// STARTED for the deployed artifact.
func TestWaitForRuntimeArtifact_Started(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Version": "1.0.0", "Status": "STARTED"}}`))
	}))
	defer server.Close()

	client := cloudintegration.New(http.DefaultClient, server.URL)

	artifact, err := waitForRuntimeArtifact(context.Background(), client, "company-codes")
	if err != nil {
		t.Fatalf("waitForRuntimeArtifact() error: %v", err)
	}
	if artifact.Status != cloudintegration.StatusStarted {
		t.Errorf("Status = %q, want %q", artifact.Status, cloudintegration.StatusStarted)
	}
}

// TestWaitForRuntimeArtifact_Error covers a deployment that SAP reports as
// failed: waitForRuntimeArtifact must return an error carrying SAP's error
// text rather than treating ERROR as a transient state to keep polling
// through. The tenant $metadata declares ErrorInformation as a navigation
// property to a media entity, so the status read carries only a __deferred
// link and the text comes from ErrorInformation/$value.
func TestWaitForRuntimeArtifact_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if strings.HasSuffix(r.URL.Path, "/ErrorInformation/$value") {
			_, _ = w.Write([]byte("invalid mapping schema\n"))
			return
		}
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Version": "1.0.0", "Status": "ERROR", "ErrorInformation": {"__deferred": {"uri": "https://host/api/v1/IntegrationRuntimeArtifacts('company-codes')/ErrorInformation"}}}}`))
	}))
	defer server.Close()

	client := cloudintegration.New(http.DefaultClient, server.URL)

	_, err := waitForRuntimeArtifact(context.Background(), client, "company-codes")
	if err == nil {
		t.Fatal("expected an error for a deployment reported as ERROR")
	}
	if !strings.Contains(err.Error(), "invalid mapping schema") {
		t.Errorf("error = %q, want it to contain SAP's error information", err.Error())
	}
}

// When the error text cannot be read, the deployment still fails with a
// generic reason instead of hiding the failure behind the read error.
func TestWaitForRuntimeArtifact_ErrorWithoutDetail(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/ErrorInformation/$value") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "no error information"}}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Version": "1.0.0", "Status": "ERROR"}}`))
	}))
	defer server.Close()

	client := cloudintegration.New(http.DefaultClient, server.URL)

	_, err := waitForRuntimeArtifact(context.Background(), client, "company-codes")
	if err == nil || !strings.Contains(err.Error(), "without further detail") {
		t.Fatalf("error = %v, want the generic deployment failure", err)
	}
}

// TestWaitForRuntimeArtifact_Timeout covers a deployment that never reaches
// a terminal state before the caller's context expires (the resource's
// configured timeout): waitForRuntimeArtifact must stop polling and return
// promptly once ctx is done, never block past it.
func TestWaitForRuntimeArtifact_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Version": "1.0.0", "Status": "STARTING"}}`))
	}))
	defer server.Close()

	client := cloudintegration.New(http.DefaultClient, server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := waitForRuntimeArtifact(ctx, client, "company-codes")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected a timeout error, got nil")
	}
	if elapsed > 5*time.Second {
		t.Errorf("waitForRuntimeArtifact() took %s after context expiry, want it to return promptly", elapsed)
	}
}

// TestWaitForRuntimeArtifact_NotFoundThenStarted covers the documented
// not-found-yet window right after a deploy is accepted, before SAP's
// asynchronous deployment pipeline has created the runtime artifact: a 404
// must be treated as "keep polling", not as a terminal failure.
func TestWaitForRuntimeArtifact_NotFoundThenStarted(t *testing.T) {
	var calls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": {"value": "not yet deployed"}}}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d": {"Id": "company-codes", "Version": "1.0.0", "Status": "STARTED"}}`))
	}))
	defer server.Close()

	client := cloudintegration.New(http.DefaultClient, server.URL)

	artifact, err := waitForRuntimeArtifact(context.Background(), client, "company-codes")
	if err != nil {
		t.Fatalf("waitForRuntimeArtifact() error: %v", err)
	}
	if artifact.Status != cloudintegration.StatusStarted {
		t.Errorf("Status = %q, want %q", artifact.Status, cloudintegration.StatusStarted)
	}
	if calls < 2 {
		t.Errorf("calls = %d, want at least 2 (a 404 followed by a successful poll)", calls)
	}
}

func fastDeploymentPolls(t *testing.T) {
	t.Helper()
	interval, maxDelay := deploymentPollInterval, deploymentPollMax
	deploymentPollInterval, deploymentPollMax = 10*time.Millisecond, 20*time.Millisecond
	t.Cleanup(func() { deploymentPollInterval, deploymentPollMax = interval, maxDelay })
}

// taskServer answers the runtime status with 404 (or the given artifact
// status) and BuildAndDeployStatus with taskStatus.
func taskServer(t *testing.T, runtimeStatus, taskStatus string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/v1/BuildAndDeployStatus"):
			_, _ = w.Write([]byte(`{"d": {"TaskId": "task-1", "Status": "` + taskStatus + `"}}`))
		case runtimeStatus == "":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error": {"code": "Not Found", "message": {"value": "not found"}}}`))
		default:
			_, _ = w.Write([]byte(`{"d": {"Id": "flow", "Version": "1.0.0", "Status": "` + runtimeStatus + `"}}`))
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// A tenant deployed a flow with SAP_ProfileId "integrationcell": the task
// ended with SUCCESS, but no Cloud Integration runtime artifact appeared.
// The wait stops with an explanation instead of running into the timeout.
func TestWaitForDeployment_TaskSucceededButNotInRuntime(t *testing.T) {
	fastDeploymentPolls(t)
	client := cloudintegration.New(http.DefaultClient, taskServer(t, "", "SUCCESS").URL)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := waitForDeployment(ctx, client, "flow", "task-1")
	if err == nil || !strings.Contains(err.Error(), "SAP_ProfileId") {
		t.Fatalf("error = %v, want the runtime profile explanation", err)
	}
	if ctx.Err() != nil {
		t.Error("the wait ran into the timeout instead of stopping early")
	}
}

func TestWaitForDeployment_TaskFailed(t *testing.T) {
	fastDeploymentPolls(t)
	client := cloudintegration.New(http.DefaultClient, taskServer(t, "", "FAIL").URL)
	_, err := waitForDeployment(context.Background(), client, "flow", "task-1")
	if err == nil || !strings.Contains(err.Error(), "FAIL") {
		t.Fatalf("error = %v, want the failed task", err)
	}
}

func TestWaitForDeployment_StartedIgnoresTask(t *testing.T) {
	fastDeploymentPolls(t)
	client := cloudintegration.New(http.DefaultClient, taskServer(t, "STARTED", "SUCCESS").URL)
	artifact, err := waitForDeployment(context.Background(), client, "flow", "task-1")
	if err != nil || artifact.Status != "STARTED" {
		t.Fatalf("artifact %+v, err %v", artifact, err)
	}
}

// Without a task ID (other artifact types) the old behaviour stays: keep
// polling until the timeout.
func TestWaitForDeployment_NoTaskWaitsForTimeout(t *testing.T) {
	fastDeploymentPolls(t)
	client := cloudintegration.New(http.DefaultClient, taskServer(t, "", "SUCCESS").URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := waitForDeployment(ctx, client, "flow", "")
	// The deadline can hit between two polls ("timed out waiting ...") or
	// during a request ("context deadline exceeded"); both mean it waited.
	if err == nil {
		t.Fatal("want the timeout, got no error")
	}
	waited := strings.Contains(err.Error(), "timed out") || errors.Is(err, context.DeadlineExceeded) ||
		strings.Contains(err.Error(), "deadline exceeded")
	if !waited {
		t.Fatalf("error = %v, want the timeout", err)
	}
}
