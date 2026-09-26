package apimanagementclassic

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is everything this client relies on in the API portal's
// Management.svc OData service. See cloudintegration.Contract for how it is
// used.
var Contract = apimeta.Contract{
	Service: "classic-api-management",
	Package: "apimanagementclassic",
	Reads: []apimeta.StructUse{
		{EntitySet: apiProvidersEntitySet, Value: APIProvider{}},
		{EntitySet: apiProxiesEntitySet, Value: APIProxy{}},
		{EntitySet: apiProductsEntitySet, Value: APIProduct{}},
		{EntitySet: apiProductsEntitySet, Value: apiProductReadWire{}},
		{EntitySet: apiProductAdditionalPropertiesEntity, Value: APIProductAdditionalProperty{}},
		{EntitySet: certificateStoreReferencesEntitySet, Value: CertificateStoreReference{}},
		{EntitySet: genericKeyMapEntriesEntitySet, Value: KeyValueMap{}},
		{EntitySet: genericKeyMapEntriesEntitySet, Value: keyValueMapReadWire{}},
		{EntitySet: "GenericKeyMapEntryValues", Value: keyValueMapEntryValueWire{}},
	},
	// Create bodies with deep inserts; never decoded.
	Writes: []apimeta.StructUse{
		{EntitySet: apiProductsEntitySet, Value: apiProductWire{}},
		{EntitySet: genericKeyMapEntriesEntitySet, Value: keyValueMapWire{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(apiProvidersEntitySet, "name", "Edm.String"),
		apimeta.Key(apiProxiesEntitySet, "name", "Edm.String"),
		apimeta.Key(apiProductsEntitySet, "name", "Edm.String"),
		apimeta.Key(apiProductAdditionalPropertiesEntity, "entityId", "Edm.String", "name", "Edm.String"),
		apimeta.Key(certificateStoreReferencesEntitySet, "name", "Edm.String"),
		apimeta.Key(genericKeyMapEntriesEntitySet, "name", "Edm.String", "scope", "Edm.String", "scopeId", "Edm.String"),
	},
	Navigations: []apimeta.NavigationUse{
		{EntitySet: apiProductsEntitySet, Property: "apiProxies"},
		{EntitySet: apiProductsEntitySet, Property: "additionalProperties"},
		{EntitySet: genericKeyMapEntriesEntitySet, Property: "genericKeyMapEntryValues"},
	},
}
