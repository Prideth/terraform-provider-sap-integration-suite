package securitycontent

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"

	v2 "github.com/Prideth/terraform-provider-sap-integration-suite/internal/client/odata/v2"
)

// The certificate chain of a key pair. SAP Help says the Key Pair API can
// import and export a key pair's certificate chain and create a certificate
// signing request, but the requests are documented only in the API
// specification on the Business Accelerator Hub. Everything here comes from
// the tenant $metadata and was verified on a tenant on 2026-10-04:
//
//   - The CSR is GET KeystoreEntries('<hex>')/SigningRequest/$value
//     (application/pkcs10, PEM). The CertificateSigningRequests set declared
//     in $metadata answered 404.
//   - The chain is PUT CertificateChainResources('<hex>')/$value. Without
//     fingerprintVerified=true SAP answered 409 "notImported", as for a
//     certificate import; with it and returnKeystoreEntries=false, 204.
//     SAP took PEM in any order, without the root, the leaf alone, DER and
//     PKCS#7, as application/pkix-cert and other content types. A
//     replacement is the same request; a certificate for another key
//     answered 400 "The public key of the CA Reply is different from the
//     public key of the keystore entry".
//   - The export is GET KeystoreEntries('<hex>')/ChainResource/$value, a
//     PEM-wrapped PKCS#7 bundle; a GET of CertificateChainResources answered
//     400. After the upload the key pair's entry shows the signed
//     certificate: its issuer, serial number and validity change.
//   - SAP has no request that removes a chain.
const (
	certificateChainResourcesEntitySet = "CertificateChainResources"
	chainResourceNavigation            = "ChainResource"
	signingRequestNavigation           = "SigningRequest"
)

func keystoreEntryNavigationPath(alias, navigation string) string {
	return keystoreEntryPath(v2.EncodeUTF8Hex(alias)) + "/" + navigation
}

// GetCertificateSigningRequest returns the PEM certificate signing request
// for a key pair's public key, to be signed by a certificate authority.
func (c *Client) GetCertificateSigningRequest(ctx context.Context, alias string) ([]byte, error) {
	return c.odata.Get(ctx, keystoreEntryNavigationPath(alias, signingRequestNavigation)+"/$value")
}

// UploadCertificateChain sends a key pair's signed certificate with its
// issuing certificates, in PEM, and replaces the chain SAP holds. It
// confirms the fingerprint like ImportCertificate does: the chain in a
// configuration is one its author chose to trust.
func (c *Client) UploadCertificateChain(ctx context.Context, alias string, pemContent []byte) error {
	path := v2.BuildPath(certificateChainResourcesEntitySet, v2.KeyPredicate(v2.EncodeUTF8Hex(alias)), "") + "/$value?" + certificateImportQuery
	_, err := c.odata.PutRaw(ctx, path, certificatePEMContentType, pemContent)
	return err
}

// ExportCertificateChain returns the chain SAP holds for a key pair, leaf
// first (see OrderCertificateChain). A key pair without an uploaded chain
// returns its self-signed certificate alone.
func (c *Client) ExportCertificateChain(ctx context.Context, alias string) ([]*x509.Certificate, error) {
	body, err := c.odata.Get(ctx, keystoreEntryNavigationPath(alias, chainResourceNavigation)+"/$value")
	if err != nil {
		return nil, err
	}
	certs, err := ParsePKCS7Certificates(body)
	if err != nil {
		return nil, err
	}
	return OrderCertificateChain(certs), nil
}

// ParsePKCS7Certificates returns the certificates of a degenerate PKCS#7
// SignedData bundle (RFC 2315), in DER or wrapped in a PEM "PKCS7" block,
// as SAP's chain export and openssl crl2pkcs7 produce it.
func ParsePKCS7Certificates(data []byte) ([]*x509.Certificate, error) {
	if block, _ := pem.Decode(data); block != nil {
		data = block.Bytes
	}
	var contentInfo struct {
		ContentType asn1.ObjectIdentifier
		Content     asn1.RawValue `asn1:"explicit,tag:0"`
	}
	if _, err := asn1.Unmarshal(data, &contentInfo); err != nil {
		return nil, fmt.Errorf("securitycontent: certificate chain is not PKCS#7: %w", err)
	}
	var signedData struct {
		Version          int
		DigestAlgorithms asn1.RawValue
		ContentInfo      asn1.RawValue
		Certificates     asn1.RawValue `asn1:"optional,tag:0"`
		CRLs             asn1.RawValue `asn1:"optional,tag:1"`
		SignerInfos      asn1.RawValue
	}
	if _, err := asn1.Unmarshal(contentInfo.Content.Bytes, &signedData); err != nil {
		return nil, fmt.Errorf("securitycontent: certificate chain is not PKCS#7 signed data: %w", err)
	}
	certs, err := x509.ParseCertificates(signedData.Certificates.Bytes)
	if err != nil {
		return nil, fmt.Errorf("securitycontent: parsing the certificates of the chain: %w", err)
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("securitycontent: the certificate chain holds no certificate")
	}
	return certs, nil
}

