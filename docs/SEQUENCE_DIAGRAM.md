# CUJU Event Flow Sequence Diagram

## Complete Event Processing Flow

```mermaid
sequenceDiagram
    participant Client
    participant HTTP API
    participant Service
    participant Deduper
    participant Queue
    participant Worker
    participant Scorer
    participant Repository
    participant Metrics

    Note over Client,Metrics: Event Submission Flow

    Client->>HTTP API: POST /events<br/>{event_id, talent_id, raw_metric, skill, ts}
    HTTP API->>HTTP API: Validate Request<br/>(required fields, RFC3339 timestamp)
    
    alt Validation Fails
        HTTP API-->>Client: 400 Bad Request
    else Validation Passes
        HTTP API->>Service: Enqueue Event
        Service->>Deduper: SeenAndRecord(event_id)
        
        alt Event Already Seen (Duplicate)
            Deduper-->>Service: true (duplicate)
            Service-->>HTTP API: 200 OK (duplicate)
            HTTP API-->>Client: 200 OK<br/>{"status": "duplicate", "duplicate": true}
            Metrics->>Metrics: Record Duplicate Event
        else New Event
            Deduper-->>Service: false (new)
            Service->>Queue: Enqueue Event
            
            alt Queue Full (Backpressure)
                Queue-->>Service: Failed
                Service-->>HTTP API: 429 Too Many Requests
                HTTP API-->>Client: 429 Too Many Requests<br/>{"code": "backpressure", "message": "backpressure"}
            else Queue Success
                Queue-->>Service: Success
                Service-->>HTTP API: 202 Accepted
                HTTP API-->>Client: 202 Accepted<br/>{"status": "accepted", "duplicate": false}
                Metrics->>Metrics: Record Event Processed
            end
            
            Note over Queue,Worker: Asynchronous Background Processing
            
            Queue->>Worker: Dequeue Event
            Worker->>Scorer: Score Event<br/>(talent_id, raw_metric, skill)
            Note over Scorer: Simulated ML Latency<br/>(80-150ms random delay)
            Scorer-->>Worker: Score Result<br/>(score = raw_metric × weight)
            Worker->>Repository: UpdateBestWithMeta<br/>(talent_id, score, event_id, skill, raw_metric)
            
            alt Score Improvement
                Repository-->>Worker: true (updated)
                Worker->>Metrics: Record Leaderboard Update
            else No Improvement
                Repository-->>Worker: false (not updated)
            end
            
            Worker->>Metrics: Record Processing Complete
        end
    end

    Note over Client,Repository: Read Operations Flow

    Client->>HTTP API: GET /leaderboard?limit=N
    HTTP API->>Service: TopN(N)
    Service->>Repository: TopN(N)
    
    Note over Repository: Treap Top-N Query<br/>O(log n + N)
    
    Repository-->>Service: Leaderboard Entries<br/>[{rank, talent_id, score}, ...]
    Service-->>HTTP API: Entries
    HTTP API-->>Client: 200 OK<br/>JSON Array with Rankings
    
    Client->>HTTP API: GET /rank/{talent_id}
    HTTP API->>Service: Rank(talent_id)
    Service->>Repository: Rank(talent_id)
    
    alt Talent Found
        Repository-->>Service: Rank Entry<br/>{rank, talent_id, score}
        Service-->>HTTP API: Entry
        HTTP API-->>Client: 200 OK<br/>JSON with Rank and Score
    else Talent Not Found
        Repository-->>Service: Error (not found)
        Service-->>HTTP API: Error
        HTTP API-->>Client: 404 Not Found
    end

    Note over Client,Metrics: Health and Monitoring

    Client->>HTTP API: GET /healthz
    HTTP API-->>Client: 200 OK<br/>Prometheus Metrics

    Client->>HTTP API: GET /stats
    HTTP API->>Service: GetStats()
    Service-->>HTTP API: Service Statistics
    HTTP API-->>Client: 200 OK<br/>JSON with Metrics

    Client->>HTTP API: GET /dashboard
    HTTP API-->>Client: 200 OK<br/>HTML Dashboard with Real-time Metrics

    Client->>HTTP API: GET /api-docs
    HTTP API-->>Client: 200 OK<br/>HTML with Interactive Swagger UI
```

## Key Components Interaction

### 1. Event Processing Pipeline
1. **Validation**: HTTP API validates request format and required fields
2. **Deduplication**: Service checks if event_id was already processed
3. **Queuing**: New events are enqueued for asynchronous processing
4. **Scoring**: Workers process events with simulated ML latency
5. **Storage**: Repository updates leaderboard with improved scores
6. **Metrics**: All operations are tracked for monitoring

### 2. Read Operations
1. **Leaderboard**: Treap-based top-N query with in-order traversal
2. **Rank Lookup**: Single treap rank calculation
3. **Caching**: Periodic snapshots provide fast access to top entries

### 3. Error Handling
- **Validation Errors**: 400 Bad Request for malformed requests
- **Duplicate Events**: 200 OK with duplicate flag
- **Not Found**: 404 for non-existent talents
- **Backpressure**: 429 Too Many Requests when queue is at capacity

### 4. Performance Characteristics
- **Event Submission**: O(1) - Hash map lookup + channel send
- **Leaderboard Query**: O(log n + N) - Treap in-order traversal
- **Rank Lookup**: O(log n) - Single treap rank calculation
- **Health Check**: O(1) - Simple status check

## Data Flow Summary

```
Client Request → HTTP API → Service → Components → Response
     ↓              ↓         ↓          ↓           ↓
  JSON Event → Validation → Dedupe → Queue → 202 Accepted
     ↓              ↓         ↓          ↓           ↓
  Read Query → Service → Repository → Treap → JSON Response
     ↓              ↓         ↓          ↓           ↓
  Health Check → Service → Status → Metrics → 200 OK
```

This sequence diagram illustrates the complete flow from user request to leaderboard storage, showing all the key components, their interactions, and the asynchronous nature of event processing.
