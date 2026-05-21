# Configuration

This document explains how to write `gitlab-labelctl` YAML configuration files.

`gitlab-labelctl` uses YAML as the desired state for GitLab labels. The
canonical entrypoint is `configs/root.yml`; it can include smaller files for
shared templates, groups, and projects.

## Minimal Config

```yaml
# yaml-language-server: $schema=../internal/schema/schema.json

version: 1

gitlab:
  url: https://gitlab.com
  auth:
    token_env: GITLAB_TOKEN

projects:
  - id: platform/backend/api
    labels:
      - name: type::bug
        color: "#D73A4A"
        description: User-visible defect
```

Run validation before using the config for diff or sync:

```bash
gitlab-labelctl validate --config configs/root.yml
```

Successful validation prints:

```text
Configuration is valid: configs/root.yml
```

With `--json`, successful validation prints:

```json
{"valid":true,"config":"configs/root.yml"}
```

The validation command still loads authentication settings, so make sure the
configured token environment variable or `.env` file is available.

For the repository-local Taskfile and Docker workflow, see
[`docs/local-development.md`](local-development.md).

## File Layout

A practical layout keeps the root file small and moves reusable data into
included files:

```text
configs/
  root.yml
  templates/
    common.yml
  groups/
    platform.yml
```

`configs/root.yml`:

```yaml
# yaml-language-server: $schema=../internal/schema/schema.json

version: 1

defaults:
  reconcile: true
  delete_unmanaged: false
  dry_run: false

gitlab:
  url: https://gitlab.com
  auth:
    token_env: GITLAB_TOKEN
  tls:
    ca_file: ""
    insecure_skip_verify: false
  timeout: 30s
  retry:
    attempts: 5
    backoff: exponential

managed_prefixes:
  - "type::"
  - "status::"
  - "priority::"

include:
  - ./templates/common.yml
  - ./groups/platform.yml

policies:
  require_scoped_labels: true
  allowed_scopes:
    - type
    - status
    - priority
  forbid_labels:
    - misc
    - asap
```

`configs/templates/common.yml`:

```yaml
templates:
  common:
    - name: type::bug
      color: "#D73A4A"
      description: Confirmed product defect
    - name: type::feature
      color: "#0E8A16"
      description: New product capability
    - name: status::triage
      color: "#FBCA04"
      description: Needs initial review
```

`configs/groups/platform.yml`:

```yaml
selectors:
  - match: "^platform/backend/"
    include_templates:
      - common

groups:
  - id: platform
    include_templates:
      - common

projects:
  - id: platform/backend/billing-api
    labels:
      - name: status::review
        color: "#5319E7"
        description: Ready for peer review
```

## Include Rules

`include` is an array of YAML files to load before validation and diffing.
Include paths are resolved relative to the file that declares them.

```yaml
include:
  - ./templates/common.yml
  - ./groups/platform.yml
```

Includes are recursive, and cycles are rejected. This is valid:

```yaml
# configs/root.yml
include:
  - ./groups/platform.yml

# configs/groups/platform.yml
include:
  - ../templates/common.yml
```

Merge behavior is important:

- maps are merged recursively;
- arrays and scalar values replace the existing value;
- when the same key exists in both files, the included file wins for arrays and
  scalars;
- if multiple includes define the same array or scalar key, the later include in
  the list wins;
- the loader tracks already visited files, so avoid including the same file more
  than once through different paths;
- avoid defining the same top-level array, such as `projects`, in both the root
  file and an included file unless replacement is intentional.

For example, if both files define `templates.common`, the included
`templates.common` array replaces the one from the including file:

```yaml
# root.yml
templates:
  common:
    - name: priority::high
      color: "#E34F4F"

include:
  - ./templates/common.yml
```

```yaml
# templates/common.yml
templates:
  common:
    - name: type::bug
      color: "#D73A4A"
```

The final `common` template contains only `type::bug`. To keep both labels,
define them in the same template array or use distinct template names.

## Top-Level Fields

| Field | Required | Purpose |
| --- | --- | --- |
| `version` | Recommended | Config format version. Must be a positive integer. |
| `defaults` | No | Defaults for dry-run and delete planning, plus reconciliation intent. |
| `gitlab` | Yes | GitLab URL, authentication variable, TLS, timeout, and retry settings. |
| `managed_prefixes` | No | Prefixes that mark labels as owned by this tool for deletion safety. |
| `include` | No | Additional YAML files to merge into this config. |
| `templates` | No | Reusable label lists. |
| `selectors` | No | Regex-based automatic template assignment. |
| `policies` | No | Validation policies for scopes and forbidden labels. |
| `groups` | No | GitLab groups whose labels should be managed. |
| `projects` | No | GitLab projects whose labels should be managed. |

## Defaults

```yaml
defaults:
  reconcile: true
  delete_unmanaged: false
  dry_run: false
```

