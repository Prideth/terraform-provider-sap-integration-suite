package partnerdirectory

import "github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"

// Contract is everything this client relies on in the Partner Directory
// part of the Cloud Integration OData service (<host>/api/v1). See
// cloudintegration.Contract for how it is used.
var Contract = apimeta.Contract{
	Service: "cloud-integration",
	Package: "partnerdirectory",
	Reads: []apimeta.StructUse{
		{EntitySet: alternativePartnersEntitySet, Value: AlternativePartner{}},
		{EntitySet: authorizedUsersEntitySet, Value: AuthorizedUser{}},
		{EntitySet: binaryParametersEntitySet, Value: BinaryParameter{}},
		{EntitySet: partnersEntitySet, Value: Partner{}},
		{EntitySet: stringParametersEntitySet, Value: StringParameter{}},
		{EntitySet: userCredentialParametersEntitySet, Value: UserCredentialParameter{}},
		{EntitySet: userCredentialParametersEntitySet, Value: createUserCredentialParameterRequest{}},
	},
	Keys: []apimeta.KeyUse{
		apimeta.Key(partnersEntitySet, "Pid", "Edm.String"),
		apimeta.Key(stringParametersEntitySet, "Pid", "Edm.String", "Id", "Edm.String"),
		apimeta.Key(binaryParametersEntitySet, "Pid", "Edm.String", "Id", "Edm.String"),
		apimeta.Key(userCredentialParametersEntitySet, "Pid", "Edm.String", "Id", "Edm.String"),
		apimeta.Key(alternativePartnersEntitySet, "Hexagency", "Edm.String", "Hexscheme", "Edm.String", "Hexid", "Edm.String"),
		apimeta.Key(authorizedUsersEntitySet, "User", "Edm.String"),
	},
	// The binary parameter size check follows the service's own MaxLength,
	// not the older 260 KB figure some SAP pages still give.
	MaxLengths: []apimeta.MaxLengthUse{
		{EntitySet: binaryParametersEntitySet, Property: "Value", Value: MaxBinaryParameterValueBytes},
	},
}
