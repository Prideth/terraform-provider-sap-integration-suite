package securitycontent

import (
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"time"
)

// CertificateMetadata is this provider's own, locally-derived view of an
// X.509 certificate's identity: subject/issuer distinguished names, serial
// number, validity period, and a canonical SHA-256 fingerprint of the
// certificate's DER bytes. Every field here is computed by parsing the
// certificate itself with Go's standard library, never by trusting an
// unconfirmed SAP-returned property name — SAP's own KeystoreEntries
// example response is truncated ("...."), and while its surrounding prose
// mentions "Subject DN and Issuer DN" existing as properties, it never
// gives their exact JSON field names or casing (see KeystoreEntry).
// Deriving this locally from the confirmed-retrievable certificate bytes
// sidesteps that gap entirely.
type CertificateMetadata struct {
	SHA256Fingerprint string
	SubjectDN         string
	IssuerDN          string
	SerialNumber      string
	NotBefore         time.Time
	NotAfter          time.Time
}

// ParseCertificatePEM parses a single PEM-encoded X.509 certificate and
// derives CertificateMetadata from it. Returns an error if pemContent is
// not a well-formed PEM certificate block.
func ParseCertificatePEM(pemContent []byte) (*CertificateMetadata, error) {
	block, _ := pem.Decode(pemContent)
	if block == nil {
		return nil, fmt.Errorf("securitycontent: content is not PEM-encoded")
	}
	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("securitycontent: PEM block type is %q, want CERTIFICATE", block.Type)
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("securitycontent: parsing X.509 certificate: %w", err)
	}

	return certificateMetadata(cert), nil
}

// certificateMetadata derives CertificateMetadata from a parsed certificate.
func certificateMetadata(cert *x509.Certificate) *CertificateMetadata {
	sum := sha256.Sum256(cert.Raw)
	return &CertificateMetadata{
		SHA256Fingerprint: hex.EncodeToString(sum[:]),
		SubjectDN:         cert.Subject.String(),
		IssuerDN:          cert.Issuer.String(),
		SerialNumber:      cert.SerialNumber.String(),
		NotBefore:         cert.NotBefore,
		NotAfter:          cert.NotAfter,
	}
}
