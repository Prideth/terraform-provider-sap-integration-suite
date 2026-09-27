package provider

import "strings"

// parseDistinguishedName splits a distinguished name such as
// "CN=tfacc, OU=Integration, O=Example\, Inc., C=DE" into attribute type and
// value pairs. Commas and plus signs separate relative names unless escaped
// with a backslash or inside double quotes; attribute types are returned in
// upper case. Values keep their characters with escapes removed.
func parseDistinguishedName(dn string) map[string]string {
	out := map[string]string{}
	var parts []string
	var cur strings.Builder
	escaped, quoted := false, false
	for _, r := range dn {
		switch {
		case escaped:
			cur.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"':
			quoted = !quoted
		case (r == ',' || r == '+' || r == ';') && !quoted:
			parts = append(parts, cur.String())
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	parts = append(parts, cur.String())
	for _, p := range parts {
		key, value, ok := strings.Cut(p, "=")
		if !ok {
			continue
		}
		key = strings.ToUpper(strings.TrimSpace(key))
		if _, dup := out[key]; dup || key == "" {
			continue // the first occurrence wins, as the leading attribute
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}
