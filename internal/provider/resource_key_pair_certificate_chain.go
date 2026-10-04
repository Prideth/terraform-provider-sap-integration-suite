package provider

import (
	"context"
	"crypto/x509"
	"sort"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/securitycontent"
)

const keyPairCertificateChainTypeName = "sapintegrationsuite_key_pair_certificate_chain"

// NewKeyPairCertificateChainResource returns a fresh resource.Resource
// implementation for sapintegrationsuite_key_pair_certificate_chain.
func NewKeyPairCertificateChainResource() resource.Resource {
	return &keyPairCertificateChainResource{}
}

type keyPairCertificateChainResource struct {
	client *securitycontent.Client
}

type keyPairCertificateChainModel struct {
	ID                types.String `tfsdk:"id"`
	KeyPairAlias      types.String `tfsdk:"key_pair_alias"`
	CertificateChain  types.String `tfsdk:"certificate_chain"`
	CertificateSHA256 types.String `tfsdk:"certificate_sha256"`
	Certificates      types.List   `tfsdk:"certificates"`
	RuntimeLocationID types.String `tfsdk:"runtime_location_id"`
}

var chainCertificateAttrTypes = map[string]attr.Type{
	"subject_dn":    types.StringType,
	"issuer_dn":     types.StringType,
	"serial_number": types.StringType,
	"not_before":    types.StringType,
	"not_after":     types.StringType,
	"sha256":        types.StringType,
}

func (r *keyPairCertificateChainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_key_pair_certificate_chain"
}

func (r *keyPairCertificateChainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the certificate chain of a key pair in the tenant keystore: the key " +
			"pair's certificate signed by a certificate authority, with the certificates that " +
			"issued it. Get the certificate signing request from " +
			"sapintegrationsuite_key_pair.certificate_signing_request, have it signed, and put " +
			"the signed certificate and its issuers here. Unofficial: SAP documents chain import " +
			"as a capability of the Key Pair API, but the requests are known only from the tenant " +
			"$metadata (PUT CertificateChainResources('<hexalias>')/$value, the export through " +
			"KeystoreEntries('<hexalias>')/ChainResource/$value) and were verified on a tenant. " +
			"SAP has no request that removes a chain: destroying this resource leaves the chain " +
			"on the key pair.",
		Attributes: map[string]schema.Attribute{
			"runtime_location_id": runtimeLocationResourceAttribute(),
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Always equal to key_pair_alias.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"key_pair_alias": schema.StringAttribute{
				Required: true,
				Description: "The alias of the key pair the chain belongs to, for example " +
					"sapintegrationsuite_key_pair.example.alias. Changing it replaces the resource.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate_chain": schema.StringAttribute{
				Required: true,
				Description: "The key pair's signed certificate and the certificates that issued it, " +
					"as one PEM bundle of CERTIFICATE blocks in any order; the root may be left out. " +
					"Any other block, above all a private key, is refused. Not sensitive: certificates " +
					"are public. SAP refuses a certificate that was not issued for this key pair's " +
					"public key. On refresh the configured text is kept as long as SAP holds the same " +
					"certificates; a different chain on the tenant replaces it, leaf first.",
			},
			"certificate_sha256": schema.StringAttribute{
				Computed: true,
				Description: "The SHA-256 fingerprint (lowercase hex) of the key pair's signed " +
					"certificate, computed locally from certificate_chain.",
			},
			"certificates": schema.ListNestedAttribute{
				Computed: true,
				Description: "The certificates of the chain, parsed locally, leaf first: the key " +
					"pair's certificate, then each issuer.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"subject_dn":    schema.StringAttribute{Computed: true, Description: "Subject distinguished name."},
						"issuer_dn":     schema.StringAttribute{Computed: true, Description: "Issuer distinguished name."},
						"serial_number": schema.StringAttribute{Computed: true, Description: "Serial number (decimal)."},
						"not_before":    schema.StringAttribute{Computed: true, Description: "Start of the validity period, RFC 3339."},
						"not_after":     schema.StringAttribute{Computed: true, Description: "End of the validity period, RFC 3339."},
						"sha256":        schema.StringAttribute{Computed: true, Description: "SHA-256 fingerprint (lowercase hex) of the certificate."},
					},
				},
			},
		},
	}
}

func (r *keyPairCertificateChainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	data, ok := req.ProviderData.(*Data)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data type", "Expected *provider.Data")
		return
	}
	if !requireOptIn(data, keyPairCertificateChainTypeName, &resp.Diagnostics) {
		return
	}
	if !requireHTTPClient(data, "resource", &resp.Diagnostics) {
		return
	}
	r.client = securitycontent.New(data.HTTPClient, data.Host)
}

// ValidateConfig refuses a chain that is not a PEM bundle of certificates,
// before anything is sent to SAP.
func (r *keyPairCertificateChainResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var chain types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, pathRoot("certificate_chain"), &chain)...)
	if resp.Diagnostics.HasError() || chain.IsNull() || chain.IsUnknown() {
		return
	}
	if _, err := securitycontent.ParseCertificateChainPEM([]byte(chain.ValueString())); err != nil {
		resp.Diagnostics.AddAttributeError(pathRoot("certificate_chain"), "Invalid certificate chain", diagnosticDetail(err))
	}
}

