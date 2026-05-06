# Architecture (engineering source of truth)

This file is the **canonical reference** for how the **Conversational Commerce Engine for Services** is built. Product intent lives in `PRODUCT_REQUIREMENTS_DOCUMENT.md`; functional scope in `REQUIREMENTS_DRAFT.md`; DynamoDB keys, GSIs, and data patterns in `DYNAMODB_ARCHITECTURE_AND_SCHEMA.md`. REST resource map, domain models, and bounded-context **services** plan: **`PLANNING.md`**. Phased delivery checklist: **`IMPLEMENTATION_PLAN.md`**.

---

## Agreed stack

| Area | Decision |
|------|----------|
| **Cloud** | AWS |
| **Compute** | **Lambda-first** — Go functions as the default deployable unit (HTTP and async workers) |
| **API** | **REST** via **Amazon API Gateway** → **AWS Lambda** |
| **HTTP (Go)** | **[chi](https://github.com/go-chi/chi)** — lightweight router and middleware inside Lambda (e.g. API Gateway proxy integration) |
| **Data** | **Amazon DynamoDB** — single-table design per `DYNAMODB_ARCHITECTURE_AND_SCHEMA.md` |
| **Optional** | **ElastiCache (Redis)**, **Amazon S3**, **EventBridge**, **SQS** as needed per use case |
| **Messaging (WhatsApp)** | Meta Cloud API and/or provider (e.g. Twilio) — **adapters** only, not core domain |
| **IaC** | **Terraform** for AWS resources |
| **CI/CD** | **GitHub Actions** — lint, test, security, artifacts, Terraform plan/apply by environment |

---

## API-first

- **REST** is the system contract: OpenAPI (or equivalent) defines resources, errors, and versioning where practical. Initial spec: **`openapi/openapi.yaml`**.
- WhatsApp webhooks, future web/mobile admins, and partners consume the **same** application use cases as first-class REST clients (thin transport adapters).

---

## DDD and hexagonal layout

- **Bounded contexts** (e.g. tenancy, booking, conversations, payments) own their ubiquitous language and domain model; avoid leaking one context’s types into another’s core.
- **Hexagonal (ports & adapters)**:
  - **Domain / application:** pure business rules and orchestration; **no** imports of AWS SDKs, chi, or DynamoDB concrete APIs.
  - **Ports:** interfaces **defined by** the application (repositories, outbound messaging, clocks, id generation, event publication).
  - **Adapters:** Lambda entrypoints, chi HTTP handlers, DynamoDB item mapping, EventBridge/SQS publishers, WhatsApp clients — live at the edges and implement ports.

This keeps Lambdas small: parse input → call a use case → map result or error to HTTP/events.

---

## AWS shape (logical)

- **Ingress:** API Gateway (REST) for public/internal APIs; WhatsApp callbacks to dedicated Lambda(s) or routes.
- **Compute:** Go Lambda functions (per bounded context or grouped by deployment boundary — decide per repo layout).
- **Data:** DynamoDB `core_table` + GSIs; S3 for blobs; optional Redis for hot reads or throttling.
- **Events:** EventBridge / SQS for domain and integration events; idempotent consumers where duplicates are possible.
- **Observability:** CloudWatch logs, metrics, alarms.

---

## Delivery

- **Terraform** owns tables, indexes, Lambdas, API Gateway, IAM, queues, and buckets; no hand-clicked production drift.
- **GitHub Actions** enforces gates before promote; apply Terraform from protected branches or manual approval as your policy requires.

---

## Summary

**AWS**, **Lambda-first** **Go** with **chi** and **REST**, **DynamoDB**, **DDD** + **hexagonal** boundaries, **Terraform**, **GitHub Actions**, and **API-first** contracts — with persistence detail in `DYNAMODB_ARCHITECTURE_AND_SCHEMA.md`.
