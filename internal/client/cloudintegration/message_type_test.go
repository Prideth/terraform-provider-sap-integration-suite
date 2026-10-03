package cloudintegration

import (
	"archive/zip"
	"bytes"
	"testing"
)

func messageTypeBundle(t *testing.T, attributes string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		"META-INF/MANIFEST.MF":                         "Manifest-Version: 1.0\nSAP-BundleType: MessageType\n",
		"src/main/resources/additionalAttributes.json": attributes,
	} {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// SAP's generated bundles name the data type twice; the cases follow the
// package export of 2026-10-03.
func TestDataTypeUsedInBundle(t *testing.T) {
	cases := []struct {
		name, attributes, want string
	}{
		{
			name:       "own data type: ID and name are the same",
			attributes: `{"dtUniqueIdinMT":"Order","dtUsedinMT":"Order","BundleSymbolicName":"OrderMessage"}`,
			want:       "Order",
		},
		{
			name:       "standard data type: the ID carries a hash suffix",
			attributes: `{"dtUniqueIdinMT":"ExchangeLogData_0123456789abcdef0123456789abcdef","dtUsedinMT":"ExchangeLogData"}`,
			want:       "ExchangeLogData_0123456789abcdef0123456789abcdef",
		},
		{
			name:       "only the name",
			attributes: `{"dtUsedinMT":"Order"}`,
			want:       "Order",
		},
		{
			name:       "no data type",
			attributes: `{"BundleSymbolicName":"Empty"}`,
			want:       "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := dataTypeUsedInBundle(messageTypeBundle(t, tc.attributes))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestDataTypeUsedInBundle_notAZip(t *testing.T) {
	if _, err := dataTypeUsedInBundle([]byte("not a zip")); err == nil {
		t.Fatal("want an error for content that is not a ZIP archive")
	}
}
