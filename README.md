# A Go Load Balancer

> **This project was built as a learning exercise to write real, idiomatic, production-quality Go — not Java in disguise, as I am a Java developer (mainly)**
>
> Every decision made here — from atomic counters over mutexes where appropriate, to consumer-defined interfaces, to explicit error handling — was made with the goal of writing Go the way Go developers actually write it.

---

## What It Does

`A Go Load Balancer` is a reverse proxy load balancer that sits in front of one or more backend services and distributes incoming HTTP traffic across their instances. It supports:

- **Round-robin** routing — take turns across healthy backends
- **Least-connections** routing — route to whichever backend is least busy
- **Active health checking** — each backend is periodically pinged; unhealthy backends are removed from rotation automatically
- **Host-based routing** — route traffic to different backend pools based on the incoming request's `Host` header
- **Graceful shutdown** — in-flight requests are drained before the process exits
- **REST API** — a `/api/v1/loadBalancers/report` endpoint exposes the current health status of all registered backends

---

## Project Structure

```
load-balancer/
  cmd/
    main.go              # Entry point — thin, just wires everything together
  internal/
    api/
      server.go          # Server struct, route registration, graceful shutdown
      handler.go         # HTTP handlers — proxy and report
    backend/
      backend.go         # Backend struct, health checking, connection tracking
    balancer/
      balancer.go        # LoadBalancer struct, delegates to Strategy
      round_robin.go     # Round-robin strategy implementation
      least_connections.go # Least-connections strategy implementation
    config/
      config.go          # YAML config loading
  config.yaml            # Your configuration file
```

---

## Configuration

Create a `config.yaml` in the same directory you run the binary from:

```yaml
apps:
  - host: api.example.com
    health_uri: /health
    timeout: 10s
    health_check_cooldown: 30s
    strategy: round_robin
    instances:
      - url: http://localhost:8081
      - url: http://localhost:8082
      - url: http://localhost:8083

  - host: payments.example.com
    health_uri: /api/health
    timeout: 5s
    health_check_cooldown: 15s
    strategy: least_connections
    instances:
      - url: http://localhost:9081
      - url: http://localhost:9082
```

### Configuration Reference

| Field | Description | Example |
|---|---|---|
| `host` | Incoming `Host` header to match | `api.example.com` |
| `health_uri` | Path to hit for health checks | `/health` |
| `timeout` | HTTP client timeout per request | `10s` |
| `health_check_cooldown` | Interval between health checks | `30s` |
| `strategy` | Routing strategy | `round_robin` or `least_connections` |
| `instances[].url` | Backend instance URL | `http://localhost:8081` |

### Strategies

**`round_robin`** — Cycles through healthy backends in order. Best for backends with roughly equal capacity and request duration.

**`least_connections`** — Routes to the backend with the fewest active connections. Better for workloads with variable request duration, as it naturally avoids overloading slow backends.

---

## Running

### Prerequisites

- Go 1.22+
- GCC (required by CGO dependencies on Linux/WSL: `sudo apt-get install gcc`)

### Run directly

```bash
go run cmd/main.go
```

The server listens on `:8080` by default.

### Build a binary

```bash
go build -o load-balancer cmd/main.go
./load-balancer
```

### Custom port

Pass a different port programmatically via `api.NewServer(port, pathToConfig)` in `main.go`.

---

## Testing

### Run all tests

```bash
go test ./...
```

### Run with race detection

```bash
go test -race ./...
```

Race detection is particularly important for this project given the concurrent health checking and request routing. Always run with `-race` before committing.

### Run benchmarks

```bash
go test -bench=. -benchmem ./...
```

Notable benchmark results on an i9-14900KF:

| Benchmark | ns/op | Notes |
|---|---|---|
| `RoundRobin.NextBackend` | ~11ns | Atomic write causes contention under parallelism |
| `LeastConnections.NextBackend` | ~8ns | Read-only atomics scale better under concurrency |
| `Backend.IsHealthy` | ~0.17ns | Essentially free — single atomic read |

### Test coverage

Tests are written as **table-driven tests** throughout — the idiomatic Go approach. Coverage includes:

- URL validation and constructor error paths with sentinel errors
- Round-robin sequencing across healthy and unhealthy backends
- Least-connections backend selection and tie-breaking
- Health check HTTP response handling via `httptest.NewServer`
- Mutex prevention of stacked concurrent health checks
- YAML config parsing with temporary files
- Race condition verification with `-race`

---

## API

### `GET /api/v1/loadBalancers/report`

Returns the current health status of all registered backends.

**Response:**

```json
{
  "apps": [
    {
      "host": "api.example.com",
      "instances": [
        { "url": "http://localhost:8081", "healthy": true },
        { "url": "http://localhost:8082", "healthy": false },
        { "url": "http://localhost:8083", "healthy": true }
      ]
    }
  ]
}
```

---

## What Was Learned Building This

This project was deliberately chosen to force idiomatic Go rather than allowing Java patterns to sneak in. Key concepts encountered naturally through the problem domain:

**Interfaces defined at the consumption site** — the `Strategy` interface lives in the `balancer` package and was extracted only when a second implementation (`LeastConnections`) was needed. It was never designed upfront.

**Explicit error handling** — every function that can fail returns an error. Sentinel errors (`ErrInvalidScheme`, `NoHealthyBackends`) allow callers to check specific failure reasons with `errors.Is`.

**Concurrency primitives used appropriately** — `atomic.Uint64` for the round-robin counter (no mutex needed for a single incrementing value), `atomic.Bool` for health state, `sync.Mutex.TryLock` to prevent stacked health checks.

**Goroutines started where their purpose is obvious** — health check goroutines are started in `LoadBalancer.StartHealthChecks`, not buried inside construction functions.

**Context for lifecycle management** — a single `context` created at server startup propagates through to all health check goroutines, stopping them cleanly on shutdown signal.

**Table-driven tests as the default** — every test package uses the standard Go table-driven pattern with anonymous structs.

---

## What This Is Not

This is not production infrastructure. It lacks:

- TLS termination
- Dynamic service discovery (backends are statically configured)
- Persistent metrics
- Access logging
- Circuit breaking
- Weighted routing

These are intentional omissions — the goal was depth of understanding over breadth of features. **API Proxy/Gateway** will tackle a production-grade API gateway with real-world deployment concerns addressed from day one.