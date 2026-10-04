package securitycontent

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// importResults answers like the tenant did on 2026-10-04.
func importResults(results ...string) string {
	return `{"d":{"results":[` + strings.Join(results, ",") + `]}}`
}

func importResult(keyID, status, details string) string {
	return `{"Id":"` + keyID + `","KeyId":"` + keyID + `","Status":"` + status + `","StatusDetails":"` + details + `"}`
}

func TestClient_ImportPGPKeys(t *testing.T) {
	var answer string
	var got []string
	var passphrase, accept, contentType string
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = append(got, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodPut {
			passphrase, accept, contentType = r.Header.Get("Passphrase"), r.Header.Get("Accept"), r.Header.Get("Content-Type")
			body, _ = io.ReadAll(r.Body)
			_, _ = w.Write([]byte(answer))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"d":{"Id":"A1","KeyId":"A1","Type":"Public"}}`))
	}))
	defer server.Close()
	c := New(http.DefaultClient, server.URL)
	ctx := context.Background()

	answer = importResults(importResult("A1", "added", ""))
	r, err := c.ImportPGPPublicKey(ctx, []byte("ARMORED"))
	if err != nil || r.KeyID != "A1" {
		t.Fatalf("public import = %+v, %v", r, err)
	}
	if got[0] != "PUT /api/v1/PgpKeyringPublicResources('pubring')/$value" || passphrase != "" ||
		accept != "application/json" || contentType != "application/pgp-keys" || string(body) != "ARMORED" {
		t.Errorf("public import sent %v, passphrase %q, accept %q, type %q", got, passphrase, accept, contentType)
	}

	got = nil
	r, err = c.ImportPGPSecretKey(ctx, []byte("SECRET"), "s3cret")
	if err != nil || r.KeyID != "A1" {
		t.Fatalf("secret import = %+v, %v", r, err)
	}
	if got[0] != "PUT /api/v1/PgpKeyringSecretResources('secring')/$value" || passphrase != "s3cret" {
		t.Errorf("secret import sent %v with passphrase %q", got, passphrase)
	}

	// HTTP 200 with "not imported" is a failure.
	answer = importResults(importResult("A1", "not imported", "An entry with the same keyId already exists"))
	if _, err := c.ImportPGPPublicKey(ctx, []byte("ARMORED")); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Errorf("not imported: err = %v", err)
	}

	// Two keys: the added one is removed again.
	got = nil
	answer = importResults(importResult("A1", "added", ""), importResult("B2", "not imported", "exists"))
	if _, err := c.ImportPGPPublicKey(ctx, []byte("TWO")); err == nil || !strings.Contains(err.Error(), "exactly one") {
		t.Errorf("two keys: err = %v", err)
	}
	if len(got) != 2 || got[1] != "DELETE /api/v1/PgpKeyEntries('A1')" {
		t.Errorf("two keys: requests %v, want the added key deleted", got)
	}

	e, err := c.GetPGPKey(ctx, "A1")
	if err != nil || e.Type != "Public" || e.HasSecret() {
		t.Errorf("get = %+v, %v", e, err)
	}
}
