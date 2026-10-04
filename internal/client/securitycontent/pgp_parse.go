package securitycontent

import (
	"bufio"
	"bytes"
	"crypto/sha1" // #nosec G505 -- the OpenPGP v4 fingerprint is defined as SHA-1 (RFC 4880, 12.2); it identifies a key, it secures nothing
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// PGPKeyInfo identifies the one key of an ASCII-armored keyring, derived
// locally so that a plan can show the key ID before SAP is called.
type PGPKeyInfo struct {
	// Fingerprint is the v4 fingerprint, 40 upper-case hex digits, as SAP
	// reports it.
	Fingerprint string
	// KeyID is the last 16 hex digits of the fingerprint, SAP's key of the
	// PgpKeyEntries set.
	KeyID string
	// Secret is true for a private key block.
	Secret bool
}

const (
	pgpTagSecretKey = 5
	pgpTagPublicKey = 6
)

// ParsePGPKey reads an ASCII-armored PGP public or private key block and
// returns the identity of its primary key. It refuses a block with more or
// less than one primary key, and version 5 and 6 keys, whose fingerprints
// are computed differently (SAP's tenant was tested with version 4 keys).
func ParsePGPKey(armored []byte) (*PGPKeyInfo, error) {
	kind, data, err := dearmorPGP(armored)
	if err != nil {
		return nil, err
	}
	var info *PGPKeyInfo
	for len(data) > 0 {
		tag, body, rest, err := nextPGPPacket(data)
		if err != nil {
			return nil, err
		}
		data = rest
		if tag != pgpTagPublicKey && tag != pgpTagSecretKey {
			continue
		}
		if info != nil {
			return nil, fmt.Errorf("securitycontent: the PGP block holds more than one primary key; put each key into its own resource")
		}
		pub, err := pgpPublicKeyPart(body)
		if err != nil {
			return nil, err
		}
		h := sha1.New() // #nosec G401 -- see the import
		h.Write([]byte{0x99, byte(len(pub) >> 8), byte(len(pub))})
		h.Write(pub)
		fp := strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
		info = &PGPKeyInfo{Fingerprint: fp, KeyID: fp[len(fp)-16:], Secret: tag == pgpTagSecretKey}
	}
	if info == nil {
		return nil, fmt.Errorf("securitycontent: the PGP block holds no key")
	}
	if info.Secret != (kind == "PRIVATE") {
		return nil, fmt.Errorf("securitycontent: the PGP %s KEY BLOCK holds a key of the other kind", kind)
	}
	return info, nil
}

// dearmorPGP decodes the first "-----BEGIN PGP PUBLIC/PRIVATE KEY BLOCK-----"
// of armored (RFC 4880, 6.2) and returns PUBLIC or PRIVATE and the packets.
func dearmorPGP(armored []byte) (string, []byte, error) {
	sc := bufio.NewScanner(bytes.NewReader(armored))
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	kind := ""
	inHeaders := false
	var b64 strings.Builder
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case kind == "":
			for _, k := range []string{"PUBLIC", "PRIVATE"} {
				if line == "-----BEGIN PGP "+k+" KEY BLOCK-----" {
					kind, inHeaders = k, true
				}
			}
		case strings.HasPrefix(line, "-----END PGP "):
			data, err := base64.StdEncoding.DecodeString(b64.String())
			if err != nil {
				return "", nil, fmt.Errorf("securitycontent: the PGP key block is not valid base64: %w", err)
			}
			return kind, data, nil
		case inHeaders:
			// Armor headers ("Comment: ...") end with an empty line.
			if line == "" {
				inHeaders = false
			} else if !strings.Contains(line, ":") {
				inHeaders = false
				b64.WriteString(line)
			}
		case strings.HasPrefix(line, "="):
			// The CRC-24 checksum line; base64 decoding checks the data.
		default:
			b64.WriteString(line)
		}
	}
	if kind == "" {
		return "", nil, fmt.Errorf("securitycontent: no ASCII-armored PGP PUBLIC KEY BLOCK or PRIVATE KEY BLOCK found")
	}
	return "", nil, fmt.Errorf("securitycontent: the PGP %s KEY BLOCK has no END line", kind)
}

