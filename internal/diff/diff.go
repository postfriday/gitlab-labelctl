package diff

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "regexp"
    "sort"
    "strings"

    "github.com/postfriday/gitlab-labelctl/internal/config"
    "github.com/postfriday/gitlab-labelctl/internal/gitlab"
    "github.com/postfriday/gitlab-labelctl/internal/reconcile"
)

type Result struct {
    Changes []reconcile.Change `json:"changes"`
}

func ComputeDiff(ctx context.Context, cfg *config.Config, client *gitlab.Client) ([]reconcile.Change, error) {
    var changes []reconcile.Change
    for _, group := range cfg.Groups {
        desired := mergeLabels(cfg, group)
        actual, err := client.ListGroupLabels(group.ID)
        if err != nil {
            return nil, err
        }
        changes = append(changes, compareLabels(group.ID, "group", desired, actual, cfg)...)
    }
    for _, project := range cfg.Projects {
        desired := mergeLabels(cfg, project)
        actual, err := client.ListProjectLabels(project.ID)
        if err != nil {
            return nil, err
        }
        changes = append(changes, compareLabels(project.ID, "project", desired, actual, cfg)...)
    }
    sort.SliceStable(changes, func(i, j int) bool {
        if changes[i].Target != changes[j].Target {
            return changes[i].Target < changes[j].Target
        }
        if changes[i].Scope != changes[j].Scope {
            return changes[i].Scope < changes[j].Scope
        }
        return changes[i].Desired.Name < changes[j].Desired.Name
    })
    return changes, nil
}

func Render(changes []reconcile.Change, out io.Writer, jsonOutput bool) {
    if jsonOutput {
        data, _ := json.MarshalIndent(Result{Changes: changes}, "", "  ")
        _, _ = out.Write(data)
        return
    }
    for _, ch := range changes {
        switch ch.Kind {
        case reconcile.Create:
            fmt.Fprintf(out, "+ create: %s/%s\n", ch.Scope, ch.Target)
            fmt.Fprintf(out, "    %s\n", ch.Desired.Name)
        case reconcile.Update:
            fmt.Fprintf(out, "~ update: %s/%s\n", ch.Scope, ch.Target)
            fmt.Fprintf(out, "    %s\n", ch.Desired.Name)
            fmt.Fprintf(out, "      color: %s -> %s\n", ch.Existing.Color, ch.Desired.Color)
            if ch.Existing.Description != ch.Desired.Description {
                fmt.Fprintf(out, "      description: %q -> %q\n", ch.Existing.Description, ch.Desired.Description)
            }
        case reconcile.Delete:
            fmt.Fprintf(out, "- delete: %s/%s\n", ch.Scope, ch.Target)
            fmt.Fprintf(out, "    %s\n", ch.Existing.Name)
        }
    }
}

func Reconcile(ctx context.Context, changes []reconcile.Change, client *gitlab.Client, continueOnError bool) error {
    return reconcile.Apply(ctx, changes, client, continueOnError)
}

func mergeLabels(cfg *config.Config, entity config.Entity) map[string]gitlab.Label {
    desired := map[string]gitlab.Label{}
    for _, tpl := range entity.IncludeTemplates {
        for _, label := range cfg.Templates[tpl] {
            desired[label.Name] = gitlab.Label{Name: label.Name, Color: label.Color, Description: label.Description}
        }
    }
    for _, selector := range cfg.Selectors {
        re, err := regexp.Compile(selector.Match)
        if err != nil {
            continue
        }
        if re.MatchString(entity.ID) {
            for _, tpl := range selector.IncludeTemplates {
                for _, label := range cfg.Templates[tpl] {
                    desired[label.Name] = gitlab.Label{Name: label.Name, Color: label.Color, Description: label.Description}
                }
            }
        }
    }
    for _, label := range entity.Labels {
        desired[label.Name] = gitlab.Label{Name: label.Name, Color: label.Color, Description: label.Description}
    }
    return desired
}

func compareLabels(target, scope string, desired map[string]gitlab.Label, actual []gitlab.Label, cfg *config.Config) []reconcile.Change {
    changes := []reconcile.Change{}
    actualMap := map[string]gitlab.Label{}
    for _, label := range actual {
        actualMap[label.Name] = label
    }
    for name, want := range desired {
        if have, ok := actualMap[name]; ok {
            if have.Color != want.Color || have.Description != want.Description {
                changes = append(changes, reconcile.Change{Kind: reconcile.Update, Target: target, Scope: scope, Desired: want, Existing: &have})
            }
            delete(actualMap, name)
        } else {
            changes = append(changes, reconcile.Change{Kind: reconcile.Create, Target: target, Scope: scope, Desired: want})
        }
    }
    for _, leftover := range actualMap {
        if isOwned(leftover.Name, cfg.ManagedPrefixes) {
            if cfg.Defaults.DeleteUnmanaged {
                changes = append(changes, reconcile.Change{Kind: reconcile.Delete, Target: target, Scope: scope, Existing: &leftover})
            }
        }
    }
    return changes
}

func isOwned(name string, prefixes []string) bool {
    if len(prefixes) == 0 {
        return true
    }
    for _, prefix := range prefixes {
        if strings.HasPrefix(name, prefix) {
            return true
        }
    }
    return false
}
