# CUJU Leaderboard System

A high-performance, in-memory leaderboard system designed for real-time talent scoring and ranking. The system processes performance events asynchronously, maintains idempotent event processing, and provides fast read access to leaderboard data.

## 🚀 How to Run

**Quick Start:**
```bash
make run
```
This will build the service, start it, and automatically open the dashboard in your browser at `http://localhost:9080/dashboard`.

**Alternative:**
```bash
make run-simple  # Runs without opening browser
```

**Docker:**
```bash
docker build -t cuju .
docker run -p 9080:9080 cuju  # Run with Docker
```

## 📚 Documentation

**Start here for comprehensive system understanding:**

* **[🏗️ Complete Architecture Documentation](docs/ARCHITECTURE.md)** - Detailed system architecture, data structures, time complexities, and design patterns
* **[📊 Data Structures & Algorithms](docs/DATA_STRUCTURES.md)** - Deep dive into treap implementation, concurrency strategy, and complexity analysis  
* **[🔄 Event Flow Sequence Diagram](docs/SEQUENCE_DIAGRAM.md)** - Visual representation of the complete event processing flow
* **[⚡ Quick Reference Guide](docs/QUICK_REFERENCE.md)** - Developer quick start guide with API examples and configuration

## 🚀 Quick Start

### Run the Service
```bash
# Build and run (opens dashboard in browser automatically)
make run

# Or run without opening browser
make run-simple

# Or with custom config
export CUJU_CONFIG=./config.yaml
./bin/cuju
```

### First Steps
1. Run `make run` to start the service
2. Dashboard opens automatically at `http://localhost:9080/dashboard`
3. Try the API examples below to test the system

### API Endpoints
* **POST /events** - Submit performance events
* **GET /leaderboard?limit=N** - Get top N leaderboard entries  
* **GET /rank/{talent_id}** - Get specific talent rank
* **GET /healthz** - Health check
* **GET /dashboard** - Web monitoring dashboard

### Example Usage
```bash
# Submit an event
curl -X POST http://localhost:9080/events \
  -H "Content-Type: application/json" \
  -d '{"event_id": "123e4567-e89b-12d3-a456-426614174000", "talent_id": "t1", "raw_metric": 42.5, "skill": "dribble", "ts": "2025-01-27T10:00:00Z"}'

# Get leaderboard
curl "http://localhost:9080/leaderboard?limit=10"

# Get rank
curl "http://localhost:9080/rank/t1"
```

## 🎯 System Overview

**Core Features:**
- **Idempotent Event Processing** - Duplicate events are detected and ignored
- **Asynchronous Scoring** - Events processed in background workers with simulated ML latency
- **High-Performance Storage** - Single treap with RWMutex for concurrent access
- **Real-time Metrics** - Comprehensive Prometheus metrics for monitoring
- **High Performance** - Optimized for sub-40ms read latencies

**Key Design Decisions:**
- **Treap Data Structure** - Chosen over simple maps for optimal Top-N query performance
- **Single Treap Architecture** - Enables concurrent updates while maintaining global ordering
- **Fixed-Point Arithmetic** - Eliminates floating-point precision issues (12 decimal places)
- **Read-Heavy Optimization** - Prioritizes leaderboard queries over individual updates

## 🧪 Performance Testing & Validation

This project includes comprehensive stress testing to validate repository performance under realistic production conditions:

### 🚀 **Comprehensive Stress Testing**
- **30M Talents**: Tests with 30 million talent records
- **All APIs Simultaneously**: UpdateBest, Rank, TopN, and Count under pressure
- **Realistic Workloads**: Production-like API call distribution
- **Extended Duration**: 15-20 minute tests for stable metrics
- **High Concurrency**: 2,000-5,000 concurrent workers

### 📊 **Key Metrics Measured**
- **P90, P95, P99 Latency**: Response time percentiles for production planning
- **Throughput**: Operations per second for each API
- **Success Rates**: Error handling and system stability
- **Memory Usage**: Resource consumption under load
- **Snapshot Impact**: Performance impact of periodic snapshots

### 🧪 **Test Scenarios Available**
- **Comprehensive**: Balanced workload, all APIs under pressure
- **Extreme Pressure**: Maximum concurrency (5K workers), 20 minutes
- **Write-Heavy**: 70% updates, heavy write operations
- **Read-Heavy**: 50% rank queries, heavy read operations
- **TopN-Heavy**: 50% topN queries, leaderboard performance

