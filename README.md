# gitlab-labelctl

Declarative CLI for GitLab label management using YAML-based desired state.

## Features

- Validate YAML config
- Preview label diffs
- Reconcile group/project labels
- Dry-run support
- Drift detection
- Export labels to YAML
- Declarative includes, templates, selectors, and ownership prefixes

## Usage

Use Taskfile tasks as the primary interface:

```bash
task labels:diff
task labels:sync
task labels:sync:dry-run
```

The utility runs inside Docker, so Go is not required on the host.

## Docker

Build the production image:

```bash
docker build --tag gitlab-labelctl:latest .
```

Run sync with Docker Compose:

```bash
docker compose run --rm labels sync --config configs/root.yml
```

## GitLab authentication

Priority order:

1. `--token`
2. `--token-file`
3. `GITLAB_TOKEN`
4. `.env`
5. CI variables such as `CI_JOB_TOKEN`

## Configuration

Use `configs/root.yml` as the desired state entrypoint.

## CI example

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
