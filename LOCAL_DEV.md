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
5. After apply, open the **`health_url`** output (or `terraform -chdir=infra/terraform output health_url`) — expect JSON `{"status":"ok","phase":"…"}`.

## Phase 2 — HTTP shell (chi + problem+json)

- **Router:** `internal/httpapi` — chi with stage strip (`API_GATEWAY_STAGE`), request ID, **recover → 500 problem+json**, chi **Logger**, routes under `/v1`.
- **Errors:** `application/problem+json` (`WriteProblem`, `WriteConflict`, `WriteUnprocessable`); 404/405/501 stubs match **OpenAPI** layout.
- **Lambda entry:** `cmd/api` uses **`aws-lambda-go-api-proxy/chi`** to forward API Gateway proxy events to the mux.
- **Apply:** redeploy Lambda after `terraform apply` so the new binary and env var are live (`make terraform-apply`).

## Phase 3 — Tenancy + catalog (DynamoDB)

- **Domain:** `internal/domain` — Business, CatalogService, Staff, Money, errors.
- **App:** `internal/app/tenancy`, `internal/app/catalog` — use cases + `internal/app/ports` interfaces.
- **Adapters:** `internal/adapters/dynamo` — single-table keys `BUSINESS#` / `META#` / `SERVICE#` / `STAFF#`, list cursors via base64 PK/SK.
- **HTTP:** Real routes when `Deps.Tenancy` + `Deps.Catalog` are wired in `cmd/api` (always in Lambda). Additional routes (customers, availability) depend on Phase 4 deps; stubs remain for bookings and later phases.
- **Auth (placeholder):**
  - Optional `PLATFORM_API_KEY` — when set, `POST /v1/platform/businesses` requires header **`X-Api-Key`**.
  - Tenant routes under `/v1/businesses/{businessId}` require **`X-Tenant-Business-Id`** equal to `{businessId}` unless `SKIP_TENANT_CHECK=true` on the Lambda (dev only).
- **Env:** `CORE_TABLE_NAME` (set by Terraform from the DynamoDB table name).

## Phase 4 — CRM + availability (read-model slots)

- **Domain:** `Customer`, `AvailabilityRule` under `internal/domain`.
- **App:** `internal/app/customers` (list / create / get / patch, resolve by phone), `internal/app/scheduling` (`PUT` rules, slot list with business timezone + optional service duration).
- **Adapters:** `CustomerRepository` (partition `CUSTOMER#…`, GSI2 `PHONE#…` / `BUSINESS#tenant`), `AvailabilityRepository` (`SK` `AVAIL#RULES`, full rule list per business).
- **Pure logic:** `internal/schedule.BuildSlots` — merges overlapping windows per staff/day, steps by slot duration in UTC.
- **HTTP (under `/v1/businesses/{businessId}`):** `GET|POST /customers`, `GET /customers/by-phone?phone=…`, `GET|PATCH /customers/{customerId}`, `PUT /availability/rules`, `GET /availability/slots?from=&to=&service_id=&staff_id=&slot_minutes=` — active when `Deps.Customers` and `Deps.Scheduling` are wired (default in `cmd/api`).

## Phase 5 — Bookings (GSI1 + idempotency)

- **Domain:** `Booking` + status transitions (`created` → `confirmed` → `completed` \| `cancelled` \| `no_show`) in `internal/domain/booking.go`.
- **App:** `internal/app/bookings` — creates with staff overlap checks, confirms with a second overlap pass, cancel / complete / no-show.
- **Adapters:** `BookingRepository` — main item `SK` `BOOKING#…`, **GSI1** `BOOKING_DATE#RFC3339Nano#bookingId`; idempotency rows `SK` `IDEMPOTENCY#sha256(key)` + `TransactWriteItems` with the booking put.
- **HTTP:** `GET|POST /bookings`, `GET /bookings/{bookingId}`, transitions under `…/confirm`, `…/cancel`, `…/complete`, `…/no-show`. **Required header:** `Idempotency-Key` on `POST /bookings`. **List:** `?from=&to=` (RFC3339). Active when `Deps.Bookings` is wired (default in `cmd/api`).
- **Events:** `bookings.EventSink` (default `NoopEvents` in Lambda; replace with EventBridge/outbox later).

Variables live in `infra/terraform/variables.tf` (`aws_region`, `project`, `environment`, etc.).

## CI

GitHub Actions runs **test**, **vet**, and **golangci-lint** on pushes and pull requests to `main` / `master`. **Use pull requests** for changes; keep `main` green.

## Module path

The module is `github.com/parama/booking`. If your organization uses a different path, update `go.mod` and imports accordingly.
