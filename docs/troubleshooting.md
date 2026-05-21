# Troubleshooting

## Common issues

- `GitLab token not found`
  - Ensure `GITLAB_TOKEN` is exported or specify `--token` / `--token-file`.
  - Support for `.env` is provided but only when the file is present in the config directory or current working directory.

- `invalid YAML config`
  - Use `gitlab-labelctl validate --config configs/root.yml`.
  - Ensure the YAML syntax is valid and schema requirements are met.

- `gitlab request failed`
  - Check GitLab base URL and token scopes.
  - Ensure network connectivity and API availability.

## Docker issues

- If the service does not start, run:

```bash
docker compose build --no-cache
```

- For debug access:

```bash
docker build -f Dockerfile.debug -t gitlab-labelctl:debug .
```
