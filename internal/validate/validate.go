package validate

import (
    "context"
    "fmt"
    "regexp"
    "strings"

    "github.com/postfriday/gitlab-labelctl/internal/config"
)

var colorRegex = regexp.MustCompile(`^#(?:[A-Fa-f0-9]{3}|[A-Fa-f0-9]{6})$`)

func Validate(ctx context.Context, cfg *config.Config) error {
    var errs []string
    if cfg.Version < 1 {
        errs = append(errs, "version must be a positive integer")
    }
    if cfg.GitLab.URL == "" {
        errs = append(errs, "gitlab.url is required")
    }
    for name, labels := range cfg.Templates {
        if strings.TrimSpace(name) == "" {
            errs = append(errs, "template names must not be empty")
        }
        errs = append(errs, validateLabels(labels, fmt.Sprintf("templates[%s]", name), cfg)...)
    }
    for _, selector := range cfg.Selectors {
        if selector.Match == "" {
            errs = append(errs, "selector match patterns must not be empty")
        } else if _, err := regexp.Compile(selector.Match); err != nil {
            errs = append(errs, fmt.Sprintf("invalid selector regex %q: %v", selector.Match, err))
        }
        for _, tpl := range selector.IncludeTemplates {
            if _, ok := cfg.Templates[tpl]; !ok {
                errs = append(errs, fmt.Sprintf("selector references missing template %q", tpl))
            }
        }
    }
    for _, group := range cfg.Groups {
        errs = append(errs, validateEntity(group, cfg, "groups")...) 
    }
    for _, project := range cfg.Projects {
        errs = append(errs, validateEntity(project, cfg, "projects")...) 
    }
    if cfg.Policies.RequireScopedLabels {
        errs = append(errs, validateScopePolicy(cfg)...)
    }
    if len(errs) > 0 {
        return fmt.Errorf("validation failed:\n%s", strings.Join(errs, "\n"))
    }
    return nil
}

func validateEntity(entity config.Entity, cfg *config.Config, section string) []string {
    var errs []string
    if strings.TrimSpace(entity.ID) == "" {
        errs = append(errs, fmt.Sprintf("%s.id must not be empty", section))
    }
    for _, tpl := range entity.IncludeTemplates {
        if _, ok := cfg.Templates[tpl]; !ok {
            errs = append(errs, fmt.Sprintf("%s %s references missing template %q", section, entity.ID, tpl))
        }
    }
    for _, label := range entity.Labels {
        errs = append(errs, validateLabels([]config.Label{label}, fmt.Sprintf("%s[%s]", section, entity.ID), cfg)...)
    }
    return errs
}

func validateLabels(labels []config.Label, prefix string, cfg *config.Config) []string {
    var errs []string
    seen := map[string]bool{}
    for _, label := range labels {
        if strings.TrimSpace(label.Name) == "" {
            errs = append(errs, fmt.Sprintf("%s label name must not be empty", prefix))
            continue
        }
        if !colorRegex.MatchString(label.Color) {
            errs = append(errs, fmt.Sprintf("%s label %q has invalid color %q", prefix, label.Name, label.Color))
        }
        if seen[label.Name] {
            errs = append(errs, fmt.Sprintf("duplicate label name %q in %s", label.Name, prefix))
        }
        seen[label.Name] = true
        if cfg.Policies.RequireScopedLabels && !strings.Contains(label.Name, "::") {
            errs = append(errs, fmt.Sprintf("label %q does not use scoped syntax", label.Name))
        }
        for _, forbidden := range cfg.Policies.ForbidLabels {
            if label.Name == forbidden {
                errs = append(errs, fmt.Sprintf("label %q is forbidden by policy", forbidden))
            }
        }
    }
    return errs
}

func validateScopePolicy(cfg *config.Config) []string {
    var errs []string
    allowed := map[string]bool{}
    for _, scope := range cfg.Policies.AllowedScopes {
        allowed[scope] = true
    }
    for _, group := range cfg.Groups {
        for _, label := range group.Labels {
            verifyScope(label.Name, allowed, &errs)
        }
    }
    for _, project := range cfg.Projects {
        for _, label := range project.Labels {
            verifyScope(label.Name, allowed, &errs)
        }
    }
    for _, labels := range cfg.Templates {
        for _, label := range labels {
            verifyScope(label.Name, allowed, &errs)
        }
    }
    return errs
}

func verifyScope(name string, allowed map[string]bool, errs *[]string) {
    parts := strings.SplitN(name, "::", 2)
    if len(parts) != 2 {
        *errs = append(*errs, fmt.Sprintf("label %q must use scope::name syntax", name))
        return
    }
    if len(allowed) > 0 && !allowed[parts[0]] {
        *errs = append(*errs, fmt.Sprintf("scope %q is not allowed for label %q", parts[0], name))
    }
}
