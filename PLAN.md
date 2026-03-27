# Event Engine Starter — Go Implementation Plan

## Overview

A domain-agnostic **event lifecycle engine** implemented in idiomatic Go, exposing
its operations as **MCP (Model Context Protocol) tools** via the Go MCP SDK.
The engine manages events through a well-defined state machine with configurable
retries, suspensions, expiry, and on-failure chaining.

An HTML frontend connects via SSE/WebSocket to demonstrate the event flow in
real time.

---

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    MCP Clients (AI / HTML UI)            │
│                                                         │
│  Tools: publish, acquire_permit, report_success,        │
│         report_failed, get_config, suspend, unsuspend   │
│  Resources: event://{id}, events://pending/{flow},      │
│             config://{eventType}, metrics://queues       │
└──────────────────────┬──────────────────────────────────┘
                       │ SSE / Streamable HTTP
                       ▼
┌─────────────────────────────────────────────────────────┐
│                   MCP Server (Go)                        │
│                                                         │
│  ┌─────────┐  ┌──────────────┐  ┌───────────────────┐  │
│  │  Tools   │  │  Resources   │  │  WebSocket Hub    │  │
│  │ handlers │  │  handlers    │  │  (live events UI) │  │
│  └────┬─────┘  └──────┬───────┘  └────────┬──────────┘  │
│       │               │                   │              │
│       ▼               ▼                   ▼              │
│  ┌──────────────────────────────────────────────────┐   │
│  │         EventLifecycleManagementService           │   │
│  │  (high-level orchestrator, retry logic, spinoffs) │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                                │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │         EventProcessingService                    │   │
│  │  (low-level state transitions, optimistic locks)  │   │
│  └──────┬───────────────┼───────────────┬───────────┘   │
│         │               │               │                │
│         ▼               ▼               ▼                │
│  ┌──────────┐   ┌──────────────┐  ┌────────────────┐   │
│  │ EventRepo│   │ PayloadRepo  │  │ ConfigRepo     │   │
│  │ (MySQL)  │   │ (Redis/Mem)  │  │ (MySQL+Cache)  │   │
│  └──────────┘   └──────────────┘  └────────────────┘   │
│                                                         │
│  ┌──────────────────────────────────────────────────┐   │
│  │              Scheduled Jobs (goroutines)           │   │
│  │  - AwaitingProcessingJob (per flow)               │   │
│  │  - ScheduledEventsJob (per flow)                  │   │
│  │  - ExpiredSuspendedJob (per flow)                 │   │
│  │  - QueueSizeMetricsJob (global)                   │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                                │
│  ┌──────────────────────┼───────────────────────────┐   │
│  │              EventDispatcher                      │   │
│  │  (NATS publish with per-message TTL)              │   │
│  └──────────────────────┬───────────────────────────┘   │
│                         │                                │
└─────────────────────────┼───────────────────────────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │    NATS Broker        │
              │  event-engine.{type}  │
              └───────────┬───────────┘
                          │
                          ▼
              ┌───────────────────────┐
              │    Adapter Stub       │
              │  (example consumer)   │
              └───────────────────────┘
```

---

## Technology Choices (Go-Idiomatic)

| Concern                  | Choice                                         | Rationale                                  |
|--------------------------|-------------------------------------------------|--------------------------------------------|
| API / Protocol           | MCP over SSE + HTTP health/metrics              | User requirement; Go SDK available          |
| Message broker           | NATS                                            | Go-native, lightweight, per-msg TTL headers |
| Primary DB               | MySQL                                           | Matches reference; well-supported in Go     |
| Payload storage          | Redis (prod) / sync.Map (test)                  | Fast KV; in-memory fallback for dev         |
| DB migrations            | golang-migrate                                  | Standard Go migration tool                  |
| In-memory cache          | Custom TTL cache (sync.RWMutex + map)           | No external dep needed for simple TTL cache |
| Scheduled jobs           | time.Ticker in goroutines                       | Native Go; no cron library needed           |
| Metrics                  | prometheus/client_golang                        | Standard; exposed via HTTP                  |
| HTTP (health/metrics/UI) | go-chi/chi                                      | Lightweight, idiomatic Go router            |
| WebSocket (live UI)      | gorilla/websocket                               | De facto Go WebSocket library               |
| DI                       | Manual constructor injection                    | Idiomatic Go — no DI framework              |

---

## Event State Machine

```
                     ┌──────────────────────────┐
                     │   AWAITING_PROCESSING    │ ◄─── Initial state
                     └────────────┬─────────────┘
                                  │ scheduler dispatches
                                  ▼
                     ┌──────────────────────────┐
                     │       DISPATCHED          │
                     └────────────┬─────────────┘
                                  │ adapter acquires permit
                                  ▼
                     ┌──────────────────────────┐
                     │    BEING_PROCESSED        │
                     └──────┬─────────────┬─────┘
                            │             │
                   success  │             │ failure
                            ▼             ▼
                     ┌────────────┐ ┌──────────┐
                     │ PROCESSED  │ │  FAILED   │
                     │  (final)   │ │          │
                     └────────────┘ └─────┬────┘
                                          │ if retries remain
                                          ▼
                                   AWAITING_PROCESSING (retry loop)
                                          │ if suspended
                                          ▼
                     ┌──────────────────────────┐
                     │       SUSPENDED           │
                     └────────────┬─────────────┘
                                  │ trigger or timeout
                                  ▼
                           AWAITING_PROCESSING (reactivated)

