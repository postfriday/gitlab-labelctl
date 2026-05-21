package reconcile

import (
    "context"
    "fmt"
    "strings"

    "github.com/postfriday/gitlab-labelctl/internal/gitlab"
)

type Kind string

const (
    Create Kind = "create"
    Update Kind = "update"
    Delete Kind = "delete"
)

type Change struct {
    Kind    Kind
    Target  string
    Scope   string
    Desired gitlab.Label
    Existing *gitlab.Label
}

func Apply(ctx context.Context, changes []Change, client *gitlab.Client, continueOnError bool) error {
    var errs []string
    for _, change := range changes {
        var err error
        switch change.Kind {
        case Create:
            switch change.Scope {
            case "project":
                err = client.CreateProjectLabel(change.Target, change.Desired)
            case "group":
                err = client.CreateGroupLabel(change.Target, change.Desired)
            }
        case Update:
            switch change.Scope {
            case "project":
                err = client.UpdateProjectLabel(change.Target, change.Desired, *change.Existing)
            case "group":
                err = client.UpdateGroupLabel(change.Target, change.Desired, *change.Existing)
            }
        case Delete:
            switch change.Scope {
            case "project":
                err = client.DeleteProjectLabel(change.Target, change.Existing.Name)
            case "group":
                err = client.DeleteGroupLabel(change.Target, change.Existing.Name)
            }
        }

        if err != nil {
            errMessage := fmt.Sprintf("%s %s/%s: %v", strings.ToUpper(string(change.Kind)), change.Scope, change.Target, err)
            if continueOnError {
                errs = append(errs, errMessage)
                continue
            }
            return fmt.Errorf(errMessage)
        }
    }
    if len(errs) > 0 {
        return fmt.Errorf("reconcile completed with errors:\n%s", strings.Join(errs, "\n"))
    }
    return nil
}
