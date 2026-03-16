# Web Page Analyzer

## Getting Started

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/) and Docker Compose

### Run the application

```bash
make run
```

This starts the app and Redis via Docker Compose. The application is available at `http://localhost:8080`.

### Other commands

```bash
make build      # build the Docker image
make test       # run unit and integration tests
make test-e2e   # run end-to-end tests
make down       # stop and remove containers
make logs       # tail application logs
make lint       # run golangci-lint
```

---

## Architecture Overview

The application is **asynchronous by design**. A synchronous approach would be unacceptable because link accessibility checking involves potentially hundreds of outbound HTTP requests, making response times unpredictable and the UX poor.

```
Client (HTMX)
    │
    ├── POST /analyze  →  validate input → enqueue job → return "processing"
    │
    └── GET /result    →  read from Redis → return current state (per-step + overall)

Background Worker
    └── Staged Pipeline → writes step results to Redis as they complete
```

---

## Technology Decisions

### Go
Required by the task. Well-suited for this workload due to first-class concurrency primitives (goroutines, channels) and low memory overhead for I/O-bound work.

### Gin (HTTP framework)
Lightweight, idiomatic, and widely adopted in the Go ecosystem. Chosen for:
- Built-in middleware support (used for rate limiting)
- Clean routing API
- No unnecessary abstractions over `net/http`

### Redis (job state store)
Chosen over in-memory storage because:
- Survives application restarts
- Allows multiple app instances to share job state (horizontal scaling)
- Native TTL support — no manual cleanup needed
- Cache TTL set to **10 minutes**, sufficient for result review without unbounded growth

### HTMX (frontend)
No JavaScript framework is warranted here. The UI is a single form + result display. HTMX handles polling and partial DOM updates without the overhead of a full SPA. This keeps the frontend maintainable and the scope focused.

### Zerolog (structured logging)
Fast, zero-allocation structured logger. Chosen over `log/slog` for performance and developer ergonomics. Every log entry includes a correlation ID (job ID) for tracing a request end-to-end.

---

## Running & Deployment

### Docker Compose
The repository ships with a `docker-compose.yml` that starts:
- **app** — the Go web server (built from the local `Dockerfile`)
- **redis** — Redis 7 with no persistence (results are ephemeral by design)

The app container depends on Redis being healthy before starting. No manual setup or local Go installation is required.

### Makefile
All common operations are exposed via `make` targets so the reviewer does not need to know the underlying commands:

| Target | Description |
|---|---|
| `make run` | Build image and start all services |
| `make build` | Build the Docker image only |
| `make test` | Run unit + integration tests inside a container |
| `make test-e2e` | Run end-to-end tests against the running stack |
| `make down` | Stop and remove all containers |
| `make logs` | Tail application logs |
| `make lint` | Run `golangci-lint` |

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `redis:6379` | Redis address |
| `PORT` | `8080` | HTTP server port |
| `CACHE_TTL_MINUTES` | `10` | Job result TTL in Redis |
| `RATE_LIMIT_RPS` | `10` | Max requests per second per IP |
| `LINK_CHECK_CONCURRENCY` | `20` | Max concurrent link accessibility checks |

---

## Input Validation & Security

### Domain-only input
The application accepts **domains only** — no raw IPs, no custom ports.

**Rationale:** The primary security concern with a URL-fetching service is Server-Side Request Forgery (SSRF), where an attacker supplies an internal address to probe internal infrastructure. Rejecting IPs and ports eliminates the most obvious attack vectors.

**Validation rules:**
- Input must be a valid domain (regex enforced)
- No IP addresses (v4 or v6)
- No port numbers
- Scheme (`http://`, `https://`, `www.`) is stripped and normalized before use

### DNS rebinding mitigation
Domain-only validation is not sufficient alone. An attacker can register a domain that resolves to a private IP (e.g. `169.254.169.254`, AWS metadata endpoint). After DNS resolution, the resolved IP is validated against private and link-local ranges before any HTTP request is made.

### Rate limiting
Applied at the Gin middleware level, per client IP. Prevents a single client from flooding the job queue. Both per-IP and global limits are applied.

---

## URL Normalization (Cache Key Strategy)

Two URLs pointing to the same page should share a cache entry. Normalization before hashing:

1. Strip protocol (`http://`, `https://`)
2. Strip `www.` prefix
3. Lowercase scheme and host
4. Keep path, query string, and fragment intact

**Decision:** path and query string are preserved because `example.com/page1` and `example.com/page2` are different pages with different content.

The normalized URL is `sha256`-hashed to produce the Redis key, avoiding key length issues with very long URLs.

**Trade-off acknowledged:** Two users requesting the same URL share a cached result. This is intentional — it reduces redundant outbound fetches. If freshness is required, a cache-bust parameter can be added as a future improvement.

---

## Staged Pipeline Design

Rather than running all analysis steps in a flat concurrent fan-out, steps are organized into **dependency-ordered stages**. A stage only starts after all steps in the previous stage have succeeded. This ensures fail-fast behavior and guarantees data dependencies are met.

