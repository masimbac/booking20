# Implementation plan

End-to-end delivery plan for the **Conversational Commerce Engine** (WhatsApp-first), aligned with **`ARCHITECTURE.md`**, **`PLANNING.md`**, **`openapi/openapi.yaml`**, and **`DYNAMODB_ARCHITECTURE_AND_SCHEMA.md`**. Treat this as a living checklist; reorder sub-tasks when dependencies change.

---

## Goals and guardrails

- **API-first:** Implement behavior behind **`openapi/openapi.yaml`**; extend the spec when the contract changes (regenerate clients/docs if you adopt codegen later).
- **Hexagonal:** Domain and application packages **must not** import AWS SDKs, chi, or DynamoDB-specific types. Persistence and HTTP live in **adapters**.
- **Tenant isolation:** Every business-scoped use case receives a **`business_id`** (or platform-only operations are explicitly gated). Repository adapters scope queries to the correct partition keys.
- **MVP (PRD):** WhatsApp booking flow, multi-tenant support, booking + availability, basic reminders — deliver minimum vertical slices per phase rather than all endpoints at once.

---

## Suggested repository layout (Go)

Not prescriptive for filenames, but useful for planning workstreams:

```text
cmd/
  api/              # Lambda bootstrap + chi mount (API Gateway proxy)
  worker-notify/    # Optional: scheduled / queue-driven worker
internal/
  domain/           # Pure types, invariants (or per-context subdirs)
  app/              # Use cases + ports (interfaces)
  adapters/
    http/           # chi handlers, request/response DTOs, problem+json
    dynamo/         # single-table mappers, repositories
    webhooks/       # WhatsApp / payment signature helpers → call app
    events/         # EventBridge / SQS publishers (optional early)
infra/
  terraform/        # Modules: dynamodb, apigw, lambda, iam, ...
openapi/
  openapi.yaml
```

---

## Phase 0 — Bootstrap and quality gates

**Objective:** Repeatable build, test, and CI before feature code accumulates.

| Task | Notes |
|------|--------|
| Initialize Go module (`go 1.22+` or team standard) | Root `go.mod` |
| Add **Makefile** or **taskfile** targets: `test`, `lint`, `fmt`, `build-lambda` | Lambda = `GOOS=linux GOARCH=amd64` (or `arm64` per platform choice) |
| Configure **golangci-lint** | Align rules with team |
| Add **GitHub Actions** workflow: lint + unit tests on PR | Optional: cache modules |
| Document local dev (optional): **SAM**, **LocalStack**, or invoke handlers with synthetic API GW events | Defer if team goes straight to dev AWS |

**Exit criteria:** CI green on an empty or stub module; merge policy defined (PR required).

---

## Phase 1 — Infrastructure baseline (Terraform)

**Objective:** A disposable dev environment with DynamoDB and a dumb API path proving deploy wiring.

| Task | Notes |
|------|--------|
| Terraform backend (S3 + DynamoDB lock) for non-dev | Dev may use local state initially |
| **`core_table`** + attributes per data doc | PK/SK, TTL if needed later |
| **GSIs:** GSI1 bookings by date, GSI2 customer by phone, GSI3 payments by booking, GSI4 notifications schedule | Names and attribute definitions locked to access patterns |
| IAM roles for Lambdas (least privilege: DynamoDB indexes only as needed) | |
| **API Gateway REST** + **Lambda proxy** integration (single `api` Lambda to start) | One `ANY /{proxy+}` or explicit routes |
| CloudWatch log groups, retention | |
| Outputs: table name, API URL | For integration tests / manual checks |

**Exit criteria:** `terraform apply` yields a reachable **health** route (e.g. `GET /v1/health`) returning 200 from Go on Lambda.

---

## Phase 2 — HTTP shell and problem details

**Objective:** chi router, versioning, and consistent errors — no business logic yet.

| Task | Notes |
|------|--------|
| Lambda **handler** bridging API Gateway V2 / proxy payload to `chi` | Use `aws-lambda-go` + adapter pattern |
| Mount routes under `/v1`; middleware: **request ID**, **recovery**, **logging** | |
| Implement **`application/problem+json`** mapper | Map domain errors to **409** / **422** where appropriate per **PLANNING.md** |
| Stub handlers for a subset of **openapi** routes returning **501** or static fixtures | Validates route table early |
| Optional: OpenAPI **request validation** middleware (custom or third-party) | |

