# gitlab-labelctl

`gitlab-labelctl` is a declarative CLI for managing GitLab labels from YAML.
It reads a desired state file, validates schema and policy rules, compares that
state with GitLab group or project labels, and can apply the resulting changes.

The CLI is intended to work the same way whether it is run as a local binary,
from a container, or in CI. Repository-local Taskfile and Docker workflows are
documented separately in [`docs/local-development.md`](docs/local-development.md).

## Features

- YAML desired state for group and project labels
- Reusable includes, templates, selectors, and ownership prefixes
- Validation for schema, scoped-label policy, and forbidden labels
- Deterministic diff output before applying changes
- Dry-run reconciliation and drift detection
- YAML export from existing GitLab labels
- Machine-readable JSON output for automation

## Basic Workflow

Use `configs/root.yml` as the desired-state entrypoint, or pass another YAML
file with `--config`.

Validate the configuration:

```bash
gitlab-labelctl validate --config configs/root.yml
```

Preview the planned changes:

```bash
gitlab-labelctl diff --config configs/root.yml
```

Preview reconciliation through the sync command without mutating GitLab:

```bash
gitlab-labelctl sync --dry-run --config configs/root.yml
```

Apply the desired state:

```bash
gitlab-labelctl sync --config configs/root.yml
```

Successful validation prints:

```text
Configuration is valid: configs/root.yml
```

If GitLab already matches the YAML desired state, `sync` prints:

```text
No changes. GitLab labels are already in sync: configs/root.yml
```

When changes are applied, `sync` prints a summary:

```text
Sync applied: 3 change(s) (1 create, 1 update, 1 delete): configs/root.yml
```

## Commands

| Command | Purpose |
| --- | --- |
| `validate` | Load and validate the YAML desired state. |
| `diff` | Compare GitLab labels with desired state and print planned changes. |
| `sync` | Apply planned create, update, and delete operations. |
| `drift` | Detect drift between GitLab and desired state. |
| `export` | Export labels from a group or project to YAML. |
| `schema` | Print the embedded JSON schema for configuration files. |
| `version` | Print version information. |

Common flags:

| Flag | Purpose |
| --- | --- |
| `--config <path>` | YAML configuration file. |
| `--token <token>` | GitLab token value. Prefer env vars or token files for routine use. |
| `--token-file <path>` | File containing the GitLab token. |
| `--dry-run` | Run without mutating GitLab. |
| `--json` | Produce machine-readable JSON output. |
| `--continue-on-error` | Continue applying non-fatal reconciliation operations. |

## Authentication

The configuration declares which environment variable contains the GitLab token:

```yaml
gitlab:
  auth:
    token_env: GITLAB_TOKEN
```

`token_env` is the variable name, not the token value. With the default config,
set `GITLAB_TOKEN` in the runtime environment or provide `--token` /
`--token-file`.

Token resolution priority:

1. `--token`
2. `--token-file`
3. environment variable named by `gitlab.auth.token_env`
4. `.env` next to the config file or in the current working directory
5. CI variables such as `CI_JOB_TOKEN` or `GITLAB_CI_TOKEN`

## Configuration

See [`docs/configuration.md`](docs/configuration.md) for the full configuration
reference. A minimal desired state looks like this:

```yaml
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

Includes are resolved relative to the file that declares them, so larger setups
can keep `configs/root.yml` small and place shared templates or project groups
in separate files.

Safety note: `defaults.reconcile` is currently informational; `sync` applies the
computed plan unless dry-run mode is enabled. `defaults.delete_unmanaged: true`
only plans deletes for labels owned by `managed_prefixes`; an empty
`managed_prefixes` list treats every label as owned.

## Export

Export labels from a project:

```bash
gitlab-labelctl export --config configs/root.yml --project platform/backend/api
```

Export labels from a group and write the YAML to a file:

```bash
gitlab-labelctl export --config configs/root.yml --group platform --output labels.yml
```

## CI Example

```yaml
labels:validate:
  script:
    - gitlab-labelctl validate --config configs/root.yml

labels:diff:
  script:
    - gitlab-labelctl diff --config configs/root.yml

labels:sync:
  script:
    - gitlab-labelctl sync --config configs/root.yml
```

## More Documentation

- [`docs/configuration.md`](docs/configuration.md): configuration reference and examples
- [`docs/local-development.md`](docs/local-development.md): local Taskfile and Docker workflows
- [`docs/architecture.md`](docs/architecture.md): package layout and implementation notes
- [`docs/troubleshooting.md`](docs/troubleshooting.md): operational troubleshooting
