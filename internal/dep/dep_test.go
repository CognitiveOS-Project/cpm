package dep

import (
	"testing"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
)

func TestResolveNoDeps(t *testing.T) {
	g := NewGraph()
	g.Add("a", &archive.Manifest{
		Name:    "a",
		Version: "1.0.0",
	})

	order, err := g.Resolve("a", "", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(order) != 1 || order[0] != "a" {
		t.Fatalf("expected [a], got %v", order)
	}
}

func TestResolveWithDeps(t *testing.T) {
	g := NewGraph()
	g.Add("a", &archive.Manifest{
		Name:         "a",
		Dependencies: map[string]string{"b": "1.0.0"},
	})
	g.Add("b", &archive.Manifest{
		Name:         "b",
		Dependencies: map[string]string{"c": "1.0.0"},
	})
	g.Add("c", &archive.Manifest{Name: "c"})

	order, err := g.Resolve("a", "", nil)
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if len(order) != 3 {
		t.Fatalf("expected 3 items, got %v", order)
	}
	if order[0] != "c" || order[1] != "b" || order[2] != "a" {
		t.Fatalf("expected [c b a], got %v", order)
	}
}

func TestCircularDep(t *testing.T) {
	g := NewGraph()
	g.Add("a", &archive.Manifest{
		Name:         "a",
		Dependencies: map[string]string{"b": "1.0.0"},
	})
	g.Add("b", &archive.Manifest{
		Name:         "b",
		Dependencies: map[string]string{"a": "1.0.0"},
	})

	_, err := g.Resolve("a", "", nil)
	if err == nil {
		t.Fatal("expected circular dependency error")
	}
}
