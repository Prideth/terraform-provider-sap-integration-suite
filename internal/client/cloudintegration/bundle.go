package cloudintegration

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// manifestPath is where every design-time artifact ZIP (integration flow,
// message mapping, script collection, value mapping) keeps its OSGi bundle
// manifest.
const manifestPath = "META-INF/MANIFEST.MF"

// BundleManifest holds the MANIFEST.MF headers of a design-time artifact ZIP
// that matter to this provider.
type BundleManifest struct {
	// SymbolicName is Bundle-SymbolicName without its directives
	// ("; singleton:=true"). A tenant test in September 2026 showed that SAP
	// rejects a content update whose SymbolicName differs from the one it
	// stored: "Could not update artifact of the package; due to change in
	// the Bundle-symbolicName".
	SymbolicName string
	Name         string
	Version      string
	// BundleType is SAP-BundleType, for example "IntegrationFlow" or
	// "MessageMapping".
	BundleType string
	// RuntimeProfile is SAP-RuntimeProfile, for example "iflmap"; empty
	// when the header is absent.
	RuntimeProfile string
	// Headers holds every header by name, continuation lines joined.
	Headers map[string]string
}

// ParseBundleManifest reads META-INF/MANIFEST.MF from an artifact ZIP.
//
// Manifest lines are at most 72 bytes; longer values continue on the next
// line, which starts with a single space (JAR file specification). SAP's own
// exports wrap even Bundle-SymbolicName this way, for example
// "Request_Employee_Dependants_-_Cloud_Connector_Expor" + "t", so the lines
// are joined before the headers are split.
func ParseBundleManifest(content []byte) (*BundleManifest, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: content is not a ZIP archive: %w", err)
	}
	var file *zip.File
	for _, f := range archive.File {
		if f.Name == manifestPath {
			file = f
			break
		}
	}
	if file == nil {
		return nil, fmt.Errorf("cloudintegration: content has no %s", manifestPath)
	}
	rc, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: opening %s: %w", manifestPath, err)
	}
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("cloudintegration: reading %s: %w", manifestPath, err)
	}

	headers := parseManifestHeaders(string(raw))
	symbolic := headers["Bundle-SymbolicName"]
	if i := strings.Index(symbolic, ";"); i >= 0 {
		symbolic = symbolic[:i]
	}
	return &BundleManifest{
		SymbolicName:   strings.TrimSpace(symbolic),
		Name:           headers["Bundle-Name"],
		Version:        headers["Bundle-Version"],
		BundleType:     headers["SAP-BundleType"],
		RuntimeProfile: headers["SAP-RuntimeProfile"],
		Headers:        headers,
	}, nil
}

// AlignBundleID returns content with Bundle-SymbolicName set to id, and the
// symbolic name the content had before.
//
// On create, SAP writes the artifact ID into Bundle-SymbolicName itself; a
// later content update whose Bundle-SymbolicName differs is rejected (tenant
// test, September 2026: 400 "Could not update artifact of the package; due
// to change in the Bundle-symbolicName" for an integration flow, 500
// BUNDLE_SYMBOLIC_NAME_CANNOT_BE_UPDATED for a message mapping). Aligning the
// uploaded copy makes a ZIP work whatever ID it was exported under.
//
// Content whose symbolic name already equals id, or that has no readable
// manifest, is returned unchanged. Otherwise only META-INF/MANIFEST.MF is
// rewritten: the symbolic name (its directives kept), and the name in
// Provide-Capability for message mappings. Every other entry is copied as
// it is, without recompressing.
func AlignBundleID(content []byte, id string) (aligned []byte, previous string, err error) {
	manifest, err := ParseBundleManifest(content)
	if err != nil || manifest.SymbolicName == id || manifest.SymbolicName == "" {
		return content, id, nil //nolint:nilerr // unreadable content is uploaded as it is; SAP reports the problem
	}

	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, "", err
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, f := range archive.File {
		if f.Name != manifestPath {
			if err := copyRawEntry(writer, f); err != nil {
				return nil, "", err
			}
			continue
		}
		text, err := rewriteManifestID(f, id)
		if err != nil {
			return nil, "", err
		}
		w, err := writer.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: f.Modified})
		if err != nil {
			return nil, "", err
		}
		if _, err := w.Write([]byte(text)); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return out.Bytes(), manifest.SymbolicName, nil
}

func copyRawEntry(writer *zip.Writer, f *zip.File) error {
	raw, err := f.OpenRaw()
	if err != nil {
		return err
	}
	header := f.FileHeader
	w, err := writer.CreateRaw(&header)
	if err != nil {
		return err
	}
	_, err = io.Copy(w, raw)
	return err
}

// rewriteManifestID returns the manifest text with the new symbolic name,
// wrapped at 72 bytes as the JAR specification requires.
func rewriteManifestID(f *zip.File, id string) (string, error) {
	rc, err := f.Open()
	if err != nil {
		return "", err
	}
	defer func() { _ = rc.Close() }()
	raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
	if err != nil {
		return "", err
	}
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n ", "")
	var lines []string
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
		switch {
		case strings.HasPrefix(line, "Bundle-SymbolicName:"):
			value := strings.TrimSpace(strings.TrimPrefix(line, "Bundle-SymbolicName:"))
			directives := ""
			if i := strings.Index(value, ";"); i >= 0 {
				directives = value[i:]
			}
			line = "Bundle-SymbolicName: " + id + directives
		case strings.HasPrefix(line, "Provide-Capability:") && strings.Contains(line, "messagemapping."):
			line = replaceCapabilityName(line, "messagemapping.", id)
		}
		lines = append(lines, line)
	}
	return foldManifestLines(lines), nil
}

// replaceCapabilityName swaps the name after prefix up to the next ";".
func replaceCapabilityName(line, prefix, id string) string {
	start := strings.Index(line, prefix)
	if start < 0 {
		return line
	}
	start += len(prefix)
	end := strings.Index(line[start:], ";")
	if end < 0 {
		return line
	}
	return line[:start] + id + line[start+end:]
}

func foldManifestLines(lines []string) string {
	var b strings.Builder
	for _, line := range lines {
		for first := true; ; first = false {
			limit := 72
			if !first {
				b.WriteString(" ")
				limit = 71
			}
			if len(line) <= limit {
				b.WriteString(line + "\r\n")
				break
			}
			b.WriteString(line[:limit] + "\r\n")
			line = line[limit:]
		}
	}
	return b.String() + "\r\n"
}

func parseManifestHeaders(text string) map[string]string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var lines []string
	for _, line := range strings.Split(text, "\n") {
		if strings.HasPrefix(line, " ") && len(lines) > 0 {
			lines[len(lines)-1] += line[1:]
			continue
		}
		lines = append(lines, line)
	}
	headers := make(map[string]string, len(lines))
	for _, line := range lines {
		name, value, ok := strings.Cut(line, ":")
		if !ok || name == "" {
			continue
		}
		headers[name] = strings.TrimSpace(value)
	}
	return headers
}
