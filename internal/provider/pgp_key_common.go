package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// pgpKeyFields are the attributes SAP reports for a PGP key, shared by the
// public and the secret key resource.
type pgpKeyFields struct {
	ID            types.String `tfsdk:"id"`
	KeyID         types.String `tfsdk:"key_id"`
	Fingerprint   types.String `tfsdk:"fingerprint"`
	Type          types.String `tfsdk:"type"`
	Algorithm     types.String `tfsdk:"algorithm"`
	KeyLength     types.Int64  `tfsdk:"key_length"`
	PrimaryUserID types.String `tfsdk:"primary_user_id"`
	ValidUntil    types.String `tfsdk:"valid_until"`
	ValidityState types.String `tfsdk:"validity_state"`
}

func (f *pgpKeyFields) fill(e *securitycontent.PGPKeyEntry) {
	f.ID = types.StringValue(e.KeyID)
	f.KeyID = types.StringValue(e.KeyID)
	f.Fingerprint = stringOrNull(e.Fingerprint)
	f.Type = stringOrNull(e.Type)
	f.Algorithm = stringOrNull(e.Algorithm)
	f.KeyLength = int64OrNull(e.KeyLength)
	f.PrimaryUserID = stringOrNull(e.PrimaryUserID)
	f.ValidUntil = odataDateToRFC3339(e.ValidUntil)
	f.ValidityState = stringOrNull(e.ValidityState)
}

// pgpKeyAttributes are the computed attributes of both PGP key resources.
func pgpKeyAttributes() map[string]schema.Attribute {
	keep := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:      true,
			Description:   "Always equal to key_id.",
			PlanModifiers: keep,
		},
		"key_id": schema.StringAttribute{
			Computed: true,
			Description: "The key ID, the last 16 hex digits of the fingerprint, upper case: SAP's key " +
				"of the PGP key. The PGP encryptor and signer steps of an integration flow reference " +
				"keys by it or by user ID.",
			PlanModifiers: keep,
		},
		"fingerprint": schema.StringAttribute{
			Computed:      true,
			Description:   "The v4 fingerprint, 40 hex digits, upper case.",
			PlanModifiers: keep,
		},
		"type": schema.StringAttribute{
			Computed: true,
			Description: "What SAP holds for the key: \"Public\", \"Secret\", or \"Secret,Public\" when " +
				"its public key was imported as well.",
		},
		"algorithm": schema.StringAttribute{
			Computed:      true,
			Description:   "The public key algorithm as SAP names it, for example \"RSA (1)\".",
			PlanModifiers: keep,
		},
		"key_length": schema.Int64Attribute{
			Computed:      true,
			Description:   "The key length in bits.",
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		},
		"primary_user_id": schema.StringAttribute{
			Computed:      true,
			Description:   "The primary user ID of the key, usually \"Name <e-mail>\".",
			PlanModifiers: keep,
		},
		"valid_until": schema.StringAttribute{
			Computed:      true,
			Description:   "The expiry of the key, RFC 3339; null for a key that does not expire.",
			PlanModifiers: keep,
		},
		"validity_state": schema.StringAttribute{
			Computed: true,
			Description: "SAP's validity state: \"Valid\", \"Critical\" (expires within 14 days) or " +
				"\"Expired\". It changes over time, so a refresh can report it as changed.",
		},
	}
}

// readPGPKey reads a key and fills fields. It returns false when the key
// is gone.
func readPGPKey(ctx context.Context, client *securitycontent.Client, keyID string, fields *pgpKeyFields) (bool, error) {
	e, err := client.GetPGPKey(ctx, keyID)
	if err != nil {
		if isNotFound(err) {
			return false, nil
		}
		return false, err
	}
	fields.fill(e)
	return true, nil
}