### 🚦 **Quick Start**
```bash
# Run all stress test scenarios
make bench-repo

# Run individual comprehensive test
go test -v -bench=BenchmarkTreapStore_30MTalents_ComprehensiveStressTest \
    -benchmem -run=^$ ./internal/adapters/repository/ \
    -timeout=30m -benchtime=15m
```

---

## 📋 Original Task Requirements

### Functional scope

#### 1. POST /events

* Body:

  ```json
  { "event_id": "uuid", "talent_id": "t1", "raw_metric": 42, "skill": "dribble", "ts": "RFC3339" }
  ```
* Idempotent on `event_id` (replays ignored).
* Returns 202 on accepted, 200 on duplicate.

#### 2. Scoring (local stub, no HTTP)

* Compute `score = f(raw_metric, skill)` with a random **80–150ms** delay to simulate ML latency.
  Example: `score = raw_metric * weight(skill)`.

#### 3. Leaderboard update

* Maintain a single Global leaderboard (Top-N by score, higher is better).
* Track per-talent best score only.

#### 4. GET /leaderboard?limit=N

* Returns Top-N:

  ```json
  [{"rank":1,"talent_id":"…","score":…}, …]
  ```

#### 5. GET /rank/{talent\_id}

* Returns the caller's current rank and score.

---

### Non-functional checks

* Safe concurrent updates (multiple POST /events in parallel).
* Basic performance: **p95 read < 40ms locally** for `GET /leaderboard?limit=50` after a burst of writes.
* Minimal observability: `/healthz` + counters (processed events, dedup hits). (Prometheus optional.)


### Example payloads

**POST /events**

```json
{
  "event_id": "8c1b7c3e-3b1f-4a19-9d49-0f5f0d1c9a11",
  "talent_id": "t-123",
  "raw_metric": 37.5,
  "skill": "dribble",
  "ts": "2025-08-28T09:15:00Z"
}
```

**GET /leaderboard?limit=3**

```json
[
  {"rank":1, "talent_id":"t-123", "score":112.5},
  {"rank":2, "talent_id":"t-777", "score":98.0},
  {"rank":3, "talent_id":"t-555", "score":91.2}
]
```

**GET /rank/t-123**

```json
{"rank":1, "talent_id":"t-123", "score":112.5}
```

---

### Format of delivery

* Send as **ZIP file** or a **GitHub/GitLab repo**.
* There will be a call where you walk through your task and thoughts.

---

## 🏗️ Architecture Summary

**Key Components:**
* **HTTP API Layer** - Request handling and validation
* **Application Service** - Component orchestration and lifecycle  
* **Repository Layer** - Single treap store for leaderboard data
* **Message Queue** - Asynchronous event processing
* **Worker Pool** - Concurrent event scoring
* **Deduplication Cache** - Idempotent event processing with LIFO eviction
* **Scoring Engine** - ML latency simulation (80-150ms)
* **Metrics System** - Comprehensive Prometheus monitoring

**Time Complexities:**
* Event Submission: **O(1)** - Hash map lookup + channel send
* Leaderboard Query: **O(log n + N)** - Treap in-order traversal
* Rank Lookup: **O(log n)** - Single treap rank calculation
* Update Operations: **O(log n)** - Treap insert/update

**Scale:**
* **MVP (Current)**: Single process, in-memory, < 1M talents, < 10K events/sec
* **Production (30M AUs)**: Distributed architecture, persistent storage, advanced caching, load balancing

For complete architectural details, see the [Architecture Documentation](docs/ARCHITECTURE.md).

## 🛠️ Development

### Build & Test
```bash
# Build the application
make build

# Run tests
make test

# Run benchmarks
make bench

# Run linting
make lint

# Run all checks
make check
```

### Configuration
Create `config.yaml` based on `config.example.yaml`:
```yaml
log_level: "info"
addr: ":9080"
queue_size: 100000
# worker_count: 16  # Defaults to 20x CPU cores
dedupe_size: 500000
scoring_latency_min_ms: 80
scoring_latency_max_ms: 150
skill_weights:
  dribble: 3.0
  shooting: 2.0
  passing: 1.2
default_skill_weight: 1.5
```

### Docker
```bash
# Build the Docker image
docker build -t cuju .

# Run the container
docker run -p 9080:9080 cuju

# Run with custom config
docker run -p 9080:9080 -v $(pwd)/config.yaml:/app/config.yaml cuju
```

### Monitoring
- **Metrics**: `http://localhost:9080/healthz` (Prometheus metrics)
- **Dashboard**: `http://localhost:9080/dashboard` (Web UI)
- **Health**: `http://localhost:9080/healthz`
- **Stats**: `http://localhost:9080/stats`
