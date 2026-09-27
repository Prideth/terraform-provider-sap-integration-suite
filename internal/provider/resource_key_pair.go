package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

// keyPairSignatureAlgorithmsByKeyType is confirmed verbatim from SAP's own
// "Generate a Key Pair" documentation's Input Properties table.
var keyPairSignatureAlgorithmsByKeyType = map[string][]string{
	"RSA": {"SHA-512/RSA", "SHA-256/RSA", "SHA-384/RSA", "SHA-224/RSA", "SHA-1/RSA"},
	"DSA": {"SHA-256/DSA", "SHA-224/DSA", "SHA-1/DSA"},
	"EC":  {"SHA-512/ECDSA", "SHA-256/ECDSA", "SHA-1/ECDSA"},
}

var keyPairKeyTypes = []string{"RSA", "DSA", "EC"}

// keyPairECCurves is confirmed verbatim from SAP's own "Generate a Key
// Pair" documentation's KeyAlgorithmParameter description.
var keyPairECCurves = []string{
	"secp160k1", "secp160r1", "secp160r2", "secp192k1", "secp192r1", "NIST P-192",
	"X9.62 prime192v1", "secp224k1", "secp224r1", "NIST P-224", "secp256k1", "secp256r1",
	"NIST P-256", "X9.62 prime256v1", "secp384r1", "NIST P-384", "secp521r1", "NIST P-521",
	"X9.62 prime192v2", "X9.62 prime192v3", "X9.62 prime239v1", "X9.62 prime239v2",
	"X9.62 prime239v3", "sect163k1", "NIST K-163", "sect163r1", "sect163r2", "NIST B-163",
	"sect193r1", "sect193r2", "sect233k1", "NIST K-233", "sect233r1", "NIST B-233",
	"sect239k1", "sect283k1", "NIST K-283", "sect283r1", "NIST B-283", "sect409k1",
	"NIST K-409", "sect409r1", "NIST B-409", "sect571k1", "NIST K-571", "sect571r1",
	"NIST B-571", "X9.62 c2tnb191v1", "X9.62 c2tnb191v2", "X9.62 c2tnb191v3",
	"X9.62 c2tnb239v1", "X9.62 c2tnb239v2", "X9.62 c2tnb239v3", "X9.62 c2tnb359v1",
	"X9.62 c2tnb431r1",
}

// NewKeyPairResource returns a fresh resource.Resource implementation for
// sapintegrationsuite_key_pair.
func NewKeyPairResource() resource.Resource {
	return &keyPairResource{}
}

type keyPairResource struct {
	client *securitycontent.Client
}

type keyPairModel struct {
	ID                    types.String `tfsdk:"id"`
	Alias                 types.String `tfsdk:"alias"`
	KeyType               types.String `tfsdk:"key_type"`
	SignatureAlgorithm    types.String `tfsdk:"signature_algorithm"`
	KeySize               types.Int64  `tfsdk:"key_size"`
	KeyAlgorithmParameter types.String `tfsdk:"key_algorithm_parameter"`
	CommonName            types.String `tfsdk:"common_name"`
	OrganizationUnit      types.String `tfsdk:"organization_unit"`
	Organization          types.String `tfsdk:"organization"`
	Locality              types.String `tfsdk:"locality"`
	State                 types.String `tfsdk:"state"`
	Country               types.String `tfsdk:"country"`
	Email                 types.String `tfsdk:"email"`
	ValidNotBefore        types.String `tfsdk:"valid_not_before"`
	ValidNotAfter         types.String `tfsdk:"valid_not_after"`
	PublicKeyOpenSSH      types.String `tfsdk:"public_key_openssh"`
	RuntimeLocationID     types.String `tfsdk:"runtime_location_id"`
}

func (r *keyPairResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_pair"
}

