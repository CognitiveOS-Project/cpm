package log

import (
	"os"
	"testing"
)

func TestLogInit(t *testing.T) {
	tmpDir := t.TempDir()
	Init(tmpDir)

	if _, err := os.Stat(tmpDir); err != nil {
		t.Fatalf("expected log dir to exist: %v", err)
	}
}

func TestLogInfo(t *testing.T) {
	Info("test info message %d", 42)
}

func TestLogWarn(t *testing.T) {
	Warn("test warn message %s", "test")
}

func TestLogError(t *testing.T) {
	Error("test error message %d", 42)
}

func TestLogAudit(t *testing.T) {
	Audit("test_event", map[string]interface{}{
		"key":   "value",
		"count": 42,
	})
}