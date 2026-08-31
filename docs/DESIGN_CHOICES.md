# Design Choices & Trade-offs

A record of the deliberate decisions made while building this project,
why each one was made, what was traded away, and what's known to be
incomplete. Written for anyone (including future-me) trying to
understand *why* the system looks the way it does, not just *what* it
does.

## Table of Contents

- [ID Generation](#id-generation)
- [Storage](#storage)
- [Caching](#caching)
- [Idempotency & Deduplication](#idempotency--deduplication)
- [Authentication & Authorization](#authentication--authorization)
- [Rate Limiting](#rate-limiting)
- [Horizontal Scaling & Coordination](#horizontal-scaling--coordination)
- [Analytics Pipeline](#analytics-pipeline)
- [Error Handling](#error-handling)
- [Migrations](#migrations)
- [Scope Decisions](#scope-decisions)
- [Known Limitations](#known-limitations)
- [Things I'd Do Differently at Larger Scale](#things-id-do-differently-at-larger-scale)

## ID Generation

**Choice:** Custom Snowflake-style ID generator (timestamp + node ID + sequence, Base62-encoded) instead of UUIDs, a database auto-increment column, or hash-based codes.

**Why:** Auto-increment requires a round-trip to the database and doesn't work well once storage is sharded. UUIDs are collision-free but long, not naturally sortable, and waste space as a primary key at billions-of-rows scale. Hash-based codes (e.g. truncated MD5 of the URL) need collision detection and retry logic, and don't give sortability. Snowflake gives short, roughly time-ordered, collision-free IDs generated locally with no coordination on the hot path.

**Trade-off:** Requires solving node-ID uniqueness across instances (see [Horizontal Scaling](#horizontal-scaling--coordination)) — complexity that auto-increment or UUIDs wouldn't need. Also relies on system clock monotonicity; clock drift is handled by refusing to generate an ID rather than risking a collision (returns an error instead of panicking, so a request fails cleanly rather than the whole process crashing).

**Verified:** Benchmarked at ~4M+ IDs/sec single-instance, fully mutex-serialized, with no meaningful degradation from 1 to 8 concurrent CPUs, confirmed this was never the write-path bottleneck.

## Storage

**Choice:** Postgres as the primary store, accessed exclusively through a `storage.Store` interface (composed of `URLStore` + `UserStore`), with the concrete implementation generated via `sqlc` from hand-written SQL rather than an ORM (GORM/Ent) or hand-written `pgx` calls.

**Why:** An ORM's auto-migration/schema-inference would obscure indexing and constraint decisions (partial indexes, dedup constraints) that were made deliberately. Hand-written `pgx` calls would mean hand-writing `rows.Scan()` boilerplate and manually keeping Go structs in sync with the schema. `sqlc` keeps schema and queries in plain, readable SQL — full control — while eliminating the repetitive, error-prone parts.

**Trade-off:** The interface boundary (`Store`) exists specifically so storage can be swapped (e.g. to ScyllaDB) without touching business logic — but this was never exercised in practice, since load testing showed the database was not the bottleneck. The abstraction cost was paid without (yet) needing the payoff.

## Caching

**Choice:** Redis, cache-aside pattern, write-through on create, 1 hour TTL, with a `Delete` method defined on the cache interface but not currently called by anything.

**Why:** Redirects outnumber writes by orders of magnitude, so solving the read path with a cache was prioritized before any other scaling work. Cache-aside (check cache, fall back to DB, populate cache) is the simplest correct pattern and keeps the cache and DB naturally consistent on the read side.

**Trade-off — staleness window:** A URL that's deactivated or expires does **not** get proactively evicted from cache. A reader who already has it cached will keep getting redirected to it for up to an hour after it should have stopped working. This was a deliberate choice: a `Delete` method exists on the interface for exactly this purpose, with the plan to wire it into a future background reaper job (tied to Phase 7's expiry-cleanup scope) rather than lowering the TTL and sacrificing hit ratio now.

**Verified:** Confirmed via `redis-cli MONITOR` that cache-aside behaves correctly under real traffic — hits skip the DB entirely, misses fall through and repopulate the cache with the correct TTL.

## Idempotency & Deduplication

**Choice:** If an authenticated user shortens the same URL twice, the second request returns the existing `short_code` instead of creating a duplicate row. Enforced by a partial unique index on `(user_id, long_url_hash)`, with an application-level lookup short-circuiting before a new insert is attempted. Anonymous requests are never deduped (no identity to dedupe against).

**Why:** Matches user expectation (shortening the same link twice shouldn't produce two different short codes) and matches how mainstream URL shorteners behave.

**Trade-off — this was designed in Phase 0/1 but not actually wired in until load testing exposed the gap.** `GetURLByUserAndHash` existed in the storage layer from early on but was never called. Concurrent writers hitting the same user+URL combination were failing on the raw database constraint violation and returning `500` instead of the intended idempotent response. Fixed by adding the lookup at the top of the create flow.

**Known remaining gap — check-then-act race:** The dedup lookup and the insert are not wrapped in a single atomic operation. Two truly simultaneous first-time requests for the same user+URL could both miss the existence check and both attempt an insert; one succeeds, one still hits the constraint violation and returns `500`. This is rare (requires sub-millisecond-scale simultaneity from the same user) and considered acceptable for now — a proper fix would use an `INSERT ... ON CONFLICT DO NOTHING RETURNING *`-style upsert instead of check-then-insert.

## Authentication & Authorization

**Choice:** API-key-based auth, not sessions/JWTs/OAuth. Keys are generated with `crypto/rand`, shown once at registration, and only their SHA-256 hash is ever persisted. No password is used at all — registration only requires an email.

**Why:** Simplest model that supports the two required tiers (anonymous, registered) without needing a login flow, password storage and hashing (bcrypt/argon2), or session management. Appropriate for an API-first product where the primary "login" is presenting a key on each request, not an interactive session.

**Trade-off:** No way to rotate or revoke a key without a database write path that doesn't exist yet (no `DELETE`/regenerate endpoint). No password means no account recovery flow is meaningful either — if a key is lost, the account is effectively unreachable via the current API surface.

**choice: auth middleware is permissive by default:** A missing `Authorization` header is treated as "proceed anonymously," not rejected. A *present but invalid* key, however, is rejected with `401` rather than silently downgraded to anonymous — deliberately, so a typo'd or revoked key surfaces as an error instead of silently and confusingly behaving like a logged-out request.

**choice: stats endpoint reuses "not found" instead of "forbidden":** When a user requests stats for a URL they don't own, the response is `404`, not `403`. This avoids leaking whether a given `short_code` exists to someone who isn't its owner.

## Rate Limiting

**Choice:** Fixed-window counter (Redis `INCR` + `EXPIRE`) rather than a sliding window or token bucket. Anonymous requests are keyed by IP, authenticated requests by `user_id`, with different limits per tier.

**Why:** Simplest correct algorithm; the well-known fixed-window boundary-burst issue (up to ~2x the intended rate right at a window edge) was judged an acceptable trade-off for the current scale and project stage, in exchange for much simpler, more obviously-correct code than a sliding window or token bucket would require.

**Trade-off — the `INCR` + `EXPIRE` pair is not atomic.** There's a narrow window where a process crash between the two calls could leave a counter key with no expiry, stuck until manually cleared. Accepted as a known, low-probability edge case rather  han adding a Lua script to make the operation atomic.

**choice: fails open, not closed.** If Redis is unreachable when a rate-limit check runs, the request is allowed through (logged, not blocked). Prioritizes availability of the core product over strict enforcement during an infrastructure hiccup — an unavailable rate limiter should not take down the write path.

## Horizontal Scaling & Coordination

**Choice:** etcd for node-ID leasing, using an atomic compare-and-put transaction per candidate ID plus a renewable lease with a TTL, rather than a hardcoded node ID, a static config-based assignment, or building the same coordination logic on top of Postgres.

**Why:** etcd's lease/TTL mechanism gives automatic cleanup when an instance crashes — a dead instance's claimed node ID becomes available again with zero additional code, since the lease simply expires. The same guarantee built on Postgres would require hand-rolled heartbeat and staleness-detection logic.

**Verified:** Ran two instances simultaneously, confirmed they acquired distinct node IDs; killed one, confirmed its ID was released and reclaimable by a new instance after the lease TTL elapsed.

## Analytics Pipeline

**Choice:** Click events are published to Kafka fully asynchronously (a detached goroutine with its own context and timeout, not the request's context) rather than written synchronously to any datastore on the redirect path. A separate consumer process batches events (1,000 events or 5 seconds, whichever comes first) before bulk-inserting into ClickHouse.

**Why:** The redirect path's latency must never depend on analytics volume or availability. Firing the publish in a goroutine with a detached context specifically avoids a subtle bug: using the original request's context would mean the publish gets cancelled the instant the HTTP response finishes, since `net/http` cancels a request's context on completion, a race that would silently drop events under real traffic.

**Why batching:** ClickHouse's `MergeTree` engine is built around bulk inserts; single-row inserts are a known anti-pattern for it.

**Trade-off:** A crash between "event read from Kafka" and "batch flushed to ClickHouse" loses that batch, there's no write-ahead log or transactional outbox on the consumer side. Acceptable for analytics data (approximate click counts), would not be acceptable if this were billing or transactional data.

**Choice: ClickHouse migrations run automatically on every consumer container startup**, unlike some production systems that treat schema changes as a deliberate, reviewed, separately-triggered step. Chosen for solo-developer convenience (avoids the "forgot to run the migration" failure mode this project hit once already) at the cost of schema changes being less visible/reviewable as a discrete deploy step.

## Error Handling

**Choice:** A single `apperrors` package holds every sentinel error the application can produce, plus the function that maps each one to an HTTP status code, kept in the same file rather than split into a separate HTTP-agnostic errors package and a handler-side mapping function.

**Why:** Single source of truth, anyone adding a new error type has one obvious place to also register its HTTP status, rather than remembering to update two files in two packages.

**Trade-off:** `apperrors` is not technically transport-agnostic anymore (it imports `net/http` for the status codes) — a theoretical future non-HTTP interface (gRPC, CLI) reusing these errors would carry an unused `net/http` dependency. Judged a non-issue for a project with one API surface.

**Choice: real errors are always logged before being converted to a generic sentinel returned to the client.** Established as a project-wide pattern after a debugging session where a swallowed error (`db error` collapsed straight to `apperrors.ErrInternal` with no log line) made a real concurrency bug essentially invisible under load testing.

## Migrations

**Choice:** `goose`, single-file migrations with `-- +goose Up` / `-- +goose Down` markers, for both Postgres and ClickHouse — chosen over `golang-migrate` (which requires two files per migration) and over declarative/diff-based tools like Atlas.

**Why:** Single-file migrations are simpler for solo, early-stage work. Hand-written SQL (rather than an ORM/diff tool inferring schema) keeps every indexing and constraint decision visible and intentional.

**Trade-off:** No down-migrations were written with the same rigor as up-migrations in every case — acceptable pre-production, would need revisiting before this ever holds real user data that can't be recreated from scratch.

## Scope Decisions

**Phase 4 (data-layer sharding to ScyllaDB): deferred, not
abandoned.** Load testing showed the *application layer*, not Postgres, was the write-path bottleneck (confirmed by resource utilization data and later by a targeted `pgxpool` connection-pool fix). Sharding the database would not have addressed the actual problem. The `storage.Store` interface exists specifically so this remains possible later without a rewrite, if it's ever actually needed.

**Phase 6 (CDN/edge distribution): dropped.** Deliberately deprioritized in favor of treating the project as feature-complete at its current scope. This is real infrastructure deployment work (a CDN account, edge compute config, DNS) rather than a distributed-systems concept the rest of the project hadn't already covered in some form (caching, horizontal scaling, coordination).

**Phase 7 (hardening): partially addressed opportunistically, not completed as a dedicated phase.** Error logging discipline and the `pgxpool` sizing fix both came out of load testing rather than a formal hardening pass. Chaos testing, circuit breakers, and the cache-eviction reaper worker remain undone.

## Known Limitations

- No way to revoke, rotate, or list API keys once issued.
- No endpoint to deactivate or delete a URL (the `is_active` column exists and is checked on redirect, but nothing sets it to `false`).
- No cache eviction on deactivation/expiry — bounded by TTL only.
- No transactional outbox for the Kafka publish — a crash at the wrong moment loses a click event (acceptable for analytics, noted for completeness).
- Check-then-act race in idempotent dedup under true concurrency (see above).
- `INCR` + `EXPIRE` non-atomicity in the rate limiter (see above).
- Root cause of the original write-path bottleneck was narrowed to `pgxpool` sizing and fixed, but not exhaustively profiled beyond that, there may be further headroom un-investigated.
- No automated test suite beyond the `idgen` benchmark; correctness was verified through manual and scripted load testing, not unit/integration tests.

## Things I'd Do Differently at Larger Scale

- Replace the fixed-window rate limiter with a sliding-window or token-bucket implementation to remove the boundary-burst edge case.
- Make the rate limiter's `INCR`+`EXPIRE` atomic via a Lua script.
- Replace check-then-insert dedup with a database-level upsert.
- Add a write-ahead/outbox pattern for the Kafka publish so a crash can't silently drop a click event.
- Treat ClickHouse (and Postgres) migrations as an explicit, reviewed deploy step rather than something that runs automatically on every container start.
- Build the deferred cache-eviction reaper worker.
- Add real automated tests (unit tests for `service`, integration tests against a test database) rather than relying on manual/load testing for correctness.