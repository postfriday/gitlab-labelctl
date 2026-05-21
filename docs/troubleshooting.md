# Troubleshooting

## Common issues

- `GitLab token not found`
  - Ensure `GITLAB_TOKEN` is exported or specify `--token` / `--token-file`.
  - Support for `.env` is provided but only when the file is present in the config directory or current working directory.

- `invalid YAML config`
  - Use `gitlab-labelctl validate --config configs/root.yml`.
  - Ensure the YAML syntax is valid and schema requirements are met.

- `validate` or `sync` appears silent
  - Successful `validate` prints `Configuration is valid: configs/root.yml`.
  - Successful no-op `sync` prints `No changes. GitLab labels are already in sync: configs/root.yml`.
  - Successful applying `sync` prints a summary such as `Sync applied: 3 change(s) (1 create, 1 update, 1 delete): configs/root.yml`.
  - If these messages are missing in Docker, rebuild the image so the task uses the latest CLI binary.

- `gitlab request failed`
  - Check GitLab base URL and token scopes.
  - Ensure network connectivity and API availability.

## Docker issues

- If the image does not build cleanly, run:

```bash
docker build --no-cache -t gitlab-labelctl:latest .
```

- For debug access:

```bash
docker build -f Dockerfile.debug -t gitlab-labelctl:debug .
```
