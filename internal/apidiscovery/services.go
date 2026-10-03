// Package apidiscovery fetches the published contracts of the SAP
// Integration Suite services this provider knows, compares them with the
// committed snapshots in testdata/api-metadata, and renders the discovery
// report. It only ever requests a service's documented root and its
// $metadata or specification; it never guesses URLs.
package apidiscovery

import (
	"os"
	"sort"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

// Capabilities name the acceptance-test suites (SAP_INTEGRATION_SUITE_ACC_<name>)
// a service belongs to.
const (
	CapCloudIntegration      = "CLOUD_INTEGRATION"
	CapSecurityContent       = "SECURITY_CONTENT"
	CapPartnerDirectory      = "PARTNER_DIRECTORY"
	CapAPIManagementClassic  = "API_MANAGEMENT_CLASSIC"
	CapAPIManagementCurrent  = "API_MANAGEMENT_CURRENT"
	CapAPIComposition        = "API_COMPOSITION"
	CapIntegrationAssessment = "INTEGRATION_ASSESSMENT"
	CapEdgeIntegrationCell   = "EDGE_INTEGRATION_CELL"
	CapDataSpaceIntegration  = "DATA_SPACE_INTEGRATION"
)

// Credentials are the OAuth client credentials of one service. Username,
// Password and Origin are an optional user login through that client
// (password grant); only services resolved with userLoginService read them.
type Credentials struct {
	TokenURL     string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Origin       string
}

// Service is one documented service root.
type Service struct {
	ID       string // snapshot name, testdata/api-metadata/<ID>.json
	Title    string
	Protocol string // apimeta.ProtocolODataV2, ProtocolODataV4 or ProtocolOpenAPI
	// Capabilities lists the acceptance suites that cover this service.
	Capabilities []string
	// Families lists the provider areas the service carries.
	Families []string
	// Evidence says where the service root is documented.
	Evidence string
	// DocumentPath is the contract document below the root; "/$metadata"
	// when empty. An OpenAPI service names its specification here.
	DocumentPath string
	// Env lists the environment variables Resolve reads.
	Env []string
	// Resolve returns the service root and credentials from the
	// environment, or the names of the variables that are missing. Nil for
	// a service whose contract comes only from an official specification.
	Resolve func(getenv func(string) string) (root string, creds Credentials, missing []string)
	// Specification names the official document to download when the
	// contract is not served by the service itself (a REST API whose
	// OpenAPI document is published on the Business Accelerator Hub). The
	// discovery then reads it from the -spec-dir directory as <ID>.json,
	// <ID>.yaml or <ID>.xml and never calls the service.
	Specification string
}

// SpecificationOnly reports whether the service's contract comes only from
// a downloaded official specification.
func (s Service) SpecificationOnly() bool { return s.Resolve == nil }

// Services is every service root the discovery knows. A root belongs here
// only with supported evidence: SAP documentation, a service key field, or
// official SAP tooling.
var Services = []Service{
	{
		ID: "cloud-integration", Title: "Cloud Integration (Integration Content, Security Content, Partner Directory, message stores)",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapCloudIntegration, CapSecurityContent, CapPartnerDirectory},
		Families:     []string{"cloud_integration", "security_content", "partner_directory", "access_policies"},
		Evidence: "SAP Help, Cloud Integration APIs: service root <tenant>/api/v1 of the API plan service key " +
			"(SAP Business Accelerator Hub package CloudIntegrationAPI).",
		Env:     mainEnv,
		Resolve: mainService("/api/v1"),
	},
	{
		ID: "classic-api-management", Title: "Classic API Management, API portal Management.svc",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapAPIManagementClassic},
		Families:     []string{"api_management_classic"},
		Evidence: "SAP Help, API Management APIs on Cloud Foundry: <api portal url>/apiportal/api/1.0/Management.svc " +
			"(service key of API Management, API portal, plan apiportal-apiaccess); SAP API Management Client SDK.",
		Env:     apiManagementEnv,
		Resolve: apiManagementService("/apiportal/api/1.0/Management.svc"),
	},
	{
		ID: "classic-api-management-transport", Title: "Classic API Management, API portal Transport API (REST)",
		Protocol:     apimeta.ProtocolOpenAPI,
		Capabilities: []string{CapAPIManagementClassic},
		Families:     []string{"api_management_classic"},
		Evidence: "Business Accelerator Hub, package APIMgmt, API \"API Portal - Transport (CF)\" (REST, " +
			"APIPortal_Transport_CF); SAP API Management Client SDK 3.0.6 imports and exports proxies through " +
			"/apiportal/api/1.0/Transport.svc/APIProxies.",
		Specification: "api.sap.com/api/APIPortal_Transport_CF, API specification download (JSON)",
	},
	{
		ID: "classic-api-management-content-archive", Title: "Classic API Management, API portal Content Archive Transport API (REST)",
		Protocol:     apimeta.ProtocolOpenAPI,
		Capabilities: []string{CapAPIManagementClassic},
		Families:     []string{"api_management_classic"},
		Evidence: "Business Accelerator Hub, package APIMgmt, API \"API Portal - Content Archive Transport (CF)\" " +
			"(REST, APIPortal_Content_Archive_Transport_CF); used by the Client SDK 3.0.6 for multi-proxy export.",
		Specification: "api.sap.com/api/APIPortal_Content_Archive_Transport_CF, API specification download (JSON)",
	},
	{
		ID: "classic-api-management-virtual-host-request", Title: "Classic API Management, virtual host requests (OData)",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapAPIManagementClassic},
		Families:     []string{"api_management_classic"},
		Evidence: "Business Accelerator Hub, package APIMgmt, API \"API Portal - Virtual Host Request (CF)\" " +
			"(OData, APIPortal_VirtualHostRequest_CF); SAP Help, Configuring a Default Domain for a Virtual Host.",
		Specification: "api.sap.com/api/APIPortal_VirtualHostRequest_CF, API specification download (EDMX)",
	},
	{
		ID: "edge-integration-cell", Title: "Cloud Integration APIs of an Edge Integration Cell runtime location",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapEdgeIntegrationCell},
		Families:     []string{"edge_integration_cell"},
		Evidence:     "SAP Help, Edge Integration Cell: runtime-specific APIs under <tenant>/location/<runtime location ID>/api/v1.",
		Env:          append(append([]string{}, mainEnv...), "SAP_INTEGRATION_SUITE_RUNTIME_LOCATION_ID"),
		Resolve: func(getenv func(string) string) (string, Credentials, []string) {
			root, creds, missing := mainService("")(getenv)
			loc := getenv("SAP_INTEGRATION_SUITE_RUNTIME_LOCATION_ID")
			if loc == "" {
				missing = append(missing, "SAP_INTEGRATION_SUITE_RUNTIME_LOCATION_ID")
			}
			return strings.TrimRight(root, "/") + "/location/" + loc + "/api/v1", creds, missing
		},
	},
	{
		ID: "api-composition-configuration", Title: "API Composition, Configuration API (business data graph)",
		Protocol:     apimeta.ProtocolODataV4,
		Capabilities: []string{CapAPIComposition},
		Families:     []string{"api_composition"},
		Evidence: "SAP Help, API Composition Configuration API: <host>/configuration/v1/sap.graph with a service key of " +
			"plan configuration; Business Accelerator Hub lists the API as OData V4. On a tenant only a key user's " +
			"token (password grant through that key's client) was allowed to read it.",
		Env: []string{"SAP_INTEGRATION_SUITE_API_COMPOSITION_HOST", "SAP_INTEGRATION_SUITE_API_COMPOSITION_TOKEN_URL",
			"SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_ID", "SAP_INTEGRATION_SUITE_API_COMPOSITION_CLIENT_SECRET"},
		Resolve: userLoginService("SAP_INTEGRATION_SUITE_API_COMPOSITION_", "HOST", "/configuration/v1/sap.graph"),
	},
	{
		ID: "integration-assessment-entities", Title: "Integration Assessment, entities API (landscape and ISA-M data)",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapIntegrationAssessment},
		Families:     []string{"integration_assessment"},
		Evidence: "Service key of Integration Assessment APIs: field \"entities\" is the service root " +
			"(SAP Help, Integration Assessment APIs).",
		Env:     integrationAssessmentEnv("ENTITIES_URL"),
		Resolve: prefixedService("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_", "ENTITIES_URL", ""),
	},
	{
		ID: "integration-assessment-management", Title: "Integration Assessment, management API (requests)",
		Protocol:     apimeta.ProtocolODataV2,
		Capabilities: []string{CapIntegrationAssessment},
		Families:     []string{"integration_assessment"},
		Evidence: "Service key of Integration Assessment APIs: field \"management\" is the service root " +
			"(SAP Help, Integration Assessment APIs).",
		Env:     integrationAssessmentEnv("MANAGEMENT_URL"),
		Resolve: prefixedService("SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_", "MANAGEMENT_URL", ""),
	},
	{
		ID: "data-space-integration", Title: "Data Space Integration API (DSIAPI, REST)",
		Protocol:      apimeta.ProtocolOpenAPI,
		Capabilities:  []string{CapDataSpaceIntegration},
		Families:      []string{"data_space_integration"},
		Evidence:      "Business Accelerator Hub, package dataspaceintegration, API DSIAPI 2.0.0 (REST).",
		Specification: "api.sap.com/api/DSIAPI, API specification download (JSON)",
	},
}

