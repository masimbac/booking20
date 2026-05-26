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
| `make terraform-backend-bootstrap-init` | `terraform init` in `infra/tf-backend-bootstrap` (one-off S3 + DynamoDB lock) |
| `make terraform-backend-bootstrap-apply` | Create remote-state bucket + lock table (`terraform.tfvars` required — see bootstrap dir) |
| `make terraform-backend-config` | Build `infra/terraform/.backend/*.hcl` from **`.env-local`** or from **`TF_REMOTE_STATE_BUCKET` + `TF_REMOTE_LOCK_TABLE`** in the environment (GitHub Actions). |
| `make terraform-init` | `terraform init` with **staging** `-backend-config` (after backend resources + config exist) |
| `make terraform-init-production` | `terraform init` with **production** backend object key (`booking20/production/terraform.tfstate` unless `TF_STATE_KEY_PREFIX` overrides) |
| `make terraform-github-actions-iam-init` | Initialise `infra/tf-github-actions-iam` (local state; IAM OIDC + deploy roles — **privileged** one-off apply) |
| `make terraform-github-actions-iam-validate` | `terraform fmt`, `validate` for GitHub Actions IAM module |
| `make terraform-github-actions-iam-{plan,apply}` | Plan/apply that module (**after** copying `terraform.tfvars.example`) |
| `make install-gh` | Install **GitHub CLI** (`gh`) via Homebrew on macOS, or print install URL elsewhere |
| `make github-actions-ensure-environments` | Create **`staging`** / **`production`** GitHub Environments via API (**`gh auth login`** required) |
| `make github-actions-sync` | Push **`TERRAFORM_*`** repo variables + optional **`AWS_ROLE_*`** secrets from **`.env-local`** |
| `make github-actions-repo-setup` | **`install-gh`** → **ensure-env** → **`github-actions-sync`** (non-destructive re-runs OK) |
| `make terraform-fmt` | `terraform fmt -recursive` |
| `make terraform-validate` | Build Lambda zip inputs, `init -backend=false`, `validate` |
| `make terraform-plan` | `terraform plan` (needs AWS credentials and initialized backend if configured) |
| `make terraform-apply` | `terraform apply` |

Override Lambda architecture: `make build-lambda LAMBDA_GOARCH=amd64`.

## Phase 1 — AWS (Terraform)

### Step A — Remote state backend (once per AWS account)

1. Copy **`.env-local.example`** to **`.env-local`** at the repo root and set **`AWS_*` credentials**, **`AWS_REGION`**, and **`TF_REMOTE_STATE_BUCKET` / `TF_REMOTE_LOCK_TABLE`** (or **`AWS_S3_BUCKET_NAME`** + **`AWS_DYNAMODB_TABLE_PREFIX`** — lock table becomes `<PREFIX>-terraform-locks`). Use **`booking20-<environment>-<resource>`** for bucket and lock names (e.g. **`booking20-staging-terraform`** + **`booking20-staging-terraform-locks`**). Bucket names **must use hyphens** (underscores are not valid DNS-style bucket names).
2. **Bootstrap** creates the bucket + DynamoDB lock table (this module keeps its **own Terraform state locally** under `infra/tf-backend-bootstrap/`, ignored by Git):
   - `cp infra/tf-backend-bootstrap/terraform.tfvars.example infra/tf-backend-bootstrap/terraform.tfvars` and edit `state_bucket_name` / `lock_table_name` **to match** `.env-local`.
   - `make terraform-backend-bootstrap-init` → `make terraform-backend-bootstrap-apply`.
3. Generate backend partials for `infra/terraform`: **`make terraform-backend-config`** writes **`infra/terraform/.backend/staging.hcl`** and **`production.hcl`** (gitignored).
4. Initialise the application stack backend: **`make terraform-init`** (**staging** state key inside the bucket). If you already had a **local** `infra/terraform/terraform.tfstate`, add **`-migrate-state`** on first init:\
   `terraform -chdir=infra/terraform init -migrate-state -backend-config="$(pwd)/infra/terraform/.backend/staging.hcl"`.

### Step C — GitHub Actions OIDC + deploy roles (once; privileged)

