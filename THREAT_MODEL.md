# Threat model (summary) — Phase 9

This is a lightweight threat model for the MVP API (Lambda + API Gateway + DynamoDB). It supports security reviews and backlog prioritization; it is **not** a formal certification document.

## Trust boundaries

1. **Public Internet → API Gateway (HTTPS)**  
   Untrusted clients. Authentication is limited to static platform API keys and tenant path/header checks (see `RUNBOOK.md`).

2. **API Gateway → Lambda**  
   Managed by AWS; assume trust of API Gateway invocation identity unless you add IAM / Lambda authorizers.

3. **Lambda → DynamoDB**  
   IAM-scoped table access; tenant isolation is **application-enforced** (partition keys and middleware), not row-level security in DynamoDB.

4. **Webhooks (`/v1/webhooks/*`)**  
   Providers should verify signatures (`WHATSAPP_APP_SECRET`, `PAYMENT_WEBHOOK_SECRET`); forged webhook bodies are in scope for abuse if secrets leak.

## Assets

- **Customer PII:** phone numbers, names (DynamoDB `CUSTOMER#` items).
- **Business operational data:** bookings, payments metadata, conversations.
- **Secrets:** platform API key, webhook signing secrets, future payment provider keys.

## Key risks and mitigations

| Risk | Mitigation (current or planned) |
|------|----------------------------------|
| Cross-tenant data access | `X-Tenant-Business-Id` vs path `businessId`; repos scope by `BUSINESS#` PK. |
| Anonymous platform abuse | `PLATFORM_API_KEY`; optional `REQUIRE_PLATFORM_API_KEY`; API Gateway throttling. |
| Credential stuffing / burst abuse | API Gateway **stage throttle**; optional app **rate limit** (`HTTP_RATE_LIMIT_*`); WAF at edge (org decision). |
| Injection via JSON bodies | Handlers decode into typed structs; no raw SQL. |
| Log leakage | Structured logs avoid printing bodies by default; redact before adding payload dumps. |
| Secrets in env | Use AWS Parameter Store / Secrets Manager for production; rotate per RUNBOOK. |
| Availability / cost | CloudWatch alarms; optional AWS Budgets; on-demand DynamoDB; monitor hot partitions. |

## Out of scope for this repository

- Fine-grained **OAuth2 / JWT** for end users (future API Gateway authorizer or middleware).
- **WAF** rules and geo blocking (configure in AWS per environment).
- **Penetration test** execution (track as an external engagement).

## Review cadence

Revisit this document when adding: new public routes, third-party webhooks, admin interfaces, or cross-region replication.
