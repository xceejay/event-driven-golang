# Event Engine Starter — Implementation Prompt

You are a senior software engineer. Your task is to generate a complete,
production-grade, domain-agnostic **event lifecycle engine** starter kit
called `event-engine-starter`.

This prompt is deliberately language-agnostic. Choose the language and
ecosystem the user specifies (Go, Java, Python, TypeScript, Rust, etc.)
and select the idiomatic libraries, patterns, and tooling for that
language. Do NOT default to Java/Spring unless explicitly asked.

Strip away ALL domain-specific concerns. Use generic placeholder names.
The result must be a reusable bootstrap repository someone can fork to
build any reliable event-driven system.

---

## REFERENCE SOURCE CODE

The following reference implementation (Java/Spring) is available at:
`/home/joel/downloads/event-engine-develop/event-engine-develop/`

Read it thoroughly before generating any code. Use it as the ground truth
for architectural decisions, state machine behaviour, method contracts,
and data model design. The reference is in Java — translate the patterns
idiomatically into the target language. Do NOT copy Java-isms.

Key paths to study:
- `src/main/java/.../model/`       — domain models and state machine
- `src/main/java/.../service/`     — service interfaces and implementations
- `src/main/java/.../job/`         — scheduled job patterns
- `src/main/java/.../repository/`  — persistence layer
- `src/main/java/.../controller/`  — API surface (gRPC)
- `src/main/java/.../config/`      — wiring and configuration
- `src/main/resources/application.yml`           — config shape
- `src/main/resources/db/changelog/init/`        — DB schema
- `src/main/resources/db/changelog/changesets/`  — seed config pattern
- `Docs/eventLifecycle.mmd`                      — state machine diagram

When reading the source, mentally replace:
- `pawapay`, `psp`, `mtn`, `momo`, `deposit`, `payout`, `remittance`
  → generic equivalents
- `DEPOSIT_FLOW`, `PAYOUT_FLOW`, etc. → `FLOW_A`, `FLOW_B`
- Any country codes, currency codes, telco names → remove entirely

---

## WHAT TO BUILD

A self-contained microservice that acts as a generic **event lifecycle
engine**. It:

- Receives events from external adapter services via a network API (gRPC
  recommended; REST acceptable if gRPC is not idiomatic in the target
  language)
- Persists and tracks events through a well-defined state machine
- Dispatches events to a message broker for adapters to consume and process
- Handles retries, suspensions, expiry, and on-failure chaining
- Drives all retry/timeout behaviour from per-event-type configuration
  stored in a relational database — no hardcoded retry logic

---

## TECHNOLOGY SELECTION GUIDE

Choose the idiomatic equivalent in the target language for each concern:

| Concern                        | Choose idiomatic equivalent for target language         |
|--------------------------------|---------------------------------------------------------|
| HTTP/RPC API (inbound)         | gRPC preferred; REST if gRPC is not idiomatic           |
| Message broker (outbound)      | ActiveMQ, RabbitMQ, NATS, or equivalent                 |
| Primary DB (event state)       | MySQL or PostgreSQL                                     |
| Payload storage                | Redis or equivalent key-value store                     |
| In-memory cache (config)       | Any in-process TTL cache                                |
| DB migrations                  | Any migration tool idiomatic to the language            |
| Scheduled jobs                 | Language-native scheduler or cron library               |
| Metrics                        | Prometheus-compatible metrics library                   |
| Testing                        | Testcontainers or equivalent for integration tests      |
| Dependency injection           | Idiomatic to the language (DI framework or manual)      |

**Broker TTL requirement**: the chosen message broker MUST support
per-message TTL (time-to-live). This is a hard architectural requirement.
If the broker does not natively support TTL, implement TTL enforcement
in the consumer.

---

## EVENT STATE MACHINE

Implement the following states as a type-safe construct (enum, const
block, discriminated union — whatever is idiomatic):

```
AWAITING_PROCESSING
     ↓  (scheduler/dispatcher picks it up)
DISPATCHED
     ↓  (adapter acquires processing permit)
BEING_PROCESSED
     ↓                        ↓
PROCESSED (terminal)     FAILED
                              ↓  (if retry attempts remain)
                         AWAITING_PROCESSING   ← retry loop
                              ↓  (if suspended waiting on external trigger)
                         SUSPENDED
                              ↓  (trigger arrives OR timeout expires)
                         AWAITING_PROCESSING   ← reactivated
```

Terminal states: `PROCESSED`, `FAILED`, `CANCELED`

