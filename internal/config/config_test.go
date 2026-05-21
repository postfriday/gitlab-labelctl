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

func TestLoadConfigRejectsCyclicIncludes(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "first.yml")
	second := filepath.Join(dir, "second.yml")
	os.WriteFile(first, []byte("version: 1\ninclude:\n  - ./second.yml\n"), 0600)
	os.WriteFile(second, []byte("include:\n  - ./first.yml\n"), 0600)

	_, err := Load(context.Background(), first)
	if err == nil {
		t.Fatal("expected cyclic include error")
	}
}

func TestResolveTokenPriority(t *testing.T) {
	dir := t.TempDir()
	tokenFile := filepath.Join(dir, "token")
	os.WriteFile(tokenFile, []byte("from-file\n"), 0600)
	t.Setenv("GITLAB_TOKEN", "from-env")

	cfg := &Config{GitLab: GitLabConfig{Auth: AuthConfig{TokenEnv: "GITLAB_TOKEN"}}}

	got, err := ResolveToken("explicit", tokenFile, cfg)
	if err != nil {
		t.Fatalf("expected explicit token to resolve, got %v", err)
	}
	if got != "explicit" {
		t.Fatalf("expected explicit token, got %q", got)
	}

	got, err = ResolveToken("", tokenFile, cfg)
	if err != nil {
		t.Fatalf("expected token file to resolve, got %v", err)
	}
	if got != "from-file" {
		t.Fatalf("expected token from file, got %q", got)
	}

	got, err = ResolveToken("", "", cfg)
	if err != nil {
		t.Fatalf("expected env token to resolve, got %v", err)
	}
	if got != "from-env" {
		t.Fatalf("expected token from env, got %q", got)
	}
}

func TestResolveTokenLoadsDotEnvBesideConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "root.yml")
	envPath := filepath.Join(dir, ".env")
	os.WriteFile(cfgPath, []byte("version: 1\n"), 0600)
	os.WriteFile(envPath, []byte("GITLAB_TOKEN=from-dotenv\n"), 0600)
	t.Setenv("GITLAB_TOKEN", "")

	cfg := &Config{
		Source: cfgPath,
		GitLab: GitLabConfig{Auth: AuthConfig{TokenEnv: "GITLAB_TOKEN"}},
	}

	got, err := ResolveToken("", "", cfg)
	if err != nil {
		t.Fatalf("expected .env token to resolve, got %v", err)
	}
	if got != "from-dotenv" {
		t.Fatalf("expected token from .env, got %q", got)
	}
}
