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
| `make terraform-init` | `terraform init` in `infra/terraform` |
| `make terraform-fmt` | `terraform fmt -recursive` |
| `make terraform-validate` | Build Lambda zip inputs, `init -backend=false`, `validate` |
| `make terraform-plan` | `terraform plan` (needs AWS credentials and initialized backend if configured) |
| `make terraform-apply` | `terraform apply` |

Override Lambda architecture: `make build-lambda LAMBDA_GOARCH=amd64`.

## Phase 1 — AWS (Terraform)

1. Configure AWS credentials (`aws configure` or environment variables for the target account).
2. From the repo root: `make terraform-init` (add `-migrate-state` / backend config when you introduce remote state).
3. `make terraform-validate` — ensures `bin/bootstrap` exists and configuration is valid (no AWS calls).
4. `make terraform-plan` then `make terraform-apply` — creates DynamoDB `core` table (4 GSIs), API Lambda (`provided.al2023` **arm64**), REST API `GET {stage}/v1/health` → Lambda.
5. After apply, open the **`health_url`** output (or `terraform -chdir=infra/terraform output health_url`) — expect JSON `{"status":"ok","phase":"1"}`.

Variables live in `infra/terraform/variables.tf` (`aws_region`, `project`, `environment`, etc.).

## CI

GitHub Actions runs **test**, **vet**, and **golangci-lint** on pushes and pull requests to `main` / `master`. **Use pull requests** for changes; keep `main` green.

## Module path

The module is `github.com/parama/booking`. If your organization uses a different path, update `go.mod` and imports accordingly.