Provide these helper predicates on the status type:
- `isFinalStatus` — true for PROCESSED, FAILED, CANCELED
- `isEligibleForDispatching` — true for AWAITING_PROCESSING only
- `isRejectableForProcessing` — true for states that must reject
  a processing permit request (anything that is not DISPATCHED)

---

## CORE DATA MODEL

### Event

Fields:
- `id` — UUID, primary key
- `eventType` — string (e.g. "ORDER_CREATED", "NOTIFICATION_REQUESTED")
- `flowType` — string or enum identifying which flow this belongs to
- `flowId` — string grouping all events in one business transaction
- `status` — EventStatus
- `version` — integer, used for optimistic locking
- `attemptsLeft` — integer
- `attemptsFailed` — integer
- `attemptScheduledAt` — timestamp, when next attempt should fire
- `attemptDueDate` — timestamp, deadline for the current attempt
- `eventProcessingDueDate` — timestamp, absolute deadline across all attempts
- `onFailEventType` — string (nullable), event type to publish on terminal failure
- `scheduleState` — IMMEDIATE or SCHEDULED
- `createdAt` — timestamp
- `updatedAt` — timestamp

### EventPayload (stored separately from Event — key-value store)

- `eventId` — UUID (key)
- `payload` — serialized JSON string (value)

Rationale: separating payload from state keeps the primary event table
lean and fast for status queries. Payloads can be large; state rows are
always small.

### EventLifecycleConfig (relational DB, cached in-process)

- `id` — auto-increment integer
- `eventType` — string, unique
- `flowType` — string
- `maxAttempts` — integer
- `eventLifespanSeconds` — integer, total time budget for all attempts
- `isSuspended` — boolean, pauses dispatch of all events of this type
- `attemptLifecycleConfigs` — JSON array of AttemptLifecycleConfig

### AttemptLifecycleConfig (embedded in EventLifecycleConfig as JSON)

- `attemptNumber` — integer (1-based)
- `delayBeforeAttemptSeconds` — integer

### EventErrorLog

- `id` — UUID
- `eventId` — UUID
- `errorMessage` — string
- `stackTrace` — string (nullable)
- `occurredAt` — timestamp

---

## FLOW TYPES

Define two example flow types:
- `FLOW_A` — represents e.g. "Order Processing"
- `FLOW_B` — represents e.g. "Notification Delivery"

**Critical design constraint**: adding a new flow type (FLOW_C) must
require ONLY:
1. Adding the new flow type identifier (enum value, constant, etc.)
2. Registering its three scheduler jobs
3. Inserting SQL rows into `event_lifecycle_config`

No other code changes should be needed.

---

## PERSISTENCE LAYER

### EventRepository

Implement these operations:

- `initiate(command)` — idempotent insert: if an event with this ID
  already exists, do nothing (INSERT ... ON CONFLICT DO NOTHING or
  equivalent)
- `load(eventId)` → Event — load by ID, error if not found
- `findByFlowAndCreatedBefore(flowType, timestamp)` → []Event
- `findByFlowAndScheduledBefore(flowType, timestamp)` → []Event
- `findByFlowAndScheduledBeforeInclusive(flowType, timestamp)` → []Event
- `markAsDispatched(command)` — optimistic lock: increment version,
  update status; return conflict error if version mismatch
- `acquireProcessingPermit(command)` — optimistic lock
- `switchToNextAttempt(command)` — optimistic lock
- `markAsSucceeded(command)` — optimistic lock
- `markAsFailed(command)` — optimistic lock
- `activateSuspended(command)` — optimistic lock
- `getQueueSize(flowType, status)` → integer

**Optimistic locking pattern**: every write command carries the event's
current `version`. The SQL update includes `WHERE version = :version`.
If 0 rows are affected, return/raise a concurrency conflict error.
Never use pessimistic locks (SELECT FOR UPDATE).

### EventPayloadRepository

- `save(eventId, payload)` — upsert
- `load(eventId)` → string
- `resolveFlowId(eventId)` → string (may cache the flowId alongside payload)

Implement two variants:
- Production: backed by Redis or equivalent
- Testing: backed by an in-process map

### EventLifecycleConfigRepository

- `findByEventType(eventType)` → EventLifecycleConfig or not-found
- `save(config)`
- `updateSuspensionState(eventType, suspended bool)`

Cache all reads in-process with a 60-second TTL. The cache is the
primary performance path — DB is only hit on cache miss.

---

## SERVICE LAYER

### EventProcessingService (low-level)

Thin layer over the repository. Owns all state transitions.
On concurrency conflict, raise/return a `RaceConditionError` (do not
retry here — retry happens at the caller).

