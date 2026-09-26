package samples

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// ExportArtifact is one artifact of a package export.
type ExportArtifact struct {
	ID          string // the export's resource ID, also the <id>_content file name
	Type        string // resourceType, for example "IFlow" or "ContentPackage"
	DisplayName string
	Version     string
	// Content is the artifact ZIP, empty for the package itself.
	Content []byte
}

// ExportArtifacts lists the artifacts of a package export from the Design
// UI: resources.cnt (base64 JSON) describes them, and each one is stored as
// <id>_content.
func ExportArtifacts(export []byte) ([]ExportArtifact, error) {
	files, err := zipFiles(export)
	if err != nil {
		return nil, err
	}
	cnt, ok := files["resources.cnt"]
	if !ok {
		return nil, fmt.Errorf("samples: export has no resources.cnt")
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(cnt)))
	if err != nil {
		return nil, fmt.Errorf("samples: resources.cnt is not base64: %w", err)
	}
	var doc struct {
		Resources []struct {
			ID              string `json:"id"`
			ResourceType    string `json:"resourceType"`
			DisplayName     string `json:"displayName"`
			SemanticVersion string `json:"semanticVersion"`
		} `json:"resources"`
	}
	if err := json.Unmarshal(decoded, &doc); err != nil {
		return nil, fmt.Errorf("samples: resources.cnt: %w", err)
	}
	artifacts := make([]ExportArtifact, 0, len(doc.Resources))
	for _, r := range doc.Resources {
		artifacts = append(artifacts, ExportArtifact{
			ID: r.ID, Type: r.ResourceType, DisplayName: r.DisplayName, Version: r.SemanticVersion,
			Content: files[r.ID+"_content"],
		})
	}
	return artifacts, nil
}

var symbolicNameHeader = regexp.MustCompile(`(?m)^Bundle-SymbolicName: [^;\n]*`)
var nameHeader = regexp.MustCompile(`(?m)^Bundle-Name: .*`)

// WithBundleID returns a copy of an artifact ZIP whose Bundle-SymbolicName
// and Bundle-Name are id, so the artifact can be created under a unique ID
// in a shared tenant. Wrapped manifest lines are joined first and the
// manifest is wrapped again at 72 bytes, as the JAR specification requires.
func WithBundleID(content []byte, id string) ([]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}
	var out bytes.Buffer
	writer := zip.NewWriter(&out)
	for _, f := range reader.File {
		data, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		if f.Name == "META-INF/MANIFEST.MF" {
			text := unfoldManifest(string(data))
			// Message mappings also name themselves in Provide-Capability
			// ("messagemapping.<ID>;version:Version=...").
			if old := strings.TrimPrefix(symbolicNameHeader.FindString(text), "Bundle-SymbolicName: "); old != "" {
				text = strings.ReplaceAll(text, "messagemapping."+old+";", "messagemapping."+id+";")
			}
			text = symbolicNameHeader.ReplaceAllString(text, "Bundle-SymbolicName: "+id)
			text = nameHeader.ReplaceAllString(text, "Bundle-Name: "+id)
			data = []byte(foldManifest(text))
		}
		w, err := writer.Create(f.Name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func unfoldManifest(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.ReplaceAll(text, "\n ", "")
}

func foldManifest(text string) string {
	var b strings.Builder
	for _, line := range strings.Split(strings.TrimRight(text, "\n"), "\n") {
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

func zipFiles(content []byte) (map[string][]byte, error) {
	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("samples: not a ZIP archive: %w", err)
	}
	files := make(map[string][]byte, len(reader.File))
	for _, f := range reader.File {
		data, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		files[f.Name] = data
	}
	return files, nil
}

// FileNames lists the entries of a ZIP archive.
func FileNames(content []byte) ([]string, error) {
	files, err := zipFiles(content)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	return names, nil
}

func readZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(io.LimitReader(rc, maxBytes))
}
