package validate

import (
    "context"
    "testing"

    "github.com/postfriday/gitlab-labelctl/internal/config"
)

func TestValidateRejectsInvalidColor(t *testing.T) {
    cfg := &config.Config{
        Version: 1,
        GitLab: config.GitLabConfig{URL: "https://gitlab.com"},
        Projects: []config.Entity{{ID: "platform/backend", Labels: []config.Label{{Name: "type::bug", Color: "red"}}}},
    }
    err := Validate(context.Background(), cfg)
    if err == nil {
        t.Fatal("expected validation error")
    }
}
