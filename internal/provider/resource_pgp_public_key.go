package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

const pgpPublicKeyTypeName = "sapintegrationsuite_pgp_public_key"

// NewPGPPublicKeyResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_pgp_public_key.
func NewPGPPublicKeyResource() resource.Resource {
	return &pgpPublicKeyResource{}
}

type pgpPublicKeyResource struct {
	client *securitycontent.Client
}

type pgpPublicKeyModel struct {
	pgpKeyFields
	PublicKey types.String `tfsdk:"public_key"`
}

func (r *pgpPublicKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pgp_public_key"
}

func (r *pgpPublicKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := pgpKeyAttributes()
	attrs["public_key"] = schema.StringAttribute{
		Required: true,
		Description: "The public key as an ASCII-armored block (\"-----BEGIN PGP PUBLIC KEY " +
			"BLOCK-----\"), with exactly one primary key and its sub keys, as gpg --armor --export " +
			"writes it. Not sensitive: public keys are public. A different key replaces the resource; " +
			"the same key in other formatting does not.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplaceIf(pgpOtherKey, "A different key replaces the resource.",
				"A different key replaces the resource."),
		},
	}
	resp.Schema = schema.Schema{
		Description: "Adds a PGP public key to the tenant's public keyring, for the PGP encryptor " +
			"and the signature verification of the PGP decryptor. Unofficial: SAP documents PGP keys " +
			"only in the Monitor UI; the requests are known from the tenant $metadata and were " +
			"verified on a tenant. Cloud runtime only.",
		Attributes: attrs,
	}
}

// pgpOtherKey replaces the resource only when the planned key has another
// fingerprint than the one in state; a reformatted block or the first plan
// after an import (no text in state) keeps it.
func pgpOtherKey(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsUnknown() || req.PlanValue.IsNull() {
		resp.RequiresReplace = true
		return
	}
	var fingerprint types.String
	req.State.GetAttribute(context.Background(), pathRoot("fingerprint"), &fingerprint)
	info, err := securitycontent.ParsePGPKey([]byte(req.PlanValue.ValueString()))
	resp.RequiresReplace = err != nil || fingerprint.IsNull() || info.Fingerprint != fingerprint.ValueString()
}

func (r *pgpPublicKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, pgpPublicKeyTypeName, &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
}

// ValidateConfig refuses anything but one armored public key.
func (r *pgpPublicKeyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var key types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("public_key"), &key)...)
	if resp.Diagnostics.HasError() || key.IsNull() || key.IsUnknown() {
		return
	}
	info, err := securitycontent.ParsePGPKey([]byte(key.ValueString()))
	switch {
	case err != nil:
		resp.Diagnostics.AddAttributeError(pathRoot("public_key"), "Invalid PGP public key", diagnosticDetail(err))
	case info.Secret:
		resp.Diagnostics.AddAttributeError(pathRoot("public_key"), "Secret key in public_key",
			"public_key holds a PGP PRIVATE KEY BLOCK. Put secret keys into sapintegrationsuite_pgp_secret_key, "+
				"whose key material is write-only, and export the public key with gpg --armor --export.")
	}
}

// ModifyPlan shows the key ID and fingerprint of a new key in the plan.
func (r *pgpPublicKeyResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || !req.State.Raw.IsNull() {
		return
	}
	var key types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, pathRoot("public_key"), &key)...)
	if key.IsUnknown() || key.IsNull() {
		return
	}
	info, err := securitycontent.ParsePGPKey([]byte(key.ValueString()))
	if err != nil {
		return
	}
	for name, value := range map[string]string{"id": info.KeyID, "key_id": info.KeyID, "fingerprint": info.Fingerprint} {
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, pathRoot(name), value)...)
	}
}

func (r *pgpPublicKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pgpPublicKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.ImportPGPPublicKey(ctx, []byte(plan.PublicKey.ValueString()))
	if err != nil {
		resp.Diagnostics.AddError("Failed to add the PGP public key", diagnosticDetail(err))
		return
	}
	found, err := readPGPKey(ctx, r.client, result.KeyID, &plan.pgpKeyFields)
	if err != nil || !found {
		resp.Diagnostics.AddError("Failed to read the PGP public key back", "SAP added key "+result.KeyID+" but did not return it: "+errText(err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pgpPublicKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pgpPublicKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := readPGPKey(ctx, r.client, state.KeyID.ValueString(), &state.pgpKeyFields)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the PGP public key", diagnosticDetail(err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only records another formatting of the same key, or the text of
// an imported key; SAP holds the key itself unchanged.
func (r *pgpPublicKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pgpPublicKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.PublicKey = plan.PublicKey
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the key, unless SAP holds a secret key for it as well:
// SAP deletes both parts together, and the secret part may be in use.
func (r *pgpPublicKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pgpPublicKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	keyID := state.KeyID.ValueString()
	entry, err := r.client.GetPGPKey(ctx, keyID)
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to read the PGP public key before deleting it", diagnosticDetail(err))
		return
	}
	if entry.HasSecret() {
		resp.Diagnostics.AddWarning("PGP key left in place",
			"SAP also holds the secret key of "+keyID+" and deletes both parts together, so the key stays. "+
				"Destroy the sapintegrationsuite_pgp_secret_key of it, or delete the key in Monitor > Manage "+
				"Security > PGP Keys.")
		return
	}
	if err := r.client.DeletePGPKey(ctx, keyID); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete the PGP public key", diagnosticDetail(err))
	}
}

func (r *pgpPublicKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("key_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("id"), req.ID)...)
}

func errText(err error) string {
	if err == nil {
		return "not found"
	}
	return err.Error()
}
