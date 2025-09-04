# CUJU Senior GO Engineer — System Design + Coding Task

### Coding Task

A Leaderboard Ingest + Query service with idempotent event intake, concurrent processing, and fast reads—but all in one process with in-memory storage (no external deps).

Build a single Go service that accepts performance events, deduplicates by `event_id`, asynchronously "scores" them (simulate 80–150ms latency), and maintains an in-memory Global leaderboard (best score per talent).

Expose:

* **POST /events**
* **GET /leaderboard?limit=N**
* **GET /rank/{talent\_id}**
* **/healthz**

Include a short design note, which should contain a sequence diagram with user interactions, a README with run/curl examples, and a few tests.
Aim for 60–90 minutes; avoid over-engineering.

---

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

---

### Performance Validation & Stress Testing

This project includes comprehensive stress testing to validate repository performance under realistic production conditions:

#### 🚀 **Comprehensive Stress Testing**
- **30M Talents**: Tests with 30 million talent records
- **All APIs Simultaneously**: UpdateBest, Rank, TopN, and Count under pressure
- **Realistic Workloads**: Production-like API call distribution
- **Extended Duration**: 15-20 minute tests for stable metrics
- **High Concurrency**: 2,000-5,000 concurrent workers

#### 📊 **Key Metrics Measured**
- **P90, P95, P99 Latency**: Response time percentiles for production planning
- **Throughput**: Operations per second for each API
- **Success Rates**: Error handling and system stability
- **Memory Usage**: Resource consumption under load
- **Snapshot Impact**: Performance impact of periodic snapshots

#### 🧪 **Test Scenarios Available**
- **Comprehensive**: Balanced workload, all APIs under pressure
- **Extreme Pressure**: Maximum concurrency (5K workers), 20 minutes
- **Write-Heavy**: 70% updates, heavy write operations
- **Read-Heavy**: 50% rank queries, heavy read operations
- **TopN-Heavy**: 50% topN queries, leaderboard performance

#### 🚦 **Quick Start**
```bash
# Run all stress test scenarios
./scripts/run_stress_tests.sh

# Run individual comprehensive test
go test -v -bench=BenchmarkTreapStore_30MTalents_ComprehensiveStressTest \
    -benchmem -run=^$ ./internal/adapters/repository/ \
    -timeout=30m -benchtime=15m
```

#### 📚 **Documentation**
- [Comprehensive Stress Testing Guide](docs/comprehensive_stress_testing.md)
- [Quick Reference](docs/stress_test_quick_reference.md)
- [Performance Analysis](docs/benchmarks.md)

---

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

### System Design

Comprehensive system design documentation is available in the `docs/` directory:

* **[Complete Architecture Documentation](docs/ARCHITECTURE.md)** - Detailed system architecture, data structures, time complexities, and design patterns
* **[Event Flow Sequence Diagram](docs/SEQUENCE_DIAGRAM.md)** - Visual representation of the complete event processing flow from user request to leaderboard storage
* **[Data Structures & Algorithms](docs/DATA_STRUCTURES.md)** - Deep dive into treap implementation, sharding strategy, and complexity analysis
* **[Quick Reference Guide](docs/QUICK_REFERENCE.md)** - Developer quick start guide with API examples and configuration

#### Quick Design Summary

**Data Model:**
* `scores[talent_id] = bestScore` - Sharded treap-based storage
* Ranking structure - O(log n) operations with deterministic tie-breaking

**Key Components:**
* **HTTP API Layer** - Request handling and validation
* **Application Service** - Component orchestration and lifecycle
* **Repository Layer** - Sharded treap store for leaderboard data
* **Message Queue** - Asynchronous event processing
* **Worker Pool** - Concurrent event scoring
* **Deduplication Cache** - Idempotent event processing
* **Scoring Engine** - ML latency simulation (80-150ms)
* **Metrics System** - Comprehensive Prometheus monitoring

**Time Complexities:**
* Event Submission: **O(1)** - Hash map lookup + channel send
* Leaderboard Query: **O(n_shard × log n_shard + N)** - K-way merge
* Rank Lookup: **O(n_shard × log n_shard)** - Global rank calculation
* Update Operations: **O(log n_shard)** - Treap insert/update

**Trade-offs:**
* **In-Memory Storage** - Ultra-fast access vs. data persistence
* **Sharded Architecture** - Concurrent writes vs. complex global operations
* **Asynchronous Processing** - High throughput vs. eventual consistency
* **Treap Data Structure** - Balanced performance vs. implementation complexity

**MVP vs Production Scale:**
* **MVP (Current)**: Single process, in-memory, < 1M talents, < 10K events/sec
* **Production (30M AUs)**: Distributed architecture, persistent storage, advanced caching, load balancing

For complete details, see the [Architecture Documentation](docs/ARCHITECTURE.md).