Terminal states: PROCESSED, FAILED, CANCELED
```

---

## Project Structure

```
event-engine-starter/
├── cmd/
│   ├── engine/
│   │   └── main.go                 # Engine entry point (MCP server + jobs + HTTP)
│   └── adapter-stub/
│       └── main.go                 # Example NATS consumer adapter
├── internal/
│   ├── model/
│   │   ├── event.go                # Event struct, EventReference
│   │   ├── status.go               # EventStatus enum + predicates
│   │   ├── schedule_state.go       # ScheduleState enum
│   │   ├── config.go               # EventLifecycleConfig, AttemptLifecycleConfig
│   │   ├── commands.go             # All state transition command structs
│   │   ├── filters.go              # Query filter structs
│   │   ├── errors.go               # RaceConditionError, domain errors
│   │   └── flow.go                 # FlowType constants (FLOW_A, FLOW_B)
│   ├── repository/
│   │   ├── interfaces.go           # EventRepository, PayloadRepository, ConfigRepository
│   │   ├── event_mysql.go          # MySQL EventRepository implementation
│   │   ├── payload_redis.go        # Redis PayloadRepository implementation
│   │   ├── payload_memory.go       # In-memory PayloadRepository (testing)
│   │   ├── config_mysql.go         # MySQL ConfigRepository implementation
│   │   └── errorlog_mysql.go       # MySQL EventErrorLog repository
│   ├── service/
│   │   ├── processing.go           # EventProcessingService (low-level)
│   │   ├── lifecycle.go            # EventLifecycleManagementService (orchestrator)
│   │   ├── dispatcher.go           # EventDispatcher (NATS publisher with TTL)
│   │   ├── configuration.go        # EventConfigurationService (cached config)
│   │   └── cache.go                # Generic TTL cache implementation
│   ├── job/
│   │   ├── runner.go               # Job runner framework (ticker + batch processing)
│   │   ├── awaiting.go             # AwaitingProcessingJob
│   │   ├── scheduled.go            # ScheduledEventsProcessingJob
│   │   ├── suspended.go            # ExpiredSuspendedEventsProcessingJob
│   │   └── metrics.go              # QueueSizeMetricsJob
│   ├── mcp/
│   │   ├── server.go               # MCP server setup, tool + resource registration
│   │   ├── tools.go                # Tool handlers (publish, acquire, report, etc.)
│   │   └── resources.go            # Resource handlers (event state, config, metrics)
│   └── web/
│       ├── handler.go              # HTTP routes (health, metrics, WebSocket, static)
│       └── hub.go                  # WebSocket hub for live event broadcasting
├── web/
│   └── index.html                  # HTML frontend (triggers events, shows live flow)
├── migrations/
│   ├── 001_create_event.up.sql
│   ├── 001_create_event.down.sql
│   ├── 002_create_event_payload.up.sql
│   ├── 002_create_event_payload.down.sql
│   ├── 003_create_lifecycle_config.up.sql
│   ├── 003_create_lifecycle_config.down.sql
│   ├── 004_create_error_log.up.sql
│   ├── 004_create_error_log.down.sql
│   └── 005_seed_config.up.sql
├── config/
│   └── config.go                   # Configuration struct + loading (env/YAML)
├── config.yaml                     # Default configuration file
├── docker-compose.yml              # MySQL + Redis + NATS
├── Dockerfile
├── Makefile
├── go.mod
├── go.sum
├── PLAN.md                         # This file
└── README.md
```

---

## MCP Integration Design

The event engine exposes its API as MCP tools and resources, allowing AI clients
(or the HTML UI acting as an MCP client) to interact with the event lifecycle.

### MCP Tools (write operations)

| Tool Name                    | Description                                | Input                                                      |
|------------------------------|--------------------------------------------|------------------------------------------------------------|
| `publish_event`              | Publish a new event into the engine        | eventType, flowType, flowId, payload, onFailEventType?     |
| `acquire_processing_permit`  | Lock an event for processing               | eventId                                                    |
| `report_success`             | Mark event as successfully processed       | eventId, spinoffEvents[]?                                  |
| `report_failure`             | Report event processing failure            | eventId, errorMessage, stackTrace?                         |
| `suspend_event_type`         | Pause dispatch of an event type            | eventType                                                  |
| `unsuspend_event_type`       | Resume dispatch of an event type           | eventType                                                  |
| `update_lifecycle_config`    | Update retry/lifecycle config              | eventType, maxAttempts, lifespanSeconds, attemptConfigs[]  |

### MCP Resources (read operations)

| Resource URI                          | Description                           |
|---------------------------------------|---------------------------------------|
| `event://{eventId}`                   | Single event state + payload          |
| `events://pending/{flowType}`         | List pending events for a flow        |
| `config://{eventType}`                | Lifecycle config for an event type    |
| `metrics://queues`                    | Queue sizes by flow type and status   |

