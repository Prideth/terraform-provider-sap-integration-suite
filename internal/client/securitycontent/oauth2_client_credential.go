package securitycontent

import (
	"context"
	"encoding/json"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

const oauth2ClientCredentialsEntitySet = "OAuth2ClientCredentials" // #nosec G101 -- an OData entity set name, not a credential value

// OAuth2ClientCredential is the READ/IDENTITY-ONLY wire representation of an
// OAuth2ClientCredentials entity (Security Content API): deliberately, this
// struct has no ClientSecret field, for the same reason UserCredential has
// no Password field — see that type's doc comment.
//
// Property names follow the OAuth2ClientCredential entity type of the tenant
// $metadata. ClientAuthentication, ScopeContentType, Resource and Audience are
// plain Edm.String values whose accepted constants SAP does not document; they
// are passed through unchanged. Custom parameters are sent only on create
// (see CustomParameter) and read separately.
type OAuth2ClientCredential struct {
	Name                 string `json:"Name"`
	Description          string `json:"Description,omitempty"`
	TokenServiceURL      string `json:"TokenServiceUrl"`
	ClientID             string `json:"ClientId"`
	Scope                string `json:"Scope,omitempty"`
	ClientAuthentication string `json:"ClientAuthentication,omitempty"`
	ScopeContentType     string `json:"ScopeContentType,omitempty"`
	Resource             string `json:"Resource,omitempty"`
	Audience             string `json:"Audience,omitempty"`
	// CustomParameters are sent by CreateOAuth2ClientCredential only.
	CustomParameters []CustomParameter `json:"-"`
}

// CustomParameter is one custom parameter of the token request. A tenant
// check of 2026-10-04 showed: SendAsPartOf takes only body, header or url
// (any letter case, stored as sent); the parameters can only be created
// together with the credential, by a deep insert in its POST; every PUT
// that does not send them deletes them, a PUT that sends them answers 400
// "CustomParameters cannot be updated using this operation", MERGE 405,
// PATCH 501, and the navigation takes no POST or DELETE. A credential with
// custom parameters can therefore only be replaced, never updated.
type CustomParameter struct {
	Key          string `json:"Key"`
	Value        string `json:"Value"`
	SendAsPartOf string `json:"SendAsPartOf"`
}

// oauth2ClientCredentialWriteRequest is the request body shape for creating
// or redeploying (editing) an OAuth2 client credential artifact. This type
// exists only to be marshaled — it is never a target of json.Unmarshal, so
// there is no code path that could accidentally decode a client secret out
// of an API response into it.
type oauth2ClientCredentialWriteRequest struct {
	Name                 string            `json:"Name"`
	Description          string            `json:"Description,omitempty"`
	TokenServiceURL      string            `json:"TokenServiceUrl"`
	ClientID             string            `json:"ClientId"`
	ClientSecret         string            `json:"ClientSecret"`
	Scope                string            `json:"Scope,omitempty"`
	ClientAuthentication string            `json:"ClientAuthentication,omitempty"`
	ScopeContentType     string            `json:"ScopeContentType,omitempty"`
	Resource             string            `json:"Resource,omitempty"`
	Audience             string            `json:"Audience,omitempty"`
	CustomParameters     []CustomParameter `json:"CustomParameters,omitempty"`
}

func oauth2ClientCredentialPath(name string) string {
	return v2.BuildPath(oauth2ClientCredentialsEntitySet, v2.KeyPredicate(name), "")
}

// GetOAuth2ClientCredential reads a single OAuth2 client credential
// artifact's identity and metadata by its Name (alias). It never requests
// or decodes a client secret.
func (c *Client) GetOAuth2ClientCredential(ctx context.Context, name string) (*OAuth2ClientCredential, error) {
	body, err := c.odata.Get(ctx, oauth2ClientCredentialPath(name))
	if err != nil {
		return nil, err
	}

	var cred OAuth2ClientCredential
	if err := v2.DecodeEntity(body, &cred); err != nil {
		return nil, err
	}
	return &cred, nil
}

// CreateOAuth2ClientCredential creates (deploys) a new OAuth2 client
// credential artifact. clientSecret is sent once, in this single request,
// and is never returned: the response is decoded into
// OAuth2ClientCredential, whose type has no ClientSecret field to receive
// it even if SAP's response body happened to include one.
func (c *Client) CreateOAuth2ClientCredential(ctx context.Context, cred OAuth2ClientCredential, clientSecret string) (*OAuth2ClientCredential, error) {
	payload, err := json.Marshal(oauth2ClientCredentialWriteRequest{ //nolint:gosec // G117: this deliberately marshals the client secret into the request body sent to SAP's Create API -- that is the whole purpose of this call, not a leak; see the write-only handling in resource_oauth2_client_credential.go for why it never reaches Terraform state or a log line
		Name:                 cred.Name,
		Description:          cred.Description,
		TokenServiceURL:      cred.TokenServiceURL,
		ClientID:             cred.ClientID,
		ClientSecret:         clientSecret,
		Scope:                cred.Scope,
		ClientAuthentication: cred.ClientAuthentication,
		ScopeContentType:     cred.ScopeContentType,
		Resource:             cred.Resource,
		Audience:             cred.Audience,
		CustomParameters:     cred.CustomParameters,
	})
	if err != nil {
		return nil, fmt.Errorf("securitycontent: encoding oauth2 client credential: %w", err)
	}

	body, err := c.odata.Post(ctx, oauth2ClientCredentialsEntitySet, payload)
	if err != nil {
		return nil, err
	}
	// SAP may accept the write with 202 and no body; read the entry back.
	if v2.EmptyBody(body) {
		return c.GetOAuth2ClientCredential(ctx, cred.Name)
	}

	var created OAuth2ClientCredential
	if err := v2.DecodeEntity(body, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateOAuth2ClientCredential redeploys (edits) an existing OAuth2 client
// credential artifact in place via PUT. SAP documents this explicitly:
// "You can edit and deploy an OAuth2 Client Credentials artifact" and
// "Every time you edit an OAuth2 Client Credentials artifact, you must
// re-enter the Client Secret" — so clientSecret is required on every call,
// and this provider models a rotation as an in-place update, not a
// replacement. The response body is deliberately not decoded; see
// UpdateUserCredential's doc comment for why the caller re-reads instead.
func (c *Client) UpdateOAuth2ClientCredential(ctx context.Context, cred OAuth2ClientCredential, clientSecret string) error {
	payload, err := json.Marshal(oauth2ClientCredentialWriteRequest{ //nolint:gosec // G117: deliberately marshals the client secret into the redeploy request body, the same documented Create-time requirement — see CreateOAuth2ClientCredential above
		Name:                 cred.Name,
		Description:          cred.Description,
		TokenServiceURL:      cred.TokenServiceURL,
		ClientID:             cred.ClientID,
		ClientSecret:         clientSecret,
		Scope:                cred.Scope,
		ClientAuthentication: cred.ClientAuthentication,
		ScopeContentType:     cred.ScopeContentType,
		Resource:             cred.Resource,
		Audience:             cred.Audience,
	})
	if err != nil {
		return fmt.Errorf("securitycontent: encoding oauth2 client credential: %w", err)
	}

	_, err = c.odata.Put(ctx, oauth2ClientCredentialPath(cred.Name), payload)
	return err
}

// oauth2CustomParameters reads only the CustomParameters navigation of a
// credential, expanded.
type oauth2CustomParameters struct {
	CustomParameters v2.ExpandedCollection[CustomParameter] `json:"CustomParameters"`
}

// ListOAuth2ClientCredentialCustomParameters returns the custom parameters
// of a credential.
func (c *Client) ListOAuth2ClientCredentialCustomParameters(ctx context.Context, name string) ([]CustomParameter, error) {
	body, err := c.odata.Get(ctx, v2.BuildPath(oauth2ClientCredentialsEntitySet, v2.KeyPredicate(name), "$expand=CustomParameters"))
	if err != nil {
		return nil, err
	}
	var cred oauth2CustomParameters
	if err := v2.DecodeEntity(body, &cred); err != nil {
		return nil, err
	}
	return cred.CustomParameters.Results, nil
}

// DeleteOAuth2ClientCredential deletes an OAuth2 client credential artifact
// by its Name.
func (c *Client) DeleteOAuth2ClientCredential(ctx context.Context, name string) error {
	return c.odata.Delete(ctx, oauth2ClientCredentialPath(name))
}
