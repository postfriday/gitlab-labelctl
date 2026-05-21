package main

import (
	"bytes"
	"encoding/json"
	"testing"
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