var mainEnv = []string{"SAP_INTEGRATION_SUITE_HOST", "SAP_INTEGRATION_SUITE_TOKEN_URL",
	"SAP_INTEGRATION_SUITE_CLIENT_ID", "SAP_INTEGRATION_SUITE_CLIENT_SECRET"}

var apiManagementEnv = []string{"SAP_INTEGRATION_SUITE_API_MANAGEMENT_HOST", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_TOKEN_URL",
	"SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_ID", "SAP_INTEGRATION_SUITE_API_MANAGEMENT_CLIENT_SECRET"}

func integrationAssessmentEnv(urlVar string) []string {
	p := "SAP_INTEGRATION_SUITE_INTEGRATION_ASSESSMENT_"
	return []string{p + urlVar, p + "TOKEN_URL", p + "CLIENT_ID", p + "CLIENT_SECRET"}
}

func mainService(path string) func(func(string) string) (string, Credentials, []string) {
	return func(getenv func(string) string) (string, Credentials, []string) {
		return resolve(getenv, "SAP_INTEGRATION_SUITE_HOST", "SAP_INTEGRATION_SUITE_", path)
	}
}

func apiManagementService(path string) func(func(string) string) (string, Credentials, []string) {
	return prefixedService("SAP_INTEGRATION_SUITE_API_MANAGEMENT_", "HOST", path)
}