### Transport

- **SSE (Server-Sent Events)** — Primary MCP transport for browser + AI clients
- HTTP server on same port hosts: MCP SSE endpoint, health check, Prometheus
  metrics, WebSocket for live UI, and static HTML files

---

## Implementation Phases

### Phase 1: Foundation (model + config)
- [ ] `internal/model/` — All domain types, status enum, commands, filters, errors
- [ ] `config/config.go` — Configuration struct and loader
- [ ] `go.mod` — All dependencies

### Phase 2: Persistence Layer
- [ ] `internal/repository/interfaces.go` — All repository interfaces
- [ ] `internal/repository/event_mysql.go` — Event repository (optimistic locking)
- [ ] `internal/repository/payload_redis.go` — Redis payload store
- [ ] `internal/repository/payload_memory.go` — In-memory payload store
- [ ] `internal/repository/config_mysql.go` — Config repository
- [ ] `internal/repository/errorlog_mysql.go` — Error log repository
- [ ] `migrations/` — All SQL migration files

### Phase 3: Service Layer
- [ ] `internal/service/cache.go` — Generic TTL cache
- [ ] `internal/service/processing.go` — Low-level event processing
- [ ] `internal/service/configuration.go` — Config service with caching
- [ ] `internal/service/dispatcher.go` — NATS dispatcher with TTL buckets
- [ ] `internal/service/lifecycle.go` — High-level orchestrator

### Phase 4: Scheduled Jobs
- [ ] `internal/job/runner.go` — Job runner framework
- [ ] `internal/job/awaiting.go` — Awaiting processing job
- [ ] `internal/job/scheduled.go` — Scheduled events job
- [ ] `internal/job/suspended.go` — Expired suspended events job
- [ ] `internal/job/metrics.go` — Queue size metrics job

### Phase 5: MCP Server + API
- [ ] `internal/mcp/server.go` — MCP server initialization
- [ ] `internal/mcp/tools.go` — All tool handlers
- [ ] `internal/mcp/resources.go` — All resource handlers

### Phase 6: HTTP + WebSocket + Frontend
- [ ] `internal/web/hub.go` — WebSocket broadcast hub
- [ ] `internal/web/handler.go` — HTTP routes
- [ ] `web/index.html` — HTML frontend

### Phase 7: Adapter Stub + Infrastructure
- [ ] `cmd/engine/main.go` — Engine wiring and startup
- [ ] `cmd/adapter-stub/main.go` — Example NATS consumer
- [ ] `docker-compose.yml` — Full stack
- [ ] `Dockerfile`
- [ ] `Makefile`

---

## Key Design Decisions

