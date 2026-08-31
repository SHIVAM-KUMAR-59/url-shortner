# Interview Prep: Questions This Project Should Prepare You For

Questions grouped by topic, roughly ordered from "you should be able to
answer this instantly" to "this requires actually thinking on the
spot." Each one maps to something real in the codebase, not a generic
textbook question, an interviewer who reads the repo could plausibly
ask any of these.

## Table of Contents

- [System Design — High Level](#system-design--high-level)
- [ID Generation](#id-generation)
- [Caching](#caching)
- [Database & Storage](#database--storage)
- [Concurrency & Idempotency](#concurrency--idempotency)
- [Authentication & Security](#authentication--security)
- [Rate Limiting](#rate-limiting)
- [Distributed Coordination](#distributed-coordination)
- [Messaging & Analytics](#messaging--analytics)
- [Load Testing & Performance Debugging](#load-testing--performance-debugging)
- [Trade-offs & Prioritization (behavioral-adjacent)](#trade-offs--prioritization-behavioral-adjacent)
- [Curveballs](#curveballs)

## System Design — High Level

1. Walk me through what happens, end to end, from a user hitting
   `POST /shorten` to receiving a short code.
2. Walk me through what happens when someone clicks a short link, every
   system it touches, in order.
3. Why did you architect the redirect path and the write path so
   differently? What does each one optimize for?
4. Which part of this system would you scale first under real load, and
   why? Which part wouldn't you touch?
5. If you had to support 10x your current throughput target tomorrow,
   what's the first thing you'd change?
6. What's the single point of failure in this architecture, if any?
7. How would this design change if you needed strong consistency
   instead of eventual consistency on the redirect path?

## ID Generation

8. Why Snowflake IDs instead of UUIDs or auto-increment?
9. Walk me through the bit layout of your ID. Why that split between
   timestamp, node ID, and sequence bits?
10. What happens if the system clock moves backwards mid-operation?
    Why did you choose to return an error instead of blocking or
    reusing the last timestamp?
11. What's the maximum IDs/sec a single node can generate with your bit
    allocation, and how do you know?
12. Why is `NextID()` guarded by a mutex, and did that turn out to be a
    bottleneck? How did you verify your answer instead of guessing?
13. How would two different instances ever generate the same ID in your
    system, if at all? What prevents it?
14. Why Base62 instead of hex or Base64 for the short code encoding?
15. Are your short codes predictable/enumerable? Does that matter, and
    if so, how would you fix it?

## Caching

16. Explain cache-aside vs. write-through vs. write-behind. Which did
    you use, and why?
17. Walk me through exactly what happens on a cache miss, step by step.
18. What's your cache's consistency guarantee? Under what circumstances
    can a client see stale data, and for how long?
19. Why didn't you actively invalidate the cache when a URL is
    deactivated? What would you build to fix that, and why haven't you
    yet?
20. Your cache TTL is an hour, how would you decide the "right" TTL
    for a system like this?
21. What happens to your system if Redis goes down entirely? Did you
    design for that, or would it just fail?
22. How would you determine your actual cache hit ratio in production,
    and what would you do if it dropped below target?

## Database & Storage

23. Walk me through your schema. Why those specific indexes, and not
    others?
24. Explain your partial unique index for dedup, why partial, and what
    would break if it weren't?
25. Why did you choose Postgres over a NoSQL store to start, given the
    project's stated scale ambitions?
26. What would migrating to a sharded/NoSQL store actually require,
    given how you structured the storage layer? What wouldn't need to
    change?
27. Why did you decide *not* to migrate to ScyllaDB, even though it was
    part of the original plan? What data led to that decision?
28. Why `sqlc` instead of an ORM or hand-written SQL calls? What did
    you give up by not using an ORM?
29. How do you hash the long URL, and why hash it at all instead of
    indexing the raw URL column?
30. What's your connection pool sizing, and how did you determine it
    wasn't the default? Walk me through how you diagnosed that.

## Concurrency & Idempotency

31. What does "idempotent" mean in the context of your `/shorten`
    endpoint specifically, and for which users does it apply?
32. Walk me through the bug you found during load testing, what was
    happening, how did you find it, and how did you fix it?
33. Your dedup check is "look up, then insert if not found", what's
    wrong with that under real concurrency, and how would you fix it
    properly?
34. Why does a duplicate long URL from two *different* users produce
    two different short codes instead of sharing one?
35. How would you make your create endpoint fully idempotent even for
    network retries (e.g. a client that times out and resends the exact
    same request)?

## Authentication & Security

36. Why API keys instead of JWTs or session cookies for this product?
37. Walk me through what's actually stored in your database for a
    user's credentials, and why never the raw key.
38. What entropy source did you use to generate API keys, and why does
    that choice matter?
39. Your stats endpoint returns 404, not 403, for a non-owner's request
    — why?
40. What happens if someone sends a malformed or garbage
    `Authorization` header versus no header at all? Why the different
    behavior?
41. What's missing from your auth system that a real production system
    would need? (key rotation, revocation, expiry)
42. How would you rate-limit login/registration attempts specifically,
    separate from your general API rate limiting?

## Rate Limiting

43. Explain the fixed-window algorithm you implemented and its known
    weakness. Why did you accept that weakness?
44. Walk me through an alternative algorithm (sliding window, token
    bucket, leaky bucket) and how it would have avoided that weakness.
45. Why did you choose to "fail open" when Redis is unreachable for
    rate limiting? What's the argument for "fail closed" instead?
46. How do you key rate limits differently for authenticated vs.
    anonymous users, and why?
47. Your `INCR`+`EXPIRE` isn't atomic, what's the actual failure mode,
    how likely is it, and how would you make it atomic?

## Distributed Coordination

48. What problem does etcd solve in your system that you couldn't
    solve with Postgres or Redis?
49. Explain the lease mechanism you used, what happens automatically
    when an instance crashes, and why?
50. Walk me through the actual transaction you use to atomically claim
    a node ID. What guarantees does that transaction give you?
51. What happens if all node IDs are exhausted? Is that handled
    gracefully?
52. Is etcd itself a single point of failure in your design? How would
    you make it highly available?
53. You said this is the same underlying technology Kubernetes uses
    for cluster state, what other coordination problems could etcd
    solve that you haven't used it for here?

## Messaging & Analytics

54. Why is the click-event publish fire-and-forget instead of
    synchronous? What would break if it were synchronous?
55. Explain the context-cancellation bug you specifically avoided by
    using a detached context for the async publish. Why does the
    original request context get cancelled?
56. What happens to a click event if your consumer crashes between
    reading it from Kafka and writing it to ClickHouse? Is that
    acceptable, and why?
57. Why batch inserts into ClickHouse instead of inserting each event
    as it arrives?
58. What is a Kafka consumer group, and why does your consumer use one?
59. Why ClickHouse specifically, instead of writing analytics into
    Postgres directly?
60. How would you guarantee exactly-once delivery for click events, if
    you needed to? What would that cost you?

## Load Testing & Performance Debugging

61. Walk me through your entire debugging process for the write-path
    bottleneck, what did you suspect first, how did you rule it out,
    and what did you find instead?
62. Why did you benchmark `idgen` in isolation instead of just
    profiling the whole system? What does that isolation buy you?
63. Explain what "requests/sec plateaus while latency climbs" tells you
    about a system, generically, not just in your project.
64. What resource metric told you the backend, not the database, was
    the bottleneck? Walk through the actual numbers.
65. Why did injecting artificial DB latency cause a non-linear collapse
    in throughput instead of a linear one? What does that tell you
    about how connection pools behave under load?
66. How would you determine the "correct" connection pool size for a
    production system, versus picking a number and testing it?
67. What's the difference between testing on `localhost` and testing
    with real network RTT? Why does that distinction matter?

## Trade-offs & Prioritization (behavioral-adjacent)

68. You dropped the CDN/edge phase entirely, how did you decide that
    was the right call instead of just running out of time?
69. You deferred database sharding, what evidence would have changed
    your mind and made you build it anyway?
70. What's one decision in this project you'd defend strongly, and one
    you're genuinely unsure about?
71. If a teammate took over this project tomorrow, what's the first
    thing you'd tell them to watch out for?
72. What would you build first if you had one more week on this
    project, and why that over everything else on your known-gaps
    list?

## Curveballs

73. Your short codes are Base62-encoded Snowflake IDs, could someone
    guess a valid, currently-unused short code and squat on it before
    a real user creates it? Is that a real risk here?
74. What happens if two people register with the same email
    simultaneously? Is that actually prevented, and how?
75. Your redirect uses a `302`, not a `301`, what's the actual
    functional difference, and why does it matter for this specific
    product?
76. If you needed to support custom domains (users bringing their own
    domain instead of yours), what would have to change in this
    architecture?
77. How would you detect and block someone using your shortener to
    generate phishing links at scale?
78. Explain exactly what happens, system-wide, in the 10 seconds after
    one of your app instances is killed with `SIGKILL`.s