Operations mirror the repository operations plus:
- `initiateIdempotently(command, payload)` — saves event and payload
  atomically (use a DB transaction)
- `load(eventId)` → Event
- `loadPayload(eventId)` → string
- `resolveFlowId(eventId)` → string
- `tryMarkAsDispatched(event)` → bool (false = lost race, not an error)
- `tryAcquireProcessingPermit(event)` → bool
- `trySwitchToNextAttempt(event, config)` → bool
- `tryMarkAsSuccessfullyProcessed(event)` → bool
- `tryMarkAsFailedToProcess(event, errorMessage)` → bool
- `tryActivateSuspended(event)` → bool

### EventLifecycleManagementService (high-level orchestrator)

This is the single entry point for all callers (API handlers and
scheduled jobs). It owns retry orchestration, spinoff event creation,
and on-fail event triggering.

Operations:
- `publish(request)` — idempotent; creates event and payload
- `tryAcquireProcessingPermit(eventPointer)` → ProcessingContext or nil/None
- `reportAsSuccessfullyProcessed(eventPointer, result)` — marks event
  as PROCESSED; if result contains spinoff events, publishes each one
- `reportAsFailed(eventPointer, errorLog)` — decides retry or terminal
  failure; on terminal failure, publishes onFailEventType if configured
- `findPendingEvents(flowType)` → []Event
- `findExpiredSuspendedEvents(flowType)` → []Event
- `processPendingEvents(flowType)` — dispatches all pending events
- `activateSuspendedEvents(flowType)` — reactivates expired suspended events
- `suspendEventType(eventType)` — pauses dispatch of this event type
- `unsuspendEventType(eventType)` — resumes dispatch

**Race condition retry policy** (apply in this service, not lower):
- On `RaceConditionError` from lower layer, retry the operation
- Max 10 retries
- Fixed 15ms delay between retries
- After 10 retries, propagate as an error

### EventDispatcher (internal, called by EventLifecycleManagementService)

Dispatches a single event to the message broker.

**TTL-aware dispatch**: select the message TTL based on remaining time
until `eventProcessingDueDate`. Use the smallest TTL bucket that is
≥ the remaining time. This prevents dead messages from being delivered
after their deadline has passed.

TTL buckets (seconds): 30, 75, 150, 300, 600, 1200, 3600, 7200, 86400

If remaining time > 7200s, use 86400s TTL.
If remaining time ≤ 0s, do not dispatch — mark as failed instead.

Queue/topic naming: `event-engine.{eventType}` (all lowercase).

Serialize the event (not just the ID) as JSON before publishing.
Adapters receive the full event from the broker and use the embedded
eventId to call back into the engine.

---

## SCHEDULED JOBS

Implement three job types per flow type, plus one metrics job.
All jobs must be:
- Stateless — safe to run concurrently on multiple instances
- Bounded — configurable batch size, max items, max failures, max runtime
- Independent — failure of one job must not affect others

### Default job parameters
- `batchSize`: 100
- `maxItems`: 100,000
- `maxFails`: 100
- `maxRuntimeMs`: 30,000

### Job 1 — AwaitingProcessingJob (one per flow type)

- Interval: 5000ms, startup delay: 2750ms
- Query: events with status AWAITING_PROCESSING, createdAt ≤ now
- Action: call `processPendingEvents(flowType)`

### Job 2 — ScheduledEventsProcessingJob (one per flow type)

- Interval: 5000ms, startup delay: 3750ms
- Query: events with scheduleState=SCHEDULED, attemptScheduledAt ≤ now
- Action: call `processPendingEvents(flowType)`

### Job 3 — ExpiredSuspendedEventsProcessingJob (one per flow type)

- Interval: 5000ms, startup delay: 4000ms
- Query: events with status SUSPENDED, attemptDueDate ≤ now
- Action: call `activateSuspendedEvents(flowType)`

### QueueSizeMetricsJob (one global)

- Interval: 60,000ms
- For each (flowType, status) combination, query count and emit as a
  gauge metric tagged with `flow_type` and `event_status`
- Expose in Prometheus format

---

## API SURFACE

### Primary API (gRPC preferred)

Define the following service contract. If using gRPC, write a proper
`.proto` file. If using REST, use equivalent HTTP endpoints with JSON.

```
Service: EventEngineApi

Publish(request) → response
  request:  eventType, flowType, flowId, payload (bytes/string),
            onFailEventType (optional)
  response: eventId

TryAcquireProcessingPermit(request) → response
  request:  eventId
  response: acquired (bool), eventId, payload (bytes/string), flowId

ReportAsSuccessfullyProcessed(request) → response
  request:  eventId,
            spinoffEvents[] (each: eventType, payload, onFailEventType?)
  response: acknowledged (bool)

ReportAsFailed(request) → response
  request:  eventId, errorMessage, stackTrace (optional)
  response: acknowledged (bool)
```

