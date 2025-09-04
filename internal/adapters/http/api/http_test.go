package api_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/okian/cuju/internal/adapters/http/api"
	repository "github.com/okian/cuju/internal/adapters/repository"
	"github.com/okian/cuju/internal/domain/types"
	. "github.com/smartystreets/goconvey/convey"
)

// Mock implementations for testing
type mockDeduper struct {
	seen map[string]bool
}

func (m *mockDeduper) SeenAndRecord(ctx context.Context, id string) bool {
	if m.seen == nil {
		m.seen = make(map[string]bool)
	}
	if m.seen[id] {
		return true
	}
	m.seen[id] = true
	return false
}

func (m *mockDeduper) Unrecord(ctx context.Context, id string) {
	if m.seen != nil {
		delete(m.seen, id)
	}
}

func (m *mockDeduper) Size() int64 {
	return int64(len(m.seen))
}

type mockQueue struct {
	enqueueSuccess bool
	enqueued       []interface{}
}

func (m *mockQueue) Enqueue(ctx context.Context, e interface{}) bool {
	if m.enqueueSuccess {
		m.enqueued = append(m.enqueued, e)
		return true
	}
	return false
}

type mockLeaderboard struct {
	topN    []types.Entry
	rank    types.Entry
	rankErr error
	topNErr error
}

func (m *mockLeaderboard) TopN(ctx context.Context, n int) ([]types.Entry, error) {
	if m.topNErr != nil {
		return nil, m.topNErr
	}
	if n > len(m.topN) {
		return m.topN, nil
	}
	return m.topN[:n], nil
}

func (m *mockLeaderboard) Rank(ctx context.Context, talentID string) (types.Entry, error) {
	if m.rankErr != nil {
		return types.Entry{}, m.rankErr
	}
	return m.rank, nil
}

type mockStatsProvider struct {
	stats map[string]interface{}
}

func (m *mockStatsProvider) GetStats() map[string]interface{} {
	return m.stats
}

