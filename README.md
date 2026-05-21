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

Taskfile loads `.env` when present. Use `CONFIG` and `IMAGE` to override the defaults, and `GITLAB_TOKEN` for GitLab authentication:

```dotenv
CONFIG=configs/root.yml
IMAGE=gitlab-labelctl:latest
GITLAB_TOKEN=...
```

## Docker

Build the production image:

```bash
docker build --tag gitlab-labelctl:latest .
```

Run sync with Docker:

```bash
docker run --rm \
  -v "$PWD:/workspace:ro" \
  -w /workspace \
  -e GITLAB_TOKEN \
  gitlab-labelctl:latest \
  sync --config configs/root.yml
```

## GitLab authentication

`configs/root.yml` declares which environment variable contains the GitLab token:

```yaml
gitlab:
  auth:
    token_env: GITLAB_TOKEN
```

`token_env` is the variable name, not the token value. With the default config, `.env` should define `GITLAB_TOKEN`, and the Docker task passes that variable into the container.

Priority order:

1. `--token`
2. `--token-file`
3. environment variable named by `gitlab.auth.token_env`, defaulting to `GITLAB_TOKEN`
4. `.env` next to the config file or in the current working directory
5. CI variables such as `CI_JOB_TOKEN` or `GITLAB_CI_TOKEN`

If `token_env` is changed, update `.env`, `.env.example`, and the Docker `-e ...` passthrough in `Taskfile.yml` to use the same variable name.

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
