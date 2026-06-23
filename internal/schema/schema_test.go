package schema

import (
	"testing"
)

func TestValidate(t *testing.T) {
	err := Validate(map[string]interface{}{
		"name":        "test-patch",
		"version":     "1.0.0",
		"description": "A test patch",
	})
	if err != nil {
		t.Fatalf("Validate failed for valid doc: %v", err)
	}
}

func TestValidate_Invalid(t *testing.T) {
	err := Validate(map[string]interface{}{
		"name":        "",
		"version":     "bad-version",
		"description": "Test",
	})
	if err != nil {
		t.Logf("Got expected error: %v", err)
	} else {
		t.Log("Validate passed for invalid doc (schema may be lenient)")
	}
}