`reconcile` documents that this config is intended for reconciliation. In the
current implementation, it does not enable or disable CLI behavior by itself.
The `sync` command reconciles whenever it is run, unless `--dry-run` or
`defaults.dry_run` makes the command render the plan only.

`delete_unmanaged` controls deletion of labels that exist in GitLab but are not
present in the desired state. Keep it `false` for initial rollout. When it is
`true`, deletion is still limited by `managed_prefixes`. It affects which
`delete` changes are added to the diff/sync plan; it does not control whether
`create` or `update` changes are generated.

`dry_run` makes `sync` render the plan without applying changes. The CLI flag
`--dry-run` has the same effect for a single run.

Safe initial setup:

```yaml
defaults:
  delete_unmanaged: false
  dry_run: true
```

After reviewing diffs, enable applying changes:

```yaml
defaults:
  delete_unmanaged: false
  dry_run: false
```

## GitLab Connection

```yaml
gitlab:
  url: https://gitlab.example.com
  auth:
    token_env: GITLAB_TOKEN
  tls:
    ca_file: /etc/ssl/certs/company-ca.pem
    insecure_skip_verify: false
  timeout: 30s
  retry:
    attempts: 5
    backoff: exponential
```

`gitlab.url` is required and should point to the GitLab base URL, not the API
path.

`gitlab.auth.token_env` is the name of the environment variable that contains
the token. It is not the token value. If omitted, it defaults to `GITLAB_TOKEN`.

Token resolution priority:

1. `--token`
2. `--token-file`
3. environment variable named by `gitlab.auth.token_env`
4. `.env` next to the config file
5. `.env` in the current working directory
6. `CI_JOB_TOKEN`
7. `GITLAB_CI_TOKEN`

Example `.env`:

```dotenv
CONFIG=configs/root.yml
IMAGE=gitlab-labelctl:latest
GITLAB_TOKEN=glpat-example
```

Never commit real tokens. Keep `.env.example` as a placeholder-only file.

For self-hosted GitLab with a private CA:

```yaml
gitlab:
  url: https://gitlab.internal.example.com
  tls:
    ca_file: ./certs/company-ca.pem
    insecure_skip_verify: false
```

Use `insecure_skip_verify: true` only for local testing or temporary debugging.

## Labels

Each label requires `name` and `color`. `description` is optional.

```yaml
labels:
  - name: type::bug
    color: "#D73A4A"
    description: Confirmed product defect
```

Colors must be GitLab-style hex colors:

```yaml
color: "#D73A4A"
color: "#FBCA04"
color: "#0E8A16"
```

Both 6-digit and 3-digit hex colors are accepted by validation, but 6-digit
colors are recommended for readability.

## Templates

Templates are named reusable arrays of labels.

```yaml
templates:
  common:
    - name: type::bug
      color: "#D73A4A"
    - name: status::triage
      color: "#FBCA04"

  backend:
    - name: type::performance
      color: "#0052CC"
```

Attach templates to groups or projects with `include_templates`:

```yaml
projects:
  - id: platform/backend/billing-api
    include_templates:
      - common
      - backend
```

If a label name appears more than once while building an entity's desired
labels, later sources override earlier ones in this order:

1. templates listed on the entity;
2. templates matched by selectors;
3. labels declared directly on the entity.

This lets a project override a shared template label:

```yaml
templates:
  common:
    - name: type::bug
      color: "#D73A4A"
      description: Generic defect

projects:
  - id: platform/backend/billing-api
    include_templates:
      - common
    labels:
      - name: type::bug
        color: "#B60205"
        description: Billing-specific defect
```

## Selectors

Selectors automatically include templates when an entity ID matches a regular
expression.

```yaml
templates:
  backend:
    - name: status::review
      color: "#5319E7"

selectors:
  - match: "^platform/backend/"
    include_templates:
      - backend
```

With this selector, `platform/backend/billing-api` gets the `backend` template.
`platform/frontend/web` does not.

Selectors apply to both group and project entity IDs. Use precise patterns when
group and project paths share similar prefixes.

## Groups And Projects

`groups` and `projects` declare the GitLab targets to manage. `id` can be a
path such as `platform/backend/billing-api` or an ID accepted by GitLab API.

```yaml
groups:
  - id: platform
    include_templates:
      - common
    labels:
      - name: priority::high
        color: "#E34F4F"

projects:
  - id: platform/backend/billing-api
    include_templates:
      - common
    labels:
      - name: status::review
        color: "#5319E7"
```

Group labels and project labels are reconciled independently. Declaring a label
on a group does not automatically declare it on every project; use templates if
you want the same labels in multiple places.

## Managed Prefixes

`managed_prefixes` limits which existing GitLab labels may be deleted when
`defaults.delete_unmanaged` is `true`.

```yaml
managed_prefixes:
  - "type::"
  - "status::"
  - "priority::"
```

