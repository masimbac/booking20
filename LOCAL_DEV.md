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

Variables live in `infra/terraform/variables.tf` (`aws_region`, `project`, `environment`, etc.).

## CI

GitHub Actions runs **test**, **vet**, and **golangci-lint** on pushes and pull requests to `main` / `master`. **Use pull requests** for changes; keep `main` green.

## Module path

The module is `github.com/parama/booking`. If your organization uses a different path, update `go.mod` and imports accordingly.
