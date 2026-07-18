package auth

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func generateTestKey(t *testing.T) ssh.Signer {
	t.Helper()
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}
	signer, err := ssh.NewSignerFromKey(privKey)
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	return signer
}

func TestPublicKeyFingerprint(t *testing.T) {
	signer := generateTestKey(t)
	fp := PublicKeyFingerprint(signer)

	if !strings.HasPrefix(fp, "SHA256:") {
		t.Errorf("fingerprint should start with SHA256:, got %s", fp)
	}

	parts := strings.SplitN(fp, ":", 2)
	if len(parts) != 2 {
		t.Fatalf("fingerprint should have format SHA256:<base64>, got %s", fp)
	}

	_, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		t.Errorf("fingerprint base64 is invalid: %v", err)
	}
}

func TestSignManifest(t *testing.T) {
	signer := generateTestKey(t)
	manifest := []byte(`{"name":"test","version":"1.0.0"}`)

	sig, err := SignManifest(signer, manifest)
	if err != nil {
		t.Fatalf("SignManifest failed: %v", err)
	}

	if sig == "" {
		t.Fatal("signature should not be empty")
	}

	decoded, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("signature should be valid base64: %v", err)
	}

	if len(decoded) < 8 {
		t.Fatal("signature too short for SSH wire format")
	}
}

func TestSignAndVerifyRoundtrip(t *testing.T) {
	signer := generateTestKey(t)
	manifest := []byte(`{"name":"hello","version":"0.1.0","description":"test"}`)

	sig, err := SignManifest(signer, manifest)
	if err != nil {
		t.Fatalf("SignManifest failed: %v", err)
	}

	hash := sha256.Sum256(manifest)

	decoded, err := base64.RawStdEncoding.DecodeString(sig)
	if err != nil {
		t.Fatalf("decode signature: %v", err)
	}

	formatLen := int(decoded[0])<<24 | int(decoded[1])<<16 | int(decoded[2])<<8 | int(decoded[3])
	format := string(decoded[4 : 4+formatLen])

	blobLen := int(decoded[4+formatLen])<<24 | int(decoded[5+formatLen])<<16 | int(decoded[6+formatLen])<<8 | int(decoded[7+formatLen])
	blob := decoded[8+formatLen : 8+formatLen+blobLen]

	sigObj := &ssh.Signature{
		Format: format,
		Blob:   blob,
	}

	err = signer.PublicKey().Verify(hash[:], sigObj)
	if err != nil {
		t.Fatalf("signature verification failed: %v", err)
	}
}

func TestSignDifferentManifests(t *testing.T) {
	signer := generateTestKey(t)

	sig1, err := SignManifest(signer, []byte(`{"name":"a"}`))
	if err != nil {
		t.Fatalf("SignManifest failed: %v", err)
	}

	sig2, err := SignManifest(signer, []byte(`{"name":"b"}`))
	if err != nil {
		t.Fatalf("SignManifest failed: %v", err)
	}

	if sig1 == sig2 {
		t.Error("different manifests should produce different signatures")
	}
}

func TestLoadPrivateKey(t *testing.T) {
	_, err := LoadPrivateKey("/nonexistent/key")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}

func TestLoadPublicKey(t *testing.T) {
	_, err := LoadPublicKey("/nonexistent/key")
	if err == nil {
		t.Error("expected error for nonexistent key")
	}
}
