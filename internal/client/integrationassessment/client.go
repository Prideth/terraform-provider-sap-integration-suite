// Package integrationassessment is the client for the Entities API of SAP
// Integration Assessment: the OData V2 service behind the landscape that
// Integration Solution Advisory Methodology assessments run against
// (vendors, applications, application instances, technologies, technology
// instances) and its reference taxonomy (deployment models and others).
//
// SAP documents the service, its entity inventory and per-tenant limits in
// SAP Help, and lists it as EntitiesAPI in the Business Accelerator Hub
// package SAPIntegrationAssessment. The field-level specification on the
// Hub needs an SAP login; the wire structs here follow the service's live
// $metadata (testdata/api-metadata/integration-assessment-entities.json),
// and every request was verified on a tenant in September 2026.
package integrationassessment

import (
	"encoding/json"
	"net/http"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// HTTPDoer is the transport the client needs: the shared retrying HTTP
// client from internal/client/http satisfies it.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client is the Integration Assessment Entities API client.
type Client struct {
	odata *v2.Client
}

// New builds a client rooted at the Entities API service root, the service
// key's "entities" value (for example
// https://intas-api.cfapps.eu10.hana.ondemand.com/intas/entities/v1).
func New(httpClient HTTPDoer, entitiesURL string) *Client {
	return &Client{odata: v2.New(httpClient, strings.TrimRight(entitiesURL, "/"))}
}

// idRef links an entity to another one in a create or update body. The
// service answered a link sent as {"__metadata": {"uri": ...}} with 400 V124
// "The navigation property contains no Id field"; {"Id": ...} works.
type idRef struct {
	ID string `json:"Id"`
}

// ref returns a link to id, or nil for an empty id.
func ref(id string) *idRef {
	if id == "" {
		return nil
	}
	return &idRef{ID: id}
}

// linkedEntity decodes a single-valued navigation property in a read: the
// linked entity when the request expanded it, {"__deferred": ...} when it
// did not, or null when nothing is linked.
type linkedEntity struct {
	ID       string          `json:"Id"`
	Deferred json.RawMessage `json:"__deferred"`
}

// id returns the linked entity's Id, or "" when nothing is linked.
func (l *linkedEntity) id() string {
	if l == nil {
		return ""
	}
	return l.ID
}

func entityPath(entitySet, id, query string) string {
	return v2.BuildPath(entitySet, v2.KeyPredicate(id), query)
}