1. Reuse **`remote_state_bucket_name`** / **`remote_lock_table_name`** from **Step A** (must match **`make terraform-backend-config`** output bucket + lock DynamoDB table).
2. **`cp infra/tf-github-actions-iam/terraform.tfvars.example infra/tf-github-actions-iam/terraform.tfvars`** and set **`github_repository`** (`OWNER/NAME`, e.g. `parama/booking-2.0`). Keep **`booking_resource_prefix`** and **`terraform_state_key_prefix`** aligned with **`infra/terraform`** `variables.project` and **`TF_STATE_KEY_PREFIX`** (default **`booking20`**).
3. If the account already has **`token.actions.githubusercontent.com`** IAM OIDC, set **`create_oidc_provider = false`** in `terraform.tfvars` (otherwise the apply will conflict). Otherwise leave **`true`** so Terraform installs it (thumbprints fetched via TLS).
4. From the repo root: **`make terraform-github-actions-iam-init`** → **`make terraform-github-actions-iam-plan`** → **`make terraform-github-actions-iam-apply`**. Uses **locally persisted Terraform state** in `infra/tf-github-actions-iam/` (gitignored — protect or move to backend later).
5. Record **`staging_deploy_role_arn`** / **`production_deploy_role_arn`** for **Step D** (GitHub Secrets).

### Step D — GitHub variables, secrets & deploy workflows

1. **Prerequisites:** Steps **A** + **C** (**`infra/tf-github-actions-iam/terraform.tfvars`** **`github_repository`** matches this repo).
2. **Install**: **`make install-gh`** (Homebrew/macOS) or **`https://cli.github.com`**. **`gh auth login`** with **`repo`** scope (**`gh auth status`**).
3. **Push Variables + Secrets (`gh`):** ensure **`.env-local`** defines **`TF_REMOTE_*`** (+ optional **`AWS_ROLE_TO_ASSUME_STAGING`** / **`AWS_ROLE_TO_ASSUME_PRODUCTION`** from IAM outputs). Run **`make github-actions-sync`** (wraps **`scripts/sync-github-actions-repo-config.sh`**). **`make github-actions-ensure-environments`** creates **`staging`/`production`** Environments via API; **`make github-actions-repo-setup`** runs **install-gh**, ensure-env, then sync. Use **`GITHUB_REPO=owner/name`** make override for a repo other than the git checkout remote. **`DRY_RUN=1`** on the script skips writes.
4. **Manual fallback:** GitHub UI **Settings → Secrets and variables → Actions** — same names as **`vars.TERRAFORM_*`** / **`secrets.AWS_ROLE_*`** consumed by **`deploy-staging/production.yml`**.
5. **`production`** environment: configure **required reviewers** under **Settings → Environments** (manual prod gate ahead of **`terraform apply`**).
6. **Automation:** **`terraform-plan-pr.yml`** (**`pull_request`**; path-filtered — **`terraform plan`** vs staging remote state — needs **`trust_github_pull_request_workflows`** / re-apply IAM). **`deploy-staging.yml`** (**`push`** to **`main`/`master`**) and **`deploy-production.yml`** (**`workflow_dispatch`** + **`environment: production`**).
7. **OIDC trust:** **`trust_staging_github_environment = true`** in **`infra/tf-github-actions-iam`** matches **`jobs.*.environment: staging`**.

### Step B — Deploy the API (`infra/terraform`)

1. **`make terraform-validate`** — ensures `bin/bootstrap` exists and configuration is valid (no AWS backend call; uses **`init -backend=false`**).
2. **Stack naming:** with **`staging.tfvars` / `production.tfvars`** resources use **`booking20-<terraform-environment>-<resource>`** (e.g. **`booking20-staging-*`** and **`booking20-prod-*`** — production keeps API stage **`prod`** per `production.tfvars`). Omit **`-var-file`** for default **`booking20-dev-*`** (`environment = "dev"`).
3. After **Step A**, run **`terraform -chdir=infra/terraform plan -var-file=environments/staging.tfvars`** then **`apply`** with the same **`-var-file`** ( **`make terraform-plan`** / **`make terraform-apply`** do **not** pass **`var-file`** yet).
4. Open **`terraform -chdir=infra/terraform output health_url`** — expect JSON `{"status":"ok","phase":"…"}`.

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

## Phase 6 — Conversations + WhatsApp webhook

- **Domain:** `Conversation`, `Message`, `ConversationChannel` in `internal/domain`.
- **App:** `internal/app/conversations` — ensure/get conversation, list/post messages, normalized webhook handling (`HELP` / `BOOK …` command router calling `bookings.CreateBooking`), optional **Meta** `X-Hub-Signature-256` when `WHATSAPP_APP_SECRET` is set.
- **Adapters:** `MessagingRepository` (`internal/adapters/dynamo/conversation_repo.go`) — conversations under `BUSINESS#…` / `CONVO#…`, index `CONVOIDX#customer#channel`, messages under `CONVO#…` / `MSG#…`, dedup `WHDEDUP#…`; `internal/adapters/outbound/whatsapp_stub.go` implements `ChannelOutbound`.
- **HTTP:** `POST /businesses/{id}/conversations`, `GET …/conversations/{conversationId}`, messages `GET|POST …/messages`; `POST /v1/webhooks/whatsapp` (no tenant header; JSON shape `business_id`, `from_e164`, `text`, optional `message_id`). Active when `Deps.Conversations` is wired (default in `cmd/api`).

