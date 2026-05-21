package gitlab

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/postfriday/gitlab-labelctl/internal/config"
)

func TestNewClientRejectsInvalidTimeout(t *testing.T) {
	_, err := NewClient(&config.Config{
		GitLab: config.GitLabConfig{
			URL:     "https://gitlab.com",
			Timeout: "not-a-duration",
		},
	})
	if err == nil {
		t.Fatal("expected invalid timeout error")
	}
}

func TestDoRequestSendsAuthHeaderAndDecodesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			t.Fatalf("expected bearer token header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Accept") != "application/json" {
			t.Fatalf("expected JSON accept header, got %q", r.Header.Get("Accept"))
		}
		if r.URL.EscapedPath() != "/api/v4/projects/platform%2Fbackend/labels" {
			t.Fatalf("expected escaped project path, got %q", r.URL.EscapedPath())
		}
		json.NewEncoder(w).Encode([]Label{{Name: "type::bug", Color: "#D73A4A"}})
	}))
	defer server.Close()

	client, err := NewClient(&config.Config{
		GitLab: config.GitLabConfig{
			URL:  server.URL,
			Auth: config.AuthConfig{Token: "secret"},
		},
	})
	if err != nil {
		t.Fatalf("expected client, got %v", err)
	}

	labels, err := client.ListProjectLabels("platform/backend")
	if err != nil {
		t.Fatalf("expected labels request to succeed, got %v", err)
	}
	if len(labels) != 1 || labels[0].Name != "type::bug" {
		t.Fatalf("expected decoded labels, got %#v", labels)
	}
}

func TestDoRequestReturnsResponseBodyForClientErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "missing scope", http.StatusForbidden)
	}))
	defer server.Close()

	client, err := NewClient(&config.Config{
		GitLab: config.GitLabConfig{URL: server.URL},
	})
	if err != nil {
		t.Fatalf("expected client, got %v", err)
	}

	err = client.doRequest(context.Background(), http.MethodGet, "/api/v4/projects/example/labels", nil, nil)
	if err == nil {
		t.Fatal("expected request error")
	}
	if !strings.Contains(err.Error(), "missing scope") {
		t.Fatalf("expected response body in error, got %v", err)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter("2"); got != 2*time.Second {
		t.Fatalf("expected 2s retry-after, got %s", got)
	}
	if got := parseRetryAfter(""); got != 0 {
		t.Fatalf("expected empty retry-after to be zero, got %s", got)
	}
}
