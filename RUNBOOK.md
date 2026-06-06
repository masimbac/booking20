# Operations runbook (Phase 9)

Use this as a starting point for on-call and production readiness reviews.

## Service overview

- **API:** Single Lambda (`cmd/api`) behind API Gateway REST. Routes are served by `chi` under `/v1/…`.
- **Data:** DynamoDB single table (`core`), GSIs for bookings, customers, payments, notifications.
- **Logs:** Lambda emits **JSON** lines via `log/slog` (`http_request` for each HTTP request).

## Alarms and notifications

If Terraform variables `alarm_notification_email` / `cost_alert_email` are set:

1. **Confirm SNS email** subscription for `alarm_notification_email` after first `terraform apply` (AWS sends a confirmation link).
2. **Lambda error alarm** fires when **Errors** sum > 0 in a five-minute window.
3. **Lambda duration alarm** fires when **Average Duration** > 5 seconds over ten minutes — tune the threshold for your latency SLO (product target: typical responses well under 2 seconds).
4. **Cost budget** emails at **80%** of `monthly_budget_usd` when `cost_alert_email` is set.

## DynamoDB backup / PITR

- **Point-in-time recovery (PITR)** is **on** when `dynamodb_point_in_time_recovery` is true **or** when `environment` is a prod-like value (`staging`, `production`, `prod`, etc.). See `infra/terraform/dynamodb.tf` (`local.dynamodb_pitr_enabled`).
- **RPO:** With PITR, restore to any second within the PITR window (AWS: up to 35 days once continuous backups are enabled).
- **RTO:** Depends on restore path (on-table restore vs export); plan table rebuild or cross-region strategy separately.

## Authentication and tenant isolation

| Surface | Mechanism |
|---------|-----------|
| Platform routes (`POST /v1/platform/…`) | Header **`X-Api-Key`** must match **`PLATFORM_API_KEY`** when that env var is set. |
| Fail-closed platform | Set **`REQUIRE_PLATFORM_API_KEY=true`**: if `PLATFORM_API_KEY` is missing, platform routes return **503**. |
| Tenant routes (`/v1/businesses/{businessId}/…`) | Header **`X-Tenant-Business-Id`** must equal `{businessId}` unless **`SKIP_TENANT_CHECK=true`** (development only). |

**JWT / Cognito** are not wired in this codebase; treat bearer integration as a follow-on behind API Gateway authorizers or app middleware.

## Application hardening env vars (Lambda)

| Variable | Purpose |
|----------|---------|
| `CORS_ALLOWED_ORIGINS` | Comma-separated list; enables CORS only for exact `Origin` matches. |
| `HTTP_RATE_LIMIT_MAX` | Max requests per client IP per window (default **off** if unset). |
| `HTTP_RATE_LIMIT_WINDOW_SEC` | Window length in seconds (default **60**). |
| `REQUIRE_PLATFORM_API_KEY` | When `true`, require a configured `PLATFORM_API_KEY`. |

Per-instance rate limits in Lambda are a **best-effort** backstop; API Gateway stage throttling (Terraform `api_gateway_throttle_*`) limits aggregate traffic.

## Key rotations

1. Generate a new `PLATFORM_API_KEY`.
2. Update Lambda environment (Terraform or console).
3. Roll out callers to send the new header value.
4. Remove the old key after traffic is fully migrated.

## Incident checklist

1. Open **CloudWatch Logs** for the API Lambda; filter by `request_id` from problem+json responses.
2. Check **Lambda Errors** and **Duration** metrics vs alarms.
3. Check **DynamoDB** throttling / `UserErrors` (on-demand mode rarely throttles; watch hot keys).
4. For suspected abuse, tighten **API Gateway throttle** variables and enable **WAF** / usage plans at the edge (not in this module by default).

## Load testing

- Script: `scripts/loadtest-health.sh` (example parallel requests to `/v1/health`).
- For booking hot paths, use your preferred tool (e.g. k6) against a non-production stage with realistic auth headers.
