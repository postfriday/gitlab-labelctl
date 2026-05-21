package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/postfriday/gitlab-labelctl/internal/gitlab"
	"github.com/postfriday/gitlab-labelctl/internal/reconcile"
)

func TestRenderValidateSuccessText(t *testing.T) {
	var out bytes.Buffer

	if err := renderValidateSuccess(&out, "configs/root.yml", false); err != nil {
		t.Fatalf("expected render to succeed, got %v", err)
	}

	want := "Configuration is valid: configs/root.yml\n"
	if out.String() != want {
		t.Fatalf("expected %q, got %q", want, out.String())
	}
}

func TestRenderValidateSuccessJSON(t *testing.T) {
	var out bytes.Buffer

	if err := renderValidateSuccess(&out, "configs/root.yml", true); err != nil {
		t.Fatalf("expected render to succeed, got %v", err)
	}

	var got struct {
		Valid  bool   `json:"valid"`
		Config string `json:"config"`
	}
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, out.String())
	}
	if !got.Valid || got.Config != "configs/root.yml" {
		t.Fatalf("unexpected JSON payload: %#v", got)
	}
}

func TestRenderSyncSuccessNoChanges(t *testing.T) {
	var out bytes.Buffer

	if err := renderSyncSuccess(&out, "configs/root.yml", nil, true, false); err != nil {
		t.Fatalf("expected render to succeed, got %v", err)
	}

	want := "No changes. GitLab labels are already in sync: configs/root.yml\n"
	if out.String() != want {
		t.Fatalf("expected %q, got %q", want, out.String())
	}
}

func TestRenderSyncSuccessAppliedSummary(t *testing.T) {
	var out bytes.Buffer
	changes := []reconcile.Change{
		{Kind: reconcile.Create, Desired: gitlab.Label{Name: "type::bug"}},
		{Kind: reconcile.Update, Desired: gitlab.Label{Name: "status::review"}},
		{Kind: reconcile.Delete, Existing: &gitlab.Label{Name: "type::old"}},
	}

	if err := renderSyncSuccess(&out, "configs/root.yml", changes, true, false); err != nil {
		t.Fatalf("expected render to succeed, got %v", err)
	}

	want := "Sync applied: 3 change(s) (1 create, 1 update, 1 delete): configs/root.yml\n"
	if out.String() != want {
		t.Fatalf("expected %q, got %q", want, out.String())
	}
}

func TestRenderSyncSuccessJSON(t *testing.T) {
	var out bytes.Buffer
	changes := []reconcile.Change{
		{Kind: reconcile.Create, Desired: gitlab.Label{Name: "type::bug"}},
	}

	if err := renderSyncSuccess(&out, "configs/root.yml", changes, true, true); err != nil {
		t.Fatalf("expected render to succeed, got %v", err)
	}

	var got syncSummary
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("expected valid JSON, got %v: %s", err, out.String())
	}
	if !got.Synced || !got.Applied || got.Changes != 1 || got.Create != 1 || got.Config != "configs/root.yml" {
		t.Fatalf("unexpected JSON payload: %#v", got)
	}
}
