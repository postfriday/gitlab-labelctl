package validate

import (
	"context"
	"testing"

	"github.com/postfriday/gitlab-labelctl/internal/config"
)

func TestValidateRejectsInvalidColor(t *testing.T) {
	cfg := &config.Config{
		Version:  1,
		GitLab:   config.GitLabConfig{URL: "https://gitlab.com"},
		Projects: []config.Entity{{ID: "platform/backend", Labels: []config.Label{{Name: "type::bug", Color: "red"}}}},
	}
	err := Validate(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateAcceptsValidScopedConfig(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		GitLab:  config.GitLabConfig{URL: "https://gitlab.com"},
		Policies: config.Policies{
			RequireScopedLabels: true,
			AllowedScopes:       []string{"type", "status", "priority"},
		},
		Templates: map[string][]config.Label{
			"common": {
				{Name: "priority::high", Color: "#E34F4F"},
			},
		},
		Selectors: []config.Selector{
			{Match: "^platform/backend", IncludeTemplates: []string{"common"}},
		},
		Projects: []config.Entity{
			{
				ID:               "platform/backend",
				IncludeTemplates: []string{"common"},
				Labels: []config.Label{
					{Name: "type::bug", Color: "#D73A4A"},
					{Name: "status::review", Color: "#5319E7"},
				},
			},
		},
	}

	if err := Validate(context.Background(), cfg); err != nil {
		t.Fatalf("expected validation to pass, got %v", err)
	}
}

func TestValidateRejectsMissingTemplateReferences(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		GitLab:  config.GitLabConfig{URL: "https://gitlab.com"},
		Selectors: []config.Selector{
			{Match: "^platform/backend", IncludeTemplates: []string{"missing-selector-template"}},
		},
		Projects: []config.Entity{
			{ID: "platform/backend", IncludeTemplates: []string{"missing-project-template"}},
		},
	}

	err := Validate(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsForbiddenAndDisallowedScopes(t *testing.T) {
	cfg := &config.Config{
		Version: 1,
		GitLab:  config.GitLabConfig{URL: "https://gitlab.com"},
		Policies: config.Policies{
			RequireScopedLabels: true,
			AllowedScopes:       []string{"type"},
			ForbidLabels:        []string{"type::forbidden"},
		},
		Projects: []config.Entity{
			{
				ID: "platform/backend",
				Labels: []config.Label{
					{Name: "status::review", Color: "#5319E7"},
					{Name: "type::forbidden", Color: "#D73A4A"},
				},
			},
		},
	}

	err := Validate(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
