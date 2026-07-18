package auth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

func LoadPrivateKey(path string) (ssh.Signer, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read key %s: %w", path, err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse key %s: %w", path, err)
	}
	return signer, nil
}

func LoadPublicKey(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read key %s: %w", path, err)
	}
	return strings.TrimSpace(string(data)), nil
}

func PublicKeyFingerprint(signer ssh.Signer) string {
	pubKey := signer.PublicKey()
	hash := sha256.Sum256(pubKey.Marshal())
	return "SHA256:" + base64.RawStdEncoding.EncodeToString(hash[:])
}

func SignManifest(signer ssh.Signer, manifestJSON []byte) (string, error) {
	hash := sha256.Sum256(manifestJSON)
	signature, err := signer.Sign(nil, hash[:])
	if err != nil {
		return "", fmt.Errorf("sign: %w", err)
	}
	wireFormat := marshalSSHSig(signature)
	return base64.RawStdEncoding.EncodeToString(wireFormat), nil
}

func marshalSSHSig(sig *ssh.Signature) []byte {
	formatBytes := []byte(sig.Format)
	blobBytes := sig.Blob

	formatLen := make([]byte, 4)
	formatLen[0] = byte(len(formatBytes) >> 24)
	formatLen[1] = byte(len(formatBytes) >> 16)
	formatLen[2] = byte(len(formatBytes) >> 8)
	formatLen[3] = byte(len(formatBytes))

	blobLen := make([]byte, 4)
	blobLen[0] = byte(len(blobBytes) >> 24)
	blobLen[1] = byte(len(blobBytes) >> 16)
	blobLen[2] = byte(len(blobBytes) >> 8)
	blobLen[3] = byte(len(blobBytes))

	result := make([]byte, 0, 8+len(formatBytes)+len(blobBytes))
	result = append(result, formatLen...)
	result = append(result, formatBytes...)
	result = append(result, blobLen...)
	result = append(result, blobBytes...)
	return result
}
