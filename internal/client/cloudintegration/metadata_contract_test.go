package cloudintegration

import (
	"testing"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/testutil/edmx"
)

// TestWireContractAgainstMetadata checks every wire struct, entity set, key
// and function import this package uses against a tenant $metadata document.
// It skips when no document is available (see internal/testutil/edmx).
func TestWireContractAgainstMetadata(t *testing.T) {
	m := edmx.Load(t)

	structs := []struct {
		entitySet string
		value     any
	}{
		{accessPoliciesEntitySet, AccessPolicy{}},
		{accessPoliciesEntitySet, accessPolicyLink{}},
		{artifactReferencesEntitySet, AccessPolicyReference{}},
		{accessPolicyRuntimeAssignmentsEntitySet, AccessPolicyRuntimeAssignment{}},
		{customTagConfigurationsEntitySet, customTagConfigurationWriteRequest{}},
		{integrationAdapterDesigntimeArtifactsEntitySet, IntegrationAdapter{}},
		{integrationDesigntimeArtifactsEntitySet, IntegrationFlow{}},
		{messageMappingDesigntimeArtifactsEntitySet, MessageMapping{}},
		{numberRangesEntitySet, numberRangeWireModel{}},
		{numberRangesEntitySet, NumberRangeState{}},
		{integrationPackagesEntitySet, Package{}},
		{integrationPackagesEntitySet, packageWriteRequest{}},
		{integrationPackagesEntitySet, packageUpdateRequest{}},
		{runtimeArtifactsEntitySet, RuntimeArtifact{}},
		{buildAndDeployStatusEntitySet, BuildAndDeployStatus{}},
		{scriptCollectionDesigntimeArtifactsEntitySet, ScriptCollection{}},
		{serviceEndpointsEntitySet, ServiceEndpoint{}},
		{"EntryPoints", EntryPoint{}},
		{"APIDefinitions", APIDefinition{}},
		{integrationFlowConfigurationsEntitySet, IntegrationFlowConfiguration{}},
		{integrationFlowConfigurationsEntitySet, integrationFlowConfigurationWrite{}},
		{valueMappingDesigntimeArtifactsEntitySet, ValueMapping{}},
	}
	for _, s := range structs {
		m.AssertStruct(t, s.entitySet, s.value)
	}
	// Create body that links the policy (deep link); never decoded.
	m.AssertWriteStruct(t, artifactReferencesEntitySet, accessPolicyReferenceCreate{})

	m.AssertKey(t, accessPoliciesEntitySet, "Id", "Edm.Int64")
	m.AssertKey(t, artifactReferencesEntitySet, "Id", "Edm.Int64")
	m.AssertKey(t, accessPolicyRuntimeAssignmentsEntitySet, "Id", "Edm.Int64")
	if et := m.EntityTypeOf(t, accessPoliciesEntitySet); et != nil && !et.Properties[accessPolicyRuntimeAssignmentsEntitySet].Navigation {
		t.Errorf("AccessPolicy has no %s navigation property", accessPolicyRuntimeAssignmentsEntitySet)
	}
	m.AssertKey(t, integrationPackagesEntitySet, "Id", "Edm.String")
	m.AssertKey(t, integrationDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String")
	m.AssertKey(t, messageMappingDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String")
	m.AssertKey(t, scriptCollectionDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String")
	m.AssertKey(t, valueMappingDesigntimeArtifactsEntitySet, "Id", "Edm.String", "Version", "Edm.String")
	m.AssertKey(t, integrationAdapterDesigntimeArtifactsEntitySet, "Id", "Edm.String")
	m.AssertKey(t, runtimeArtifactsEntitySet, "Id", "Edm.String")
	m.AssertKey(t, buildAndDeployStatusEntitySet, "TaskId", "Edm.String")
	m.AssertFunctionImport(t, "DeployIntegrationDesigntimeArtifact", "POST", "Id", "Version")
	m.AssertKey(t, numberRangesEntitySet, "Name", "Edm.String")
	m.AssertKey(t, customTagConfigurationsEntitySet, "Id", "Edm.String")
	m.AssertKey(t, integrationFlowConfigurationsEntitySet, "ParameterKey", "Edm.String")
	if et := m.EntityTypeOf(t, integrationDesigntimeArtifactsEntitySet); et != nil && !et.Properties["Configurations"].Navigation {
		t.Error("IntegrationDesigntimeArtifact has no Configurations navigation property")
	}

	m.AssertFunctionImport(t, "DeployIntegrationDesigntimeArtifact", "POST", "Id", "Version")
	m.AssertFunctionImport(t, "DeployMessageMappingDesigntimeArtifact", "POST", "Id", "Version")
	m.AssertFunctionImport(t, "DeployScriptCollectionDesigntimeArtifact", "POST", "Id", "Version")
	m.AssertFunctionImport(t, "DeployValueMappingDesigntimeArtifact", "POST", "Id", "Version")
	m.AssertFunctionImport(t, "DeployIntegrationAdapterDesigntimeArtifact", "POST", "Id")
	for _, fi := range []string{"IntegrationDesigntimeArtifactSaveAsVersion", "MessageMappingDesigntimeArtifactSaveAsVersion", "ScriptCollectionDesigntimeArtifactSaveAsVersion"} {
		m.AssertFunctionImport(t, fi, "POST", "Id", "SaveAsVersion")
	}
}
