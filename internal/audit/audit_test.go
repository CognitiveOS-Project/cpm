package audit

import (
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

func TestCheck_NoRequirements(t *testing.T) {
	err := Check(nil, &Result{AvailableRAMMB: 512})
	if err != nil {
		t.Fatalf("Check with nil req should pass: %v", err)
	}
}

func TestCheck_RAM(t *testing.T) {
	err := Check(&archive.HardwareReq{MinRAMMB: 1024}, &Result{AvailableRAMMB: 512})
	if err == nil {
		t.Fatal("expected error for insufficient RAM")
	}
}

func TestCheck_RAMSufficient(t *testing.T) {
	err := Check(&archive.HardwareReq{MinRAMMB: 512}, &Result{AvailableRAMMB: 1024})
	if err != nil {
		t.Fatalf("expected pass: %v", err)
	}
}

func TestCheck_Storage(t *testing.T) {
	err := Check(&archive.HardwareReq{MinStorageMB: 100}, &Result{AvailableStorageMB: 50})
	if err == nil {
		t.Fatal("expected error for insufficient storage")
	}
}

func TestCheck_NPU(t *testing.T) {
	err := Check(&archive.HardwareReq{NPURequired: true}, &Result{NPUAvailable: false})
	if err == nil {
		t.Fatal("expected error for missing NPU")
	}
}
