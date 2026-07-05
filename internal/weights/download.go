package weights

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func Download(ctx context.Context, url, dest string, expectedSHA256 string, fn ProgressFn) error {
	if fn == nil {
		fn = NoopProgress
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "cpm/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	total := resp.ContentLength

	tmpDest := dest + ".cpm-partial"
	f, err := os.Create(tmpDest)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}

	hasher := sha256.New()
	writer := io.MultiWriter(f, hasher)

	buf := make([]byte, 256*1024) // 256 KB
	var written int64

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			wn, writeErr := writer.Write(buf[:n])
			if writeErr != nil {
				f.Close()
				os.Remove(tmpDest)
				return fmt.Errorf("write: %w", writeErr)
			}
			if wn != n {
				f.Close()
				os.Remove(tmpDest)
				return fmt.Errorf("short write")
			}
			written += int64(n)
			fn(written, total)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			f.Close()
			os.Remove(tmpDest)
			return fmt.Errorf("read: %w", readErr)
		}
	}

	f.Close()
	fn(written, total)
	fmt.Fprintf(os.Stderr, "\n")

	if expectedSHA256 != "" {
		actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
		if actualSHA256 != expectedSHA256 {
			os.Remove(tmpDest)
			return fmt.Errorf("SHA-256 mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
		}
	}

	if err := os.Rename(tmpDest, dest); err != nil {
		os.Remove(tmpDest)
		return fmt.Errorf("rename: %w", err)
	}

	return nil
}

func ModelDir(kind Kind) string {
	base := "/cognitiveos/models"
	switch kind {
	case KindRaw:
		return filepath.Join(base, "raw")
	case KindWide:
		return filepath.Join(base, "wide", "active")
	default:
		return base
	}
}
