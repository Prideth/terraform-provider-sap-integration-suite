package securitycontent

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"
)

// The fixtures are public keys gpg 2.4 generated for these tests; the
// fingerprints are the ones gpg reported.
func TestParsePGPKey(t *testing.T) {
	for _, c := range []struct{ file, fingerprint string }{
		{"pgp-public-rsa.asc", "D8DE705CC863FED0B2A56EB6AB8AF9015AD46E69"},
		{"pgp-public-ed25519.asc", "1C8F04EA151EB299D9AF2C4C8B9EE279220B519E"},
	} {
		info, err := ParsePGPKey(readFixture(t, c.file))
		if err != nil {
			t.Fatalf("%s: %v", c.file, err)
		}
		if info.Fingerprint != c.fingerprint || info.KeyID != c.fingerprint[24:] || info.Secret {
			t.Errorf("%s: %+v, want fingerprint %s", c.file, info, c.fingerprint)
		}
	}
}

// rearmor wraps packets as an armored block of the given kind.
func rearmor(kind string, packets []byte) []byte {
	return []byte("-----BEGIN PGP " + kind + " KEY BLOCK-----\nComment: test\n\n" +
		base64.StdEncoding.EncodeToString(packets) + "\n=AAAA\n-----END PGP " + kind + " KEY BLOCK-----\n")
}

func TestParsePGPKey_SecretAndRefusals(t *testing.T) {
	_, packets, err := dearmorPGP(readFixture(t, "pgp-public-rsa.asc"))
	if err != nil {
		t.Fatal(err)
	}
	tag, body, _, err := nextPGPPacket(packets)
	if err != nil || tag != pgpTagPublicKey {
		t.Fatalf("first packet tag %d, %v", tag, err)
	}
	// A secret key packet starts with the public key part; the secret
	// fields that follow do not change the fingerprint.
	secretBody := append(append([]byte{}, body...), 0, 1, 2, 3)
	secret := append([]byte{0xc0 | pgpTagSecretKey, 0xff, 0, 0, byte(len(secretBody) >> 8), byte(len(secretBody))}, secretBody...)
	info, err := ParsePGPKey(rearmor("PRIVATE", secret))
	if err != nil || !info.Secret || info.Fingerprint != "D8DE705CC863FED0B2A56EB6AB8AF9015AD46E69" {
		t.Errorf("secret key: %+v, %v", info, err)
	}

	two := append(append([]byte{}, packets...), packets...)
	fixture := readFixture(t, "pgp-public-rsa.asc")
	noEnd := fixture[:bytes.Index(fixture, []byte("-----END"))]
	for name, c := range map[string]struct {
		in   []byte
		want string
	}{
		"two keys":          {rearmor("PUBLIC", two), "more than one primary key"},
		"secret as public":  {rearmor("PUBLIC", secret), "other kind"},
		"public as private": {rearmor("PRIVATE", packets), "other kind"},
		"no armor":          {[]byte("just text"), "no ASCII-armored"},
		"no end":            {noEnd, "no END line"},
		"truncated":         {rearmor("PUBLIC", packets[:20]), "truncated"},
	} {
		if _, err := ParsePGPKey(c.in); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}