**Exit criteria:** Postman/curl against deployed API hits real Lambda + chi; errors are structured Problem JSON.

---

## Phase 3 — Tenancy and catalog (vertical slice)

**Objective:** First persisted entities; proves single-table writes and tenant scoping.

| Task | Notes |
|------|--------|
| Domain + use cases: **Business**, **Service**, **Staff** (minimal fields matching **openapi** schemas) | |
| **DynamoDB adapters:** `BUSINESS#` partition, `META#`, `SERVICE#`, `STAFF#` SKs | Integration tests with **dynamodb local** or real dev table |
| Implement **REST:** `POST /platform/businesses`, `GET/PATCH /businesses/{id}` | Platform auth may be a static API key or IAM-only in dev |
| Implement **REST:** services and staff CRUD under `/businesses/{id}/…` | |
| Authorization placeholder: assert caller’s tenant matches `businessId` on business-scoped routes | |

**Exit criteria:** Create a business, add services and staff, read back from DynamoDB through the API only.

---

## Phase 4 — CRM and availability

**Objective:** Customers and bookable time; prerequisites for booking.

| Task | Notes |
|------|--------|
| Domain + repo: **Customer**; **GSI2** `PHONE#e164` for lookup | Enforce uniqueness policy per tenant |
| REST: customers list/create/get/patch, **GET …/customers/by-phone** | |
| **Availability rules** model (start simple: weekly blocks per staff or business) | |
| REST: **PUT …/availability/rules**, **GET …/availability/slots** | MVP: compute slots in app from rules + service duration; optimize later |
| Unit tests for slot generation and timezone edge cases (**business timezone**) | |

**Exit criteria:** For one business, customer resolved by phone; slots returned for a date range without overlapping staff double-book at read time (creation enforced in Phase 5).

---

## Phase 5 — Bookings (core aggregate)

**Objective:** End-to-end appointment lifecycle driving MVP value.

| Task | Notes |
|------|--------|
| Domain: **Booking** state machine (`created` → `confirmed` → `completed` \| `cancelled` \| `no_show`) | Explicit transition methods; reject illegal jumps |
| **GSI1** writes for booking date queries; conflict check before create (staff + time window) | |
| REST: **POST /bookings** with **Idempotency-Key** persistence (conditional writes or idempotency table — choose one design) | |
| REST: list by range, get, **confirm / cancel / complete / no-show** | |
| Emit **domain events** (outbox pattern or direct EventBridge publish) for `BookingCreated`, etc. | Start with synchronous publish; refine with outbox if needed |

**Exit criteria:** Two clients cannot claim the same staff slot; idempotent replay of booking creation does not double-book.

---

## Phase 6 — Conversations and WhatsApp

**Objective:** Inbound messages trigger the same use cases as REST.

| Task | Notes |
|------|--------|
| Domain + repo: **Conversation**, **Message** (keys per data doc: `CUSTOMER#`, `CONVO#`, `MSG#`) | |
| REST: ensure conversation, get conversation, list/post messages | |
| **Webhook Lambda** (or shared `api` with `/webhooks/whatsapp`): verify signature, normalize payload → **AppendInboundMessage** / command router | |
| Outbound **port** `ChannelOutbound`; adapter calls WhatsApp provider | |
| Minimal conversation **state** machine to drive booking steps over chat | |

**Exit criteria:** Message in WhatsApp creates/updates customer/booking through application use cases; outbound reply sent via provider.

---

## Phase 7 — Payments

**Objective:** Deposits/full pay aligned with bookings.

