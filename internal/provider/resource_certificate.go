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

// NewCertificateResource returns a fresh resource.Resource implementation
// for sapintegrationsuite_certificate.
func NewCertificateResource() resource.Resource {
	return &certificateResource{}
}

type certificateResource struct {
	client *securitycontent.Client
}

type certificateModel struct {
	ID                types.String `tfsdk:"id"`
	Alias             types.String `tfsdk:"alias"`
	Certificate       types.String `tfsdk:"certificate"`
	CertificateSHA256 types.String `tfsdk:"certificate_sha256"`
	SubjectDN         types.String `tfsdk:"subject_dn"`
	IssuerDN          types.String `tfsdk:"issuer_dn"`
	SerialNumber      types.String `tfsdk:"serial_number"`
	RuntimeLocationID types.String `tfsdk:"runtime_location_id"`
}

func (r *certificateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_certificate"
}

func (r *certificateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a trusted X.509 certificate (keystore entry) in the tenant keystore. " +
			"Backed by the public Security Content OData V2 API: PUT " +
			"CertificateResources('<hexalias>')/$value imports a certificate, and the same request " +
			"with update=true replaces the certificate of an existing alias. Both confirm the " +
			"certificate's fingerprint (fingerprintVerified=true), which the UI asks for when you " +
			"import a self-signed or otherwise untrusted certificate: listing a certificate here is " +
			"the decision to trust it, so compare certificate_sha256 with the fingerprint your " +
			"partner gave you before you apply. Creating fails if the alias already exists on the " +
			"tenant; import it instead. certificate content is X.509 public certificate data, " +
			"not a secret, and is never marked Sensitive — never store a private key here; SAP " +
			"generates and retains private key material separately and this provider never " +
			"requests or exposes it (see sapintegrationsuite_key_pair). Deleting this resource uses " +
			"SAP's documented keystore mass-deletion operation with exactly the one alias this " +
			"resource owns, never a caller-supplied list. See docs/guides/security-content.md, " +
			"including how this provider handles SAP-owned keystore entries.",
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
				Required: true,
				Description: "The keystore entry's alias. This is the entity's actual identity " +
					"(SAP's OData key is this alias' hex encoding, computed internally — you never " +
					"provide it), so it is RequiresReplace: SAP documents a separate, explicit " +
					"rename operation (PUT KeystoreEntries('<hex>')?renameAlias=<new>) this resource " +
					"does not use, to keep lifecycle behavior predictable.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"certificate": schema.StringAttribute{
				Required: true,
				Description: "The X.509 certificate in PEM format " +
					"(\"-----BEGIN CERTIFICATE-----...-----END CERTIFICATE-----\"). Not Sensitive: " +
					"public certificate content is not confidential. On refresh, if the certificate " +
					"actually deployed on the tenant is byte-identical in substance to this value " +
					"(compared via certificate_sha256, not raw text — see that attribute), this " +
					"provider keeps your own PEM text unchanged in state rather than replacing it " +
					"with SAP's own re-serialization, so differences in line wrapping or line endings " +
					"never produce a spurious plan diff. A genuinely different certificate on the " +
					"tenant does update this value, surfacing real drift.",
			},
			"certificate_sha256": schema.StringAttribute{
				Computed: true,
				Description: "The SHA-256 fingerprint (lowercase hex) of the certificate's DER " +
					"bytes, computed locally by this provider from the certificate content — not a " +
					"value SAP's API returns. Used internally to detect genuine certificate changes " +
					"independent of PEM text formatting; exposed because it is generally more useful " +
					"for policy checks and cross-referencing than comparing raw PEM text.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"subject_dn": schema.StringAttribute{
				Computed: true,
				Description: "The certificate's subject distinguished name, parsed locally from " +
					"the certificate content (Go's crypto/x509), not a raw SAP-returned field — see " +
					"docs/guides/security-content.md for why.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"issuer_dn": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate's issuer distinguished name, parsed locally from the certificate content.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"serial_number": schema.StringAttribute{
				Computed:    true,
				Description: "The certificate's serial number (decimal), parsed locally from the certificate content.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// ModifyPlan computes the fingerprint, subject, issuer and serial number of
// the planned certificate. They are derived locally from the PEM, so a plan
// that changes the certificate shows the new values instead of carrying the
// old ones over from state (which Terraform then reports as an inconsistent
// result after apply). If the PEM is not known yet, they become unknown.
func (r *certificateResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy
	}
	var plan certificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if plan.Certificate.IsUnknown() || plan.Certificate.IsNull() {
		plan.CertificateSHA256 = types.StringUnknown()
		plan.SubjectDN = types.StringUnknown()
		plan.IssuerDN = types.StringUnknown()
		plan.SerialNumber = types.StringUnknown()
	} else {
		meta, err := securitycontent.ParseCertificatePEM([]byte(plan.Certificate.ValueString()))
		if err != nil {
			return // reported by Create or Update with the attribute path
		}
		applyCertificateMetadata(&plan, meta)
	}
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *certificateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *certificateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan certificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	alias := plan.Alias.ValueString()
	content := []byte(plan.Certificate.ValueString())

	meta, err := securitycontent.ParseCertificatePEM(content)
	if err != nil {
		resp.Diagnostics.AddAttributeError(pathRoot("certificate"), "Invalid certificate", diagnosticDetail(err))
		return
	}

	if err := client.ImportCertificate(ctx, alias, content); err != nil {
		resp.Diagnostics.AddError("Failed to import SAP Integration Suite certificate", diagnosticDetail(err))
		return
	}

	plan.ID = types.StringValue(alias)
	applyCertificateMetadata(&plan, meta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *certificateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state certificateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, state.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	alias := state.Alias.ValueString()
	remotePEM, err := client.GetCertificate(ctx, alias)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read SAP Integration Suite certificate", diagnosticDetail(err))
		return
	}

	remoteMeta, err := securitycontent.ParseCertificatePEM(remotePEM)
	if err != nil {
		resp.Diagnostics.AddError("Failed to parse certificate returned by SAP Integration Suite", diagnosticDetail(err))
		return
	}

	// Only overwrite the practitioner's own PEM text with SAP's if the
	// certificate identity actually changed (compared via a canonical
	// fingerprint) — never merely because SAP re-serialized the same
	// certificate with different line wrapping or line endings. See the
	// "certificate" attribute's schema description.
	if localMeta, localErr := securitycontent.ParseCertificatePEM([]byte(state.Certificate.ValueString())); localErr != nil || localMeta.SHA256Fingerprint != remoteMeta.SHA256Fingerprint {
		state.Certificate = types.StringValue(string(remotePEM))
	}

	state.ID = types.StringValue(alias)
	applyCertificateMetadata(&state, remoteMeta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *certificateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan certificateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, ok := locatedClient(r.client, plan.RuntimeLocationID, &resp.Diagnostics)
	if !ok {
		return
	}

	alias := plan.Alias.ValueString()
	content := []byte(plan.Certificate.ValueString())

	meta, err := securitycontent.ParseCertificatePEM(content)
	if err != nil {
		resp.Diagnostics.AddAttributeError(pathRoot("certificate"), "Invalid certificate", diagnosticDetail(err))
		return
	}

	if err := client.UpdateCertificate(ctx, alias, content); err != nil {
		resp.Diagnostics.AddError("Failed to update SAP Integration Suite certificate", diagnosticDetail(err))
		return
	}

	plan.ID = types.StringValue(alias)
	applyCertificateMetadata(&plan, meta)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Delete uses SAP's documented keystore mass-deletion operation with
// exactly the one alias this resource owns — never a caller-supplied list
// of unrelated aliases, and never a guessed "DELETE /KeystoreEntries(...)"
// endpoint SAP does not document. If the alias belongs to an SAP-owned
// entry, SAP's own server-side protection rejects this call; this provider
// surfaces that error rather than attempting to detect ownership itself
// (no confirmed API field exists to do so — see
// docs/guides/security-content.md).
func (r *certificateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state certificateModel
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
		resp.Diagnostics.AddError("Failed to delete SAP Integration Suite certificate", diagnosticDetail(err))
	}
}

func (r *certificateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	loc, parts, err := splitLocatedImportID(req.ID, 1)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, pathRoot("alias"), parts[0])...)
	setImportedRuntimeLocation(ctx, loc, resp.State.SetAttribute, &resp.Diagnostics)
}

func applyCertificateMetadata(m *certificateModel, meta *securitycontent.CertificateMetadata) {
	m.CertificateSHA256 = types.StringValue(meta.SHA256Fingerprint)
	m.SubjectDN = types.StringValue(meta.SubjectDN)
	m.IssuerDN = types.StringValue(meta.IssuerDN)
	m.SerialNumber = types.StringValue(meta.SerialNumber)
}
