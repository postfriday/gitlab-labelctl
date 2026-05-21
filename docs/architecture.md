# Architecture

`gitlab-labelctl` is structured as a clean, modular CLI with distinct layers:

- `cmd/` contains the Cobra command tree and CLI wiring.
- `internal/config/` handles YAML loading, include resolution, and token resolution.
- `internal/gitlab/` wraps GitLab API access and retry semantics.
- `internal/diff/` computes planned label changes and deterministic diffs.
- `internal/reconcile/` applies create/update/delete operations.
- `internal/validate/` enforces schema and policy rules.
- `internal/schema/` embeds JSON schema for IDE autocomplete.

The repository-local workflow is Docker-first; `Taskfile.yml` exposes the main
development commands through `docker build` and `docker run`.