// ModifyPlan derives the fingerprint and the certificate list from the
// planned chain, so that a changed chain shows its new values in the plan.
func (r *keyPairCertificateChainResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan keyPairCertificateChainModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.CertificateChain.IsUnknown() || plan.CertificateChain.IsNull() {
		plan.CertificateSHA256 = types.StringUnknown()
		plan.Certificates = types.ListUnknown(types.ObjectType{AttrTypes: chainCertificateAttrTypes})
	} else {
		certs, err := securitycontent.ParseCertificateChainPEM([]byte(plan.CertificateChain.ValueString()))
		if err != nil {
			return // reported by ValidateConfig
		}
		resp.Diagnostics.Append(applyChainMetadata(&plan, certs)...)
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *keyPairCertificateChainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	r.upload(ctx, req.Plan.Get, &resp.State, &resp.Diagnostics)
}

func (r *keyPairCertificateChainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	r.upload(ctx, req.Plan.Get, &resp.State, &resp.Diagnostics)
}

// upload sends the planned chain; SAP replaces whatever chain the key pair
// had, so create and update are the same request.
func (r *keyPairCertificateChainResource) upload(ctx context.Context, getPlan func(context.Context, any) diag.Diagnostics, state interface {
	Set(context.Context, any) diag.Diagnostics
}, diags *diag.Diagnostics) {
	var plan keyPairCertificateChainModel
	diags.Append(getPlan(ctx, &plan)...)
	if diags.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, diags)
	if !ok {
		return
	}
	content := []byte(plan.CertificateChain.ValueString())
	certs, err := securitycontent.ParseCertificateChainPEM(content)
	if err != nil {
		diags.AddAttributeError(pathRoot("certificate_chain"), "Invalid certificate chain", diagnosticDetail(err))
		return
	}
	alias := plan.KeyPairAlias.ValueString()
	if err := client.UploadCertificateChain(ctx, alias, content); err != nil {
		diags.AddError("Failed to upload the certificate chain of SAP Integration Suite key pair "+alias, diagnosticDetail(err))
		return
	}
	plan.ID = types.StringValue(alias)
	diags.Append(applyChainMetadata(&plan, certs)...)
	diags.Append(state.Set(ctx, &plan)...)
}

func (r *keyPairCertificateChainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state keyPairCertificateChainModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}
	alias := state.KeyPairAlias.ValueString()
	remote, err := client.ExportCertificateChain(ctx, alias)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read the certificate chain of SAP Integration Suite key pair "+alias, diagnosticDetail(err))
		return
	}
	// Keep the configured text while SAP holds the same certificates; any
	// other chain, including the self-signed certificate of a key pair that
	// was regenerated, replaces it, so the next plan uploads the chain again.
	local, localErr := securitycontent.ParseCertificateChainPEM([]byte(state.CertificateChain.ValueString()))
	certs := local
	if localErr != nil || !sameCertificates(local, remote) {
		state.CertificateChain = types.StringValue(string(securitycontent.EncodeCertificateChainPEM(remote)))
		certs = remote
	}
	state.ID = types.StringValue(alias)
	resp.Diagnostics.Append(applyChainMetadata(&state, certs)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Delete leaves the chain on the key pair: SAP has no request that removes
// it. The key pair keeps the signed certificate until it is regenerated.
func (r *keyPairCertificateChainResource) Delete(_ context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning("Certificate chain left on the key pair",
		"SAP has no request that removes a certificate chain, so the key pair keeps the signed "+
			"certificate and its chain. Replace the key pair to return to a self-signed certificate.")
}

func (r *keyPairCertificateChainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("key_pair_alias"), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
}

// sameCertificates compares two chains by the certificates they hold,
// regardless of order and PEM formatting.
func sameCertificates(a, b []*x509.Certificate) bool {
	fingerprints := func(certs []*x509.Certificate) []string {
		out := make([]string, 0, len(certs))
		for _, m := range securitycontent.CertificateChainMetadata(certs) {
			out = append(out, m.SHA256Fingerprint)
		}
		sort.Strings(out)
		return out
	}
	fa, fb := fingerprints(a), fingerprints(b)
	if len(fa) != len(fb) {
		return false
	}
	for i := range fa {
		if fa[i] != fb[i] {
			return false
		}
	}
	return true
}

func applyChainMetadata(m *keyPairCertificateChainModel, certs []*x509.Certificate) diag.Diagnostics {
	meta := securitycontent.CertificateChainMetadata(certs)
	values := make([]attr.Value, 0, len(meta))
	for _, c := range meta {
		values = append(values, types.ObjectValueMust(chainCertificateAttrTypes, map[string]attr.Value{
			"subject_dn":    types.StringValue(c.SubjectDN),
			"issuer_dn":     types.StringValue(c.IssuerDN),
			"serial_number": types.StringValue(c.SerialNumber),
			"not_before":    types.StringValue(c.NotBefore.UTC().Format(time.RFC3339)),
			"not_after":     types.StringValue(c.NotAfter.UTC().Format(time.RFC3339)),
			"sha256":        types.StringValue(c.SHA256Fingerprint),
		}))
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: chainCertificateAttrTypes}, values)
	m.Certificates = list
	if len(meta) > 0 {
		m.CertificateSHA256 = types.StringValue(meta[0].SHA256Fingerprint)
	}
	return diags
}