// nextPGPPacket splits off the first packet (RFC 4880, 4.2).
func nextPGPPacket(data []byte) (tag int, body, rest []byte, err error) {
	if len(data) < 2 || data[0]&0x80 == 0 {
		return 0, nil, nil, fmt.Errorf("securitycontent: malformed PGP packet")
	}
	var length, hdr int
	if data[0]&0x40 != 0 { // new format
		tag = int(data[0] & 0x3f)
		switch o := int(data[1]); {
		case o < 192:
			length, hdr = o, 2
		case o < 224 && len(data) >= 3:
			length, hdr = ((o-192)<<8)+int(data[2])+192, 3
		case o == 255 && len(data) >= 6:
			length, hdr = int(data[2])<<24|int(data[3])<<16|int(data[4])<<8|int(data[5]), 6
		default:
			return 0, nil, nil, fmt.Errorf("securitycontent: unsupported PGP packet length")
		}
	} else { // old format
		tag = int(data[0]>>2) & 0x0f
		switch data[0] & 3 {
		case 0:
			length, hdr = int(data[1]), 2
		case 1:
			if len(data) < 3 {
				return 0, nil, nil, fmt.Errorf("securitycontent: malformed PGP packet")
			}
			length, hdr = int(data[1])<<8|int(data[2]), 3
		case 2:
			if len(data) < 5 {
				return 0, nil, nil, fmt.Errorf("securitycontent: malformed PGP packet")
			}
			length, hdr = int(data[1])<<24|int(data[2])<<16|int(data[3])<<8|int(data[4]), 5
		default:
			return 0, nil, nil, fmt.Errorf("securitycontent: unsupported PGP packet length")
		}
	}
	if length < 0 || hdr+length > len(data) {
		return 0, nil, nil, fmt.Errorf("securitycontent: truncated PGP packet")
	}
	return tag, data[hdr : hdr+length], data[hdr+length:], nil
}

// pgpPublicKeyPart returns the public key material at the start of a v4
// public or secret key packet body (RFC 4880, 5.5.2; RFC 6637 and the
// EdDSA draft for the curve algorithms): version, creation time, algorithm
// and the algorithm's public fields.
func pgpPublicKeyPart(body []byte) ([]byte, error) {
	if len(body) < 6 {
		return nil, fmt.Errorf("securitycontent: truncated PGP key packet")
	}
	if body[0] != 4 {
		return nil, fmt.Errorf("securitycontent: PGP key version %d is not supported; use a version 4 key", body[0])
	}
	pos := 6
	mpi := func() error {
		if pos+2 > len(body) {
			return fmt.Errorf("securitycontent: truncated PGP key packet")
		}
		bits := int(body[pos])<<8 | int(body[pos+1])
		pos += 2 + (bits+7)/8
		if pos > len(body) {
			return fmt.Errorf("securitycontent: truncated PGP key packet")
		}
		return nil
	}
	field := func() error { // one length octet, then that many octets
		if pos >= len(body) {
			return fmt.Errorf("securitycontent: truncated PGP key packet")
		}
		pos += 1 + int(body[pos])
		if pos > len(body) {
			return fmt.Errorf("securitycontent: truncated PGP key packet")
		}
		return nil
	}
	var steps []func() error
	switch body[5] {
	case 1, 2, 3: // RSA: n, e
		steps = []func() error{mpi, mpi}
	case 16: // ElGamal: p, g, y
		steps = []func() error{mpi, mpi, mpi}
	case 17: // DSA: p, q, g, y
		steps = []func() error{mpi, mpi, mpi, mpi}
	case 19, 22: // ECDSA, EdDSA: curve OID, point
		steps = []func() error{field, mpi}
	case 18: // ECDH: curve OID, point, KDF parameters
		steps = []func() error{field, mpi, field}
	default:
		return nil, fmt.Errorf("securitycontent: PGP public key algorithm %d is not supported", body[5])
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	return body[:pos], nil
}
