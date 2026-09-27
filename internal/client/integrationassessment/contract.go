package integrationassessment

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is everything this client relies on in the Integration
// Assessment Entities API. The contract test checks it against the
// committed $metadata snapshot; the metadata acceptance test checks it
// against a live service.
var Contract = apimeta.Contract{
	Service: "integration-assessment-entities",
	Package: "integrationassessment",
	Reads: []apimeta.StructUse{
		{EntitySet: vendorEntitySet, Value: Vendor{}},
		{EntitySet: applicationEntitySet, Value: Application{}},
		{EntitySet: applicationInstanceEntitySet, Value: ApplicationInstance{}},
		{EntitySet: technologyEntitySet, Value: Technology{}},
		{EntitySet: technologyInstanceEntitySet, Value: TechnologyInstance{}},
		{EntitySet: deploymentModelEntitySet, Value: DeploymentModel{}},
	},
	Writes: []apimeta.StructUse{
		{EntitySet: vendorEntitySet, Value: vendorWrite{}},
		{EntitySet: applicationEntitySet, Value: applicationCreate{}},
		{EntitySet: applicationEntitySet, Value: applicationUpdate{}},
		{EntitySet: applicationInstanceEntitySet, Value: applicationInstanceCreate{}},
		{EntitySet: applicationInstanceEntitySet, Value: applicationInstanceUpdate{}},
		{EntitySet: technologyEntitySet, Value: technologyCreate{}},
		{EntitySet: technologyEntitySet, Value: technologyUpdate{}},
		{EntitySet: technologyInstanceEntitySet, Value: technologyInstanceCreate{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(vendorEntitySet, "Id", "Edm.String"),
		apimeta.Key(applicationEntitySet, "Id", "Edm.String"),
		apimeta.Key(applicationInstanceEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyInstanceEntitySet, "Id", "Edm.String"),
	},
	Navigations: []apimeta.NavigationUse{
		{EntitySet: applicationEntitySet, Property: "Vendor"},
		{EntitySet: applicationInstanceEntitySet, Property: "Application"},
		{EntitySet: applicationInstanceEntitySet, Property: "DeploymentModel"},
		{EntitySet: technologyEntitySet, Property: "Vendor"},
		{EntitySet: technologyInstanceEntitySet, Property: "Technology"},
		{EntitySet: technologyInstanceEntitySet, Property: "DeploymentModel"},
	},
}