// ParseCertificateChainPEM parses a PEM bundle of certificates and returns
// them leaf first. It refuses any block that is not a certificate, above
// all a private key: the chain is public data, and a key pasted into it by
// mistake must not be sent anywhere.
func ParseCertificateChainPEM(pemContent []byte) ([]*x509.Certificate, error) {
	var certs []*x509.Certificate
	rest := pemContent
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("securitycontent: the certificate chain contains a %q block; it may only contain CERTIFICATE blocks", block.Type)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("securitycontent: parsing certificate %d of the chain: %w", len(certs)+1, err)
		}
		certs = append(certs, cert)
	}
	if len(bytes.TrimSpace(rest)) > 0 {
		return nil, fmt.Errorf("securitycontent: the certificate chain contains text that is not PEM")
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("securitycontent: the certificate chain holds no PEM certificate")
	}
	return OrderCertificateChain(certs), nil
}

// OrderCertificateChain sorts a chain leaf first, each certificate followed
// by its issuer. The leaf is the certificate that issued none of the
// others; certificates that do not belong to the path follow in their
// original order.
func OrderCertificateChain(certs []*x509.Certificate) []*x509.Certificate {
	if len(certs) < 2 {
		return certs
	}
	used := make(map[*x509.Certificate]bool, len(certs))
	var ordered []*x509.Certificate
	for c := chainLeaf(certs); c != nil && !used[c]; c = unusedIssuer(certs, c, used) {
		ordered = append(ordered, c)
		used[c] = true
	}
	for _, c := range certs {
		if !used[c] {
			ordered = append(ordered, c)
		}
	}
	return ordered
}

func selfSigned(c *x509.Certificate) bool { return bytes.Equal(c.RawSubject, c.RawIssuer) }

// chainLeaf returns the first certificate that issued none of the others.
func chainLeaf(certs []*x509.Certificate) *x509.Certificate {
	for _, c := range certs {
		if !issuedAnother(certs, c) {
			return c
		}
	}
	return nil
}

func issuedAnother(certs []*x509.Certificate, c *x509.Certificate) bool {
	for _, o := range certs {
		if o != c && !selfSigned(o) && bytes.Equal(o.RawIssuer, c.RawSubject) {
			return true
		}
	}
	return false
}

// unusedIssuer returns the issuer of c among the certificates not placed
// yet, or nil at the end of the path.
func unusedIssuer(certs []*x509.Certificate, c *x509.Certificate, used map[*x509.Certificate]bool) *x509.Certificate {
	if selfSigned(c) {
		return nil
	}
	for _, o := range certs {
		if !used[o] && bytes.Equal(o.RawSubject, c.RawIssuer) {
			return o
		}
	}
	return nil
}

// CertificateChainMetadata describes each certificate of a chain, in the
// order given.
func CertificateChainMetadata(certs []*x509.Certificate) []CertificateMetadata {
	out := make([]CertificateMetadata, 0, len(certs))
	for _, c := range certs {
		out = append(out, *certificateMetadata(c))
	}
	return out
}

// EncodeCertificateChainPEM writes certificates as one PEM bundle.
func EncodeCertificateChainPEM(certs []*x509.Certificate) []byte {
	var b bytes.Buffer
	for _, c := range certs {
		_ = pem.Encode(&b, &pem.Block{Type: "CERTIFICATE", Bytes: c.Raw})
	}
	return b.Bytes()
}

// SameCertificateRequest reports whether two PEM certificate signing
// requests ask for the same thing: the same subject and public key. SAP
// builds a key pair's CSR when it is read; after a chain upload it signed it
// with another algorithm (SHA-512 before, SHA-256 after, tenant run of
// 2026-10-04), so the text changed while the request did not.
func SameCertificateRequest(a, b []byte) bool {
	parse := func(p []byte) *x509.CertificateRequest {
		block, _ := pem.Decode(p)
		if block == nil {
			return nil
		}
		csr, err := x509.ParseCertificateRequest(block.Bytes)
		if err != nil {
			return nil
		}
		return csr
	}
	ca, cb := parse(a), parse(b)
	return ca != nil && cb != nil &&
		bytes.Equal(ca.RawSubject, cb.RawSubject) && bytes.Equal(ca.RawSubjectPublicKeyInfo, cb.RawSubjectPublicKeyInfo)
}