### 1. MCP as the API surface (not raw gRPC/REST)
The Java reference uses gRPC. We replace this with MCP tools/resources, which
provides a structured, discoverable API that AI clients can consume natively.
The HTML frontend uses the MCP SSE transport or a thin WebSocket bridge.

### 2. NATS instead of ActiveMQ
NATS is Go-native, lightweight, and supports per-message headers (used for TTL
enforcement on the consumer side). Queue groups provide competing consumer
semantics equivalent to JMS queues.

### 3. Optimistic locking via SQL WHERE version = ?
Identical to the Java reference. Every state transition command carries the
current version. If UPDATE affects 0 rows, return RaceConditionError. The
lifecycle management service retries up to 10 times with 15ms delay.

### 4. Manual DI via constructors
Go convention — no DI framework. All dependencies injected via New* constructors.
Interfaces defined in the repository package, implementations selected at startup.

### 5. Goroutine-based scheduler instead of Spring @Scheduled
Each job runs in its own goroutine with a time.Ticker. Batch processing with
configurable limits (batch size, max items, max failures, max runtime).

### 6. TTL enforcement at consumer
NATS doesn't enforce per-message TTL natively. We set a `TTL` header on publish
and check `time.Since(publishedAt) > ttl` on consume. Messages past TTL are
silently discarded.

---

## Adding a New Flow Type

To add `FLOW_C`, you need exactly:

1. **Add constant** in `internal/model/flow.go`:
   ```go
   const FlowTypeC FlowType = "FLOW_C"
   ```

2. **Register 3 jobs** in `cmd/engine/main.go`:
   ```go
   job.NewAwaitingProcessingJob(model.FlowTypeC, lifecycleSvc, cfg.Jobs.FlowC.Awaiting)
   job.NewScheduledEventsJob(model.FlowTypeC, lifecycleSvc, cfg.Jobs.FlowC.Scheduled)
   job.NewExpiredSuspendedJob(model.FlowTypeC, lifecycleSvc, cfg.Jobs.FlowC.Suspended)
   ```

3. **Insert config rows** in SQL:
   ```sql
   INSERT INTO event_lifecycle_config (event_type, flow_type, max_attempts, ...)
   VALUES ('FLOW_C_STEP_1_REQUESTED', 'FLOW_C', 3, ...);
   ```

No other code changes needed.

---

## Configuration Shape (config.yaml)

```yaml
api:
  http_port: 8080           # Health, metrics, MCP SSE, WebSocket, static files

db:
  host: localhost
  port: 3306
  name: event_engine
  user: root
  password: password
  max_open_conns: 80
  max_idle_conns: 10

payload_store:
  type: redis               # or "memory"
  host: localhost
  port: 6379

broker:
  url: nats://localhost:4222

jobs:
  flow_a:
    awaiting:
      interval_ms: 5000
      startup_delay_ms: 2750
      batch_size: 100
      max_items: 100000
      max_fails: 100
      max_runtime_ms: 30000
    scheduled:
      interval_ms: 5000
      startup_delay_ms: 3750
      batch_size: 100
      max_items: 100000
      max_fails: 100
      max_runtime_ms: 30000
    suspended:
      interval_ms: 5000
      startup_delay_ms: 4000
      batch_size: 100
      max_items: 100000
      max_fails: 100
      max_runtime_ms: 30000
  flow_b:
    awaiting:
      interval_ms: 5000
      startup_delay_ms: 2750
    scheduled:
      interval_ms: 5000
      startup_delay_ms: 3750
    suspended:
      interval_ms: 5000
      startup_delay_ms: 4000

metrics:
  enabled: true
```

---

## HTML Frontend

A single-page app that:
- Connects to the MCP server via SSE transport
- Calls MCP tools to publish events, acquire permits, report results
- Shows live event state transitions via WebSocket
- Displays the state machine visually with events moving through states
- Provides controls for suspend/unsuspend and config management
- Shows queue metrics and event error logs

---

## Definition of Done

1. `docker-compose up` starts MySQL + Redis + NATS + Engine
2. MCP tools are discoverable and callable via any MCP client
3. HTML frontend can trigger and observe full event lifecycle
4. Adapter stub processes events end-to-end
5. Adding FLOW_C requires only: 1 constant + 3 job registrations + SQL rows
6. All state transitions use optimistic locking
7. Scheduled jobs are stateless and safe for concurrent execution
8. TTL-aware dispatch prevents stale message delivery
