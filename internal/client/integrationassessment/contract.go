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
		{EntitySet: domainEntitySet, Value: Domain{}},
		{EntitySet: styleEntitySet, Value: Style{}},
		{EntitySet: keyCharacteristicEntitySet, Value: KeyCharacteristic{}},
		{EntitySet: keyCharacteristicValueEntitySet, Value: KeyCharacteristicValue{}},
		{EntitySet: recommendationDegreeEntitySet, Value: RecommendationDegree{}},
		{EntitySet: technologyDomainEntitySet, Value: TechnologyDomain{}},
		{EntitySet: technologyStyleEntitySet, Value: TechnologyStyle{}},
		{EntitySet: technologyKeyCharacteristicEntitySet, Value: TechnologyKeyCharacteristic{}},
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
		{EntitySet: technologyInstanceEntitySet, Value: technologyInstanceUpdate{}},
		{EntitySet: technologyDomainEntitySet, Value: technologyDomainCreate{}},
		{EntitySet: technologyStyleEntitySet, Value: technologyStyleCreate{}},
		{EntitySet: technologyKeyCharacteristicEntitySet, Value: technologyKeyCharacteristicCreate{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(vendorEntitySet, "Id", "Edm.String"),
		apimeta.Key(applicationEntitySet, "Id", "Edm.String"),
		apimeta.Key(applicationInstanceEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyInstanceEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyDomainEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyStyleEntitySet, "Id", "Edm.String"),
		apimeta.Key(technologyKeyCharacteristicEntitySet, "Id", "Edm.String"),
	},
	Navigations: []apimeta.NavigationUse{
		{EntitySet: applicationEntitySet, Property: "Vendor"},
		{EntitySet: applicationInstanceEntitySet, Property: "Application"},
		{EntitySet: applicationInstanceEntitySet, Property: "DeploymentModel"},
		{EntitySet: technologyEntitySet, Property: "Vendor"},
		{EntitySet: technologyInstanceEntitySet, Property: "Technology"},
		{EntitySet: technologyInstanceEntitySet, Property: "DeploymentModel"},
		{EntitySet: keyCharacteristicValueEntitySet, Property: "KeyCharacteristic"},
		{EntitySet: technologyDomainEntitySet, Property: "Technology"},
		{EntitySet: technologyDomainEntitySet, Property: "Domain"},
		{EntitySet: technologyStyleEntitySet, Property: "Technology"},
		{EntitySet: technologyStyleEntitySet, Property: "Style"},
		{EntitySet: technologyKeyCharacteristicEntitySet, Property: "Technology"},
		{EntitySet: technologyKeyCharacteristicEntitySet, Property: "KeyCharacteristicValue"},
		{EntitySet: technologyKeyCharacteristicEntitySet, Property: "RecommendationDegree"},
	},
}