```
Stage 1 — URL Validation (blocking, fail-fast)
    └── Validate domain format
    └── DNS resolve + private IP check (SSRF guard)

Stage 2 — Fetch HTML (blocking, fail-fast)
    └── HTTP GET with timeout
    └── Store raw HTML in shared job state

Stage 3 — Concurrent Analysis (all run in parallel)
    ├── HTML version detection
    ├── Page title extraction
    ├── Heading counts (h1–h6)
    ├── Login form detection
    └── Link extraction (populates link list for Stage 4)

Stage 4 — Link Accessibility (concurrent, semaphore-bounded)
    └── HTTP HEAD per link (max 20 concurrent via semaphore)
    └── Classifies each link as internal, external, or inaccessible
```

**If any stage fails**, subsequent stages do not run. The job state in Redis is immediately updated to `failed` with the error detail.

### Step Interface

Each step is self-contained and declares which stage it belongs to:

```go
type Step interface {
    Name()  string
    Stage() int
    Run(ctx context.Context, state *State) error
}
```

Adding a new analysis step requires only implementing this interface and registering it. The pipeline runner requires no modification — open/closed principle.

### Concurrency Primitive: `errgroup`

Each stage runs its steps with `golang.org/x/sync/errgroup`. This provides:
- WaitGroup semantics (wait for all steps in the stage)
- Automatic error propagation (first error cancels the context for the whole stage)
- Clean context cancellation through the pipeline

### Shared State

Steps within a stage communicate via a shared `State` struct (protected by `sync.RWMutex`). Stage 4 reads the link list populated by Stage 3 — this is safe because stages are sequential even though steps within a stage are concurrent.

---

## Redis Schema

Job state is stored as a **Redis Hash** (not a JSON blob). This allows individual steps to update their own fields atomically via `HSET` without read-modify-write cycles or locking, avoiding race conditions between concurrent goroutines.

```
Key:    job:{sha256(normalizedURL)}
TTL:    10 minutes

Fields:
    overall_status              →  pending | processing | done | failed
    overall_error               →  "" | "error description"

    step:url_validation:status  →  pending | done | failed
    step:fetch_html:status      →  pending | done | failed
    step:title:status           →  pending | done | failed
    step:title:data             →  "Page Title Here"
    step:headings:status        →  pending | done | failed
    step:headings:data          →  {"h1":1,"h2":3,"h3":5}
    step:links:status           →  pending | done | failed
    step:links:data             →  {"internal":12,"external":4,"inaccessible":2}
    step:login_form:status      →  pending | done | failed
    step:login_form:data        →  "true" | "false"
    step:html_version:status    →  pending | done | failed
    step:html_version:data      →  "HTML5"
```

Steps update Redis as they complete, enabling **progressive frontend updates** — the UI shows results for completed steps while others are still running.

---

## Frontend Polling

HTMX polls `GET /result?url={url}` on a short interval (1.5s). The endpoint returns the current Redis hash state. HTMX swaps the result partial as data changes.

Polling stops when `overall_status` is `done` or `failed`.

**Trade-off acknowledged:** Server-Sent Events (SSE) would be more efficient (server pushes instead of client polling). HTMX supports SSE natively. Polling was chosen for simplicity and reliability — SSE connections require more careful connection lifecycle management under load. This is noted as a future improvement.

---

## Error Handling

| Scenario | Behavior |
|---|---|
| Invalid domain format | Stage 1 fails immediately, `overall_status: failed`, error shown to user |
| DNS resolves to private IP | Stage 1 fails, SSRF guard message returned |
| URL unreachable (timeout, 4xx, 5xx) | Stage 2 fails, HTTP status code + description stored and shown |
| Step-level failure (e.g. malformed HTML) | That step marked `failed`, pipeline continues where possible |
| Redis unavailable | Returns 503, logged with zerolog |

---

## Testing Strategy

- **Unit tests** for each pipeline step in isolation (mock HTTP client, mock HTML input)
- **Integration tests** for the full pipeline against a local test HTTP server
- **E2E tests** for the full stack (submit URL → poll → verify result) using a controlled test server
- **Table-driven tests** for URL validation and normalization edge cases

---

## Assumptions Made

1. **No authentication required** — the task does not mention users or sessions.
2. **No persistence beyond Redis TTL** — results are ephemeral; no database needed.
3. **Domain-only input** is a reasonable constraint that simplifies security without meaningfully reducing usefulness.
4. **Link accessibility uses HEAD requests** — faster and sufficient; falls back to GET if HEAD is rejected (405).
5. **Internal vs external link classification** is based on comparing the link's host to the analyzed URL's host.
6. **Login form detection** is heuristic — presence of `<input type="password">` within a `<form>`.

---

## Possible Improvements

1. **SSE instead of polling** — push results to the client as each step completes
2. **Job deduplication UI** — inform the user if a cached result is being served and how old it is
3. **Configurable step registry** — enable/disable steps via config without redeployment
4. **Metrics & tracing** — Prometheus metrics per step (duration, error rate), OpenTelemetry traces
5. **Auth + per-user history** — store past analyses per user
6. **Webhook support** — notify an external URL when analysis is complete
7. **Depth option** — allow crawling N levels deep from the submitted URL
