package apicomposition

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is what this client relies on in the Configuration API's
// $metadata, which a key user's token read from a tenant on 2026-10-03
// (snapshot testdata/api-metadata/api-composition-configuration.json).
var Contract = apimeta.Contract{
	Service: "api-composition-configuration",
	Package: "apicomposition",
	Reads: []apimeta.StructUse{
		{EntitySet: graphConfigurationResource, Value: GraphConfiguration{}},
	},
	Writes: []apimeta.StructUse{
		{EntitySet: graphConfigurationResource, Value: GraphConfigurationInput{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(graphConfigurationResource, "businessDataGraphIdentifier", "Edm.String"),
	},
}
