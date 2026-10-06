// Package apimanagementclassic implements the client for SAP's classic API
// Management API Portal service (Management.svc under /apiportal/api/1.0),
// distinct from both Cloud Integration's OData V2 services and from the
// current, API-artifact-centric API Management model, which this project
// has confirmed has no public API at all. The API Portal's service is
// confirmed, by direct inspection of the official "SAP API Management
// Standalone Service" user guide, to use the identical OData V2 response
// envelope ({"d": {...}} / {"d": {"results": [...]}}), error format
// ({"error": {"code", "message": {"lang", "value"}}}), and single-quoted
// key-predicate convention this project's internal/client/odata/v2 package
// already implements for Cloud Integration — so this package builds
// directly on top of it rather than duplicating that decoding logic, while
// keeping its own entity types and operations, its own authentication
// (the apiportal-apiaccess service plan, never Cloud Integration's OAuth
// client), and its own base path.
package apimanagementclassic

import (
	"net/http"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// HTTPDoer is the transport this client needs: the shared retrying
// *http.Client from internal/client/http satisfies this directly.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the Classic API Management (API Portal) client.
type Client struct {
	odata *v2.Client
	// transport addresses Transport.svc, the API portal's import and export
	// service for API proxy bundles, next to Management.svc.
	transport *v2.Client
	// configuration addresses Configuration.svc under /apiportal/operations/1.0,
	// where virtual hosts are created, changed and deleted. Only a key with
	// the role APIManagement.SelfService.Administrator may write there.
	configuration *v2.Client
}

// New builds a Classic API Management client. host is the API Portal
// application URL from the apiportal-apiaccess service key's "url" field
// (for example https://tenant.prod-eu10.apiportal.cfapps.eu10.hana.ondemand.com);
// the service's "/apiportal/api/1.0/Management.svc" path is appended
// automatically.
func New(httpClient HTTPDoer, host string) *Client {
	base := strings.TrimRight(host, "/")
	root := base + "/apiportal/api/1.0"
	odata := v2.New(httpClient, root+"/Management.svc")
	return &Client{
		odata:         odata,
		transport:     odata.WithBaseURL(root + "/Transport.svc"),
		configuration: odata.WithBaseURL(base + "/apiportal/operations/1.0/Configuration.svc"),
	}
}
