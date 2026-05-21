package export

import (
    "context"
    "fmt"
    "io"

    "github.com/postfriday/gitlab-labelctl/internal/gitlab"
    "gopkg.in/yaml.v3"
)

func Run(ctx context.Context, client *gitlab.Client, projectRef, groupRef string, out io.Writer) error {
    var labels []gitlab.Label
    var err error
    if projectRef != "" {
        labels, err = client.ListProjectLabels(projectRef)
    } else {
        labels, err = client.ListGroupLabels(groupRef)
    }
    if err != nil {
        return err
    }
    payload := map[string]interface{}{"labels": labels}
    data, err := yaml.Marshal(payload)
    if err != nil {
        return fmt.Errorf("failed to encode export YAML: %w", err)
    }
    _, err = out.Write(data)
    return err
}
