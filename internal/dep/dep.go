package dep

import (
	"fmt"

	"github.com/CognitiveOS-Project/cpm/internal/archive"
	"github.com/CognitiveOS-Project/cpm/internal/patch"
)

type Graph struct {
	nodes map[string]*archive.Manifest
}

func NewGraph() *Graph {
	return &Graph{nodes: make(map[string]*archive.Manifest)}
}

func (g *Graph) Add(name string, m *archive.Manifest) {
	g.nodes[name] = m
}

func (g *Graph) Resolve(name string, version string, seen map[string]bool) ([]string, error) {
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[name] {
		return nil, fmt.Errorf("circular dependency detected: %s", name)
	}
	seen[name] = true

	m, ok := g.nodes[name]
	if !ok {
		return nil, fmt.Errorf("dependency %q not found", name)
	}

	var order []string
	for depName := range m.Dependencies {
		deps, err := g.Resolve(depName, "", seen)
		if err != nil {
			return nil, fmt.Errorf("resolving %s: %w", depName, err)
		}
		order = append(order, deps...)
	}
	order = append(order, name)
	return order, nil
}

func CheckDependents(name string) []string {
	installed, err := patch.List()
	if err != nil {
		return nil
	}
	var dependents []string
	for _, p := range installed {
		if p.Manifest.Dependencies != nil {
			if _, ok := p.Manifest.Dependencies[name]; ok {
				dependents = append(dependents, p.Manifest.Name)
			}
		}
	}
	return dependents
}
