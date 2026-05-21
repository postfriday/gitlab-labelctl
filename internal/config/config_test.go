package config

import (
    "context"
    "os"
    "path/filepath"
    "testing"
)

func TestLoadConfigWithIncludes(t *testing.T) {
    dir := t.TempDir()
    root := filepath.Join(dir, "root.yml")
    include := filepath.Join(dir, "common.yml")
    os.WriteFile(include, []byte("templates:\n  common:\n    - name: type::bug\n      color: \"#D73A4A\"\n"), 0600)
    os.WriteFile(root, []byte("version: 1\ninclude:\n  - ./common.yml\n"), 0600)

    cfg, err := Load(context.Background(), root)
    if err != nil {
        t.Fatalf("expected load to succeed, got %v", err)
    }
    if cfg.Version != 1 {
        t.Fatalf("expected version 1, got %d", cfg.Version)
    }
    if len(cfg.Templates["common"]) != 1 {
        t.Fatalf("expected template to be loaded")
    }
}
