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
* **GET /healthz** - Health check (Prometheus metrics)
* **GET /stats** - Service statistics (JSON format)
* **GET /dashboard** - Web monitoring dashboard (HTML with real-time metrics)
* **GET /api-docs** - Interactive API documentation (Swagger UI with ReDoc)

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
* Rank Lookup: **O(1)** - Snapshot-based lookup
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
queue_size: 200000
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

## 🎯 Design Trade-offs & Scaling Considerations

### In-Memory vs Persistent Storage Trade-offs

**Current Choice: In-Memory Storage**

**Why In-Memory + Snapshots Were Omitted:**
- **Simplicity**: Eliminates complex snapshot management, recovery logic, and disk I/O coordination
- **Performance**: Direct memory access provides optimal latency for real-time leaderboards
- **Development Speed**: Faster iteration and testing without persistence layer complexity
- **Resource Efficiency**: No disk space management, backup strategies, or data migration concerns

**Trade-offs Accepted:**
- **Data Loss Risk**: System restart loses all leaderboard state
- **Limited Scale**: Memory capacity constrains maximum talent count
- **No Historical Data**: Cannot track score changes over time
- **Single Point of Failure**: No data redundancy or recovery mechanisms

### Eventual vs Strong Consistency

**Current Choice: Eventual Consistency**

**Design Rationale:**
- **Performance Priority**: Strong consistency would require distributed locking, significantly impacting latency
- **User Experience**: Leaderboard updates can be slightly delayed (seconds) without affecting user experience
- **Scalability**: Eventual consistency enables horizontal scaling and better throughput
- **Fault Tolerance**: System remains available even if some components are temporarily unavailable

**Consistency Model:**
- **Event Processing**: Asynchronous with eventual consistency (80-150ms delay)
- **Read Operations**: Strong consistency within single process (immediate consistency)
- **Cross-Region**: Would require eventual consistency for global deployment

### MVP vs Production Scale Design Differences

#### **MVP (Current): Closed Beta, Few Hundred Users**

**Architecture Characteristics:**
- **Single Process**: All components in one application instance
- **In-Memory Storage**: Fast access, simple deployment
- **Basic Monitoring**: Health checks and basic metrics
- **Local Deployment**: Single region, single data center
- **Simple Configuration**: YAML-based config with environment overrides

**Performance Profile:**
- **Scale**: < 1,000 concurrent users, < 10,000 events/second
- **Latency**: Sub-40ms read operations, 80-150ms event processing
- **Memory**: < 10GB for full dataset
- **Availability**: Single point of failure, manual recovery

**Operational Model:**
- **Deployment**: Simple binary deployment or Docker container
- **Monitoring**: Basic health checks and Prometheus metrics
- **Scaling**: Vertical scaling (more CPU/memory)
- **Recovery**: Manual restart, data loss on failure

#### **Production Scale: 30M MAUs, Multi-Country (Brazil, Germany, South Africa)**

**Architecture Characteristics:**
- **Distributed Microservices**: Separate services for API, processing, storage, and monitoring
- **Persistent Storage**: Redis clusters + PostgreSQL for durability and analytics
- **Advanced Monitoring**: Distributed tracing, advanced metrics, alerting
- **Multi-Region**: Active-active deployment across regions
- **Dynamic Configuration**: Feature flags, A/B testing, runtime configuration

**Performance Profile:**
- **Scale**: 30M+ concurrent users, 100K+ events/second
- **Latency**: < 20ms read operations globally, < 100ms event processing
- **Memory**: Distributed across multiple nodes, persistent storage
- **Availability**: 99.99% uptime with automatic failover

**Operational Model:**
- **Deployment**: Kubernetes with auto-scaling, blue-green deployments
- **Monitoring**: Comprehensive observability with distributed tracing
- **Scaling**: Horizontal auto-scaling based on load
- **Recovery**: Automatic failover, data replication, backup/restore

#### **Key Architectural Changes for Production Scale**

**Data Layer:**
```yaml
# MVP (Current)
storage: in-memory
consistency: eventual (single process)
backup: none

# Production Scale
storage: 
  - Redis clusters (hot data)
  - PostgreSQL (persistent data)
  - S3 (analytics/archival)
consistency: eventual (cross-region)
backup: automated, multi-region replication
```

**Processing Layer:**
```yaml
# MVP (Current)
workers: 20x CPU cores
queue: in-memory channel
scoring: synchronous simulation

# Production Scale
workers: auto-scaling based on queue depth
queue: Kafka with partitioning
scoring: async ML service integration
```

**Deployment Model:**
```yaml
# MVP (Current)
deployment: single binary
scaling: vertical (more resources)
regions: single
availability: manual recovery

# Production Scale
deployment: Kubernetes microservices
scaling: horizontal auto-scaling
regions: active-active (Brazil, Germany, South Africa)
availability: automatic failover
```

**Monitoring & Observability:**
```yaml
# MVP (Current)
metrics: Prometheus
logging: structured logs
tracing: none
alerting: basic health checks

# Production Scale
metrics: Prometheus + Grafana + AlertManager
logging: centralized logging (ELK stack)
tracing: distributed tracing (Jaeger/Zipkin)
alerting: sophisticated alerting with runbooks
```

### Regional Considerations for Multi-Country Deployment

**Latency Optimization:**
- **Edge Caching**: CDN for static content, edge compute for read operations
- **Data Locality**: Regional data centers to minimize cross-region latency
- **Read Replicas**: Local read replicas for leaderboard queries

**Compliance & Data Sovereignty:**
- **Data Residency**: Ensure data stays within country boundaries
- **GDPR Compliance**: Data processing and retention policies
- **Local Regulations**: Compliance with Brazilian, German, and South African regulations

**Network & Infrastructure:**
- **Cross-Region Connectivity**: Reliable, low-latency connections between regions
- **Disaster Recovery**: Regional failover capabilities
- **Load Distribution**: Intelligent routing based on user location

This design evolution represents the natural progression from a focused MVP to a production-scale system, with each architectural decision optimized for the specific scale and requirements of the deployment phase.

### Monitoring
- **Metrics**: `http://localhost:9080/healthz` (Prometheus metrics)
- **Dashboard**: `http://localhost:9080/dashboard` (Web UI with real-time metrics)
- **Stats**: `http://localhost:9080/stats` (JSON service statistics)
- **API Docs**: `http://localhost:9080/api-docs` (Interactive Swagger UI)
