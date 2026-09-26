package samples

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	envDownload = "SAP_SAMPLES_DOWNLOAD"
	envCache    = "SAP_SAMPLES_CACHE"
	maxBytes    = 32 << 20
)

// Get returns the verified bytes of the named sample, downloading it into
// the cache first when downloading is allowed. It skips the test when the
// sample is not cached and downloading is off or fails, and fails the test
// when the bytes do not match the pinned hash.
func Get(t testing.TB, name string) []byte {
	t.Helper()
	s, ok := Lookup(name)
	if !ok {
		t.Fatalf("samples: no sample named %q", name)
	}
	path := filepath.Join(cacheDir(t), s.SHA256+".zip")
	if data, err := os.ReadFile(path); err == nil { //nolint:gosec // G304: path is the test cache directory joined with a pinned SHA-256 from the catalog
		verify(t, s, data)
		return data
	}
	if os.Getenv(envDownload) != "1" && os.Getenv("TF_ACC") != "1" {
		t.Skipf("sample %s is not cached; set %s=1 to download it from github.com/SAP-samples/%s", name, envDownload, s.Repo)
	}
	data, err := download(s)
	if err != nil {
		t.Skipf("sample %s could not be downloaded: %v", name, err)
	}
	verify(t, s, data)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Logf("samples: caching %s failed: %v", name, err)
	}
	return data
}

// URL is where a sample is downloaded from.
func (s Sample) URL() string {
	segments := strings.Split(s.Path, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	return fmt.Sprintf("https://raw.githubusercontent.com/SAP-samples/%s/%s/%s", s.Repo, s.Commit, strings.Join(segments, "/"))
}

func verify(t testing.TB, s Sample, data []byte) {
	t.Helper()
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != s.SHA256 {
		t.Fatalf("samples: %s has SHA-256 %s, want %s (pinned to %s@%s)", s.Name, got, s.SHA256, s.Repo, s.Commit)
	}
}

func cacheDir(t testing.TB) string {
	t.Helper()
	dir := os.Getenv(envCache)
	if dir == "" {
		base, err := os.UserCacheDir()
		if err != nil {
			base = os.TempDir()
		}
		dir = filepath.Join(base, "terraform-provider-sap-integration-suite", "samples")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil { //nolint:gosec // G703: dir is the developer's own cache location (SAP_SAMPLES_CACHE or the user cache directory)
		t.Fatalf("samples: creating cache %s: %v", dir, err)
	}
	return dir
}

func download(s Sample) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.URL(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req) //nolint:gosec // G107: fixed raw.githubusercontent.com URL from the pinned catalog
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", s.URL(), resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, maxBytes))
}