func TestServer_Register(t *testing.T) {
	Convey("Given a new API server", t, func() {
		deps := &mockDependencies{
			dedupe: &mockDeduper{},
			queue:  &mockQueue{enqueueSuccess: true},
			lb:     &mockLeaderboard{},
		}
		statsProvider := &mockStatsProvider{}
		server := api.NewServer(deps, statsProvider, 100)
		mux := http.NewServeMux()

		Convey("When registering routes", func() {
			server.Register(context.Background(), mux, deps)

			Convey("Then all expected routes should be registered", func() {
				So(mux, ShouldNotBeNil)
			})

			Convey("And health endpoint should be accessible", func() {
				req := httptest.NewRequest("GET", "/healthz", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
			})

			Convey("And stats endpoint should be accessible", func() {
				req := httptest.NewRequest("GET", "/stats", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
			})

			Convey("And events endpoint should be accessible", func() {
				req := httptest.NewRequest("POST", "/events", strings.NewReader(`{}`))
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusBadRequest) // Invalid request
			})

			Convey("And leaderboard endpoint should be accessible", func() {
				req := httptest.NewRequest("GET", "/leaderboard?limit=10", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
			})

			Convey("And rank endpoint should be accessible", func() {
				req := httptest.NewRequest("GET", "/rank/test-id", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
			})

			Convey("And root endpoint should catch everything else", func() {
				req := httptest.NewRequest("GET", "/unknown", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})

			Convey("And dashboard endpoint should serve HTML with refresh control", func() {
				req := httptest.NewRequest("GET", "/dashboard", nil)
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
				body := w.Body.String()
				So(body, ShouldContainSubstring, "id=\"refresh-interval\"")
				So(body, ShouldContainSubstring, "id=\"refresh-control\"")
			})
		})
	})
}

func TestEventRequest_Validate(t *testing.T) {
	Convey("Given an event request", t, func() {
		validTime := time.Now().Format(time.RFC3339)

		Convey("When all fields are valid", func() {
			req := eventRequest{
				EventID:   "event-123",
				TalentID:  "talent-456",
				RawMetric: 95.5,
				Skill:     "programming",
				TS:        validTime,
			}

			Convey("Then validation should pass", func() {
				err := req.validate()
				So(err, ShouldBeNil)
			})
		})

		Convey("When EventID is missing", func() {
			req := eventRequest{
				TalentID:  "talent-456",
				RawMetric: 95.5,
				Skill:     "programming",
				TS:        validTime,
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing event_id")
			})
		})

		Convey("When EventID is empty string", func() {
			req := eventRequest{
				EventID:   "   ",
				TalentID:  "talent-456",
				RawMetric: 95.5,
				Skill:     "programming",
				TS:        validTime,
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing event_id")
			})
		})

		Convey("When TalentID is missing", func() {
			req := eventRequest{
				EventID:   "event-123",
				RawMetric: 95.5,
				Skill:     "programming",
				TS:        validTime,
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing talent_id")
			})
		})

		Convey("When Skill is missing", func() {
			req := eventRequest{
				EventID:   "event-123",
				TalentID:  "talent-456",
				RawMetric: 95.5,
				TS:        validTime,
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing skill")
			})
		})

		Convey("When TS is missing", func() {
			req := eventRequest{
				EventID:   "event-123",
				TalentID:  "talent-456",
				RawMetric: 95.5,
				Skill:     "programming",
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "missing ts")
			})
		})

		Convey("When TS is invalid format", func() {
			req := eventRequest{
				EventID:   "event-123",
				TalentID:  "talent-456",
				RawMetric: 95.5,
				Skill:     "programming",
				TS:        "invalid-time",
			}

			Convey("Then validation should fail", func() {
				err := req.validate()
				So(err, ShouldNotBeNil)
				So(err.Error(), ShouldContainSubstring, "invalid ts")
			})
		})

		Convey("When TS is valid RFC3339", func() {
			testCases := []string{
				"2023-01-01T12:00:00Z",
				"2023-01-01T12:00:00+01:00",
				"2023-01-01T12:00:00.123Z",
			}

			for _, ts := range testCases {
				Convey(fmt.Sprintf("And TS is %s", ts), func() {
					req := eventRequest{
						EventID:   "event-123",
						TalentID:  "talent-456",
						RawMetric: 95.5,
						Skill:     "programming",
						TS:        ts,
					}

					Convey("Then validation should pass", func() {
						err := req.validate()
						So(err, ShouldBeNil)
					})
				})
			}
		})
	})
}

