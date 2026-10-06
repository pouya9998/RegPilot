# Contributing to RegPilot

Thanks for your interest in contributing.

## Development

RegPilot is fully Dockerized. You do not need a local Go or Node toolchain.

```bash
cp .env.example .env
docker compose up --build
```

### Tests

```bash
make test
```

### Guidelines

- Keep the app lightweight — avoid new infrastructure dependencies.
- Do not store registry images in SQLite.
- Never expose registry passwords in API responses.
- Never execute arbitrary user-supplied shell commands.
- Prefer clear confirmation dialogs for destructive actions.
- Add tests for authentication, authorization, and registry operations.

## Pull requests

1. Fork and create a feature branch.
2. Keep changes focused.
3. Ensure `make test` and `make docker-build` succeed.
4. Update docs when behavior changes.