func prefixedService(prefix, urlVar, path string) func(func(string) string) (string, Credentials, []string) {
	return func(getenv func(string) string) (string, Credentials, []string) {
		return resolve(getenv, prefix+urlVar, prefix, path)
	}
}

// userLoginService is prefixedService plus the optional <prefix>USERNAME,
// <prefix>PASSWORD and <prefix>ORIGIN of a user login. They are never
// reported as missing.
func userLoginService(prefix, urlVar, path string) func(func(string) string) (string, Credentials, []string) {
	return func(getenv func(string) string) (string, Credentials, []string) {
		root, creds, missing := resolve(getenv, prefix+urlVar, prefix, path)
		creds.Username = getenv(prefix + "USERNAME")
		creds.Password = getenv(prefix + "PASSWORD")
		creds.Origin = getenv(prefix + "ORIGIN")
		return root, creds, missing
	}
}

func resolve(getenv func(string) string, urlVar, credPrefix, path string) (string, Credentials, []string) {
	var missing []string
	get := func(name string) string {
		v := getenv(name)
		if v == "" {
			missing = append(missing, name)
		}
		return v
	}
	root := get(urlVar)
	creds := Credentials{
		TokenURL:     get(credPrefix + "TOKEN_URL"),
		ClientID:     get(credPrefix + "CLIENT_ID"),
		ClientSecret: get(credPrefix + "CLIENT_SECRET"),
	}
	if root != "" {
		root = strings.TrimRight(root, "/") + path
	}
	return root, creds, missing
}

// Lookup returns the service with an ID.
func Lookup(id string) (Service, bool) {
	for _, s := range Services {
		if s.ID == id {
			return s, true
		}
	}
	return Service{}, false
}

// Configured reports whether the environment has everything the service
// needs, and lists what is missing (variable names only, never values).
func (s Service) Configured() (bool, []string) {
	if s.SpecificationOnly() {
		return false, []string{"official specification (" + s.Specification + ")"}
	}
	_, _, missing := s.Resolve(os.Getenv)
	sort.Strings(missing)
	return len(missing) == 0, missing
}

func (s Service) documentPath() string {
	if s.DocumentPath == "" {
		return "/$metadata"
	}
	return s.DocumentPath
}
