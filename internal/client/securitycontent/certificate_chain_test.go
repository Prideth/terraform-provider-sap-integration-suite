package securitycontent

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// The fixtures are a throw-away root, intermediate and leaf made with
// openssl (crl2pkcs7 for the bundle), the way SAP's export and a CA's reply
// look. They hold public certificates only.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func subjects(certs []*x509.Certificate) string {
	var names []string
	for _, c := range certs {
		names = append(names, c.Subject.CommonName)
	}
	return strings.Join(names, ",")
}

func TestParsePKCS7Certificates(t *testing.T) {
	certs, err := ParsePKCS7Certificates(readFixture(t, "chain-leaf-intermediate-root.p7c"))
	if err != nil {
		t.Fatal(err)
	}
	if got := subjects(OrderCertificateChain(certs)); got != "tfacc-test-leaf,tfacc-test-intermediate,tfacc-test-root" {
		t.Errorf("chain = %s", got)
	}
	if _, err := ParsePKCS7Certificates([]byte("not a bundle")); err == nil {
		t.Error("garbage parsed as PKCS#7")
	}
}

func TestParseCertificateChainPEM(t *testing.T) {
	chain := readFixture(t, "chain-leaf-intermediate-root.pem")
	certs, err := ParseCertificateChainPEM(chain)
	if err != nil {
		t.Fatal(err)
	}
	want := "tfacc-test-leaf,tfacc-test-intermediate,tfacc-test-root"
	if got := subjects(certs); got != want {
		t.Errorf("chain = %s, want %s", got, want)
	}

	// Root first, as SAP lists it, comes back leaf first.
	reversed := EncodeCertificateChainPEM([]*x509.Certificate{certs[2], certs[1], certs[0]})
	again, err := ParseCertificateChainPEM(reversed)
	if err != nil {
		t.Fatal(err)
	}
	if got := subjects(again); got != want {
		t.Errorf("reversed chain = %s, want %s", got, want)
	}

	for name, bad := range map[string]string{
		"private key": string(chain) + "-----BEGIN PRIVATE KEY-----\nMIIB\n-----END PRIVATE KEY-----\n",
		"empty":       "",
		"no PEM":      "just text",
		"trailing":    string(chain) + "trailing text",
	} {
		if _, err := ParseCertificateChainPEM([]byte(bad)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
}

func TestClient_CertificateChainRequests(t *testing.T) {
	p7 := readFixture(t, "chain-leaf-intermediate-root.p7c")
	var putPath, putType string
	var putBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/KeystoreEntries('6b6579')/SigningRequest/$value":
			_, _ = w.Write([]byte("-----BEGIN CERTIFICATE REQUEST-----\nMIIB\n-----END CERTIFICATE REQUEST-----\n"))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/KeystoreEntries('6b6579')/ChainResource/$value":
			_, _ = w.Write(p7)
		case r.Method == http.MethodPut:
			putPath = r.URL.Path + "?" + r.URL.RawQuery
			putType = r.Header.Get("Content-Type")
			putBody, _ = io.ReadAll(r.Body)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	c := New(http.DefaultClient, server.URL)
	ctx := context.Background()

	csr, err := c.GetCertificateSigningRequest(ctx, "key")
	if err != nil || !strings.Contains(string(csr), "CERTIFICATE REQUEST") {
		t.Fatalf("CSR = %q, %v", csr, err)
	}
	if err := c.UploadCertificateChain(ctx, "key", []byte("PEM")); err != nil {
		t.Fatal(err)
	}
	if putPath != "/api/v1/CertificateChainResources('6b6579')/$value?fingerprintVerified=true&returnKeystoreEntries=false" ||
		putType != "application/pkix-cert" || string(putBody) != "PEM" {
		t.Errorf("upload = %s, %s, %q", putPath, putType, putBody)
	}
	certs, err := c.ExportCertificateChain(ctx, "key")
	if err != nil {
		t.Fatal(err)
	}
	if got := subjects(certs); got != "tfacc-test-leaf,tfacc-test-intermediate,tfacc-test-root" {
		t.Errorf("exported chain = %s", got)
	}
}

func TestSameCertificateRequest(t *testing.T) {
	newCSR := func(key *rsa.PrivateKey, cn string, alg x509.SignatureAlgorithm) []byte {
		der, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
			Subject: pkix.Name{CommonName: cn, Country: []string{"DE"}}, SignatureAlgorithm: alg}, key)
		if err != nil {
			t.Fatal(err)
		}
		return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: der})
	}
	key, _ := rsa.GenerateKey(rand.Reader, 2048)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)
	sha512 := newCSR(key, "tfacc", x509.SHA512WithRSA)
	if !SameCertificateRequest(sha512, newCSR(key, "tfacc", x509.SHA256WithRSA)) {
		t.Error("another signature algorithm made the request differ")
	}
	if SameCertificateRequest(sha512, newCSR(other, "tfacc", x509.SHA512WithRSA)) {
		t.Error("another key counted as the same request")
	}
	if SameCertificateRequest(sha512, newCSR(key, "tfacc-renewed", x509.SHA512WithRSA)) {
		t.Error("another subject counted as the same request")
	}
	if SameCertificateRequest(sha512, []byte("not a CSR")) {
		t.Error("garbage counted as the same request")
	}
}
