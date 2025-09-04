# CUJU Senior GO Engineer — System Design + Coding Task

### Coding Task

A Leaderboard Ingest + Query service with idempotent event intake, concurrent processing, and fast reads—but all in one process with in-memory storage (no external deps).

Build a single Go service that accepts performance events, deduplicates by `event_id`, asynchronously “scores” them (simulate 80–150ms latency), and maintains an in-memory Global leaderboard (best score per talent).

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

* Returns the caller’s current rank and score.

---

### Non-functional checks

* Safe concurrent updates (multiple POST /events in parallel).
* Basic performance: **p95 read < 40ms locally** for `GET /leaderboard?limit=50` after a burst of writes.
* Minimal observability: `/healthz` + counters (processed events, dedup hits). (Prometheus optional.)

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

One short page (could be in README) covering:

* **Sequence Diagram (event flow):** From User request to create an event to Leaderboard store.
* **Data model:**

  * `scores[talent_id] = bestScore`
  * ranking structure
* **Trade-offs:** Why in-mem + snapshot omitted, eventual vs. strong consistency.
* **Thoughts:** Design differences between MVP for a few hundred users (closed Beta) vs. a service catering to **30M AUs** in multiple countries (Brazil, Germany, South Africa).
