package registry

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"results":[{"name":"test-patch","version":"1.0.0","description":"test patch","license":"MIT","checksum_sha256":"abc123"}],"total":1,"page":1}`))
	}))
	defer srv.Close()

	client := New(srv.URL)
	results, err := client.Search("test", SearchOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results.Results))
	}
	if results.Results[0].Name != "test-patch" {
		t.Fatalf("expected name test-patch, got %s", results.Results[0].Name)
	}
}

func TestClientGetMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/patches/test-patch/1.0.0" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"name":"test-patch","version":"1.0.0","description":"test patch","checksum_sha256":"abc123","download_url":"http://example.com/pkg.cgp","status":"active"}`))
	}))
	defer srv.Close()

	client := New(srv.URL)
	meta, err := client.GetMetadata("test-patch", "1.0.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Name != "test-patch" {
		t.Fatalf("expected name test-patch, got %s", meta.Name)
	}
	if meta.Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", meta.Version)
	}
}

func TestClientGetVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/patches/test-patch/versions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(`[{"version":"1.0.0","status":"active"},{"version":"0.9.0","status":"deprecated"}]`))
	}))
	defer srv.Close()

	client := New(srv.URL)
	versions, err := client.GetVersions("test-patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(versions) != 2 {
		t.Fatalf("expected 2 versions, got %d", len(versions))
	}
	if versions[0].Version != "1.0.0" {
		t.Fatalf("expected version 1.0.0, got %s", versions[0].Version)
	}
}

func TestClientGetDependencies(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/patches/test-patch/dependencies" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"name":"test-patch","version":"1.0.0","dependencies":[{"name":"dep-patch","version":"1.0.0","dependencies":[]}]}`))
	}))
	defer srv.Close()

	client := New(srv.URL)
	tree, err := client.GetDependencies("test-patch")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tree.Name != "test-patch" {
		t.Fatalf("expected name test-patch, got %s", tree.Name)
	}
	if len(tree.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(tree.Dependencies))
	}
	if tree.Dependencies[0].Name != "dep-patch" {
		t.Fatalf("expected dep name dep-patch, got %s", tree.Dependencies[0].Name)
	}
}