## Phase 7 — Payments (GSI3 + confirm gate)

- **Domain:** `Payment`, `PaymentKind`, `PaymentStatus`, and `SucceededPaymentSatisfiesConfirm` policy in `internal/domain/payment.go` (**full** pay must match catalog **Money** when the service has a price; otherwise a succeeded **full** or **deposit** suffices).
- **App:** `internal/app/payments` — create intent via `PaymentCheckoutProvider` (stub in `internal/adapters/paymentstub`), list/get, webhook apply.
- **Adapters:** `PaymentRepository` — `BUSINESS#` / `PAYMENT#…`, **GSI3** `BOOKING#…` / `PAYMENT#…`, idempotency keys `IDEMPOTENCY#PAY#…` (separate namespace from booking idempotency).
- **Bookings:** `ConfirmBooking` calls `confirmPaymentGate` when `Payments` repository is wired (always in `cmd/api`): confirmation **409** until policy is satisfied.
- **HTTP:** `POST|GET /businesses/{id}/payments`, `GET …/bookings/{bookingId}/payments`, `POST /v1/webhooks/payments/{provider}` — normalized JSON `business_id`, `payment_id`, `status`, `external_ref`; optional **`PAYMENT_WEBHOOK_SECRET`** + `X-Payment-Signature: sha256=…`.

## Phase 8 — Notifications (GSI4 + dispatch worker hook)

- **Domain:** `Notification`, kinds, and statuses in `internal/domain/notification.go` (aligned with OpenAPI).
- **App:** `internal/app/notifications` — create/list, `DispatchDue` (query due rows, WhatsApp via `ChannelOutbound`, mark sent/failed), `ScheduleBookingReminder` on new bookings (24h before start, or ~2 minutes ahead if that time is already past).
- **Adapters:** `NotificationRepository` (`internal/adapters/dynamo/notification_repo.go`) — base keys `BUSINESS#…` / `NOTIF#…`, **GSI4** due queue `GSI4PK=NOTIFICATION` and `GSI4SK=SCHED#<padded UnixNano>#<notificationId>`; completed rows move to `NOTIFICATION#SENT` / `NOTIFICATION#FAILED` so they leave the due partition.
- **HTTP:** `GET|POST /businesses/{id}/notifications` (`?status=&cursor=&limit=` on GET). Platform **`POST /v1/platform/notifications/dispatch-due`** (same optional `X-Api-Key` gate as `POST /platform/businesses`) runs one batch of due sends — wire **EventBridge** (or similar) to call it on a schedule in your environment.
- **Wiring:** `cmd/api` registers `notifyOnBookingCreated` with `bookings.Application.Events` so a successful `CreateBooking` schedules the reminder; conversations reuse the same `WhatsAppStub` outbound instance as notifications.

## Phase 9 — Hardening and production readiness

- **Logging:** JSON request logs via `log/slog` (`http_request`: `request_id`, `method`, `path`, `status`, `duration_ms`, …). Panics log through the same logger.
- **HTTP:** Security headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`). Optional **CORS** allowlist (`CORS_ALLOWED_ORIGINS`, comma-separated). Optional per-IP **rate limit** in Lambda (`HTTP_RATE_LIMIT_MAX`, `HTTP_RATE_LIMIT_WINDOW_SEC`).
- **Auth:** `REQUIRE_PLATFORM_API_KEY=true` fails closed (**503**) if `PLATFORM_API_KEY` is not configured — use in prod for platform routes.
- **Terraform:** API Gateway stage **throttling** (`api_gateway_throttle_rate_limit` / `_burst_limit`); **CloudWatch** Lambda error + duration alarms (optional **SNS** email via `alarm_notification_email` — confirm subscription); optional **AWS Budgets** email (`cost_alert_email`, `monthly_budget_usd`). **DynamoDB PITR** toggled via `dynamodb_point_in_time_recovery` or auto for prod-like `environment` values.
- **Docs:** `RUNBOOK.md` (on-call, RPO/RTO, rotations), `THREAT_MODEL.md` (trust boundaries). Example load script: `scripts/loadtest-health.sh`.

Variables live in `infra/terraform/variables.tf` (`aws_region`, `project`, `environment`, etc.).

## CI

GitHub Actions runs **test**, **vet**, and **golangci-lint** on pushes and pull requests to `main` / `master`. **Use pull requests** for changes; keep `main` green.

## Module path

The module is `github.com/parama/booking`. If your organization uses a different path, update `go.mod` and imports accordingly.
