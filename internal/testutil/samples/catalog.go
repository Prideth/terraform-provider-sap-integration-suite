// Package samples gives tests real SAP Integration Suite content from SAP's
// public sample repositories (github.com/SAP-samples, Apache-2.0).
//
// Each sample is pinned to a commit and a SHA-256 hash. Files are downloaded
// on demand into a local cache and verified before use, so a test always sees
// exactly the bytes listed here. Nothing from these repositories is copied
// into this repository.
//
// Downloading happens only when SAP_SAMPLES_DOWNLOAD=1 or TF_ACC=1 is set;
// otherwise a test whose sample is not cached yet is skipped. The cache lives
// in $SAP_SAMPLES_CACHE, or in the user cache directory.
package samples

// Kind is what a sample file contains.
type Kind string

const (
	// KindIntegrationFlow is a single integration flow artifact ZIP, the
	// content sapintegrationsuite_integration_flow uploads.
	KindIntegrationFlow Kind = "IntegrationFlow"
	// KindMessageMapping is a single message mapping artifact ZIP.
	KindMessageMapping Kind = "MessageMapping"
	// KindPackageExport is an integration package exported from the Design
	// UI: resources.cnt plus one <id>_content ZIP per artifact.
	KindPackageExport Kind = "PackageExport"
	// KindAPIProxy is a Classic API Management API proxy bundle.
	KindAPIProxy Kind = "APIProxy"
	// KindPolicyTemplate is a Classic API Management policy template.
	KindPolicyTemplate Kind = "PolicyTemplate"
)

// Sample is one pinned file from an SAP-samples repository.
type Sample struct {
	Name   string
	Repo   string // repository in the SAP-samples organization
	Commit string
	Path   string
	SHA256 string
	Kind   Kind
	// BundleID is the Bundle-SymbolicName of an artifact ZIP.
	BundleID string
	// Deployable is true only for content that does nothing on its own once
	// deployed: no timer start event and no polling sender, so it runs only
	// when someone calls its endpoint. Live tests deploy nothing else.
	Deployable bool
	Notes      string
}

// Catalog lists every pinned sample. Hashes were taken on 2026-09-26.
var Catalog = []Sample{
	{
		Name: "spend-account-dim-map", Repo: "btp-spend-analysis",
		Commit: "fabb838cc68009e23aabe04efadb214c97492f4d",
		Path:   "src/version-two/spend-analysis/Ariba_AccountDimMap.zip",
		SHA256: "eecbe01863f1065faa862b41ce1cc2cd274b8759bf854d5de9acd5a2b3624798",
		Kind:   KindMessageMapping, BundleID: "Ariba_AccountDimMap", Deployable: true,
		Notes: "Message mapping between two XSDs, no external calls.",
	},
	{
		Name: "spend-supplier-dim-map", Repo: "btp-spend-analysis",
		Commit: "fabb838cc68009e23aabe04efadb214c97492f4d",
		Path:   "src/version-two/spend-analysis/Ariba_SupplierDimMap.zip",
		SHA256: "8edaf8e7a5a9663cc2ec90dc666b0a71e15439194cbf3f2bf426b13fa0bf6afc",
		Kind:   KindMessageMapping, BundleID: "Ariba_SupplierDimMap", Deployable: true,
		Notes: "A larger message mapping.",
	},
	{
		Name: "spend-step-0-flow", Repo: "btp-spend-analysis",
		Commit: "fabb838cc68009e23aabe04efadb214c97492f4d",
		Path:   "src/version-two/spend-analysis/AribaSpend_Step_0.zip",
		SHA256: "dc92bb6a0309f2dfddc88d794538f0943fc9249e565ce62c476c5881501dff6d",
		Kind:   KindIntegrationFlow, BundleID: "AribaSpend_Step_0",
		Notes: "Integration flow with a timer start event: design time only, never deploy.",
	},
	{
		Name: "codejam-package-export", Repo: "connecting-systems-services-integration-suite-codejam",
		Commit: "b4fbfeb2d0c1fc6e6e94874e88e3dd2ece513f00",
		Path:   "assets/cloud-integration/Connecting Systems CodeJam - Export.zip",
		SHA256: "46a727cafe0c4fb0fe4864037289be0a2f62cdf5fd588b0c42f007730568d0e8",
		Kind:   KindPackageExport,
		Notes: "Seven integration flows; several wrap Bundle-SymbolicName over two manifest lines " +
			"and carry externalized parameters.",
	},
	{
		Name: "eventhub-package-export", Repo: "event-driven-integrations-e-bite",
		Commit: "77c3c5535918445425c1ec4b50e97394519ba8b6",
		Path:   "03-event-hub/cloud-integration/EventHub_Sample.zip",
		SHA256: "5e6f196e62a92c9ceb8854b635ddc1dfcb121fda527dd231a8a610348367191f",
		Kind:   KindPackageExport, Deployable: true,
		Notes: "One integration flow with only an HTTPS sender and no receiver: safe to deploy.",
	},
	{
		Name: "codejam-api-proxy", Repo: "connecting-systems-services-integration-suite-codejam",
		Commit: "b4fbfeb2d0c1fc6e6e94874e88e3dd2ece513f00",
		Path:   "exercises/06-expose-integration-flow-api-management/assets/api-management/Request_Employee_Dependants_v1_BeforePolicies.zip",
		SHA256: "15a2f4469fecf3c3a71adcd34556234a0c7c5dc208eba9b047e02c1b82e842b9",
		Kind:   KindAPIProxy,
		Notes:  "API proxy bundle for a future API proxy resource.",
	},
	{
		Name: "policy-template-performance", Repo: "integration-suite-learning-journey",
		Commit: "fe681076310edb62a2f62e5f7a8d7288c0245fb7",
		Path:   "src/rev_21/Performance_Traceability.zip",
		SHA256: "06a5983d05f43acda73e2b06becb02e0ae6dc8b0b47f34c6a450fce95b675e48",
		Kind:   KindPolicyTemplate,
		Notes:  "API Management policy template.",
	},
}

// Lookup returns the catalog entry with the given name.
func Lookup(name string) (Sample, bool) {
	for _, s := range Catalog {
		if s.Name == name {
			return s, true
		}
	}
	return Sample{}, false
}
