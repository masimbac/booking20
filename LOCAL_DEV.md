# Local development

Phase 0 bootstrap: run tests and linters locally before opening a PR.

## Prerequisites

- Go **1.22+** (see `go.mod`)
- [golangci-lint](https://golangci-lint.run/welcome/install/) for `make lint`

## Common commands

| Command | Purpose |
|---------|--------|
| `make test` | Run tests with race detector |
| `make vet` | `go vet ./...` |
| `make fmt` | Format with `go fmt` |
| `make fmt-check` | Fail if files need formatting |
| `make lint` | Run golangci-lint |
| `make build-lambda` | Linux binary for Lambda (`bin/bootstrap`, **arm64** by default) |

Override Lambda architecture: `make build-lambda LAMBDA_GOARCH=amd64`.

## CI

GitHub Actions runs **test**, **vet**, and **golangci-lint** on pushes and pull requests to `main` / `master`. **Use pull requests** for changes; keep `main` green.

## Module path

The module is `github.com/parama/booking`. If your organization uses a different path, update `go.mod` and imports accordingly.