func (r *keyPairResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Generates an SAP-managed key pair as a persistent tenant keystore entry. " +
			"Backed by the public Security Content OData V2 API: POST KeyPairGenerationRequests " +
			"(confirmed field-for-field against SAP's own documented Input Properties table). The " +
			"private key never leaves SAP — this resource has no field for it, and no operation " +
			"this provider performs ever requests one. There is no separate \"SSH Key\" object in " +
			"SAP's API: the tenant keystore's own UI documentation uses the identical attribute set " +
			"for both, so an RSA key pair generated here can also be exported in OpenSSH format via " +
			"public_key_openssh — see docs/guides/security-content.md. SAP documents no update " +
			"operation for a generated key pair, so every attribute that defines the generated key " +
			"material is RequiresReplace.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Always equal to alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"alias": schema.StringAttribute{
				Required:    true,
				Description: "The keystore entry alias to create. Must not already exist on the tenant.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_type": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "One of \"RSA\" (default), \"DSA\", \"EC\" — confirmed as SAP's " +
					"complete, fixed enum for this field.",
				Default:    stringdefault.StaticString("RSA"),
				Validators: []validator.String{stringOneOfValidator{values: keyPairKeyTypes}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"signature_algorithm": schema.StringAttribute{
				Optional: true,
				Description: "The signing algorithm, from a fixed enum that depends on key_type " +
					"(RSA: SHA-512/RSA (SAP's default), SHA-256/RSA, SHA-384/RSA, SHA-224/RSA, " +
					"SHA-1/RSA; DSA: SHA-256/DSA, SHA-224/DSA, SHA-1/DSA; EC: SHA-512/ECDSA, " +
					"SHA-256/ECDSA, SHA-1/ECDSA). Left unset, SAP applies its own default; this " +
					"provider cannot confirm this value is returned by a subsequent read, so it is " +
					"not Computed — it is trusted from your configuration once generation succeeds, " +
					"the same as every other write-only-in-practice generation parameter below.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key_size": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Description: "Key size in bits. Must be specified for key_type RSA or DSA. For " +
					"key_type EC, either key_size (112-571) or key_algorithm_parameter must be " +
					"specified. SAP's default is 4096 when left unset for RSA/DSA.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"key_algorithm_parameter": schema.StringAttribute{
				Optional: true,
				Description: "The standard name of an Elliptic Curve domain parameter (for " +
					"example \"secp256r1\", \"NIST P-256\"), used only for key_type \"EC\" instead " +
					"of key_size. See docs/guides/security-content.md for the complete confirmed " +
					"list of accepted values.",
				Validators: []validator.String{stringOneOfValidator{values: keyPairECCurves}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"common_name": schema.StringAttribute{
				Required:    true,
				Description: "Common name (CN) of the certificate's subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization_unit": schema.StringAttribute{
				Optional:    true,
				Description: "Organizational unit (OU) of the subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"organization": schema.StringAttribute{
				Optional:    true,
				Description: "Organization (O) of the subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"locality": schema.StringAttribute{
				Optional:    true,
				Description: "Locality (L) of the subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"state": schema.StringAttribute{
				Optional:    true,
				Description: "State or province (ST) of the subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"country": schema.StringAttribute{
				Required: true,
				Description: "Country or region (C) of the subject distinguished name. SAP " +
					"documents this as \"two characters required.\"",
				Validators: []validator.String{stringLenExactlyValidator{n: 2}},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "E-mail (E) of the subject distinguished name.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid_not_before": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Lower boundary of the certificate's validity period, RFC 3339 " +
					"(for example \"2026-01-01T00:00:00Z\"). Left unset, SAP defaults to the " +
					"current time.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"valid_not_after": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Upper boundary of the certificate's validity period, RFC 3339. " +
					"Left unset, SAP defaults to the current time plus three years.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"public_key_openssh": schema.StringAttribute{
				Computed: true,
				Description: "The public key in OpenSSH format (\"ssh-rsa AAAA...\"), read via " +
					"SAP's confirmed KeystoreEntries('<hexalias>')/Sshkey/$value export. Only " +
					"populated for key_type RSA or DSA — SAP documents EC as unsupported for this " +
					"export; left null for an EC key pair rather than surfacing an error, since not " +
					"every practitioner generating an EC key pair needs an SSH representation of it.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ValidateConfig enforces SAP's documented conditional rules that a plain
// schema cannot express: key_size is mandatory for RSA/DSA, and EC
// requires either key_size or key_algorithm_parameter.
func (r *keyPairResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config keyPairModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	keyType := config.KeyType.ValueString()
	if config.KeyType.IsUnknown() {
		return
	}
	if keyType == "" {
		keyType = "RSA"
	}

	switch keyType {
	case "RSA", "DSA":
		if config.KeySize.IsNull() && !config.KeySize.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				pathRoot("key_size"),
				"key_size is required for this key_type",
				fmt.Sprintf("key_size must be specified when key_type is %q.", keyType),
			)
		}
		if !config.KeyAlgorithmParameter.IsNull() && config.KeyAlgorithmParameter.ValueString() != "" {
			resp.Diagnostics.AddAttributeError(
				pathRoot("key_algorithm_parameter"),
				"key_algorithm_parameter is only valid for key_type \"EC\"",
				fmt.Sprintf("key_algorithm_parameter was set, but key_type is %q.", keyType),
			)
		}
	case "EC":
		hasKeySize := !config.KeySize.IsNull() && !config.KeySize.IsUnknown()
		hasCurve := !config.KeyAlgorithmParameter.IsNull() && config.KeyAlgorithmParameter.ValueString() != ""
		if !hasKeySize && !hasCurve {
			resp.Diagnostics.AddAttributeError(
				pathRoot("key_algorithm_parameter"),
				"key_size or key_algorithm_parameter is required for key_type \"EC\"",
				"Either key_size (112-571) or key_algorithm_parameter must be specified when key_type is \"EC\".",
			)
		}
		if hasKeySize {
			n := config.KeySize.ValueInt64()
			if n < 112 || n > 571 {
				resp.Diagnostics.AddAttributeError(
					pathRoot("key_size"),
					"key_size out of range for key_type \"EC\"",
					"SAP documents key_size for key_type \"EC\" as null or from 112 through 571.",
				)
			}
		}
	}

	if !config.SignatureAlgorithm.IsNull() && !config.SignatureAlgorithm.IsUnknown() {
		alg := config.SignatureAlgorithm.ValueString()
		valid := keyPairSignatureAlgorithmsByKeyType[keyType]
		if !containsString(valid, alg) {
			resp.Diagnostics.AddAttributeError(
				pathRoot("signature_algorithm"),
				"signature_algorithm is not valid for this key_type",
				fmt.Sprintf("%q is not one of the values SAP documents for key_type %q: %v", alg, keyType, valid),
			)
		}
	}
}

func (r *keyPairResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
}

func (r *keyPairResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keyPairModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	genReq, diags := keyPairGenerationRequestFromModel(plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := client.GenerateKeyPair(ctx, genReq); err != nil {
		resp.Diagnostics.AddError("Failed to generate SAP Integration Suite key pair", diagnosticDetail(err))
		return
	}

	r.readAfterWrite(ctx, plan.Alias.ValueString(), plan, &resp.Diagnostics, &resp.State)
}

func (r *keyPairResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keyPairModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.readAfterWrite(ctx, state.Alias.ValueString(), state, &resp.Diagnostics, &resp.State)
}

// readAfterWrite re-reads the confirmed-readable subset of a key pair's
// state (key_type, key_size, valid_not_before, valid_not_after, and the
// OpenSSH public key export) from KeystoreEntries, fills subject fields that
// are still empty from SubjectDN, and leaves the generation-only parameters
// this API does not return (signature_algorithm, key_algorithm_parameter)
// exactly as they are in base — trusted from the last successful write, the
// same pattern this provider uses wherever an API's Read cannot verify
// everything its Create accepted.
func (r *keyPairResource) readAfterWrite(ctx context.Context, alias string, base keyPairModel, diags *diag.Diagnostics, state *tfsdk.State) {
	client, ok := locatedClient(r.client, base.RuntimeLocationID, diags)
	if !ok {
		return
	}
	entry, err := client.GetKeystoreEntry(ctx, alias)
	if err != nil {
		if isNotFound(err) {
			state.RemoveResource(ctx)
			return
		}
		diags.AddError("Failed to read SAP Integration Suite key pair", diagnosticDetail(err))
		return
	}

	base.ID = types.StringValue(alias)
	base.KeyType = stringOrNull(entry.KeyType)
	base.KeySize = int64OrNull(entry.KeySize)
	base.ValidNotBefore = stringOrNull(entry.ValidNotBefore)
	base.ValidNotAfter = stringOrNull(entry.ValidNotAfter)
	fillSubjectFromDN(&base, entry.SubjectDN)

	if entry.KeyType == "RSA" || entry.KeyType == "DSA" {
		if pub, sshErr := client.GetSSHPublicKey(ctx, alias); sshErr == nil {
			base.PublicKeyOpenSSH = types.StringValue(string(pub))
		} else {
			base.PublicKeyOpenSSH = types.StringNull()
		}
	} else {
		base.PublicKeyOpenSSH = types.StringNull()
	}

	diags.Append(state.Set(ctx, &base)...)
}

// Update should be unreachable: every field this resource exposes carries
// RequiresReplace, since SAP documents no operation to update a generated
// key pair's material in place. This exists only to satisfy the
// resource.Resource interface.
func (r *keyPairResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan keyPairModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readAfterWrite(ctx, plan.Alias.ValueString(), plan, &resp.Diagnostics, &resp.State)
}

// Delete uses the same documented keystore mass-deletion operation as
// sapintegrationsuite_certificate, with exactly the one alias this
// resource owns — see that resource's Delete for the full reasoning about
// SAP-owned entries.
func (r *keyPairResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keyPairModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	err := client.DeleteKeystoreEntries(ctx, []string{state.Alias.ValueString()})
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite key pair", diagnosticDetail(err))
	}
}

// fillSubjectFromDN sets the subject fields that are still empty (after an
// import) from the entry's SubjectDN, which KeystoreEntries returns. Values
// already in state come from the configuration and stay as they are. Without
// this, the first apply after an import would see every configured subject
// field as a change and, since they are RequiresReplace, regenerate the key
// pair (a tenant run in September 2026 showed the gap).
func fillSubjectFromDN(m *keyPairModel, subjectDN string) {
	if subjectDN == "" {
		return
	}
	dn := parseDistinguishedName(subjectDN)
	fill := func(field *types.String, keys ...string) {
		if !field.IsNull() && !field.IsUnknown() {
			return
		}
		for _, k := range keys {
			if v, ok := dn[k]; ok && v != "" {
				*field = types.StringValue(v)
				return
			}
		}
	}
	fill(&m.CommonName, "CN")
	fill(&m.OrganizationUnit, "OU")
	fill(&m.Organization, "O")
	fill(&m.Locality, "L")
	fill(&m.State, "ST", "S")
	fill(&m.Country, "C")
	fill(&m.Email, "EMAILADDRESS", "E")
}

// ImportState recovers the alias; the next read fills the subject fields
// from the entry's SubjectDN (see fillSubjectFromDN). signature_algorithm and
// key_algorithm_parameter cannot be read back, so a configuration that sets
// them plans a replacement after an import — see
// docs/guides/security-content.md.
func (r *keyPairResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("alias"), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
}

func keyPairGenerationRequestFromModel(m keyPairModel) (securitycontent.KeyPairGenerationRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	req := securitycontent.KeyPairGenerationRequest{
		Alias:                 m.Alias.ValueString(),
		KeyType:               m.KeyType.ValueString(),
		SignatureAlgorithm:    m.SignatureAlgorithm.ValueString(),
		KeyAlgorithmParameter: m.KeyAlgorithmParameter.ValueString(),
		CommonName:            m.CommonName.ValueString(),
		OrganizationUnit:      m.OrganizationUnit.ValueString(),
		Organization:          m.Organization.ValueString(),
		Locality:              m.Locality.ValueString(),
		State:                 m.State.ValueString(),
		Country:               m.Country.ValueString(),
		Email:                 m.Email.ValueString(),
	}
	if !m.KeySize.IsNull() && !m.KeySize.IsUnknown() {
		req.KeySize = int(m.KeySize.ValueInt64())
	}
	if !m.ValidNotBefore.IsNull() && m.ValidNotBefore.ValueString() != "" {
		t, err := time.Parse(time.RFC3339, m.ValidNotBefore.ValueString())
		if err != nil {
			diags.AddAttributeError(pathRoot("valid_not_before"), "Invalid date", "Must be RFC 3339, for example \"2026-01-01T00:00:00Z\": "+err.Error())
		} else {
			req.ValidNotBefore = &t
		}
	}
	if !m.ValidNotAfter.IsNull() && m.ValidNotAfter.ValueString() != "" {
		t, err := time.Parse(time.RFC3339, m.ValidNotAfter.ValueString())
		if err != nil {
			diags.AddAttributeError(pathRoot("valid_not_after"), "Invalid date", "Must be RFC 3339, for example \"2026-01-01T00:00:00Z\": "+err.Error())
		} else {
			req.ValidNotAfter = &t
		}
	}

	return req, diags
}

func containsString(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

// stringOneOfValidator rejects a config value not present in values,
// unless it is null or unknown.
type stringOneOfValidator struct {
	values []string
}

func (v stringOneOfValidator) Description(context.Context) string {
	return fmt.Sprintf("must be one of: %v", v.values)
}
func (v stringOneOfValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v stringOneOfValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if !containsString(v.values, req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", v.Description(ctx)+", got: "+req.ConfigValue.ValueString())
	}
}

// stringLenExactlyValidator rejects a config value whose length is not
// exactly n, unless it is null or unknown.
type stringLenExactlyValidator struct {
	n int
}

func (v stringLenExactlyValidator) Description(context.Context) string {
	return fmt.Sprintf("must be exactly %d characters", v.n)
}
func (v stringLenExactlyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}
func (v stringLenExactlyValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if len(req.ConfigValue.ValueString()) != v.n {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid value", v.Description(ctx)+", got: "+strconv.Itoa(len(req.ConfigValue.ValueString()))+" characters")
	}
}
