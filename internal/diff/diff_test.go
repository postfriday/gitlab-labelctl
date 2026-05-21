package diff

import (
    "testing"

    "github.com/postfriday/gitlab-labelctl/internal/config"
    "github.com/postfriday/gitlab-labelctl/internal/gitlab"
)

func TestCompareLabelsCreatesMissingLabel(t *testing.T) {
    cfg := &config.Config{Defaults: config.Defaults{DeleteUnmanaged: false}, ManagedPrefixes: []string{"type::"}}
    desired := map[string]gitlab.Label{"type::bug": {Name: "type::bug", Color: "#D73A4A"}}
    actual := []gitlab.Label{}
    changes := compareLabels("platform/backend", "project", desired, actual, cfg)
    if len(changes) != 1 || changes[0].Kind != "create" {
        t.Fatalf("expected create change, got %#v", changes)
    }
}