### Management API (gRPC or REST)

```
Service: EventEngineManagementApi

GetEventLifecycleConfig(eventType) → EventLifecycleConfig
UpdateEventLifecycleConfig(config) → acknowledged
SuspendEventType(eventType)        → acknowledged
UnsuspendEventType(eventType)      → acknowledged
```

---

## DATABASE SCHEMA

Implement using whatever migration tool is idiomatic for the target
language (Flyway, Liquibase, golang-migrate, Alembic, Prisma, etc.).

### Schema

```sql
-- Event state table (primary, always small rows)
CREATE TABLE IF NOT EXISTS event (
  id                        CHAR(36)     NOT NULL,
  event_type                VARCHAR(255) NOT NULL,
  flow_type                 VARCHAR(100) NOT NULL,
  flow_id                   VARCHAR(255) NOT NULL,
  status                    VARCHAR(50)  NOT NULL,
  version                   BIGINT       NOT NULL DEFAULT 0,
  attempts_left             INT          NOT NULL,
  attempts_failed           INT          NOT NULL DEFAULT 0,
  attempt_scheduled_at      DATETIME(3)  NULL,
  attempt_due_date          DATETIME(3)  NULL,
  event_processing_due_date DATETIME(3)  NULL,
  on_fail_event_type        VARCHAR(255) NULL,
  schedule_state            VARCHAR(50)  NOT NULL DEFAULT 'IMMEDIATE',
  created_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
                                         ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  INDEX idx_event_flow_status    (flow_type, status),
  INDEX idx_event_flow_scheduled (flow_type, schedule_state, attempt_scheduled_at),
  INDEX idx_event_created        (created_at)
);

-- Payload stored separately to keep event table lean
CREATE TABLE IF NOT EXISTS event_payload (
  event_id   CHAR(36)  NOT NULL,
  payload    LONGTEXT  NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (event_id)
);

-- Per-event-type retry/lifecycle configuration
CREATE TABLE IF NOT EXISTS event_lifecycle_config (
  id                        BIGINT       NOT NULL AUTO_INCREMENT,
  event_type                VARCHAR(255) NOT NULL UNIQUE,
  flow_type                 VARCHAR(100) NOT NULL,
  max_attempts              INT          NOT NULL,
  event_lifespan_seconds    BIGINT       NOT NULL,
  is_suspended              BOOLEAN      NOT NULL DEFAULT FALSE,
  attempt_lifecycle_configs JSON         NOT NULL,
  created_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at                DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3)
                                         ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id)
);

-- Error log (append-only, never updated)
CREATE TABLE IF NOT EXISTS event_error_log (
  id            CHAR(36)    NOT NULL,
  event_id      CHAR(36)    NOT NULL,
  error_message TEXT        NULL,
  stack_trace   LONGTEXT    NULL,
  occurred_at   DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  INDEX idx_error_log_event (event_id)
);
```

### Example seed data (two flows, demonstrating the config pattern)

```sql
INSERT INTO event_lifecycle_config
  (event_type, flow_type, max_attempts, event_lifespan_seconds,
   is_suspended, attempt_lifecycle_configs)
VALUES
  -- FLOW_A: 3-step retry with escalating delays
  ('FLOW_A_STEP_1_REQUESTED', 'FLOW_A', 3, 300, false,
   '[{"attemptNumber":1,"delayBeforeAttemptSeconds":0},
     {"attemptNumber":2,"delayBeforeAttemptSeconds":30},
     {"attemptNumber":3,"delayBeforeAttemptSeconds":60}]'),

  ('FLOW_A_STEP_1_EXPIRED', 'FLOW_A', 1, 60, false,
   '[{"attemptNumber":1,"delayBeforeAttemptSeconds":0}]'),

  ('FLOW_A_STEP_2_REQUESTED', 'FLOW_A', 5, 3600, false,
   '[{"attemptNumber":1,"delayBeforeAttemptSeconds":0},
     {"attemptNumber":2,"delayBeforeAttemptSeconds":60},
     {"attemptNumber":3,"delayBeforeAttemptSeconds":120},
     {"attemptNumber":4,"delayBeforeAttemptSeconds":300},
     {"attemptNumber":5,"delayBeforeAttemptSeconds":600}]'),

  -- FLOW_B: notification with 3 attempts
  ('FLOW_B_NOTIFICATION_REQUESTED', 'FLOW_B', 3, 600, false,
   '[{"attemptNumber":1,"delayBeforeAttemptSeconds":0},
     {"attemptNumber":2,"delayBeforeAttemptSeconds":60},
     {"attemptNumber":3,"delayBeforeAttemptSeconds":120}]'),

  ('FLOW_B_NOTIFICATION_FAILED', 'FLOW_B', 1, 120, false,
   '[{"attemptNumber":1,"delayBeforeAttemptSeconds":0}]');
```

