# Cursor agent notes — booking-2.0

Context for Cursor Agent (and humans) working in this repo. For deeper phased feature notes, env setup, and deploy steps, read **LOCAL_DEV.md** and **RUNBOOK.md**.

## Agent rules and skills

**Global (all projects on this machine):** Karpathy behavioral guidelines from [andrej-karpathy-skills](https://github.com/multica-ai/andrej-karpathy-skills) live under `~/.cursor/`:

| Artifact | Path | When it applies |
|----------|------|-----------------|
| Karpathy guidelines (User Rules) | **Cursor Settings → Rules → User Rules** | Always, every project — paste from `~/.cursor/user-rules-karpathy.txt` |
| Karpathy guidelines (skill) | `~/.cursor/skills/karpathy-guidelines/SKILL.md` | On demand / skill invocation |

**This repo only:**

| Artifact | Path | When it applies |
|----------|------|-----------------|
| Go project patterns (rule) | `.cursor/rules/booking-go-patterns.mdc` | When editing `**/*.go` files |
| Cross-tool instructions | `CLAUDE.md` | Claude Code and other tools that read root instruction files |

Confirm project rules under **Settings → Rules**. When updating the four Karpathy principles, keep **User Rules**, `~/.cursor/skills/karpathy-guidelines/SKILL.md`, and `CLAUDE.md` in sync.

## What this is

- **Go API** packaged as an **AWS Lambda** behind API Gateway (**chi** mux via `aws-lambda-go-api-proxy`).
- **DynamoDB** single-table design; repositories live under `internal/adapters/dynamo`.
- **Business logic** is split into **`internal/domain`** (entities, validation, pure rules) and **`internal/app/<feature>`** (use cases wired to **`internal/app/ports`** interfaces).

**Module:** `github.com/parama/booking`  
**Go version:** See `go.mod` (currently 1.24.x toolchain).

## Layout (high signal)

| Path | Role |
|------|------|
| `cmd/api/main.go` | Lambda entry: AWS config, repo wiring, `httpapi.NewRouter`, `lambda.Start`. |
| `internal/httpapi/` | Chi router, middleware (CORS, rate limit, problem+json recover), handlers, error mapping. |
| `internal/domain/` | Types, domain errors (`ErrNotFound`, `ErrConflict`, `ErrInvalid`), booking/payment/scheduling rules. |
| `internal/app/<bounded context>/` | Application services (`bookings`, `tenancy`, `catalog`, …). |
| `internal/app/ports/` | Interfaces consumed by HTTP/app layers; implemented by adapters. |
| `internal/adapters/dynamo/` | DynamoDB repos, key layout, cursored lists. |
| `internal/adapters/paymentstub/`, `outbound/` | Stubs / outbound channels for local Lambda defaults. |
| `internal/schedule/` | Pure slot building (`BuildSlots`). |
| `internal/phase0/` | Bootstrap phase string surfaced on `GET /v1/health`. |
| `infra/terraform/` | DynamoDB table, Lambda, API Gateway, monitoring. |
| `infra/terraform/environments/` | Naming for **staging** vs **production** stacks (`staging.tfvars`, `production.tfvars`). |
| `infra/tf-backend-bootstrap/` | **One-off** Terraform to create **S3 + DynamoDB lock** for remote backend (uses local bootstrap state only). |
| `.env-local` (+ **`.env-local.example`**) | Local AWS + backend bucket/table names (**gitignored**, never commit). |
| `infra/terraform/.backend/` | Generated `-backend-config` stubs (`staging.hcl`, `production.hcl`) via **`make terraform-backend-config`** — ignore in Git. |
| `infra/tf-github-actions-iam/` | **One-off** IAM bootstrap: GitHub OIDC + deploy roles **`booking20-staging-github-deploy`** / **`booking20-production-github-deploy`** (override via `staging_role_name` / `production_role_name`). Staging role trusts **`repo:<org>/<name>`** for branch refs + optional **`environment:staging`** (**`trust_staging_github_environment`**) + **`pull_request*`** (**`trust_github_pull_request_workflows`** for PR plans). **Local Terraform state** — copy **`terraform.tfvars.example`** → **`terraform.tfvars`** (gitignored). |
| `.github/workflows/terraform-plan-pr.yml` | Path-filtered **`pull_request`** **`terraform plan`** (staging **`.tfvars`** + remote state) before merge. |
| `.github/workflows/deploy-staging.yml` | **`push`** → **`terraform apply`** **`environments/staging.tfvars`**. |
| `.github/workflows/deploy-production.yml` | **`workflow_dispatch`**: **`environment: production`**, **`secrets.AWS_ROLE_TO_ASSUME_PRODUCTION`**, **`terraform apply`** **`environments/production.tfvars`**. |

## Commands

From repo root (see **Makefile**):

- **Tests:** `make test` — `go test -count=1 -race ./...`
- **Static checks:** `make vet`, `make lint` (needs `golangci-lint`)
- **Format:** `make fmt` / `make fmt-check`
- **Lambda binary:** `make build-lambda` — Linux **arm64** `bin/bootstrap` (override `LAMBDA_GOARCH` if needed)
- **Terraform backend:** **`make terraform-backend-bootstrap-{init,apply}`** once → **`make terraform-backend-config`** (**`scripts/render-backend-config.sh`**: **`.env-local`** locally, or **`TF_REMOTE_STATE_BUCKET` + `TF_REMOTE_LOCK_TABLE`** in CI matching repo **Variables `TERRAFORM_*`** — **LOCAL_DEV Step D**) → **`make terraform-init`** (**staging**) or **`terraform-init-production`**.
- **GitHub deploy IAM / repo config:** **`make terraform-github-actions-iam-{init,plan,apply}`** (**`infra/tf-github-actions-iam/terraform.tfvars`**) + **`make install-gh`** + **`make github-actions-{ensure-environments,sync,repo-setup}`** / **`gh auth login`** (**LOCAL_DEV Step D**). **`make terraform-github-actions-iam-validate`**.
- **Terraform app:** **`make terraform-{fmt,validate,plan,apply}`** — app init uses **`init -backend=false`** for validate CI; **`plan`/`apply`** need prior **`terraform-init`** plus **`-var-file=environments/<env>.tfvars`** on the **`terraform`** CLI until Makefile grows wrappers.

Before opening a PR, at minimum run **`make test`**, **`make vet`**, and **`make lint`** (and fix fmt if `fmt-check` fails).

## HTTP conventions

- **Base path:** `/v1`; API Gateway stage is stripped when `API_GATEWAY_STAGE` is set (`StripAPIStagePrefix`).
- **Errors:** RFC 7807-style **`application/problem+json`** (`WriteProblem`, `WriteConflict`, `WriteUnprocessable`). Domain errors map through `internal/httpapi/errmap.go`.
- **Real vs stub routes:** Phase 3+ routes register when **`Deps.Tenancy` and `Deps.Catalog`** are non-nil (`internal/httpapi/router.go`). Other domains follow the same deps pattern (customers, scheduling, bookings, payments, conversations, notifications).
- **Auth / tenancy (placeholder):**
  - Optional platform routes: **`X-Api-Key`** vs `PLATFORM_API_KEY`; **`REQUIRE_PLATFORM_API_KEY`** enforces configuration.
  - Tenant routes expect **`X-Tenant-Business-Id`** aligned with path **`SKIP_TENANT_CHECK`** for dev-only relaxation.
- **Bookings:** `POST /bookings` expects **`Idempotency-Key`** (see LOCAL_DEV Phase 5).

## Lambda environment (representative)

| Variable | Purpose |
|----------|---------|
| `CORE_TABLE_NAME` | **Required.** DynamoDB table name. |
| `API_GATEWAY_STAGE` | Stage prefix stripping for chi. |
| `PLATFORM_API_KEY` / `REQUIRE_PLATFORM_API_KEY` | Platform key auth toggles. |
| `SKIP_TENANT_CHECK` | Dev-only tenant header bypass. |
| `CORS_ALLOWED_ORIGINS` | Parsed list for CORS. |
| `HTTP_RATE_LIMIT_MAX` / `HTTP_RATE_LIMIT_WINDOW_SEC` | Optional sliding-window rate limit. |
| `WHATSAPP_APP_SECRET` | Meta webhook signature verification when set. |

## Patterns for changes

1. **Prefer extending existing ports and repos** rather than ad-hoc Dynamo access from handlers.
2. **Keep handlers thin:** decode → call app → map errors with `mapAppErr` or explicit problem responses.
3. **Time:** inject `Now func() time.Time` on apps (see `cmd/api/main.go`) for testability.
4. **IDs and money:** ULIDs where used; **`internal/domain/money`** for catalog/pricing consistency.
5. **Infra:** table keys/GSIs live in Terraform and adapter code together — update both when adding access patterns.

## Other docs

- **LOCAL_DEV.md** — phased delivery notes, curl-style behavior, Dynamo key shapes.
- **RUNBOOK.md** — operations.
- **THREAT_MODEL.md** — security framing.
