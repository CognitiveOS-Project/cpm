package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
)

type mockRegistry struct {
	publishFunc     func(token string, req registry.PublishRequest) error
	publishSSHFunc  func(fingerprint, signature string, req registry.PublishRequest) error
	registerFunc    func(publicKey string) (*registry.RegisterResponse, error)
}

func (m *mockRegistry) Search(query string, opts registry.SearchOptions) (*registry.SearchResult, error) {
	return nil, nil
}
func (m *mockRegistry) GetMetadata(name, version string) (*registry.PatchMetadata, error) {
	return nil, nil
}
func (m *mockRegistry) GetVersions(name string) ([]registry.VersionInfo, error) {
	return nil, nil
}
func (m *mockRegistry) GetDependencies(name string) (*registry.DependencyTree, error) {
	return nil, nil
}
func (m *mockRegistry) Unlock(name, version, code string) error {
	return nil
}
func (m *mockRegistry) Download(name, version string, opts registry.DownloadOptions) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockRegistry) Publish(token string, req registry.PublishRequest) error {
	return m.publishFunc(token, req)
}
func (m *mockRegistry) PublishSSH(fingerprint, signature string, req registry.PublishRequest) error {
	if m.publishSSHFunc != nil {
		return m.publishSSHFunc(fingerprint, signature, req)
	}
	return nil
}
func (m *mockRegistry) RegisterPublicKey(publicKey string) (*registry.RegisterResponse, error) {
	if m.registerFunc != nil {
		return m.registerFunc(publicKey)
	}
	return nil, nil
}

func createTestCGP(t *testing.T, name, version string) string {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, fmt.Sprintf("%s-%s.cgp", name, version))

	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	manifest := archive.Manifest{
		Name:    name,
		Version: version,
	}
	mB, _ := json.Marshal(manifest)

	hdr := &tar.Header{
		Name: "cognitive.json",
		Mode: 0644,
		Size: int64(len(mB)),
	}
	_ = tw.WriteHeader(hdr)
	_, _ = tw.Write(mB)

	tw.Close()
	gw.Close()

	return path
}

func TestPublishCmd(t *testing.T) {
	// Setup env
	os.Setenv("CPM_REGISTRY_TOKEN", "test-token")
	defer os.Unsetenv("CPM_REGISTRY_TOKEN")
	registryURL = "https://registry.test"

	name := "test-patch"
	version := "1.0.0"
	path := createTestCGP(t, name, version)
	downloadURL := "https://download.test/patch.cgp"

	t.Run("successful publish", func(t *testing.T) {
		var capturedReq registry.PublishRequest
		var capturedFingerprint string
		mock := &mockRegistry{
			publishFunc: func(token string, req registry.PublishRequest) error {
				capturedReq = req
				return nil
			},
			publishSSHFunc: func(fingerprint, signature string, req registry.PublishRequest) error {
				capturedFingerprint = fingerprint
				capturedReq = req
				return nil
			},
		}
		registryClient = mock
		defer func() { registryClient = nil }()

		publishDownloadURL = downloadURL
		publishTags = []string{"test", "mock"}
		publishScope = "testorg"
		publishVisibility = "public"
		publishKeyPath = "/nonexistent/key"

		err := publishCmd.RunE(nil, []string{path})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if capturedReq.Name != name || capturedReq.Version != version {
			t.Errorf("expected %s v%s, got %s v%s", name, version, capturedReq.Name, capturedReq.Version)
		}
		if capturedReq.DownloadURL != downloadURL {
			t.Errorf("expected download URL %s, got %s", downloadURL, capturedReq.DownloadURL)
		}
		if len(capturedReq.Tags) != 2 || capturedReq.Tags[0] != "test" {
			t.Errorf("tags mismatch: %v", capturedReq.Tags)
		}
		if capturedReq.Scope != "testorg" {
			t.Errorf("scope mismatch: %s", capturedReq.Scope)
		}
		if capturedFingerprint != "" {
			t.Logf("used SSH auth (fingerprint: %s)", capturedFingerprint)
		}

		publishKeyPath = ""
	})

	t.Run("missing download URL", func(t *testing.T) {
		publishDownloadURL = ""
		err := publishCmd.RunE(nil, []string{path})
		if err == nil || !contains(err.Error(), "ERROR:P006") {
			t.Fatalf("expected ERROR:P006, got %v", err)
		}
	})

	t.Run("missing token and no SSH key", func(t *testing.T) {
		os.Unsetenv("CPM_REGISTRY_TOKEN")
		publishDownloadURL = downloadURL
		publishKeyPath = "/nonexistent/key"
		err := publishCmd.RunE(nil, []string{path})
		if err == nil || !contains(err.Error(), "ERROR:P005") {
			t.Fatalf("expected ERROR:P005, got %v", err)
		}
		publishKeyPath = ""
		os.Setenv("CPM_REGISTRY_TOKEN", "test-token")
	})

	t.Run("invalid file", func(t *testing.T) {
		publishDownloadURL = downloadURL
		err := publishCmd.RunE(nil, []string{"non-existent.cgp"})
		if err == nil || !contains(err.Error(), "ERROR:P001") {
			t.Fatalf("expected ERROR:P001, got %v", err)
		}
	})

	t.Run("invalid manifest", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "invalid.cgp")
		_ = os.WriteFile(tmpFile, []byte("not a gzip"), 0644)
		publishDownloadURL = downloadURL
		err := publishCmd.RunE(nil, []string{tmpFile})
		if err == nil || !contains(err.Error(), "ERROR:P002") {
			t.Fatalf("expected ERROR:P002, got %v", err)
		}
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
