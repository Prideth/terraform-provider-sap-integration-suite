package apidiscovery

import (
	"sort"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

// Status says what the provider makes of one entity set or operation of a
// service.
type Status string

const (
	// StatusUsed: a client contract reads, writes or calls it. Derived
	// from the contracts, never written by hand.
	StatusUsed Status = "used"
	// StatusCandidate: it could back a resource or data source; Feature
	// names the catalog entry that tracks it.
	StatusCandidate Status = "candidate"
	// StatusExcluded: deliberately not modelled; Reason says why.
	StatusExcluded Status = "excluded"
	// StatusUnclassified: nobody has looked at it yet. The classification
	// test fails on this status.
	StatusUnclassified Status = "unclassified"
)

// rule classifies an entity set or operation by exact name or by prefix.
type rule struct {
	Name    string
	Prefix  string
	Status  Status
	Feature string // catalog key, for candidates
	Reason  string // for exclusions; also a note for candidates
}

func (r rule) matches(name string) bool {
	if r.Prefix != "" {
		return strings.HasPrefix(name, r.Prefix)
	}
	return name == r.Name
}

func candidate(name, feature, note string) rule {
	return rule{Name: name, Status: StatusCandidate, Feature: feature, Reason: note}
}

func excluded(name, reason string) rule {
	return rule{Name: name, Status: StatusExcluded, Reason: reason}
}

// excludedFor is an exclusion the catalog records under a feature key, so
// the report can link the entity to the decision users see.
func excludedFor(name, feature, reason string) rule {
	return rule{Name: name, Status: StatusExcluded, Feature: feature, Reason: reason}
}

func excludedPrefix(prefix, feature, reason string) rule {
	return rule{Prefix: prefix, Status: StatusExcluded, Feature: feature, Reason: reason}
}

// each applies one constructor to several names.
func each(names []string, f func(string) rule) []rule {
	out := make([]rule, 0, len(names))
	for _, n := range names {
		out = append(out, f(n))
	}
	return out
}

func concat(groups ...[]rule) []rule {
	var out []rule
	for _, g := range groups {
		out = append(out, g...)
	}
	return out
}

// Row is the classification of one entity set or operation.
type Row struct {
	Kind     string // "entity set" or "operation"
	Name     string
	Type     string // entity type, or the operation's HTTP method
	Status   Status
	Feature  string
	Reason   string
	Packages []string // client packages using it, for StatusUsed
}

// Classify returns a row for every entity set and import operation of a
// service contract: used by the provider (from the client contracts), or
// classified by the hand-written table below. Exact names win over
// prefixes.
func Classify(s *apimeta.Service) []Row {
	usage := UsageOf(s.ID)
	rules := classification[s.ID]
	lookup := func(name string) (rule, bool) {
		for _, r := range rules {
			if r.Prefix == "" && r.matches(name) {
				return r, true
			}
		}
		for _, r := range rules {
			if r.Prefix != "" && r.matches(name) {
				return r, true
			}
		}
		return rule{}, false
	}
	classify := func(kind, name, typ string, pkgs []string) Row {
		row := Row{Kind: kind, Name: name, Type: typ}
		if len(pkgs) > 0 {
			row.Status, row.Packages = StatusUsed, append([]string(nil), pkgs...)
			sort.Strings(row.Packages)
			return row
		}
		if r, ok := lookup(name); ok {
			row.Status, row.Feature, row.Reason = r.Status, r.Feature, r.Reason
			return row
		}
		row.Status = StatusUnclassified
		return row
	}

	var rows []Row
	for _, es := range s.EntitySets {
		rows = append(rows, classify("entity set", es.Name, es.EntityType, usage.EntitySets[es.Name]))
	}
	for _, sg := range s.Singletons {
		rows = append(rows, classify("singleton", sg.Name, sg.EntityType, usage.EntitySets[sg.Name]))
	}
	for _, op := range s.Operations {
		if op.Bound || op.Kind == apimeta.KindFunction || op.Kind == apimeta.KindAction {
			continue // reached through an entity or an import
		}
		rows = append(rows, classify("operation", op.Name, op.HTTPMethod, usage.Operations[op.Name]))
	}
	return rows
}

// classification holds, per service, every entity set and operation the
// provider does not use, with what it is: a candidate for a resource or data
// source (and the catalog entry that tracks it), or deliberately excluded.
// Exclusion reasons follow the provider's scope rule: Terraform manages
// desired configuration, not runtime data, monitoring, or workflows.
var classification = map[string][]rule{
	"cloud-integration":      cloudIntegrationClassification,
	"classic-api-management": classicAPIManagementClassification,
}

const (
	reasonMessageMonitoring = "Runtime data of message processing (logs, attachments, run steps); monitoring, not configuration."
	reasonMessageStores     = "Runtime messages and queues filled by deployed integration flows; operations on them are support actions, not desired state."
	reasonRuntimeRecords    = "Runtime records written by integration flows (ID mapping, idempotency); not configuration."
	reasonB2BMonitoring     = "Trading Partner Management interchange monitoring and reprocessing; runtime data and support actions."
	reasonTrace             = "Trace data of integration flows running with log level Trace; diagnostics."
	reasonLogs              = "Tenant log files and audit logs; read-only diagnostics."
	reasonLocks             = "Edit and runtime locks; releasing one is a support action, not configuration."
	reasonGuidelines        = "Design guideline and validation checks run on demand; their results are reports. A data source could expose them later."
	reasonRuntimeNodes      = "Runtime node and profile information of the tenant; read-only system information."
	reasonTopology          = "Connection topology derived from deployed content; read-only."
	reasonRuntimeSync       = "Distribution status of keystores and keyrings to runtimes; monitoring, not configuration."
	reasonCSR               = "Certificate signing requests and signed-response uploads for key pairs; a certificate-authority workflow, not desired state."
	reasonOAuthCode         = "Authorization-code credentials get their refresh token from an interactive browser consent, which Terraform cannot complete; " +
		"the operations belong to that consent flow."
	reasonArchivingKPIs = "Statistics of the archiving runs; monitoring."
	reasonDesignVersion = "Versioning is covered by the content resources through their version attribute."
)

var cloudIntegrationClassification = concat(
	// Design-time content the provider does not manage yet.
	[]rule{
		candidate("DataTypeDesigntimeArtifacts", "cloud_integration.data_type", "Versioned design-time artifact with $value content, like message mappings."),
		candidate("DataTypeDesigntimeArtifactSaveAsVersion", "cloud_integration.data_type", ""),
		candidate("MessageTypeDesigntimeArtifacts", "cloud_integration.message_type", "Versioned design-time artifact with $value content."),
		candidate("MessageTypeDesigntimeArtifactSaveAsVersion", "cloud_integration.message_type", ""),
		candidate("FaultMessageTypeDesigntimeArtifacts", "cloud_integration.message_type", "Fault message types; the catalog tracks them with message types."),
		candidate("FaultMessageTypeDesigntimeArtifactSaveAsVersion", "cloud_integration.message_type", ""),
		candidate("ServiceInterfaceDesigntimeArtifacts", "cloud_integration.service_interface", "Versioned design-time artifact with $value content."),
		candidate("ServiceInterfaceDesigntimeArtifactSaveAsVersion", "cloud_integration.service_interface", ""),
		candidate("ValueMappingDesigntimeArtifactSaveAsVersion", "cloud_integration.design_time_versioning", "The value mapping resource does not save versions yet."),
		candidate("ValMapSchema", "cloud_integration.value_mapping_entry", "Agency/identifier pairs of a value mapping."),
		candidate("ValMaps", "cloud_integration.value_mapping_entry", "Value pairs of one agency/identifier pair."),
		candidate("DefaultValMaps", "cloud_integration.value_mapping_entry", "Default value of one agency/identifier pair."),
		candidate("UpsertValMaps", "cloud_integration.value_mapping_entry", ""),
		candidate("UpdateDefaultValMap", "cloud_integration.value_mapping_entry", ""),
		candidate("DeleteValMaps", "cloud_integration.value_mapping_entry", ""),
		candidate("CustomTags", "cloud_integration.custom_tag_configuration", "Custom tag values of an integration package (IntegrationPackages/CustomTags)."),
		excluded("Resources", "Single files inside design-time artifacts; the content resources manage the artifact as a whole ZIP, and editing files one by one would fight it."),
		excluded("CopyIntegrationPackage", "Copies an SAP-provided package from the Discover catalog; a one-off action. A resource for copied standard packages is possible later."),
		excluded("IntegrationDesigntimeLocks", reasonLocks),
		excluded("Locks", reasonLocks),
		excluded("DesignGuidelines", reasonGuidelines),
		excluded("DesignGuidelineExecutionResults", reasonGuidelines),
		excluded("ExecuteIntegrationDesigntimeArtifactsGuidelines", reasonGuidelines),
		excluded("ExecuteScriptCollectionDesigntimeArtifactsGuidelines", reasonGuidelines),
		excluded("ValidateIntegrationDesigntimeArtifact", reasonGuidelines),
		excluded("RuntimeArtifactErrorInformations", "Read through the ErrorInformation navigation of IntegrationRuntimeArtifacts, which the deployment resources use."),
	},
	// Security material.
	[]rule{
		candidate("CertificateChainResources", "security.certificate_chain", "Media entity for uploading a chain to a key pair (KeystoreEntries/ChainResource)."),
		candidate("ChainCertificates", "security.certificate_chain", "Certificates of a key pair's chain (KeystoreEntries/ChainCertificates)."),
		candidate("KeyPairResources", "security.key_pair", "Upload of an externally created key pair (PKCS#12 with password); private material would be write-only."),
		candidate("RSAKeyGenerationRequests", "security.key_pair", "Key pair generation from an RSA file."),
		candidate("CustomParameters", "security.oauth2_client_credential", "Custom token request parameters of an OAuth2 client credential."),
		excludedFor("SSHKeyGenerationRequests", "security.ssh_key", "Generates an SSH key; the catalog records SSH keys as out of scope."),
		excludedFor("SSHKeyResources", "security.ssh_key", "Upload of an SSH key; the catalog records SSH keys as out of scope."),
		excluded("CertificateSigningRequests", reasonCSR),
		excluded("HistoryKeystoreEntries", "Earlier versions of keystore entries the tenant keeps for rollback; restoring one is a support action."),
		excluded("Keystores", "Keystore list with its distribution status; the keystore itself is not configurable, only its entries."),
		excluded("KeyringRuntimeAssignment", reasonRuntimeSync),
		excluded("RuntimeSyncInfos", reasonRuntimeSync),
		excluded("SecurityArtifacts", "Name-only listing of security material; nothing to configure beyond the typed entity sets."),
		excluded("OAuth2AuthorizationCodes", reasonOAuthCode),
		excluded("OAuth2AuthorizationCodeCopy", reasonOAuthCode),
		excluded("OAuth2AuthorizationCodeFullAuthUrl", reasonOAuthCode),
		excluded("OAuth2AuthorizationCodeRefreshTokenUpdate", reasonOAuthCode),
		excluded("OAuthTokenFromCode", reasonOAuthCode),
	},
	// PGP keyrings, keys, sub keys and user IDs; secret keyrings carry
	// private material, which would be write-only.
	each([]string{
		"PgpKeyrings", "PgpPublicKeyrings", "PgpSecretKeyrings", "PgpKeyringPublicResources",
		"PgpKeyringSecretResources", "PgpKeyEntries", "PgpKeyEntryImportResults", "PgpKeyPublicResources",
		"PgpKeySecretResources", "PgpSubKeys", "PgpUserIds",
	}, func(n string) rule { return candidate(n, "security.pgp_keyring", "") }),
	// Tenant settings.
	[]rule{
		candidate("ArchivingConfigurations", "cloud_integration.archiving", "Tenant-wide archiving settings; a singleton needing a destructive gate in tests."),
		candidate("activateArchivingConfiguration", "cloud_integration.archiving", ""),
		candidate("B2BArchivingConfigurations", "cloud_integration.archiving", "Archiving of B2B interchange payloads."),
		candidate("activateB2BArchivingConfiguration", "cloud_integration.archiving", ""),
		excluded("ArchivingKeyPerformanceIndicators", reasonArchivingKPIs),
		excluded("B2BArchivingKeyPerformanceIndicators", reasonArchivingKPIs),
		excluded("ExternalLoggingActivationStatus", "Tenant-wide switch for external logging; owning it in Terraform would change every flow's logging, and the catalog has no entry for it yet."),
		excluded("ExternalLoggingEvents", reasonLogs),
		excluded("activateExternalLogging", "Tenant-wide switch for external logging (see ExternalLoggingActivationStatus)."),
		excluded("deactivateExternalLogging", "Tenant-wide switch for external logging (see ExternalLoggingActivationStatus)."),
		excluded("ExtendedFieldsConfigs", "Search field layout of business document monitoring; monitoring configuration without a catalog entry."),
		excluded("CustomObjects", "Search field values of business document monitoring; runtime data."),
	},
	// Runtime data and monitoring.
	[]rule{
		excludedPrefix("MessageProcessingLog", "cloud_integration.message_processing_logs", reasonMessageMonitoring),
		excluded("CancelMessageProcessingLog", reasonMessageMonitoring),
		excludedPrefix("MessageStoreEntr", "cloud_integration.message_stores", reasonMessageStores),
		excludedPrefix("Jms", "cloud_integration.message_stores", reasonMessageStores),
		excludedPrefix("Messaging", "cloud_integration.message_stores", reasonMessageStores),
		excludedFor("Queues", "cloud_integration.message_stores", reasonMessageStores),
		excludedFor("QueueStates", "cloud_integration.message_stores", reasonMessageStores),
		excluded("MoveMessagingMessages", reasonMessageStores),
		excluded("RetryMessagingMessages", reasonMessageStores),
		excluded("activateQueue", reasonMessageStores),
		excluded("deactivateQueue", reasonMessageStores),
		excludedFor("DataStores", "cloud_integration.data_store", "Created implicitly by a flow's Data Store Write step; runtime data."),
		excludedFor("DataStoreEntries", "cloud_integration.data_store_entry", "Runtime messages written by flows; runtime data."),
		excludedFor("XiDataStores", "cloud_integration.data_store", "Temporary storage of the XI adapter; runtime data."),
		excludedFor("XiDataStoreArtifacts", "cloud_integration.data_store", "Temporary storage of the XI adapter; runtime data."),
		excludedFor("Variables", "cloud_integration.variable", "Written by a flow's Write Variables step; runtime data."),
		excludedPrefix("IdMap", "", reasonRuntimeRecords),
		excluded("IdempotentRepositoryEntries", reasonRuntimeRecords),
		excluded("GenericIdempotentRepositoryEntries", reasonRuntimeRecords),
		excluded("MDIDeltaToken", "Delta token of the SAP Master Data Integration adapter; runtime state."),
		excludedPrefix("BusinessDocument", "", reasonB2BMonitoring),
		excluded("FunctionalAcknowledgements", reasonB2BMonitoring),
		excluded("TechnicalAcknowledgements", reasonB2BMonitoring),
		excluded("OrphanedInterchanges", reasonB2BMonitoring),
		excluded("ErrorDetails", reasonB2BMonitoring),
		excluded("CommunicationProtocolHeaders", reasonB2BMonitoring),
		excluded("massInterchangeProcess", reasonB2BMonitoring),
		excluded("singleInterchangeProcess", reasonB2BMonitoring),
		excludedPrefix("Trace", "", reasonTrace),
		excluded("LogFiles", reasonLogs),
		excluded("LogFileArchives", reasonLogs),
		excluded("AuditLogs", reasonLogs),
		excluded("NodeProfiles", reasonRuntimeNodes),
		excluded("Roles", reasonRuntimeNodes),
		excluded("WNNodes", reasonRuntimeNodes),
		excluded("IntegrationConnections", reasonTopology),
		excluded("IntegrationFlows", reasonTopology),
	},
)

const (
	reasonAPIMRevision  = "Revision history of API proxies; the current state is what a resource manages."
	reasonAPIProxyPart  = "Part of an API proxy (endpoints, flows, steps, resources); managed with the proxy as a whole, not one by one."
	reasonAPIMInternals = "Internal bookkeeping of the API portal (tenant cloning, content package import); not configuration."
)

var classicAPIManagementClassification = concat(
	[]rule{
		candidate("APIProxyDeployments", "api_management.classic.api_proxy_deployment", ""),
		candidate("Policies", "api_management.classic.policy", "Policies are also part of the proxy bundle."),
		candidate("VirtualHosts", "api_management.classic.virtual_host", ""),
		candidate("CertificateStores", "api_management.classic.certificate_store", ""),
		candidate("Certificates", "api_management.classic.certificate_store", ""),
		candidate("CacheResources", "api_management.classic.cache_resource", ""),
		candidate("KeyMapEntries", "api_management.classic.environment_key_value_map", ""),
		candidate("KeyMapEntryValues", "api_management.classic.environment_key_value_map", ""),
		candidate("Applications", "api_management.classic.application", ""),
		candidate("ApplicationAdditionalPropertys", "api_management.classic.application", ""),
		candidate("Developers", "api_management.classic.application", ""),
		candidate("Agents", "api_management.classic.application", "Application-shaped (app key and secret, products); its role is not documented yet."),
		candidate("AgentAdditionalPropertys", "api_management.classic.application", ""),
		candidate("RatePlans", "api_management.classic.rate_plan", ""),
		candidate("ACLProductLinkages", "api_management.classic.product_access_control", ""),
		candidate("APIProviderAdditionalPropertys", "api_management.classic.api_provider", "Custom attributes of an API provider."),
		candidate("PolicyTemplateContainers", "api_management.classic.policy_template", ""),
		candidate("TemplatePolicys", "api_management.classic.policy_template", ""),
		candidate("TemplateFileResources", "api_management.classic.policy_template", ""),
		excluded("Bills", "Monetization bills generated from usage; runtime data."),
		excluded("DestinationAndUrlMappings", "Mapping of BTP destinations to target URLs maintained by the portal; not documented as configuration."),
		excluded("ApiportalCloneMappings", reasonAPIMInternals),
		excluded("ContentPackageMappers", reasonAPIMInternals),
		excluded("GetAllRevisions", reasonAPIMRevision),
	},
	each([]string{
		"APIProxyEndPoints", "APITargetEndPoints", "APIResources", "ConditionalFlowRules", "DefaultFaultRules",
		"FaultRules", "FlowRules", "RouteRules", "Steps", "Streams", "EndPointProperties", "Resources",
		"FileResources", "Documentations",
	}, func(n string) rule { return excludedFor(n, "api_management.classic.api_proxy", reasonAPIProxyPart) }),
	each([]string{
		"APIProxyRevisions", "APIProxyEndPointRevisions", "APITargetEndPointRevisions", "APIResourceRevisions", "AttachmentRevisions",
		"ConditionalFlowRuleRevisions", "DefaultFaultRuleRevisions", "DocumentationRevisions", "EndPointPropertyRevisions",
		"FaultRuleRevisions", "FileResourceRevisions", "FlowRuleRevisions", "PolicyRevisions", "ResourceRevisions",
		"RouteRuleRevisions", "StepRevisions", "StreamRevisions",
	}, func(n string) rule { return excluded(n, reasonAPIMRevision) }),
)
