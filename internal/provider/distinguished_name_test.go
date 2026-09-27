package provider

import (
	"reflect"
	"testing"
)

func TestParseDistinguishedName(t *testing.T) {
	for _, tc := range []struct {
		dn   string
		want map[string]string
	}{
		{"CN=tfacc-key-pair, C=DE", map[string]string{"CN": "tfacc-key-pair", "C": "DE"}},
		{"CN=a,OU=Integration,O=Example\\, Inc.,L=Walldorf,ST=BW,C=DE,EMAILADDRESS=x@example.com",
			map[string]string{"CN": "a", "OU": "Integration", "O": "Example, Inc.", "L": "Walldorf", "ST": "BW", "C": "DE", "EMAILADDRESS": "x@example.com"}},
		{`CN="Quoted, Name", c=de`, map[string]string{"CN": "Quoted, Name", "C": "de"}},
		{"CN=multi+OU=valued", map[string]string{"CN": "multi", "OU": "valued"}},
		{"", map[string]string{}},
	} {
		if got := parseDistinguishedName(tc.dn); !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%q: got %v, want %v", tc.dn, got, tc.want)
		}
	}
}