---

## CONFIGURATION SHAPE

The config file format should be idiomatic to the target language
(YAML, TOML, ENV, etc.). It must expose at minimum:

```
# API ports
api.grpc.port         = 9090
api.http.port         = 8080   # health/metrics only

# Primary database
db.host               = localhost
db.port               = 3306
db.name               = event_engine
db.username           = root
db.password           = password

# Payload store (Redis or equivalent)
payload_store.host    = localhost
payload_store.port    = 6379

# Message broker
broker.url            = localhost:61616

# Scheduler config per flow type
jobs.flow_a.awaiting.interval_ms         = 5000
jobs.flow_a.awaiting.startup_delay_ms    = 2750
jobs.flow_a.awaiting.batch_size          = 100
jobs.flow_a.awaiting.max_items           = 100000
jobs.flow_a.scheduled.interval_ms        = 5000
jobs.flow_a.scheduled.startup_delay_ms   = 3750
jobs.flow_a.suspended.interval_ms        = 5000
jobs.flow_a.suspended.startup_delay_ms   = 4000

jobs.flow_b.awaiting.interval_ms         = 5000
jobs.flow_b.awaiting.startup_delay_ms    = 2750
# ... same pattern

# Metrics
metrics.prometheus.enabled = true
metrics.prometheus.port    = 9091
```

---

## PROJECT STRUCTURE

Use idiomatic project structure for the target language. At minimum,
separate these concerns into distinct modules/packages/directories:

```
event-engine-starter/
├── api/           — gRPC/REST handlers
├── service/       — orchestration (lifecycle management, dispatcher)
├── repository/    — persistence (event, payload, config)
├── model/         — domain types (Event, EventStatus, EventLifecycleConfig, ...)
├── job/           — scheduled jobs
├── config/        — wiring and configuration loading
├── proto/         — .proto files (if gRPC)
├── migrations/    — DB migration scripts
├── adapter-stub/  — example adapter (separate module/package)
├── docker-compose.yml
├── Dockerfile
└── README.md
```

---

## ADAPTER CONTRACT (document + stub)

In the README, explain the adapter integration pattern:

1. Adapter calls `Publish` to submit a new event to the engine
2. Adapter subscribes to broker queue `event-engine.{eventType}`
3. Adapter receives a dispatched event message from the broker
4. Adapter calls `TryAcquireProcessingPermit` to lock the event
   (if `acquired = false`, discard — another instance got it)
5. Adapter performs its work (calls external API, etc.)
6. Adapter calls `ReportAsSuccessfullyProcessed` (with optional
   spinoff events to chain) or `ReportAsFailed`

Provide a working `adapter-stub` that demonstrates this contract
against a running engine instance using test/mock business logic.
The stub must be runnable standalone.

---

## CONSTRAINTS

- Zero domain-specific code — no payments, no logistics, no specific
  business logic of any kind
- Every interface/contract must have at least one concrete implementation
- All state transitions must use optimistic locking — never pessimistic
- All scheduled jobs must be stateless and safe to run on N instances
  simultaneously without coordination
- Payload storage implementation must be swappable without changing
  business logic (interface/dependency inversion)
- The in-process config cache must have a configurable TTL
- Every public interface/contract must have documentation explaining
  its invariants and error conditions
- Include integration tests for: repository layer (real DB via
  containers), service layer (real DB + broker), and API handlers
- README must cover: architecture overview, state machine diagram,
  how to add a new flow type (step by step), how to configure retry
  behaviour, how to write an adapter

---

## DEFINITION OF DONE

1. Project builds and all tests pass with a single command
2. Zero hardcoded domain logic anywhere in the codebase
3. Adding a new flow type FLOW_C requires exactly:
   - One new identifier (constant/enum value)
   - Three job registrations
   - SQL INSERT rows into event_lifecycle_config
   — and nothing else
4. `docker-compose.yml` starts the full stack (DB + payload store +
   broker + engine) and the engine is reachable
5. The adapter-stub can successfully publish, process, and complete
   an event end-to-end against the running engine
6. README is complete and accurate
