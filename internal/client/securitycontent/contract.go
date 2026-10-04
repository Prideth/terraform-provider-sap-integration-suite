package securitycontent

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is everything this client relies on in the Security Content part
// of the Cloud Integration OData service (<host>/api/v1). See
// cloudintegration.Contract for how it is used.
var Contract = apimeta.Contract{
	Service: "cloud-integration",
	Package: "securitycontent",
	Reads: []apimeta.StructUse{
		{EntitySet: oauth2ClientCredentialsEntitySet, Value: OAuth2ClientCredential{}},
		{EntitySet: oauth2ClientCredentialsEntitySet, Value: oauth2ClientCredentialWriteRequest{}},
		{EntitySet: userCredentialsEntitySet, Value: UserCredential{}},
		{EntitySet: userCredentialsEntitySet, Value: userCredentialWriteRequest{}},
		{EntitySet: secureParametersEntitySet, Value: SecureParameter{}},
		{EntitySet: secureParametersEntitySet, Value: secureParameterWriteRequest{}},
		{EntitySet: keystoreEntriesEntitySet, Value: KeystoreEntry{}},
		{EntitySet: keyPairGenerationRequestsEntitySet, Value: keyPairGenerationWireRequest{}},
		{EntitySet: keystoreResourcesEntitySet, Value: deleteKeystoreEntriesRequest{}},
	},
	Navigations: []apimeta.NavigationUse{
		{EntitySet: keystoreEntriesEntitySet, Property: chainResourceNavigation},
		{EntitySet: keystoreEntriesEntitySet, Property: signingRequestNavigation},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(keystoreEntriesEntitySet, "Hexalias", "Edm.String"),
		apimeta.Key(certificateResourcesEntitySet, "Hexalias", "Edm.String"),
		apimeta.Key(certificateChainResourcesEntitySet, "Hexalias", "Edm.String"),
		apimeta.Key(oauth2ClientCredentialsEntitySet, "Name", "Edm.String"),
		apimeta.Key(userCredentialsEntitySet, "Name", "Edm.String"),
		apimeta.Key(secureParametersEntitySet, "Name", "Edm.String"),
		apimeta.Key(keystoreResourcesEntitySet, "Name", "Edm.String"),
	},
	MaxLengths: []apimeta.MaxLengthUse{
		{EntitySet: secureParametersEntitySet, Property: "Name", Value: MaxSecureParameterNameLength},
		{EntitySet: secureParametersEntitySet, Property: "SecureParam", Value: MaxSecureParameterValueLength},
	},
}
