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

const pgpSecretKeyTypeName = "sapintegrationsuite_pgp_secret_key"

// NewPGPSecretKeyResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_pgp_secret_key.
func NewPGPSecretKeyResource() resource.Resource {
	return &pgpSecretKeyResource{}
}

type pgpSecretKeyResource struct {
	client *securitycontent.Client
}

// pgpSecretKeyModel carries the write-only fields only because the
// framework decodes every attribute; Terraform always sends them as null in
// plan and state, and Create reads them from the configuration.
type pgpSecretKeyModel struct {
	pgpKeyFields
	SecretKeyWO        types.String `tfsdk:"secret_key_wo"`
	PassphraseWO       types.String `tfsdk:"passphrase_wo"`
	SecretKeyWOVersion types.String `tfsdk:"secret_key_wo_version"`
}

func (r *pgpSecretKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pgp_secret_key"
}

func (r *pgpSecretKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	attrs := pgpKeyAttributes()
	attrs["secret_key_wo"] = schema.StringAttribute{
		Required:  true,
		Sensitive: true,
		WriteOnly: true,
		Description: "The secret key as an ASCII-armored block (\"-----BEGIN PGP PRIVATE KEY " +
			"BLOCK-----\"), protected by passphrase_wo, as gpg --armor --export-secret-keys writes it. " +
			"Write-only: Terraform sends it once and never stores it in plan or state. SAP re-encrypts " +
			"the key with a passphrase of its own. Requires Terraform 1.11 or later.",
	}
	attrs["passphrase_wo"] = schema.StringAttribute{
		Required:  true,
		Sensitive: true,
		WriteOnly: true,
		Description: "The passphrase that protects secret_key_wo, sent once in a request header. " +
			"Write-only: never stored. SAP refuses a wrong one.",
	}
	attrs["secret_key_wo_version"] = schema.StringAttribute{
		Required: true,
		Description: "Any value; change it to send a new secret key. Terraform cannot see changes " +
			"of the write-only key, so this value decides when the key is replaced: the old key is " +
			"deleted and the new one added.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
				// The first plan after an import only records the version.
				resp.RequiresReplace = !req.StateValue.IsNull()
			}, "Changing the version replaces the key.", "Changing the version replaces the key."),
		},
	}
	resp.Schema = schema.Schema{
		Description: "Adds a PGP secret key to the tenant's secret keyring, for the PGP decryptor " +
			"and the PGP signer. The key and its passphrase are write-only. Unofficial: SAP documents " +
			"PGP keys only in the Monitor UI; the requests are known from the tenant $metadata and " +
			"were verified on a tenant. Cloud runtime only.",
		Attributes: attrs,
	}
}

func (r *pgpSecretKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, pgpSecretKeyTypeName, &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
}

// ValidateConfig refuses anything but one armored secret key. The error
// never repeats the key.
func (r *pgpSecretKeyResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var key types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("secret_key_wo"), &key)...)
	if resp.Diagnostics.HasError() || key.IsNull() || key.IsUnknown() {
		return
	}
	info, err := securitycontent.ParsePGPKey([]byte(key.ValueString()))
	switch {
	case err != nil:
		resp.Diagnostics.AddAttributeError(pathRoot("secret_key_wo"), "Invalid PGP secret key", diagnosticDetail(err))
	case !info.Secret:
		resp.Diagnostics.AddAttributeError(pathRoot("secret_key_wo"), "Public key in secret_key_wo",
			"secret_key_wo holds a PGP PUBLIC KEY BLOCK; use sapintegrationsuite_pgp_public_key for it.")
	}
}

func (r *pgpSecretKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pgpSecretKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	var key, passphrase types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("secret_key_wo"), &key)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("passphrase_wo"), &passphrase)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.ImportPGPSecretKey(ctx, []byte(key.ValueString()), passphrase.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to add the PGP secret key", diagnosticDetail(err))
		return
	}
	found, err := readPGPKey(ctx, r.client, result.KeyID, &plan.pgpKeyFields)
	if err != nil || !found {
		resp.Diagnostics.AddError("Failed to read the PGP secret key back", "SAP added key "+result.KeyID+" but did not return it: "+errText(err))
		return
	}
	plan.SecretKeyWO, plan.PassphraseWO = types.StringNull(), types.StringNull()
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *pgpSecretKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pgpSecretKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	found, err := readPGPKey(ctx, r.client, state.KeyID.ValueString(), &state.pgpKeyFields)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read the PGP secret key", diagnosticDetail(err))
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only records the version after an import; every other change
// replaces the key.
func (r *pgpSecretKeyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state pgpSecretKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.SecretKeyWOVersion = plan.SecretKeyWOVersion
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete removes the key from both keyrings: SAP deletes its public part
// with it.
func (r *pgpSecretKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pgpSecretKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeletePGPKey(ctx, state.KeyID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete the PGP secret key", diagnosticDetail(err))
	}
}

func (r *pgpSecretKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("key_id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("id"), req.ID)...)
}
