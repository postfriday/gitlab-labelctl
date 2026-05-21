# AGENTS.md

Guidance for coding agents working in this repository.

## Project Overview

`gitlab-labelctl` is a Go CLI for declarative GitLab label management. It reads YAML desired state, validates it, computes diffs against GitLab, and can reconcile group or project labels.

Primary entrypoints:

- CLI wiring: `cmd/gitlab-labelctl/main.go`
- Config loading and token resolution: `internal/config/`
- GitLab API wrapper: `internal/gitlab/`
- Diff planning and rendering: `internal/diff/`
- Reconciliation: `internal/reconcile/`
- Validation and policy checks: `internal/validate/`
- Embedded JSON schema: `internal/schema/`
- Example desired state: `configs/root.yml`

## Development Workflow

Use the Taskfile and Docker workflow when exercising the full CLI:

```bash
task labels:validate
task labels:diff
task labels:sync:dry-run
```

Useful local Go checks:

```bash
go test ./...
go test ./internal/config ./internal/diff ./internal/validate
gofmt -w <changed-go-files>
```

The runtime is intended to be Docker-first. Do not assume users have Go installed on the host when documenting normal usage, but it is fine to use Go tooling for local development and tests.

## Configuration Notes

- Treat `configs/root.yml` as the canonical desired-state entrypoint.
- Includes are resolved relative to the including file. Preserve that behavior when changing config loading.
- YAML config should stay schema-friendly and human-readable.
- Label colors should use GitLab-style hex colors such as `"#D73A4A"`.
- Managed prefixes and scoped label policies are intentional safety controls. Be cautious when changing `delete_unmanaged`, `managed_prefixes`, or validation rules.

## Authentication And Safety

GitLab token resolution priority is:

1. `--token`
2. `--token-file`
3. configured token env var from `gitlab.auth.token_env`, normally `GITLAB_TOKEN`
4. `.env`
5. CI variables such as `CI_JOB_TOKEN` or `GITLAB_CI_TOKEN`

`gitlab.auth.token_env` in `configs/root.yml` is not a token value; it is the name of the environment variable the CLI should read. Keep Taskfile `.env` loading, `.env.example`, and Docker `-e ...` passthrough aligned with that configured name. If `token_env` changes from `GITLAB_TOKEN`, update the Taskfile passthrough and examples at the same time.

Never commit real tokens or generated secret files. Prefer dry-run flows for examples and verification. Avoid running mutating sync commands against real GitLab unless explicitly requested.

## Code Style

- Follow the existing small-package structure; keep CLI wiring thin and domain behavior in `internal/*`.
- Keep behavior deterministic where output is user-facing, especially diffs and rendered plans.
- Prefer table-driven or focused unit tests for config, diff, and validation behavior.
- Use `context.Context` consistently on operations that may touch IO or GitLab.
- Use structured YAML parsing instead of ad hoc string manipulation.
- Run `gofmt` on changed Go files.

## Testing Expectations

For Go changes, run:

```bash
go test ./...
```

For config or docs-only changes, at least verify examples and commands are still accurate. If Docker workflows are affected, prefer:

```bash
task labels:validate
task labels:diff
```

If a check cannot be run because Docker, network, GitLab credentials, or host tooling is unavailable, say so explicitly in the final response.

## Documentation

- Keep `README.md` focused on quick usage.
- Put operational failure guidance in `docs/troubleshooting.md`.
- Put structure and layer explanations in `docs/architecture.md`.
- When adding flags or commands, update README examples and troubleshooting notes if user-facing behavior changes.