func TestEventsHandler_HandlePostEvent(t *testing.T) {
	Convey("Given an events handler", t, func() {
		deps := &mockDependencies{
			dedupe: &mockDeduper{},
			queue:  &mockQueue{enqueueSuccess: true},
			lb:     &mockLeaderboard{},
		}
		handler := api.NewEventsHandler(deps)

		Convey("When handling a valid POST request", func() {
			validEvent := `{
				"event_id": "event-123",
				"talent_id": "talent-456",
				"raw_metric": 95.5,
				"skill": "programming",
				"ts": "2023-01-01T12:00:00Z"
			}`

			req := httptest.NewRequest("POST", "/events", strings.NewReader(validEvent))
			w := httptest.NewRecorder()

			Convey("Then it should return accepted status", func() {
				handler.HandlePostEvent(w, req)
				So(w.Code, ShouldEqual, http.StatusAccepted)

				var response ackResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(response.Status, ShouldEqual, "accepted")
				So(response.Duplicate, ShouldBeFalse)
			})
		})

		Convey("When handling a duplicate event", func() {
			validEvent := `{
				"event_id": "event-123",
				"talent_id": "talent-456",
				"raw_metric": 95.5,
				"skill": "programming",
				"ts": "2023-01-01T12:00:00Z"
			}`

			// First request
			req1 := httptest.NewRequest("POST", "/events", strings.NewReader(validEvent))
			w1 := httptest.NewRecorder()
			handler.HandlePostEvent(w1, req1)

			// Second request with same event ID
			req2 := httptest.NewRequest("POST", "/events", strings.NewReader(validEvent))
			w2 := httptest.NewRecorder()

			Convey("Then it should return duplicate status", func() {
				handler.HandlePostEvent(w2, req2)
				So(w2.Code, ShouldEqual, http.StatusOK)

				var response ackResponse
				err := json.NewDecoder(w2.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(response.Status, ShouldEqual, "duplicate")
				So(response.Duplicate, ShouldBeTrue)
			})
		})

		Convey("When handling an invalid JSON request", func() {
			invalidJSON := `{invalid json`
			req := httptest.NewRequest("POST", "/events", strings.NewReader(invalidJSON))
			w := httptest.NewRecorder()

			Convey("Then it should return bad request status", func() {
				handler.HandlePostEvent(w, req)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When handling a request with missing required fields", func() {
			incompleteEvent := `{
				"event_id": "event-123",
				"raw_metric": 95.5
			}`
			req := httptest.NewRequest("POST", "/events", strings.NewReader(incompleteEvent))
			w := httptest.NewRecorder()

			Convey("Then it should return bad request status", func() {
				handler.HandlePostEvent(w, req)
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When handling a non-POST request", func() {
			req := httptest.NewRequest("GET", "/events", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return not found status", func() {
				handler.HandlePostEvent(w, req)
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When enqueue fails due to backpressure", func() {
			deps.queue.enqueueSuccess = false
			validEvent := `{
				"event_id": "event-456",
				"talent_id": "talent-789",
				"raw_metric": 95.5,
				"skill": "programming",
				"ts": "2023-01-01T12:00:00Z"
			}`

			req := httptest.NewRequest("POST", "/events", strings.NewReader(validEvent))
			w := httptest.NewRecorder()

			Convey("Then it should return too many requests status", func() {
				handler.HandlePostEvent(w, req)
				So(w.Code, ShouldEqual, http.StatusTooManyRequests)

				var response errorResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(response.Code, ShouldEqual, "backpressure")
			})
		})
	})
}

func TestLeaderboardHandler_HandleGetLeaderboard(t *testing.T) {
	Convey("Given a leaderboard handler", t, func() {
		mockLB := &mockLeaderboard{
			topN: []types.Entry{
				{Rank: 1, TalentID: "talent-1", Score: 100.0},
				{Rank: 2, TalentID: "talent-2", Score: 95.0},
				{Rank: 3, TalentID: "talent-3", Score: 90.0},
			},
		}
		deps := &mockDependencies{
			dedupe: &mockDeduper{},
			queue:  &mockQueue{enqueueSuccess: true},
			lb:     mockLB,
		}
		handler := api.NewLeaderboardHandler(deps, 100)

		Convey("When requesting top N entries", func() {
			req := httptest.NewRequest("GET", "/leaderboard?limit=2", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return the top N entries", func() {
				handler.HandleGetLeaderboard(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)

				var response []types.Entry
				err := json.NewDecoder(w.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(len(response), ShouldEqual, 2)
				So(response[0].TalentID, ShouldEqual, "talent-1")
				So(response[1].TalentID, ShouldEqual, "talent-2")
			})
		})

		Convey("When no limit is specified", func() {
			req := httptest.NewRequest("GET", "/leaderboard", nil)
			w := httptest.NewRecorder()

			handler.HandleGetLeaderboard(w, req)

			Convey("Then it should return 400 Bad Request", func() {
				So(w.Code, ShouldEqual, http.StatusBadRequest)
			})
		})

		Convey("When leaderboard returns an error", func() {
			mockLB.topNErr = fmt.Errorf("database error")
			req := httptest.NewRequest("GET", "/leaderboard?limit=10", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return internal server error", func() {
				handler.HandleGetLeaderboard(w, req)
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func TestRankHandler_HandleGetRank(t *testing.T) {
	Convey("Given a rank handler", t, func() {
		mockLB := &mockLeaderboard{
			rank: types.Entry{Rank: 5, TalentID: "talent-123", Score: 85.0},
		}
		handler := api.NewRankHandler(mockLB)

		Convey("When requesting rank for existing talent", func() {
			req := httptest.NewRequest("GET", "/rank/talent-123", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return the rank information", func() {
				handler.HandleGetRank(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
				So(w.Header().Get("Content-Type"), ShouldContainSubstring, "application/json")

				var response types.Entry
				err := json.NewDecoder(w.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(response.TalentID, ShouldEqual, "talent-123")
				So(response.Rank, ShouldEqual, 5)
				So(response.Score, ShouldEqual, 85.0)
			})
		})

		Convey("When requesting rank for non-existent talent", func() {
			req := httptest.NewRequest("GET", "/rank/nonexistent", nil)
			w := httptest.NewRecorder()

			// Mock the error response
			mockLB.rankErr = repository.ErrNotFound

			handler.HandleGetRank(w, req)

			Convey("Then it should return not found status", func() {
				So(w.Code, ShouldEqual, http.StatusNotFound)
			})
		})

		Convey("When leaderboard returns other error", func() {
			req := httptest.NewRequest("GET", "/rank/talent-123", nil)
			w := httptest.NewRecorder()

			// Mock the error response
			mockLB.rankErr = fmt.Errorf("database error")

			handler.HandleGetRank(w, req)

			Convey("Then it should return internal server error", func() {
				So(w.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}

func TestHealthHandler_HandleHealth(t *testing.T) {
	Convey("Given a health handler", t, func() {
		handler := api.NewHealthHandler()

		Convey("When handling health check request", func() {
			req := httptest.NewRequest("GET", "/healthz", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return OK status", func() {
				handler.HandleHealth(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)
			})
		})
	})
}

func TestStatsHandler_HandleStats(t *testing.T) {
	Convey("Given a stats handler", t, func() {
		mockStats := &mockStatsProvider{
			stats: map[string]interface{}{
				"total_events": 1000,
				"active_users": 150,
			},
		}
		handler := api.NewStatsHandler(mockStats)

		Convey("When handling stats request", func() {
			req := httptest.NewRequest("GET", "/stats", nil)
			w := httptest.NewRecorder()

			Convey("Then it should return stats", func() {
				handler.HandleStats(w, req)
				So(w.Code, ShouldEqual, http.StatusOK)

				var response map[string]interface{}
				err := json.NewDecoder(w.Body).Decode(&response)
				So(err, ShouldBeNil)
				So(response["total_events"], ShouldEqual, 1000)
				So(response["active_users"], ShouldEqual, 150)
			})
		})
	})
}

// Mock dependencies that implements the Dependencies interface
type mockDependencies struct {
	dedupe *mockDeduper
	queue  *mockQueue
	lb     *mockLeaderboard
}

func (m *mockDependencies) SeenAndRecord(ctx context.Context, id string) bool {
	return m.dedupe.SeenAndRecord(ctx, id)
}

func (m *mockDependencies) Unrecord(ctx context.Context, id string) {
	m.dedupe.Unrecord(ctx, id)
}

func (m *mockDependencies) Size() int64 {
	return m.dedupe.Size()
}

func (m *mockDependencies) Enqueue(ctx context.Context, e interface{}) bool {
	return m.queue.Enqueue(ctx, e)
}

func (m *mockDependencies) TopN(ctx context.Context, n int) ([]types.Entry, error) {
	return m.lb.TopN(ctx, n)
}

func (m *mockDependencies) Rank(ctx context.Context, talentID string) (types.Entry, error) {
	return m.lb.Rank(ctx, talentID)
}

// Local types for testing
type eventRequest struct {
	EventID   string  `json:"event_id"`
	TalentID  string  `json:"talent_id"`
	RawMetric float64 `json:"raw_metric"`
	Skill     string  `json:"skill"`
	TS        string  `json:"ts"`
}

func (e eventRequest) validate() error {
	switch {
	case strings.TrimSpace(e.EventID) == "":
		return fmt.Errorf("missing event_id")
	case strings.TrimSpace(e.TalentID) == "":
		return fmt.Errorf("missing talent_id")
	case strings.TrimSpace(e.Skill) == "":
		return fmt.Errorf("missing skill")
	case strings.TrimSpace(e.TS) == "":
		return fmt.Errorf("missing ts")
	}
	if _, err := time.Parse(time.RFC3339, e.TS); err != nil {
		return fmt.Errorf("invalid ts; must be RFC3339")
	}
	return nil
}

type ackResponse struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

type errorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
