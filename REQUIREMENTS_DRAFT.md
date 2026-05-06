# REQUIREMENTS_DRAFT.md

## 🧭 Vision

> **“I’m building a conversational commerce engine for services.”**

A platform that enables businesses (starting with salons) to manage bookings, payments, customer interactions, and operations entirely through conversational interfaces, with WhatsApp as the primary entry point.

The system is API-first, multi-tenant, and extensible, allowing multiple service-based businesses to onboard and operate independently.

---

## 🎯 Objectives

- Enable customers to book services via WhatsApp seamlessly
- Provide businesses with booking management, CRM, and payment tools
- Support multiple businesses (multi-tenant architecture)
- Expose all functionality via APIs for extensibility
- Deliver a low-friction, no-app-required experience
- Lay the foundation for a generic service commerce engine

---

## 👥 Users

### 1. End Customers
- Interact via WhatsApp
- Book, reschedule, cancel appointments
- Receive reminders and notifications
- Make payments or upload proof of payment

### 2. Business Owners
- Manage services, staff, availability
- View and manage bookings
- Track customers and revenue
- Send promotions and notifications

### 3. Platform Admin
- Manage tenants (businesses)
- Monitor system usage
- Configure global settings

---

## 🧱 Core Functional Requirements

### 1. Multi-Tenant Business Management
- Create and manage multiple businesses
- Each business has:
  - Name, location, contact details
  - Services offered
  - Staff members
- Data isolation between businesses

---

### 2. WhatsApp Conversation Interface
- Receive and process incoming messages
- Respond with text, buttons, lists, and media
- Support booking, browsing, and support flows

---

### 3. Conversation Engine
- Maintain user session state across messages
- Track conversation progress
- Store contextual data per session

---

### 4. Booking Management
- Service catalog
- Staff assignment
- Availability management
- Booking lifecycle (Created, Confirmed, Completed, Cancelled, No-show)

---

### 5. Payments Integration
- Generate payment links
- Support deposits or full payments
- Handle webhook confirmations
- Optional proof-of-payment uploads

---

### 6. CRM
- Store customer profiles
- Track booking history and preferences

---

### 7. Notifications Engine
- Booking confirmations
- Reminders
- Promotions
- Follow-ups

---

### 8. Admin Dashboard
- Calendar view
- Manage services and staff
- Customer management
- Revenue tracking

---

### 9. API-First Architecture
- **REST** is the primary external contract; behavior is specified and delivered **API-first** (e.g. OpenAPI-driven design, contract tests)
- Conversational channels (WhatsApp), admin tools, and third parties integrate via the same REST surface

---

### 10. Event-Driven Architecture
- Emit events (BookingCreated, PaymentReceived, etc.)
- Enable integrations and scalability

---

### 11. AI-Assisted Interaction (Optional)
- Natural language understanding
- Personalization and recommendations

---

## 🗄️ Non-Functional Requirements

- Scalability
- Performance
- Reliability
- Security
- Compliance

---

## 🧰 Technology Stack & Architecture (Agreed)

Full stack and layering rules: **`ARCHITECTURE.md`** (canonical). Summary below.

### Platform

- **Cloud:** AWS  
- **Compute:** **Lambda-first** — business logic ships as Go functions; scale and ops align with serverless by default  
- **API exposure:** Amazon API Gateway (REST) → Lambda  
- **HTTP in Go:** **[chi](https://github.com/go-chi/chi)** for lightweight routing and middleware inside Lambda (e.g. via API Gateway–Lambda proxy integration)

### Data & async

- **Primary store:** **Amazon DynamoDB** (single-table design; see `DYNAMODB_ARCHITECTURE_AND_SCHEMA.md`)  
- **Caching (optional):** ElastiCache (Redis) where read-heavy or session-adjacent paths justify it  
- **Async / events:** EventBridge, SQS (and similar AWS primitives) for domain and integration events  
- **Object storage:** S3 for media and documents  

### Conversations & integrations

- **WhatsApp:** Meta Cloud API and/or provider (e.g. Twilio), implemented as **adapters** behind application ports (hexagonal)

### Code structure

- **Language:** **Go**  
- **DDD:** Model the problem with explicit **bounded contexts** (e.g. booking, tenancy, conversations, payments) and ubiquitous language in domain types  
- **Hexagonal architecture:**  
  - **Domain / application core:** no imports of AWS SDKs, chi, or DynamoDB types  
  - **Ports:** interfaces defined by the application (e.g. repositories, message senders, event publishers)  
  - **Adapters:** Lambda + REST (chi), DynamoDB repositories, WhatsApp webhooks, EventBridge/SQS in **infrastructure** packages  

### Delivery

- **IaC:** **Terraform** for AWS resources (tables, Lambdas, API Gateway, IAM, queues, buckets)  
- **CI/CD:** **GitHub Actions** — lint, test, security checks, container or zip build artifacts, Terraform plan/apply in controlled environments  

### API-first workflow

- REST resources and error model are defined before channel-specific UX; WhatsApp flows call the same use cases as authenticated REST clients.

---

## 🚀 MVP Scope

### Phase 1
- WhatsApp integration
- Basic booking flow
- Availability + booking creation
- Reminders

### Phase 2
- Payments
- CRM
- Admin dashboard

### Phase 3
- Multi-tenant onboarding
- Promotions
- AI enhancements

---

## 🔮 Future Enhancements

- Multi-channel support
- Marketplace
- Analytics
- Loyalty programs
- Integrations

---

## 💡 Key Differentiator

- WhatsApp-first experience
- API-first design
- Conversational UX over traditional UI

---

## 📌 Summary

This platform starts as a WhatsApp-based booking system but evolves into a scalable conversational commerce engine for service businesses. Implementation is **API-first REST** on **AWS** (**Lambda-first** Go + **chi**, **DynamoDB**), with **DDD** and **hexagonal** boundaries, **Terraform**, and **GitHub Actions**.