| Task | Notes |
|------|--------|
| Domain + repo: **Payment**; **GSI3** for booking → payments | |
| REST: create payment (**Idempotency-Key**), get payment, list by booking | |
| Provider adapter: create checkout session / payment link | Stub provider acceptable for dev |
| **POST /webhooks/payments/{provider}`**: verify, map to **RecordPaymentWebhook** | |

**Exit criteria:** Booking can move to confirmed only when business rules + payment rules satisfied (define rule in code explicitly).

---

## Phase 8 — Notifications and workers

**Objective:** Reminders and operational visibility (MVP “basic reminders”).

| Task | Notes |
|------|--------|
| Domain + repo: **Notification**; **GSI4** for scheduled work | |
| REST: create/list notifications | |
| **Worker Lambda** (EventBridge schedule or SQS): query due notifications, send via `ChannelOutbound`, handle retries + DLQ | |
| Wire **BookingCreated** → schedule reminder event | |

**Exit criteria:** Time-shifted booking receives reminder without manual API call.

---

## Phase 9 — Hardening and production readiness

**Objective:** NFRs from PRD: **<2s** typical latency, security, operability.

| Task | Notes |
|------|--------|
| Authentication: replace placeholders (**JWT / Cognito / API keys**) per product decision | Enforce platform vs tenant roles |
| Rate limiting (API Gateway / usage plans or edge WAF) | |
| WAF, TLS-only, CORS policy for future browser clients | |
| Structured logging, metrics, tracing (X-Ray optional) | Alarm on 5xx and latency SLO |
| Backup/point-in-time recovery policy for DynamoDB | Document RPO/RTO |
| Load test hot paths (list bookings, create booking) | Tune RCU/WCU or on-demand |

**Exit criteria:** Threat model reviewed; runbooks for on-call; cost alerts configured.

---

## Testing strategy

| Layer | Scope |
|-------|--------|
| **Unit** | Domain transitions, slot math, idempotency key logic, webhook signature parsing |
| **Integration** | DynamoDB adapters against local or dev table; contract tests against **openapi** (schemathesis or Dredd optional) |
| **E2E** (later) | Script: create business → catalog → customer → slot → booking → reminder |

Run integration tests in CI only if a DynamoDB-compatible endpoint is available in pipeline; otherwise nightly against dev AWS.

---

## CI/CD (GitHub Actions) progression

1. **PR:** `go test`, `golangci-lint`, Terraform **`fmt` + `validate`** (and `terraform plan` with read-only creds if feasible).  
2. **Main:** Build Lambda zips (or container images), upload artifact, **`terraform apply`** to staging with approval gate.  
3. **Production:** Manual approval + canary or single region deploy per org policy.

---

## Documentation maintenance

| When | Update |
|------|--------|
| New endpoint or breaking change | **`openapi/openapi.yaml`** first, then handlers |
| New access pattern | **`DYNAMODB_ARCHITECTURE_AND_SCHEMA.md`** (and GSIs — prefer additive changes) |
| Context boundaries shift | **`PLANNING.md`** services section |

---

## Risks and mitigations

| Risk | Mitigation |
|------|------------|
| Single-table hot partition on one huge tenant | Monitor consumed capacity; consider sharding prefix in PK later |
| Idempotency bugs double-charge or double-book | Centralize idempotency storage; integration tests for replay |
| WhatsApp webhook duplicates | Provider id de-duplication + idempotent message append |
| Lambda + chi cold starts | Keep binaries lean; ARM64 if cost/latency win; avoid heavy init in `init()` |
| Terraform drift | Exclusive management via CI apply; no console edits |

---

## Milestone summary (TL;DR)

| Phase | Milestone |
|-------|-----------|
| 0 | CI + module bootstrapped |
| 1 | Terraform + DynamoDB + API GW + health Lambda |
| 2 | chi + problem+json + route skeleton |
| 3 | Business, services, staff persisted |
| 4 | Customers + availability + slots |
| 5 | Bookings + idempotency + transitions + events |
| 6 | Conversations + WhatsApp in/out |
| 7 | Payments + provider webhooks |
| 8 | Notifications + scheduled worker |
| 9 | Auth, limits, observability, prod readiness |

---

## References

- `ARCHITECTURE.md` — stack, DDD/hexagonal rules  
- `PLANNING.md` — routes, bounded contexts, Lambda split  
- `openapi/openapi.yaml` — REST contract  
- `DYNAMODB_ARCHITECTURE_AND_SCHEMA.md` — keys and GSIs  
- `PRODUCT_REQUIREMENTS_DOCUMENT.md` — MVP and NFRs  
