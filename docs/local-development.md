# Local Development

This repository provides a Docker-first local workflow through `Taskfile.yml`.
The tasks build and run the production container, so Go is not required on the
host for normal CLI usage.

## Requirements

- Docker
- Task

## Environment

`Taskfile.yml` loads `.env` when present. Start from `.env.example` and set the
values needed for your environment:

```dotenv
CONFIG=configs/root.yml
IMAGE=gitlab-labelctl:latest
GITLAB_TOKEN=...
```

`CONFIG` selects the desired-state entrypoint, and `IMAGE` controls the local
Docker image tag. `GITLAB_TOKEN` must match `gitlab.auth.token_env` in the
configuration:

```yaml
gitlab:
  auth:
    token_env: GITLAB_TOKEN
```

If `token_env` changes from `GITLAB_TOKEN`, update `.env`, `.env.example`, and
the Docker `-e ...` passthrough in `Taskfile.yml` at the same time.

## Taskfile Workflow

Validate the YAML configuration:

```bash
task labels:validate
```

Preview planned changes:

```bash
task labels:diff
```

Run reconciliation in dry-run mode:

```bash
task labels:sync:dry-run
```

Apply reconciliation:

```bash
task labels:sync
```

The label tasks use the production Docker image and pass through
`GITLAB_TOKEN`. They read the repository into the container as a read-only
workspace.

## Docker Commands

Build the production image:

```bash
task docker:build
```

Equivalent direct Docker command:

```bash
docker build --pull --tag gitlab-labelctl:latest .
```

Run the CLI directly in Docker:

```bash
docker run --rm \
  -v "$PWD:/workspace:ro" \
  -w /workspace \
  -e GITLAB_TOKEN \
  gitlab-labelctl:latest \
  sync --config configs/root.yml
```

Build the debug image when shell access inside the image is useful:

```bash
task docker:build:debug
```

Equivalent direct Docker command:

```bash
docker build --pull --tag gitlab-labelctl:debug -f Dockerfile.debug .
```

## Tests

Run Go tests through the Dockerized Go toolchain:

```bash
task test
```

If Go is available on the host, focused local checks are also useful:

```bash
go test ./...
go test ./internal/config ./internal/diff ./internal/validate
```

Run `gofmt` on changed Go files before committing Go changes:

```bash
gofmt -w <changed-go-files>
```
