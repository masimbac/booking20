# DYNAMODB_ARCHITECTURE_AND_SCHEMA.md

## 🧭 Overview

This document defines the DynamoDB-based data model and AWS architecture for a **Conversational Commerce Engine for Services (WhatsApp-first)**.

Primary database: Amazon DynamoDB  
Architecture: Serverless (**Lambda-first**), event-driven, multi-tenant  

**Companion engineering choices:** see **`ARCHITECTURE.md`** (canonical). In short: Go (HTTP via **chi**), **REST** behind API Gateway, **DDD** + **hexagonal** layering, **Terraform**, **GitHub Actions**. Persistence adapters implement repository **ports** against this schema.

---

## 🧱 Core Design Principles

- Single-table design
- Access pattern-driven modeling
- Multi-tenancy via partition keys
- Denormalization for performance
- Event-driven architecture

---

## 🗄️ DynamoDB Table Design

### Table: core_table

| PK | SK | EntityType | Attributes |
|----|----|------------|-----------|

---

## 🔑 Key Patterns

- PK = BUSINESS#{businessId}
- SK = ENTITY#{entityType}#{entityId}

---

## 🧩 Entities

### Business
PK: BUSINESS#123  
SK: META#123  

### Customer
PK: BUSINESS#123  
SK: CUSTOMER#456  

### Service
PK: BUSINESS#123  
SK: SERVICE#789  

### Staff
PK: BUSINESS#123  
SK: STAFF#111  

### Booking
PK: BUSINESS#123  
SK: BOOKING#2026-05-10T10:00#999  

### Payment
PK: BUSINESS#123  
SK: PAYMENT#999  

### Conversation
PK: CUSTOMER#456  
SK: CONVO#123  

### Message
PK: CONVO#123  
SK: MSG#timestamp  

### Notification
PK: BUSINESS#123  
SK: NOTIF#timestamp  

### Event
PK: EVENT  
SK: timestamp#uuid  

---

## 🔍 Access Patterns

### Get services for business
PK = BUSINESS#id  
SK begins_with SERVICE#

### Get bookings by date (GSI1)
GSI1PK = BUSINESS#id  
GSI1SK = BOOKING_DATE#timestamp  

### Get customer by phone (GSI2)
GSI2PK = PHONE#number  

### Get messages in conversation
PK = CONVO#id  

---

## 📊 GSIs

### GSI1: Bookings by Date
- PK: BUSINESS#{id}
- SK: BOOKING_DATE#{timestamp}

### GSI2: Customer by Phone
- PK: PHONE#{phone}

### GSI3: Payments by Booking
- PK: BOOKING#{bookingId}

### GSI4: Notifications by Schedule
- PK: NOTIFICATION
- SK: scheduled_at

---

## ☁️ AWS Architecture

### Messaging
- WhatsApp API (Twilio / Meta), inbound webhooks and outbound sends as **adapters** to the application core

### Backend (API-first, REST)
- **AWS Lambda** (Go) — default deployment unit for HTTP and async workers  
- **Amazon API Gateway** (REST) — public REST surface; Lambda handlers use **chi** for routing/middleware where applicable  
- Internal **hexagonal** layout: domain/use-case packages depend on **ports**; Lambda entrypoints are thin **adapters**

### Data
- **DynamoDB** (primary) — `core_table` and GSIs per this document  
- **ElastiCache (Redis)** — optional, for cache or rate limiting as needed

### Async/Eventing
- **EventBridge** / **SQS** — emit/consume domain events (e.g. booking lifecycle, payment confirmations)

### Storage
- **S3** (media, documents)

### Infrastructure as code & delivery
- **Terraform** — DynamoDB tables, GSIs, Lambdas, API Gateway, IAM, queues, buckets, and related AWS configuration  
- **GitHub Actions** — CI (test, lint, security) and CD (artifact build, Terraform plan/apply per environment)

### Monitoring
- **CloudWatch** (logs, metrics, alarms)

---

## 🔁 Event Flow Example

User → WhatsApp → **Lambda** (webhook adapter)  
→ **Application use case** (domain; persists via DynamoDB **repository adapter**)  
→ **EventBridge** (domain event)  
→ **Lambda** (subscriber adapter)  
→ Notification / side effects  

The same booking creation **use case** can be invoked from **REST** (API Gateway → chi router → handler) with identical domain logic.

---

## ⚖️ Trade-offs

### Pros
- Scalable
- Serverless
- Low latency

### Cons
- No joins
- Requires careful design
- GSI management needed

---

## 🚀 Summary

This design supports:
- Multi-tenant WhatsApp booking
- High-scale conversational interactions
- Event-driven automation
- Extensible service commerce platform
- **REST** APIs implemented in **Go** on **Lambda** with **chi**, **Terraform**-managed AWS resources, and **GitHub Actions**-driven delivery
