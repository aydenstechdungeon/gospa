package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCollectDocsRoutes(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "routing", "api"), 0750); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "page.templ"), []byte("root"), 0600); err != nil {
		t.Fatalf("write page.templ failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "routing", "api", "page.templ"), []byte("nested"), 0600); err != nil {
		t.Fatalf("write nested page.templ failed: %v", err)
	}

	routes, err := collectDocsRoutes(root)
	if err != nil {
		t.Fatalf("collectDocsRoutes failed: %v", err)
	}
	if !routes["/docs"] {
		t.Fatal("expected /docs route")
	}
	if !routes["/docs/routing/api"] {
		t.Fatal("expected nested /docs/routing/api route")
	}
}

func TestLoadSearchEntries(t *testing.T) {
	tmp := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	path := "docs_search_index.json"
	content := `[{"url":"/docs"},{"url":"/docs/api/core"}]`
	if err := os.WriteFile(filepath.Join(tmp, path), []byte(content), 0600); err != nil {
		t.Fatalf("write json failed: %v", err)
	}

	entries, err := loadSearchEntries("docs_search_index.json")
	if err != nil {
		t.Fatalf("loadSearchEntries failed: %v", err)
	}
	if len(entries) != 2 || entries[0].URL != "/docs" || entries[1].URL != "/docs/api/core" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestLoadSearchEntriesRejectsUnsafePaths(t *testing.T) {
	absPath := filepath.Join(string(filepath.Separator), "tmp", "x.json")
	if _, err := loadSearchEntries(absPath); err == nil {
		t.Fatalf("expected absolute path %q to be rejected", absPath)
	}

	if _, err := loadSearchEntries("../docs_search_index.json"); err == nil {
		t.Fatal("expected parent-relative path to be rejected")
	}
}
