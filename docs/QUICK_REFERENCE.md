# CUJU Quick Reference Guide

## 🚀 Quick Start

```bash
# Build and run
go build -o cuju cmd/main.go
./cuju

# With custom config
export CUJU_CONFIG=./config.yaml
./cuju
```

## 📊 API Endpoints

| Endpoint | Method | Purpose | Time Complexity |
|----------|--------|---------|-----------------|
| `/events` | POST | Submit performance event | O(1) |
| `/leaderboard?limit=N` | GET | Get top N leaderboard | O(log n + N) |
| `/rank/{talent_id}` | GET | Get talent rank | O(log n) |
| `/healthz` | GET | Health check (Prometheus metrics) | O(1) |
| `/stats` | GET | Service statistics | O(1) |
| `/dashboard` | GET | Web monitoring dashboard | O(1) |
| `/api-docs` | GET | Interactive API documentation | O(1) |

## 🔧 Configuration

```yaml
# config.yaml
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

## 📝 Example Usage

### Submit Event
```bash
curl -X POST http://localhost:9080/events \
  -H "Content-Type: application/json" \
  -d '{
    "event_id": "123e4567-e89b-12d3-a456-426614174000",
    "talent_id": "t1",
    "raw_metric": 42.5,
    "skill": "dribble",
    "ts": "2025-01-27T10:00:00Z"
  }'
```

### Get Leaderboard
```bash
curl "http://localhost:9080/leaderboard?limit=10"
```

### Get Rank
```bash
curl "http://localhost:9080/rank/t1"
```

## 🏗️ Architecture Overview

```
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│   Client    │───▶│   HTTP API   │───▶│   Service   │
└─────────────┘    └──────────────┘    └─────────────┘
                                              │
                                              ▼
┌─────────────┐    ┌──────────────┐    ┌─────────────┐
│ Repository  │◀───│   Worker     │◀───│    Queue    │
│ (Single Treap)│   │    Pool      │    │             │
└─────────────┘    └──────────────┘    └─────────────┘
       │                   │
       ▼                   ▼
┌─────────────┐    ┌──────────────┐
│  Deduper    │    │   Scorer     │
│             │    │              │
└─────────────┘    └──────────────┘
```

## 📈 Performance Characteristics

| Operation | Time Complexity | Space Complexity | Notes |
|-----------|----------------|------------------|-------|
| Event Submission | O(1) | O(1) | Hash map + channel |
| Leaderboard Query | O(log n + N) | O(N) | In-order traversal |
| Rank Lookup | O(log n) | O(1) | In-order traversal |
| Update Best | O(log n) | O(1) | Treap operation |
| Deduplication | O(1) | O(M) | Hash map lookup |

Where:
- N = number of talents
- M = deduplication cache size

## 🎯 Key Features

- ✅ **Idempotent Processing** - Duplicate events are detected and ignored
- ✅ **Asynchronous Scoring** - Events processed in background workers
- ✅ **High-Performance Storage** - Optimized treap-based storage
- ✅ **Real-time Metrics** - Comprehensive Prometheus monitoring
- ✅ **High Performance** - Sub-40ms read latencies
- ✅ **Graceful Shutdown** - Clean resource cleanup

## 🔍 Monitoring

- **Health Check**: `GET /healthz` (Prometheus metrics)
- **Dashboard**: `GET /dashboard` (Web UI with real-time metrics)
- **Stats**: `GET /stats` (JSON statistics)
- **API Documentation**: `GET /api-docs` (Interactive Swagger UI)

## 🧪 Testing

```bash
# Unit tests
go test ./...

# Benchmarks
go test -bench=. ./internal/adapters/repository/

# Stress tests
go test -bench=BenchmarkTreapStore_30MTalents_ComprehensiveStressTest \
  -benchmem -run=^$ ./internal/adapters/repository/ \
  -timeout=30m -benchtime=15m
```

## 📚 Documentation

- **[Complete Architecture](ARCHITECTURE.md)** - Detailed system design
- **[Sequence Diagram](SEQUENCE_DIAGRAM.md)** - Event flow visualization
- **[OpenAPI Spec](openapi.yaml)** - API specification
- **[Data Structures](DATA_STRUCTURES.md)** - Performance testing and complexity analysis

## 🚨 Troubleshooting

### Common Issues

1. **Queue Full (503 Service Unavailable)**
   - Increase `queue_size` in config
   - Add more workers with `worker_count`

2. **High Memory Usage**
   - Reduce `dedupe_size` for deduplication cache
   - Monitor with `/stats` endpoint

3. **Slow Read Performance**
   - Optimize worker count for better throughput
   - Check metrics at `/metrics`

### Performance Tuning

- **High Write Load**: Increase `worker_count` and `queue_size`
- **High Read Load**: Optimize snapshot interval for better performance
- **Memory Constraints**: Reduce `dedupe_size` and `queue_size`

## 🔄 Development Workflow

```bash
# Make changes
git add .
git commit -m "feature: add new functionality"

# Run service
make run

# Test changes
go test ./...
```

## 📦 Dependencies

- **Go 1.24+**
- **Koanf** - Configuration management
- **Prometheus** - Metrics collection
- **Zap** - Structured logging
- **UUID** - Event ID generation

## 🎯 Design Principles

1. **Performance First** - Optimized for speed and low latency
2. **Simplicity** - Clean, maintainable code structure
3. **Observability** - Comprehensive monitoring and metrics
4. **Reliability** - Idempotent processing and error handling
5. **Scalability** - Optimized treap architecture for high performance
