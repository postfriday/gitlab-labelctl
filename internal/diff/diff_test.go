package diff

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/postfriday/gitlab-labelctl/internal/config"
	"github.com/postfriday/gitlab-labelctl/internal/gitlab"
	"github.com/postfriday/gitlab-labelctl/internal/reconcile"
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

func TestCompareLabelsUpdatesChangedLabel(t *testing.T) {
	cfg := &config.Config{}
	desired := map[string]gitlab.Label{
		"type::bug": {Name: "type::bug", Color: "#D73A4A", Description: "Bug"},
	}
	actual := []gitlab.Label{
		{Name: "type::bug", Color: "#000000", Description: "Old"},
	}

	changes := compareLabels("platform/backend", "project", desired, actual, cfg)
	if len(changes) != 1 || changes[0].Kind != reconcile.Update {
		t.Fatalf("expected update change, got %#v", changes)
	}
	if changes[0].Existing == nil || changes[0].Existing.Color != "#000000" {
		t.Fatalf("expected existing label details, got %#v", changes[0].Existing)
	}
}

func TestCompareLabelsDeletesOnlyManagedUnmanagedLabels(t *testing.T) {
	cfg := &config.Config{
		Defaults:        config.Defaults{DeleteUnmanaged: true},
		ManagedPrefixes: []string{"type::"},
	}
	actual := []gitlab.Label{
		{Name: "type::old", Color: "#D73A4A"},
		{Name: "external", Color: "#C2E0C6"},
	}

	changes := compareLabels("platform/backend", "project", map[string]gitlab.Label{}, actual, cfg)
	if len(changes) != 1 || changes[0].Kind != reconcile.Delete {
		t.Fatalf("expected one delete change, got %#v", changes)
	}
	if changes[0].Existing == nil || changes[0].Existing.Name != "type::old" {
		t.Fatalf("expected managed label to be deleted, got %#v", changes[0].Existing)
	}
}

func TestMergeLabelsAppliesTemplatesSelectorsAndEntityOverrides(t *testing.T) {
	cfg := &config.Config{
		Templates: map[string][]config.Label{
			"common": {
				{Name: "priority::high", Color: "#E34F4F"},
				{Name: "type::bug", Color: "#000000", Description: "template"},
			},
			"backend": {
				{Name: "status::review", Color: "#5319E7"},
			},
		},
		Selectors: []config.Selector{
			{Match: "^platform/backend/", IncludeTemplates: []string{"backend"}},
		},
	}
	entity := config.Entity{
		ID:               "platform/backend/api",
		IncludeTemplates: []string{"common"},
		Labels: []config.Label{
			{Name: "type::bug", Color: "#D73A4A", Description: "entity"},
		},
	}

	labels := mergeLabels(cfg, entity)
	if len(labels) != 3 {
		t.Fatalf("expected 3 merged labels, got %#v", labels)
	}
	if labels["type::bug"].Color != "#D73A4A" || labels["type::bug"].Description != "entity" {
		t.Fatalf("expected entity label to override template, got %#v", labels["type::bug"])
	}
	if _, ok := labels["status::review"]; !ok {
		t.Fatalf("expected selector template label to be included, got %#v", labels)
	}
}

func TestRenderTextAndJSON(t *testing.T) {
	changes := []reconcile.Change{
		{
			Kind:    reconcile.Create,
			Scope:   "project",
			Target:  "platform/backend",
			Desired: gitlab.Label{Name: "type::bug", Color: "#D73A4A"},
		},
	}

	var text bytes.Buffer
	Render(changes, &text, false)
	if !strings.Contains(text.String(), "+ create: project/platform/backend") {
		t.Fatalf("expected text render to include create header, got %q", text.String())
	}
	if !strings.Contains(text.String(), "type::bug") {
		t.Fatalf("expected text render to include label name, got %q", text.String())
	}

	var encoded bytes.Buffer
	Render(changes, &encoded, true)
	var result Result
	if err := json.Unmarshal(encoded.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON render, got %v: %s", err, encoded.String())
	}
	if len(result.Changes) != 1 || result.Changes[0].Desired.Name != "type::bug" {
		t.Fatalf("expected rendered JSON change, got %#v", result)
	}
}
