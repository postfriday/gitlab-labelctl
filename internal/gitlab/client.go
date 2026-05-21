package gitlab

import (
    "bytes"
    "context"
    "crypto/tls"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "net/url"
    "path"
    "strings"
    "time"

    "github.com/postfriday/gitlab-labelctl/internal/config"
)

type Client struct {
    baseURL    *url.URL
    httpClient *http.Client
    token      string
    retry      config.Retry
}

type Label struct {
    Name        string `json:"name" yaml:"name"`
    Color       string `json:"color" yaml:"color"`
    Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

func NewClient(cfg *config.Config) (*Client, error) {
    if cfg.GitLab.URL == "" {
        return nil, fmt.Errorf("gitlab.url is required")
    }
    parsed, err := url.Parse(cfg.GitLab.URL)
    if err != nil {
        return nil, fmt.Errorf("invalid GitLab URL: %w", err)
    }
    timeout := 30 * time.Second
    if cfg.GitLab.Timeout != "" {
        parsedTimeout, err := time.ParseDuration(cfg.GitLab.Timeout)
        if err != nil {
            return nil, fmt.Errorf("invalid timeout value: %w", err)
        }
        timeout = parsedTimeout
    }
    transport := http.DefaultTransport.(*http.Transport).Clone()
    if cfg.GitLab.TLS.InsecureSkipVerify {
        transport.TLSClientConfig = cloneTLSConfig(transport.TLSClientConfig)
        transport.TLSClientConfig.InsecureSkipVerify = true
    }
    if cfg.GitLab.TLS.CAFile != "" {
        pool, err := loadCACertPool(cfg.GitLab.TLS.CAFile)
        if err != nil {
            return nil, err
        }
        transport.TLSClientConfig = cloneTLSConfig(transport.TLSClientConfig)
        transport.TLSClientConfig.RootCAs = pool
    }
    return &Client{
        baseURL: parsed,
        httpClient: &http.Client{Transport: transport, Timeout: timeout},
        token: cfg.GitLab.Auth.Token,
        retry: cfg.GitLab.Retry,
    }, nil
}

func (c *Client) doRequest(ctx context.Context, method, endpoint string, body io.Reader, result interface{}) error {
    urlStr := c.baseURL.ResolveReference(&url.URL{Path: path.Join(c.baseURL.Path, endpoint)}).String()
    req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
    if err != nil {
        return err
    }
    if c.token != "" {
        req.Header.Set("Authorization", "Bearer "+c.token)
    }
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "application/json")

    attempts := 1
    if c.retry.Attempts > 0 {
        attempts = c.retry.Attempts
    }

    for i := 0; i < attempts; i++ {
        res, err := c.httpClient.Do(req)
        if err != nil {
            if i+1 == attempts {
                return err
            }
            time.Sleep(backoffDelay(i))
            continue
        }
        defer res.Body.Close()
        if res.StatusCode >= 200 && res.StatusCode < 300 {
            if result == nil {
                return nil
            }
            return json.NewDecoder(res.Body).Decode(result)
        }
        if res.StatusCode == 429 || res.StatusCode >= 500 {
            if i+1 == attempts {
                return fmt.Errorf("gitlab request failed: %s", res.Status)
            }
            retryAfter := parseRetryAfter(res.Header.Get("Retry-After"))
            if retryAfter <= 0 {
                retryAfter = backoffDelay(i)
            }
            time.Sleep(retryAfter)
            continue
        }
        data, _ := io.ReadAll(res.Body)
        return fmt.Errorf("gitlab request failed: %s: %s", res.Status, strings.TrimSpace(string(data)))
    }
    return fmt.Errorf("gitlab request exhausted")
}

func (c *Client) ListProjectLabels(projectID string) ([]Label, error) {
    var out []Label
    err := c.doRequest(context.Background(), http.MethodGet, fmt.Sprintf("/api/v4/projects/%s/labels", url.PathEscape(projectID)), nil, &out)
    return out, err
}

func (c *Client) ListGroupLabels(groupID string) ([]Label, error) {
    var out []Label
    err := c.doRequest(context.Background(), http.MethodGet, fmt.Sprintf("/api/v4/groups/%s/labels", url.PathEscape(groupID)), nil, &out)
    return out, err
}

func (c *Client) CreateProjectLabel(projectID string, label Label) error {
    payload, _ := json.Marshal(label)
    return c.doRequest(context.Background(), http.MethodPost, fmt.Sprintf("/api/v4/projects/%s/labels", url.PathEscape(projectID)), bytes.NewReader(payload), nil)
}

func (c *Client) UpdateProjectLabel(projectID string, label Label, existing Label) error {
    payload, _ := json.Marshal(map[string]string{"name": existing.Name, "new_name": label.Name, "color": label.Color, "description": label.Description})
    return c.doRequest(context.Background(), http.MethodPut, fmt.Sprintf("/api/v4/projects/%s/labels", url.PathEscape(projectID)), bytes.NewReader(payload), nil)
}

func (c *Client) DeleteProjectLabel(projectID, name string) error {
    return c.doRequest(context.Background(), http.MethodDelete, fmt.Sprintf("/api/v4/projects/%s/labels?name=%s", url.PathEscape(projectID), url.QueryEscape(name)), nil, nil)
}

func (c *Client) CreateGroupLabel(groupID string, label Label) error {
    payload, _ := json.Marshal(label)
    return c.doRequest(context.Background(), http.MethodPost, fmt.Sprintf("/api/v4/groups/%s/labels", url.PathEscape(groupID)), bytes.NewReader(payload), nil)
}

func (c *Client) UpdateGroupLabel(groupID string, label Label, existing Label) error {
    payload, _ := json.Marshal(map[string]string{"name": existing.Name, "new_name": label.Name, "color": label.Color, "description": label.Description})
    return c.doRequest(context.Background(), http.MethodPut, fmt.Sprintf("/api/v4/groups/%s/labels", url.PathEscape(groupID)), bytes.NewReader(payload), nil)
}

func (c *Client) DeleteGroupLabel(groupID, name string) error {
    return c.doRequest(context.Background(), http.MethodDelete, fmt.Sprintf("/api/v4/groups/%s/labels?name=%s", url.PathEscape(groupID), url.QueryEscape(name)), nil, nil)
}

func backoffDelay(attempt int) time.Duration {
    base := 500 * time.Millisecond
    return time.Duration(1<<attempt) * base
}

func parseRetryAfter(value string) time.Duration {
    if value == "" {
        return 0
    }
    if seconds, err := time.ParseDuration(value + "s"); err == nil {
        return seconds
    }
    if t, err := http.ParseTime(value); err == nil {
        return time.Until(t)
    }
    return 0
}

func cloneTLSConfig(src *tls.Config) *tls.Config {
    if src == nil {
        return &tls.Config{}
    }
    return src.Clone()
}