With this config, an unmanaged GitLab label named `type::old` may be deleted,
but a label named `customer-visible` is left alone.

If `managed_prefixes` is empty, every label is considered owned by the tool for
deletion purposes. This is risky for existing GitLab installations.

The delete decision is therefore a two-key safety check:

```yaml
defaults:
  delete_unmanaged: true

managed_prefixes:
  - "type::"
```

With this configuration, GitLab label `type::old` can be planned for deletion if
it is absent from YAML. GitLab label `customer-visible` is not deleted because it
does not match the managed prefix.

Recommended rollout:

```yaml
defaults:
  delete_unmanaged: false

managed_prefixes:
  - "type::"
  - "status::"
```

After repeated clean diffs, enable deletion only if those prefixes accurately
describe the labels owned by this repository:

```yaml
defaults:
  delete_unmanaged: true
```

## Policies

Policies make validation fail before the CLI talks to GitLab.

```yaml
policies:
  require_scoped_labels: true
  allowed_scopes:
    - type
    - status
    - priority
  forbid_labels:
    - misc
    - asap
```

`require_scoped_labels` requires every label to use `scope::name` syntax.

`allowed_scopes` restricts the part before `::`. If the list is empty, any scope
is allowed as long as scoped syntax is used.

`forbid_labels` rejects exact label names.

Valid:

```yaml
- name: type::bug
  color: "#D73A4A"
```

Invalid when `require_scoped_labels: true`:

```yaml
- name: bug
  color: "#D73A4A"
```

Invalid when `allowed_scopes` does not contain `workflow`:

```yaml
- name: workflow::blocked
  color: "#B60205"
```

## Diff And Sync Behavior

For each configured group and project, the CLI builds desired labels from
templates, selectors, and inline labels. It then compares desired labels with
GitLab labels.

Planned changes:

- `create`: label exists in YAML but not in GitLab;
- `update`: label exists in both places but color or description differs;
- `delete`: label exists in GitLab but not in YAML, `delete_unmanaged` is
  enabled, and the label is owned by `managed_prefixes`.

For non-empty plans, `diff` and `sync --dry-run` render the same changes without
applying them. `sync` applies the rendered plan when dry-run mode is off.
`defaults.reconcile` is not a separate execution gate in the current CLI; treat
the command you run as the source of truth for whether reconciliation is
attempted.

When the sync plan is empty, `sync` prints an explicit no-op message:

```text
No changes. GitLab labels are already in sync: configs/root.yml
```

After applying changes, `sync` prints a summary:

```text
Sync applied: 3 change(s) (1 create, 1 update, 1 delete): configs/root.yml
```

Preview changes:

```bash
gitlab-labelctl diff --config configs/root.yml
```

Preview sync without applying:

```bash
gitlab-labelctl sync --dry-run --config configs/root.yml
```

Apply changes:

```bash
gitlab-labelctl sync --config configs/root.yml
```

Render machine-readable output:

```bash
gitlab-labelctl diff --json --config configs/root.yml
```

For `sync --json` when changes are applied, the success output is a summary:

```json
{"synced":true,"applied":true,"config":"configs/root.yml","changes":3,"create":1,"update":1,"delete":1}
```

For `sync --dry-run --json`, the output remains a diff payload with `changes`.

## Schema Support

`internal/schema/schema.json` is the embedded JSON schema. Add this modeline at
the top of root config files for YAML language server support:

```yaml
# yaml-language-server: $schema=../internal/schema/schema.json
```

For files below `configs/templates/` or `configs/groups/`, adjust the relative
path:

```yaml
# yaml-language-server: $schema=../../internal/schema/schema.json
```

The CLI can also print the schema:

```bash
gitlab-labelctl schema --config configs/root.yml
```

## Complete Example

```yaml
# yaml-language-server: $schema=../internal/schema/schema.json

version: 1

defaults:
  reconcile: true
  delete_unmanaged: false
  dry_run: false

gitlab:
  url: https://gitlab.com
  auth:
    token_env: GITLAB_TOKEN
  tls:
    ca_file: ""
    insecure_skip_verify: false
  timeout: 30s
  retry:
    attempts: 5
    backoff: exponential

managed_prefixes:
  - "type::"
  - "status::"
  - "priority::"

templates:
  common:
    - name: type::bug
      color: "#D73A4A"
      description: Confirmed defect
    - name: status::triage
      color: "#FBCA04"
      description: Needs initial review

selectors:
  - match: "^platform/backend/"
    include_templates:
      - common

policies:
  require_scoped_labels: true
  allowed_scopes:
    - type
    - status
    - priority
  forbid_labels:
    - misc
    - asap

groups:
  - id: platform
    include_templates:
      - common

projects:
  - id: platform/backend/billing-api
    labels:
      - name: priority::high
        color: "#E34F4F"
        description: High priority work
```
