# url-shortener

A production-grade, high-throughput URL shortener — built incrementally,
phase by phase, with an emphasis on the design decisions and trade-offs
behind each layer, not just a working demo.

## Table of Contents

- [Overview](#overview)
- [Non-Functional Targets](#non-functional-targets)
- [Tech Stack](#tech-stack)
- [Architecture](#architecture)
  - [System Diagram](#system-diagram)
  - [Request Paths](#request-paths)
  - [Component Responsibilities](#component-responsibilities)
- [Repository Structure](#repository-structure)
- [Product Decisions](#product-decisions)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Running Everything with Docker Compose](#running-everything-with-docker-compose)
  - [Environment Variables](#environment-variables)
- [API Reference](#api-reference)
- [Database Schema](#database-schema)
- [Migrations](#migrations)
- [Load Testing](#load-testing)
- [Build Phases](#build-phases)
- [Known Trade-offs & Deferred Work](#known-trade-offs--deferred-work)
- [Roadmap](#roadmap)

## Overview

This is not a beginner-scale URL shortener — it's built to handle large
data volumes, high throughput, and horizontal scaling, with every major
design decision made deliberately: caching strategy, ID generation,
storage sharding readiness, coordination for multi-instance deployment,
and a fully decoupled analytics pipeline.

The project was built in explicit phases, each with its own scope, so
that correctness could be validated before scale-oriented complexity was
layered on top.

## Non-Functional Targets

| Metric | Target |
|---|---|
| Read throughput | 50,000 redirects/sec |
| Write throughput | 500 shortens/sec |
| Redirect latency (p99) | < 50ms, ideally < 10ms |
| Availability | 99.9%+ |
| Cache hit ratio | 95%+ |

## Tech Stack

| Layer | Choice |
|---|---|
| Application | Go |
| ID generation | Custom Snowflake-style generator |
| Cache | Redis |
| Primary storage | Postgres (ScyllaDB migration path kept open via interface) |
| Coordination | etcd (node-ID leasing for horizontal scaling) |
| Event streaming | Kafka (KRaft mode) |
| Analytics store | ClickHouse |
| Migrations | goose (Postgres and ClickHouse dialects) |
| DB access | sqlc (typed queries generated from hand-written SQL) |
| Containerization | Docker + Docker Compose |
| Frontend | Next.js (dashboard only — never touches the redirect hot path) |

## Architecture

### System Diagram

```mermaid
flowchart TD
    Client([Client / Browser])

    Client -->|GET /short_code| Edge[CDN / Edge Compute — planned]
    Client -->|Dashboard traffic| FE[Next.js Frontend]

    Edge -->|cache miss| LB[Load Balancer / Gateway — planned]
    FE -->|API calls| Backend

    LB --> Backend[Go Backend]

    Backend --> Cache[(Redis Cache)]
    Backend --> DB[(Postgres)]
    Cache -->|cache miss| DB

    Backend -.async fire-and-forget.-> Kafka[[Kafka]]
    Kafka --> Consumer[Kafka Consumer]
    Consumer -->|batched inserts| ClickHouse[(ClickHouse)]

    Etcd[[etcd]] -.assigns node ID at boot.-> Backend
```

### Request Paths

**Redirect path** — `GET /{short_code}` (the hot path):
Backend → Redis cache-aside → Postgres only on a cold miss → `302`
redirect → click event fired asynchronously to Kafka (never blocks the
response).

**Write path** — `POST /api/v1/shorten`:
Auth middleware (optional) → rate-limit middleware → idempotent dedup
check for authenticated users → Snowflake ID generation → Base62 encode
→ Postgres insert → cache write-through.

**Analytics path** (fully decoupled):
Redirect → Kafka producer (fire-and-forget, detached context) → Kafka
topic → consumer batches events (1,000 events or 5s, whichever first) →
bulk insert into ClickHouse.

### Component Responsibilities

- **Go backend (stateless)** — owns Snowflake ID generation, Base62
  encoding, request handling, auth, and rate limiting. Any instance can
  serve any request.
- **Redis** — cache-aside for `short_code -> long_url`; also backs the
  fixed-window rate limiter (separate keyspace/interface from the URL
  cache).
- **Postgres** — source of truth for URLs and users, accessed through a
  `storage.Store` interface (composed of `URLStore` + `UserStore`) so the
  underlying database can be swapped without touching business logic.
- **etcd** — leases a unique node ID to each app instance at boot via an
  atomic compare-and-put transaction, with a renewable lease so a crashed
  instance's ID is automatically freed after its TTL expires.
- **Kafka** — receives click events fully asynchronously; a producer
  failure never fails a redirect.
- **ClickHouse** — append-optimized analytics store (`MergeTree` engine),
  fed via batched inserts from a dedicated consumer process.

## Repository Structure

```
url-shortener/
├── docker-compose.yml
├── backend/
│   ├── cmd/
│   │   ├── server/            # HTTP API entrypoint
│   │   └── consumer/          # Kafka → ClickHouse consumer entrypoint
│   ├── internal/
│   │   ├── api/
│   │   │   ├── handler/       # Thin HTTP controllers + middleware
│   │   │   └── service/       # Business logic, returns apperrors
│   │   ├── apperrors/         # Sentinel errors + HTTP status mapping
│   │   ├── cache/             # Cache interface + Redis implementation
│   │   ├── config/            # Env-based config loading
│   │   ├── events/            # Click event contract + Kafka publisher
│   │   ├── idgen/             # Snowflake ID generator
│   │   ├── nodelease/         # etcd-based node-ID leasing
│   │   ├── ratelimit/         # Rate limiter interface + Redis implementation
│   │   ├── storage/           # Store interfaces + sqlc-backed Postgres impl
│   │   │   └── db/            # sqlc-generated code
│   │   └── utils/             # Email normalization, API key gen/hash
│   ├── pkg/base62/            # Base62 encode/decode (dependency-free)
│   ├── migrations/            # Postgres migrations (goose)
│   ├── migrations_clickhouse/ # ClickHouse migrations (goose)
│   ├── Dockerfile             # API server image (runs migrations on boot)
│   └── Dockerfile.consumer    # Consumer image (runs ClickHouse migrations on boot)
├── frontend/                  # Next.js dashboard (create/manage links, stats, API key)
├── deployments/
├── docs/
└── scripts/
```

## Product Decisions

- **Custom aliases** — supported for authenticated users.
- **Auth** — API-key-gated, with an anonymous low-volume tier. No
  payment integration; only two tiers exist (anonymous vs registered),
  no paid plan.
- **Expiry** — opt-in; links persist indefinitely unless an expiry is
  set.
- **Idempotent dedup** — shortening the same URL twice under the same
  authenticated account returns the existing `short_code` instead of
  creating a duplicate. Different users get different codes. Anonymous
  requests are never deduped.
- **API keys** — generated via `crypto/rand`, shown once at registration,
  only their SHA-256 hash is ever stored.

## Getting Started

### Prerequisites

- Docker and Docker Compose
- Go 1.23+ (only needed for local, non-containerized development)

### Running Everything with Docker Compose

One command brings up Postgres, Redis, etcd, Kafka, ClickHouse, the API
server, and the Kafka consumer — with health checks ensuring each
service only starts once its dependencies are actually ready, and
migrations (both Postgres and ClickHouse) applied automatically on
startup:

```bash
docker compose up --build
```

The API is then available at `http://localhost:8080`.

### Environment Variables

Set via a `.env` file (see `env_file` in `docker-compose.yml`):

| Variable | Purpose |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `PORT` | API server port (default `8080`) |
| `REDIS_URL` | Redis connection string |
| `ETCD_ENDPOINTS` | etcd endpoint(s) for node-ID leasing |
| `KAFKA_BROKERS` | Comma-separated Kafka broker addresses |
| `CLICKHOUSE_ADDR` | ClickHouse native-protocol address |
| `CLICKHOUSE_USER` / `CLICKHOUSE_PASSWORD` | ClickHouse credentials |

## API Reference

| Method & Path | Auth | Description |
|---|---|---|
| `POST /api/v1/users` | None | Register and receive an API key (shown once) |
| `POST /api/v1/shorten` | Optional | Create a short URL; rate-limited; deduped per authenticated user |
| `GET /{short_code}` | None | Redirect to the original URL (`302`) |

Authenticated requests pass `Authorization: Bearer <api_key>`.

## Database Schema

**Postgres** — `users`, `urls` (short_code, long_url, long_url_hash,
user_id, is_custom_alias, expires_at, is_active), with:
- A unique index on `short_code` (the hot-path lookup)
- A partial unique index on `(user_id, long_url_hash)` for per-user
  idempotent dedup
- A partial index on `expires_at` for future cleanup jobs

**ClickHouse** — `click_events(short_code, clicked_at)`, `MergeTree`
engine, ordered by `(short_code, clicked_at)`.

## Migrations

Both Postgres and ClickHouse migrations use `goose`. Postgres migrations
run automatically on every API server container startup. ClickHouse
migrations run automatically on every consumer container startup.

To run manually:

```bash
goose -dir migrations postgres "$DATABASE_URL" up
goose -dir migrations_clickhouse clickhouse "clickhouse://user:pass@host:9000/default" up
```

## Load Testing

Load testing was performed with [`hey`](https://github.com/rakyll/hey)
against the fully containerized stack.

**Results (single node):**
- Redirect (cache-hit) throughput: ~41,400 req/sec, p99 latency 2.1ms
- Write path (idempotent dedup hits): ~16,800 req/sec, 0 errors across
  168k+ requests
- Write path plateaus around ~8–8.7k req/sec under sustained new-URL
  writes regardless of concurrency, with backend CPU (not Postgres) as
  the primary bottleneck

Load-testing surfaced and led to fixing a real bug: idempotent
per-user dedup was designed but never wired in, causing concurrent
duplicate-URL requests to fail on a database constraint violation
instead of returning the existing short code.

## Build Phases

- [x] **Phase 0** — API contract + non-functional targets
- [x] **Phase 1** — Core correctness (single-node Postgres, Snowflake IDs, no cache)
- [x] **Phase 2** — Redis caching layer (cache-aside, write-through)
- [x] **Phase 3** — Auth, rate limiting, horizontal scaling via etcd node-ID leasing
- [ ] **Phase 4** — Data layer sharding (Postgres → ScyllaDB, deferred — load testing showed the backend, not the database, is the current bottleneck)
- [x] **Phase 5** — Kafka + ClickHouse analytics pipeline
- [ ] **Phase 6** — Edge/global distribution
- [ ] **Phase 7** — Hardening (idempotency edge cases, chaos testing, TTL/expiry cleanup worker)

## Known Trade-offs & Deferred Work

- **Backend write-path bottleneck** — load testing points to the
  application layer (possibly the Snowflake generator's mutex, possibly
  `pgxpool` connection pool sizing under real network latency) as the
  current throughput ceiling, not Postgres. Not yet root-caused.
- **Check-then-act dedup race** — under true concurrency, two
  simultaneous first-time requests for the same user+URL could both miss
  the dedup check and one could fail. Rare and self-correcting; not yet
  hardened with a transaction or upsert.
- **Cache staleness on deactivation** — cached entries for a URL that's
  deactivated or expires are not actively evicted; they age out via TTL
  (1 hour). A `Delete` method exists on the cache interface for a future
  background reaper to use, but nothing calls it yet.
- **`/stats` endpoint** — scoped in the original API design, not yet
  built. `click_events` in ClickHouse already has the data.
- **ClickHouse migrations run automatically on every consumer startup**
  — convenient for solo development, but a deliberate trade-off some
  production setups avoid in favor of reviewed, explicit schema changes.

## Roadmap

1. Build the `/stats` endpoint against ClickHouse
2. Root-cause the write-path bottleneck (mutex contention vs. connection
   pool sizing)
3. Phase 6 — CDN/edge distribution for the redirect path
4. Phase 7 — hardening: TTL/expiry cleanup worker (also invalidates
   cache), chaos testing, idempotency keys, circuit breakers