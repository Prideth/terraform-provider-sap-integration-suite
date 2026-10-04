package securitycontent

import (
	"context"
	"fmt"
	"strings"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// PGP keys of the Cloud runtime. SAP Help documents PGP keys only in the
// Monitor UI (Manage Security > PGP Keys); everything here comes from the
// tenant $metadata and was verified on a tenant on 2026-10-04:
//
//   - The Cloud runtime has exactly the keyrings pubring and secring; other
//     names answered 404 "PUBLIC_KEYRING ... does not exist", a POST with a
//     Slug 403.
//   - PUT PgpKeyringPublicResources('pubring')/$value with an armored (or
//     binary) keyring adds its keys and answers, with Accept JSON, a list of
//     PgpKeyEntryImportResults. A key that exists already is reported with
//     Status "not imported" and HTTP 200, so the status must be checked.
//   - The secret keyring takes the secret key with the passphrase that
//     protects it in the request header Passphrase (without it 400
//     "Passphrase Request Header missing or empty", a wrong one 400
//     "Decrypting Secret Key with applied passphrase failed"). SAP then
//     re-encrypts the key with a passphrase of its own.
//   - PgpKeyEntries('<KeyId>') reads one key and DELETE removes it from both
//     keyrings (an entry of Type "Secret,Public" entirely); it answers 404
//     afterwards.
//   - A public key sent to the secret keyring, or a secret key to the public
//     one, answers 400 "Uploaded content does not contain any PGP ... Key".
const (
	pgpKeyEntriesEntitySet             = "PgpKeyEntries"
	pgpKeyringPublicResourcesEntitySet = "PgpKeyringPublicResources"
	pgpKeyringSecretResourcesEntitySet = "PgpKeyringSecretResources"
	pgpPublicKeyring                   = "pubring"
	pgpSecretKeyring                   = "secring"
	pgpKeysContentType                 = "application/pgp-keys"
	pgpPassphraseHeader                = "Passphrase"
	pgpImportStatusAdded               = "added"
)

// PGPKeyEntry is one key of the tenant's PGP keyrings. Type is "Public",
// "Secret" or "Secret,Public". Date properties are Edm.DateTimeOffset
// literals, kept as strings.
type PGPKeyEntry struct {
	ID               string `json:"Id"`
	KeyID            string `json:"KeyId"`
	Fingerprint      string `json:"Fingerprint,omitempty"`
	Type             string `json:"Type,omitempty"`
	CreatedTime      string `json:"CreatedTime,omitempty"`
	Algorithm        string `json:"Algorithm,omitempty"`
	KeyLength        int    `json:"KeyLength,omitempty"`
	KeyFlagsLabel    string `json:"KeyFlagsLabel,omitempty"`
	ValidityState    string `json:"ValidityState,omitempty"`
	ValidUntil       string `json:"ValidUntil,omitempty"`
	ModifiedOn       string `json:"ModifiedOn,omitempty"`
	PrimaryUserID    string `json:"PrimaryUserId,omitempty"`
	ErrorInformation string `json:"ErrorInformation,omitempty"`
}

// HasSecret reports whether the entry holds a secret key.
func (e PGPKeyEntry) HasSecret() bool { return strings.Contains(e.Type, "Secret") }

// PGPImportResult is SAP's answer for one key of an uploaded keyring.
type PGPImportResult struct {
	ID            string `json:"Id"`
	KeyID         string `json:"KeyId"`
	PrimaryUserID string `json:"PrimaryUserId,omitempty"`
	Status        string `json:"Status"`
	StatusDetails string `json:"StatusDetails,omitempty"`
}

func pgpKeyEntryPath(keyID string) string {
	return v2.BuildPath(pgpKeyEntriesEntitySet, v2.KeyPredicate(keyID), "")
}

// ImportPGPPublicKey adds the keys of an armored or binary public keyring
// to the tenant's public keyring and returns the key that was added. It
// fails when the keyring holds more than one key, or when SAP did not add
// it, for example because a key with the same key ID exists already.
func (c *Client) ImportPGPPublicKey(ctx context.Context, keyring []byte) (*PGPImportResult, error) {
	path := v2.BuildPath(pgpKeyringPublicResourcesEntitySet, v2.KeyPredicate(pgpPublicKeyring), "") + "/$value"
	return c.importPGPKey(ctx, path, nil, keyring)
}

// ImportPGPSecretKey adds a secret key to the tenant's secret keyring.
// passphrase is the one that protects the key in keyring; it is sent once,
// in the Passphrase header, and never stored.
func (c *Client) ImportPGPSecretKey(ctx context.Context, keyring []byte, passphrase string) (*PGPImportResult, error) {
	path := v2.BuildPath(pgpKeyringSecretResourcesEntitySet, v2.KeyPredicate(pgpSecretKeyring), "") + "/$value"
	return c.importPGPKey(ctx, path, map[string]string{pgpPassphraseHeader: passphrase}, keyring)
}

func (c *Client) importPGPKey(ctx context.Context, path string, headers map[string]string, keyring []byte) (*PGPImportResult, error) {
	body, err := c.odata.PutRawJSON(ctx, path, pgpKeysContentType, headers, keyring)
	if err != nil {
		return nil, err
	}
	var results []PGPImportResult
	if err := v2.DecodeCollection(body, &results); err != nil {
		return nil, err
	}
	if len(results) != 1 {
		// Several keys in one keyring would be several objects; remove what
		// SAP added and refuse, so nothing stays behind unmanaged.
		var ids []string
		for _, r := range results {
			ids = append(ids, r.KeyID)
			if r.Status == pgpImportStatusAdded {
				_ = c.DeletePGPKey(ctx, r.KeyID)
			}
		}
		return nil, fmt.Errorf("securitycontent: the keyring must hold exactly one PGP key, SAP found %d (%s); keys it added were removed again", len(results), strings.Join(ids, ", "))
	}
	r := results[0]
	if r.Status != pgpImportStatusAdded {
		return nil, fmt.Errorf("securitycontent: SAP did not add PGP key %s: %s (%s)", r.KeyID, r.Status, r.StatusDetails)
	}
	return &r, nil
}

// GetPGPKey reads one key by its key ID.
func (c *Client) GetPGPKey(ctx context.Context, keyID string) (*PGPKeyEntry, error) {
	body, err := c.odata.Get(ctx, pgpKeyEntryPath(keyID))
	if err != nil {
		return nil, err
	}
	var e PGPKeyEntry
	if err := v2.DecodeEntity(body, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

// DeletePGPKey removes a key from both keyrings: its public and, if there
// is one, its secret part.
func (c *Client) DeletePGPKey(ctx context.Context, keyID string) error {
	return c.odata.Delete(ctx, pgpKeyEntryPath(keyID))
}
