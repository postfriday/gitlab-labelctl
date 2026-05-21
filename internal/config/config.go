package config

import (
    "context"
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"

    "gopkg.in/yaml.v3"
)

type Config struct {
    Version         int                    `yaml:"version"`
    Defaults        Defaults               `yaml:"defaults"`
    GitLab          GitLabConfig           `yaml:"gitlab"`
    ManagedPrefixes []string               `yaml:"managed_prefixes"`
    Include         []string               `yaml:"include"`
    Templates       map[string][]Label     `yaml:"templates"`
    Selectors       []Selector             `yaml:"selectors"`
    Policies        Policies               `yaml:"policies"`
    Groups          []Entity               `yaml:"groups"`
    Projects        []Entity               `yaml:"projects"`
    Source          string                 `yaml:"-"`
}

type Defaults struct {
    Reconcile      bool `yaml:"reconcile"`
    DeleteUnmanaged bool `yaml:"delete_unmanaged"`
    DryRun         bool `yaml:"dry_run"`
}

type GitLabConfig struct {
    URL     string     `yaml:"url"`
    Auth    AuthConfig `yaml:"auth"`
    TLS     TLSConfig  `yaml:"tls"`
    Timeout string     `yaml:"timeout"`
    Retry   Retry      `yaml:"retry"`
}

type AuthConfig struct {
    TokenEnv string `yaml:"token_env"`
    Token    string `yaml:"-"`
}

type TLSConfig struct {
    CAFile             string `yaml:"ca_file"`
    InsecureSkipVerify bool   `yaml:"insecure_skip_verify"`
}

type Retry struct {
    Attempts int    `yaml:"attempts"`
    Backoff  string `yaml:"backoff"`
}

type Selector struct {
    Match            string   `yaml:"match"`
    IncludeTemplates []string `yaml:"include_templates"`
}

type Policies struct {
    RequireScopedLabels bool     `yaml:"require_scoped_labels"`
    AllowedScopes       []string `yaml:"allowed_scopes"`
    ForbidLabels        []string `yaml:"forbid_labels"`
}

type Entity struct {
    ID               string   `yaml:"id"`
    IncludeTemplates []string `yaml:"include_templates"`
    Labels           []Label  `yaml:"labels"`
}

type Label struct {
    Name        string `yaml:"name"`
    Color       string `yaml:"color"`
    Description string `yaml:"description,omitempty"`
}

var defaultTokenEnv = "GITLAB_TOKEN"
var envVarNameRegex = regexp.MustCompile(`^[A-Z_][A-Z0-9_]*$`)

func Load(ctx context.Context, path string) (*Config, error) {
    raw, err := loadConfigFile(filepath.Clean(path), map[string]bool{})
    if err != nil {
        return nil, err
    }
    cfg := Config{}
    if err := yaml.Unmarshal(raw, &cfg); err != nil {
        return nil, fmt.Errorf("invalid YAML config: %w", err)
    }
    if cfg.GitLab.Auth.TokenEnv == "" {
        cfg.GitLab.Auth.TokenEnv = defaultTokenEnv
    }
    cfg.Source = path
    return &cfg, nil
}

func loadConfigFile(path string, visited map[string]bool) ([]byte, error) {
    absPath, err := filepath.Abs(path)
    if err != nil {
        return nil, err
    }
    if visited[absPath] {
        return nil, fmt.Errorf("cyclic include detected: %s", absPath)
    }
    visited[absPath] = true

    raw, err := os.ReadFile(absPath)
    if err != nil {
        return nil, err
    }

    base := map[string]interface{}{}
    if err := yaml.Unmarshal(raw, &base); err != nil {
        return nil, fmt.Errorf("invalid YAML in %s: %w", absPath, err)
    }

    includes, _ := base["include"].([]interface{})
    for _, inc := range includes {
        includePath := fmt.Sprint(inc)
        if includePath == "" {
            continue
        }
        resolved := filepath.Join(filepath.Dir(absPath), includePath)
        includedRaw, err := loadConfigFile(resolved, visited)
        if err != nil {
            return nil, err
        }
        includedMap := map[string]interface{}{}
        if err := yaml.Unmarshal(includedRaw, &includedMap); err != nil {
            return nil, fmt.Errorf("invalid YAML in included file %s: %w", resolved, err)
        }
        mergeMaps(base, includedMap)
    }

    output, err := yaml.Marshal(base)
    if err != nil {
        return nil, err
    }
    return output, nil
}

func mergeMaps(dst, src map[string]interface{}) {
    for key, value := range src {
        if existing, ok := dst[key]; ok {
            switch existingTyped := existing.(type) {
            case map[string]interface{}:
                if valueTyped, ok := value.(map[string]interface{}); ok {
                    mergeMaps(existingTyped, valueTyped)
                    continue
                }
            }
        }
        dst[key] = value
    }
}

func ResolveToken(explicit, tokenFile string, cfg *Config) (string, error) {
    if explicit != "" {
        return explicit, nil
    }
    if tokenFile != "" {
        data, err := os.ReadFile(tokenFile)
        if err != nil {
            return "", fmt.Errorf("failed to read token file: %w", err)
        }
        return strings.TrimSpace(string(data)), nil
    }
    if value := os.Getenv(cfg.GitLab.Auth.TokenEnv); value != "" {
        return value, nil
    }
    if cfg.Source != "" {
        if err := loadDotEnv(filepath.Dir(cfg.Source)); err == nil {
            if value := os.Getenv(cfg.GitLab.Auth.TokenEnv); value != "" {
                return value, nil
            }
        }
    }
    if err := loadDotEnv("."); err == nil {
        if value := os.Getenv(cfg.GitLab.Auth.TokenEnv); value != "" {
            return value, nil
        }
    }
    if value := os.Getenv("CI_JOB_TOKEN"); value != "" {
        return value, nil
    }
    if value := os.Getenv("GITLAB_CI_TOKEN"); value != "" {
        return value, nil
    }
    return "", fmt.Errorf("GitLab token not found: provide --token, --token-file, %s, or CI variables", cfg.GitLab.Auth.TokenEnv)
}

func loadDotEnv(dir string) error {
    path := filepath.Join(dir, ".env")
    data, err := os.ReadFile(path)
    if err != nil {
        return err
    }
    for _, line := range strings.Split(string(data), "\n") {
        line = strings.TrimSpace(line)
        if line == "" || strings.HasPrefix(line, "#") {
            continue
        }
        parts := strings.SplitN(line, "=", 2)
        if len(parts) != 2 {
            continue
        }
        key := strings.TrimSpace(parts[0])
        if !envVarNameRegex.MatchString(key) {
            continue
        }
        value := strings.TrimSpace(parts[1])
        os.Setenv(key, strings.Trim(value, " \t\"'"))
    }
    return nil
}
