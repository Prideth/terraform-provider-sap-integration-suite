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
