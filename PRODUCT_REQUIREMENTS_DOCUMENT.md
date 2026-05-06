# PRODUCT_REQUIREMENTS_DOCUMENT.md (PRD)

## 🧭 Product Name
Conversational Commerce Engine for Services (WhatsApp-first)

---

## 🎯 Product Vision

Build a multi-tenant, API-first conversational platform that enables service-based businesses (starting with salons) to manage bookings, payments, and customer interactions entirely through WhatsApp.

---

## 🎯 Goals & Success Metrics

### Goals
- Enable end-to-end booking via WhatsApp
- Reduce friction vs traditional booking apps
- Support multiple businesses on a single platform
- Provide extensible APIs for future integrations

### Success Metrics (MVP)
- ≥ 70% of bookings completed via WhatsApp without human intervention  
- < 2 minutes average booking completion time  
- ≥ 90% successful message delivery rate  
- ≥ 80% reminder effectiveness (reduced no-shows)

---

## 👥 Personas

### Customer (End User)
- Wants fast, simple booking via chat
- Prefers not to install apps

### Business Owner
- Needs booking management and automation

### Platform Admin
- Manages tenants and system usage

---

## 🧱 Epics & Key Features

### Multi-Tenant Management
- Create/manage businesses
- Isolated data per tenant

### WhatsApp Interface
- Messaging (text, buttons, media)
- Booking and support flows

### Conversation Engine
- Stateful sessions
- Context-aware flows

### Booking System
- Services, availability, scheduling
- Booking lifecycle management

### Payments
- Deposit/full payment support
- Webhook confirmations

### CRM
- Customer profiles
- Booking history

### Notifications
- Confirmations, reminders, promotions

### Admin Dashboard
- Calendar, staff, bookings, revenue

### API-First Platform
- **REST** APIs as the system contract; WhatsApp, future UIs, and integrations consume the same APIs
- Features are designable and testable **API-first** (OpenAPI or equivalent as the source of truth for HTTP boundaries)

---

## 🏗️ Technical Direction (Agreed)

| Area | Decision |
|------|----------|
| **Cloud** | AWS |
| **Compute** | **Lambda-first** (serverless handlers as the default unit of deployment) |
| **Data** | Amazon **DynamoDB** (single-table model per platform data doc) |
| **Language / HTTP** | **Go**, lightweight routing with **[chi](https://github.com/go-chi/chi)** |
| **API style** | **REST** |
| **Structure** | **Domain-Driven Design (DDD)** bounded contexts + **Hexagonal** (ports & adapters): domain and application cores independent of AWS, HTTP, and persistence details |
| **IaC** | **Terraform** for AWS resources |
| **CI/CD** | **GitHub Actions** (test, build, deploy pipelines) |

Product features in this PRD are intended to surface through versioned REST resources; conversational and admin experiences are clients of that API.

Canonical engineering detail: **`ARCHITECTURE.md`**.

---

## ⚙️ Non-Functional Requirements

- Performance: <2s response time
- Scalability: multi-tenant ready
- Reliability: retries and idempotency
- Security: auth + isolation

---

## 🚀 MVP Scope

- WhatsApp booking flow
- Multi-tenant support
- Booking + availability
- Basic reminders

---

## 🔮 Future Enhancements

- AI-powered flows
- Marketplace
- Integrations

---

## 📌 Summary

A WhatsApp-first conversational commerce engine designed to scale across service industries. Engineering delivery targets **AWS** with **Lambda-first** Go services, **REST** (chi), **DynamoDB**, **DDD** and **hexagonal** structure, **Terraform** for infrastructure, and **GitHub Actions** for CI/CD—all behind an **API-first** contract.
