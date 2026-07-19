package cmd

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/registry"
	"golang.org/x/crypto/ssh"
)

type mockRegistry struct {
	publishFunc         func(token string, req registry.PublishRequest) error
	publishSSHFunc      func(fingerprint, signature string, req registry.PublishRequest) error
	publishOfficialFunc func(fingerprint, signature string, req registry.PublishRequest, metadataJSON, cgpData []byte) error
	registerFunc        func(publicKey string) (*registry.RegisterResponse, error)
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
func (m *mockRegistry) PublishOfficial(fingerprint, signature string, req registry.PublishRequest, metadataJSON, cgpData []byte) error {
	if m.publishOfficialFunc != nil {
		return m.publishOfficialFunc(fingerprint, signature, req, metadataJSON, cgpData)
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
	t.Helper()
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

func createTempSSHKey(t *testing.T) string {
	t.Helper()
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(t.TempDir(), "id_ed25519")
	privBytes, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		t.Fatal(err)
	}
	pemBytes := pem.EncodeToMemory(privBytes)
	if err := os.WriteFile(keyPath, pemBytes, 0600); err != nil {
		t.Fatal(err)
	}
	_ = signer // used to verify key loads correctly
	return keyPath
}

func TestPublishCmd(t *testing.T) {
	registryURL = "https://registry.test"

	name := "test-patch"
	version := "1.0.0"
	path := createTestCGP(t, name, version)
	downloadURL := "https://download.test/patch.cgp"

	t.Run("proxy publish with download-url", func(t *testing.T) {
		os.Setenv("CPM_REGISTRY_TOKEN", "test-token")
		defer os.Unsetenv("CPM_REGISTRY_TOKEN")

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
		publishDownloadURL = ""
		publishTags = nil
		publishScope = ""
		publishVisibility = "public"
	})

	t.Run("official publish without download-url", func(t *testing.T) {
		keyPath := createTempSSHKey(t)

		var capturedMeta []byte
		var capturedCGP []byte
		var capturedFingerprint string
		mock := &mockRegistry{
			publishOfficialFunc: func(fingerprint, signature string, req registry.PublishRequest, metadataJSON, cgpData []byte) error {
				capturedFingerprint = fingerprint
				capturedMeta = metadataJSON
				capturedCGP = cgpData
				return nil
			},
		}
		registryClient = mock
		defer func() { registryClient = nil }()

		publishDownloadURL = ""
		publishKeyPath = keyPath
		publishTags = []string{"official"}
		publishScope = ""
		publishVisibility = "public"

		err := publishCmd.RunE(nil, []string{path})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if capturedFingerprint == "" {
			t.Error("expected SSH fingerprint, got empty")
		}
		if len(capturedMeta) == 0 {
			t.Error("expected non-empty metadata JSON")
		}
		if len(capturedCGP) == 0 {
			t.Error("expected non-empty cgp data")
		}

		var parsedReq registry.PublishRequest
		if err := json.Unmarshal(capturedMeta, &parsedReq); err != nil {
			t.Fatalf("failed to parse metadata: %v", err)
		}
		if parsedReq.Name != name || parsedReq.Version != version {
			t.Errorf("expected %s v%s in metadata, got %s v%s", name, version, parsedReq.Name, parsedReq.Version)
		}
		if len(parsedReq.Tags) != 1 || parsedReq.Tags[0] != "official" {
			t.Errorf("tags mismatch in metadata: %v", parsedReq.Tags)
		}

		publishKeyPath = ""
		publishTags = nil
	})

	t.Run("official publish too large", func(t *testing.T) {
		keyPath := createTempSSHKey(t)

		// Create a .cgp file that exceeds 32 MB
		bigPath := filepath.Join(t.TempDir(), "big.cgp")
		f, err := os.Create(bigPath)
		if err != nil {
			t.Fatal(err)
		}
		// Write 33 MB of zeros
		big := make([]byte, 33<<20)
		if _, err := f.Write(big); err != nil {
			t.Fatal(err)
		}
		f.Close()

		// We need a valid tar.gz with cognitive.json for ReadManifest to work.
		// Instead, test the size check by writing a valid .cgp that's large.
		// Create a valid archive then append padding.
		validCGP := createTestCGP(t, "big-pkg", "1.0.0")
		validData, _ := os.ReadFile(validCGP)
		padded := append(validData, make([]byte, 33<<20)...)
		paddedPath := filepath.Join(t.TempDir(), "big-padded.cgp")
		os.WriteFile(paddedPath, padded, 0644)

		mock := &mockRegistry{}
		registryClient = mock
		defer func() { registryClient = nil }()

		publishDownloadURL = ""
		publishKeyPath = keyPath

		err = publishCmd.RunE(nil, []string{paddedPath})
		if err == nil || !contains(err.Error(), "ERROR:P012") {
			t.Fatalf("expected ERROR:P012 for oversized file, got %v", err)
		}
		if !contains(err.Error(), "32 MB") {
			t.Errorf("error should mention 32 MB limit: %v", err)
		}
		if !contains(err.Error(), "ghr:") {
			t.Errorf("error should mention UPR flow (ghr:): %v", err)
		}

		publishKeyPath = ""
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
		publishDownloadURL = ""
	})

	t.Run("invalid file", func(t *testing.T) {
		publishDownloadURL = downloadURL
		err := publishCmd.RunE(nil, []string{"non-existent.cgp"})
		if err == nil || !contains(err.Error(), "ERROR:P001") {
			t.Fatalf("expected ERROR:P001, got %v", err)
		}
		publishDownloadURL = ""
	})

	t.Run("invalid manifest", func(t *testing.T) {
		tmpFile := filepath.Join(t.TempDir(), "invalid.cgp")
		_ = os.WriteFile(tmpFile, []byte("not a gzip"), 0644)
		publishDownloadURL = downloadURL
		err := publishCmd.RunE(nil, []string{tmpFile})
		if err == nil || !contains(err.Error(), "ERROR:P002") {
			t.Fatalf("expected ERROR:P002, got %v", err)
		}
		publishDownloadURL = ""
	})
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
