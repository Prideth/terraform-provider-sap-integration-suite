package cloudintegration

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is everything this client relies on in the Cloud Integration
// OData service (<host>/api/v1): the wire structs it decodes or sends, the
// keys it addresses entities with, the navigation properties it follows and
// the function imports it calls. The contract test checks it against the
// committed snapshot of the service's $metadata, the metadata acceptance
// test against a live tenant, and the discovery report uses it to tell
// which entity sets the provider uses.
var Contract = apimeta.Contract{
	Service: "cloud-integration",
	Package: "cloudintegration",
	Reads: []apimeta.StructUse{
		{EntitySet: accessPoliciesEntitySet, Value: AccessPolicy{}},
		{EntitySet: accessPoliciesEntitySet, Value: accessPolicyLink{}},
		{EntitySet: artifactReferencesEntitySet, Value: AccessPolicyReference{}},
		{EntitySet: accessPolicyRuntimeAssignmentsEntitySet, Value: AccessPolicyRuntimeAssignment{}},
		{EntitySet: customTagConfigurationsEntitySet, Value: customTagConfigurationWriteRequest{}},
		{EntitySet: integrationAdapterDesigntimeArtifactsEntitySet, Value: IntegrationAdapter{}},
		{EntitySet: integrationDesigntimeArtifactsEntitySet, Value: IntegrationFlow{}},
		{EntitySet: messageMappingDesigntimeArtifactsEntitySet, Value: MessageMapping{}},
		{EntitySet: dataTypeDesigntimeArtifactsEntitySet, Value: DataType{}},
		{EntitySet: messageTypeDesigntimeArtifactsEntitySet, Value: MessageType{}},
		{EntitySet: faultMessageTypeDesigntimeArtifactsEntitySet, Value: MessageType{}},
		{EntitySet: serviceInterfaceDesigntimeArtifactsEntitySet, Value: ServiceInterface{}},
		{EntitySet: numberRangesEntitySet, Value: numberRangeWireModel{}},
		{EntitySet: numberRangesEntitySet, Value: NumberRangeState{}},
		{EntitySet: integrationPackagesEntitySet, Value: Package{}},
		{EntitySet: integrationPackagesEntitySet, Value: packageWriteRequest{}},
		{EntitySet: integrationPackagesEntitySet, Value: packageUpdateRequest{}},
		{EntitySet: runtimeArtifactsEntitySet, Value: RuntimeArtifact{}},
		{EntitySet: buildAndDeployStatusEntitySet, Value: BuildAndDeployStatus{}},
		{EntitySet: scriptCollectionDesigntimeArtifactsEntitySet, Value: ScriptCollection{}},
		{EntitySet: serviceEndpointsEntitySet, Value: ServiceEndpoint{}},
		{EntitySet: "EntryPoints", Value: EntryPoint{}},
		{EntitySet: "APIDefinitions", Value: APIDefinition{}},
		{EntitySet: integrationFlowConfigurationsEntitySet, Value: IntegrationFlowConfiguration{}},
		{EntitySet: integrationFlowConfigurationsEntitySet, Value: integrationFlowConfigurationWrite{}},
		{EntitySet: valueMappingDesigntimeArtifactsEntitySet, Value: ValueMapping{}},
	},
	Writes: []apimeta.StructUse{
		// Content update body, sent to each versioned design-time entity set.
		{EntitySet: integrationDesigntimeArtifactsEntitySet, Value: designtimeUpdateRequest{}},
		{EntitySet: messageMappingDesigntimeArtifactsEntitySet, Value: designtimeUpdateRequest{}},
		{EntitySet: dataTypeDesigntimeArtifactsEntitySet, Value: dataTypeCreate{}},
		{EntitySet: dataTypeDesigntimeArtifactsEntitySet, Value: dataTypeUpdate{}},
		{EntitySet: messageTypeDesigntimeArtifactsEntitySet, Value: messageTypeCreate{}},
		{EntitySet: messageTypeDesigntimeArtifactsEntitySet, Value: messageTypeUpdate{}},
		{EntitySet: faultMessageTypeDesigntimeArtifactsEntitySet, Value: messageTypeCreate{}},
		{EntitySet: faultMessageTypeDesigntimeArtifactsEntitySet, Value: messageTypeUpdate{}},
		{EntitySet: serviceInterfaceDesigntimeArtifactsEntitySet, Value: serviceInterfaceCreate{}},
		{EntitySet: scriptCollectionDesigntimeArtifactsEntitySet, Value: designtimeUpdateRequest{}},
		// Create body that links the policy (deep link); never decoded.
		{EntitySet: artifactReferencesEntitySet, Value: accessPolicyReferenceCreate{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(accessPoliciesEntitySet, "Id", "Edm.Int64"),
		apimeta.Key(artifactReferencesEntitySet, "Id", "Edm.Int64"),
		apimeta.Key(accessPolicyRuntimeAssignmentsEntitySet, "Id", "Edm.Int64"),
		apimeta.Key(integrationPackagesEntitySet, "Id", "Edm.String"),
		apimeta.Key(integrationDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(messageMappingDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(dataTypeDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(messageTypeDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(faultMessageTypeDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(serviceInterfaceDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(scriptCollectionDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(valueMappingDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String"),
		apimeta.Key(integrationAdapterDesigntimeArtifactsEntitySet, "Id", "Edm.String"),
		apimeta.Key(runtimeArtifactsEntitySet, "Id", "Edm.String"),
		apimeta.Key(buildAndDeployStatusEntitySet, "TaskId", "Edm.String"),
		apimeta.Key(numberRangesEntitySet, "Name", "Edm.String"),
		apimeta.Key(customTagConfigurationsEntitySet, "Id", "Edm.String"),
		apimeta.Key(integrationFlowConfigurationsEntitySet, "ParameterKey", "Edm.String"),
	},
	Navigations: []apimeta.NavigationUse{
		{EntitySet: accessPoliciesEntitySet, Property: accessPolicyRuntimeAssignmentsEntitySet},
		{EntitySet: integrationDesigntimeArtifactsEntitySet, Property: "Configurations"},
		{EntitySet: runtimeArtifactsEntitySet, Property: "ErrorInformation"},
	},
	Operations: []apimeta.OperationUse{
		{Name: "DeployIntegrationDesigntimeArtifact", HTTPMethod: "POST", Parameters: []string{"Id", "Version"}},
		{Name: "DeployMessageMappingDesigntimeArtifact", HTTPMethod: "POST", Parameters: []string{"Id", "Version"}},
		{Name: "DeployScriptCollectionDesigntimeArtifact", HTTPMethod: "POST", Parameters: []string{"Id", "Version"}},
		{Name: "DeployValueMappingDesigntimeArtifact", HTTPMethod: "POST", Parameters: []string{"Id", "Version"}},
		{Name: "DeployIntegrationAdapterDesigntimeArtifact", HTTPMethod: "POST", Parameters: []string{"Id"}},
		{Name: "IntegrationDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: "MessageMappingDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: "DataTypeDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: "MessageTypeDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: "FaultMessageTypeDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: serviceInterfaceSaveAsVersion, HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
		{Name: "ScriptCollectionDesigntimeArtifactSaveAsVersion", HTTPMethod: "POST", Parameters: []string{"Id", "SaveAsVersion"}},
	},
}
