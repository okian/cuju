package model_test

import (
	"testing"
	"time"

	model "github.com/okian/cuju/internal/domain/model"
	"github.com/smartystreets/goconvey/convey"
)

func TestEvent(t *testing.T) {
	convey.Convey("Given an Event struct", t, func() {
		convey.Convey("When creating a new event", func() {
			eventID := "event-123"
			talentID := "talent-456"
			rawMetric := 95.5
			skill := "programming"
			ts := time.Now()

			event := model.Event{
				EventID:   eventID,
				TalentID:  talentID,
				RawMetric: rawMetric,
				Skill:     skill,
				TS:        ts,
			}

			convey.Convey("Then it should have the correct values", func() {
				convey.So(event.EventID, convey.ShouldEqual, eventID)
				convey.So(event.TalentID, convey.ShouldEqual, talentID)
				convey.So(event.RawMetric, convey.ShouldEqual, rawMetric)
				convey.So(event.Skill, convey.ShouldEqual, skill)
				convey.So(event.TS, convey.ShouldEqual, ts)
			})
		})

		convey.Convey("When creating an event with zero values", func() {
			event := model.Event{}

			convey.Convey("Then it should have default values", func() {
				convey.So(event.EventID, convey.ShouldEqual, "")
				convey.So(event.TalentID, convey.ShouldEqual, "")
				convey.So(event.RawMetric, convey.ShouldEqual, 0.0)
				convey.So(event.Skill, convey.ShouldEqual, "")
				convey.So(event.TS, convey.ShouldEqual, time.Time{})
			})
		})

		convey.Convey("When creating an event with negative metric", func() {
			event := model.Event{
				EventID:   "event-neg",
				TalentID:  "talent-789",
				RawMetric: -10.5,
				Skill:     "testing",
				TS:        time.Now(),
			}

			convey.Convey("Then it should accept negative values", func() {
				convey.So(event.RawMetric, convey.ShouldEqual, -10.5)
			})
		})

		convey.Convey("When creating an event with very large metric", func() {
			event := model.Event{
				EventID:   "event-large",
				TalentID:  "talent-999",
				RawMetric: 999999.999,
				Skill:     "performance",
				TS:        time.Now(),
			}

			convey.Convey("Then it should accept large values", func() {
				convey.So(event.RawMetric, convey.ShouldEqual, 999999.999)
			})
		})

		convey.Convey("When creating an event with empty skill", func() {
			event := model.Event{
				EventID:   "event-empty-skill",
				TalentID:  "talent-111",
				RawMetric: 50.0,
				Skill:     "",
				TS:        time.Now(),
			}

			convey.Convey("Then it should accept empty skill", func() {
				convey.So(event.Skill, convey.ShouldEqual, "")
			})
		})

		convey.Convey("When creating an event with past timestamp", func() {
			pastTime := time.Now().Add(-24 * time.Hour)
			event := model.Event{
				EventID:   "event-past",
				TalentID:  "talent-222",
				RawMetric: 75.0,
				Skill:     "history",
				TS:        pastTime,
			}

			convey.Convey("Then it should accept past timestamps", func() {
				convey.So(event.TS, convey.ShouldEqual, pastTime)
			})
		})

		convey.Convey("When creating an event with future timestamp", func() {
			futureTime := time.Now().Add(24 * time.Hour)
			event := model.Event{
				EventID:   "event-future",
				TalentID:  "talent-333",
				RawMetric: 85.0,
				Skill:     "prediction",
				TS:        futureTime,
			}

			convey.Convey("Then it should accept future timestamps", func() {
				convey.So(event.TS, convey.ShouldEqual, futureTime)
			})
		})
	})
}

func TestTalentScore(t *testing.T) {
	convey.Convey("Given a TalentScore struct", t, func() {
		convey.Convey("When creating a new talent score", func() {
			talentID := "talent-123"
			score := 87.5

			talentScore := model.TalentScore{
				TalentID: talentID,
				Score:    score,
			}

			convey.Convey("Then it should have the correct values", func() {
				convey.So(talentScore.TalentID, convey.ShouldEqual, talentID)
				convey.So(talentScore.Score, convey.ShouldEqual, score)
			})
		})

		convey.Convey("When creating a talent score with zero values", func() {
			talentScore := model.TalentScore{}

			convey.Convey("Then it should have default values", func() {
				convey.So(talentScore.TalentID, convey.ShouldEqual, "")
				convey.So(talentScore.Score, convey.ShouldEqual, 0.0)
			})
		})

		convey.Convey("When creating a talent score with negative score", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-neg",
				Score:    -15.0,
			}

			convey.Convey("Then it should accept negative scores", func() {
				convey.So(talentScore.Score, convey.ShouldEqual, -15.0)
			})
		})

		convey.Convey("When creating a talent score with very high score", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-high",
				Score:    100.0,
			}

			convey.Convey("Then it should accept high scores", func() {
				convey.So(talentScore.Score, convey.ShouldEqual, 100.0)
			})
		})

		convey.Convey("When creating a talent score with decimal precision", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-precise",
				Score:    92.857,
			}

			convey.Convey("Then it should maintain decimal precision", func() {
				convey.So(talentScore.Score, convey.ShouldEqual, 92.857)
			})
		})
	})
}

func TestEventValidation(t *testing.T) {
	convey.Convey("Given event validation scenarios", t, func() {
		convey.Convey("When creating an event with valid data", func() {
			event := model.Event{
				EventID:   "valid-event-123",
				TalentID:  "valid-talent-456",
				RawMetric: 88.5,
				Skill:     "valid-skill",
				TS:        time.Now(),
			}

			convey.Convey("Then it should be a valid event", func() {
				convey.So(event.EventID, convey.ShouldNotBeEmpty)
				convey.So(event.TalentID, convey.ShouldNotBeEmpty)
				convey.So(event.Skill, convey.ShouldNotBeEmpty)
				convey.So(event.TS, convey.ShouldNotBeZeroValue)
			})
		})

		convey.Convey("When creating an event with minimal data", func() {
			event := model.Event{
				EventID:  "minimal",
				TalentID: "minimal",
			}

			convey.Convey("Then it should have minimal required fields", func() {
				convey.So(event.EventID, convey.ShouldNotBeEmpty)
				convey.So(event.TalentID, convey.ShouldNotBeEmpty)
				convey.So(event.RawMetric, convey.ShouldEqual, 0.0)
				convey.So(event.Skill, convey.ShouldEqual, "")
				convey.So(event.TS, convey.ShouldEqual, time.Time{})
			})
		})

		convey.Convey("When creating multiple events", func() {
			events := []model.Event{
				{
					EventID:   "event-1",
					TalentID:  "talent-1",
					RawMetric: 90.0,
					Skill:     "skill-1",
					TS:        time.Now(),
				},
				{
					EventID:   "event-2",
					TalentID:  "talent-2",
					RawMetric: 85.0,
					Skill:     "skill-2",
					TS:        time.Now().Add(time.Minute),
				},
				{
					EventID:   "event-3",
					TalentID:  "talent-3",
					RawMetric: 95.0,
					Skill:     "skill-3",
					TS:        time.Now().Add(2 * time.Minute),
				},
			}

			convey.Convey("Then all events should be valid", func() {
				for _, event := range events {
					convey.So(event.EventID, convey.ShouldNotBeEmpty)
					convey.So(event.TalentID, convey.ShouldNotBeEmpty)
					convey.So(event.Skill, convey.ShouldNotBeEmpty)
					convey.So(event.TS, convey.ShouldNotBeZeroValue)
				}
			})
		})
	})
}

func TestTalentScoreValidation(t *testing.T) {
	convey.Convey("Given talent score validation scenarios", t, func() {
		convey.Convey("When creating a talent score with valid data", func() {
			talentScore := model.TalentScore{
				TalentID: "valid-talent-123",
				Score:    89.5,
			}

			convey.Convey("Then it should be a valid talent score", func() {
				convey.So(talentScore.TalentID, convey.ShouldNotBeEmpty)
				convey.So(talentScore.Score, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating multiple talent scores", func() {
			talentScores := []model.TalentScore{
				{TalentID: "talent-1", Score: 90.0},
				{TalentID: "talent-2", Score: 85.5},
				{TalentID: "talent-3", Score: 92.0},
				{TalentID: "talent-4", Score: 78.5},
				{TalentID: "talent-5", Score: 95.0},
			}

			convey.Convey("Then all talent scores should be valid", func() {
				for _, ts := range talentScores {
					convey.So(ts.TalentID, convey.ShouldNotBeEmpty)
					convey.So(ts.Score, convey.ShouldNotBeNil)
				}
			})

			convey.Convey("And scores should be in expected range", func() {
				for _, ts := range talentScores {
					convey.So(ts.Score, convey.ShouldBeGreaterThanOrEqualTo, 0.0)
					convey.So(ts.Score, convey.ShouldBeLessThanOrEqualTo, 100.0)
				}
			})
		})
	})
}

func TestModelEdgeCases(t *testing.T) {
	convey.Convey("Given model edge cases", t, func() {
		convey.Convey("When creating an event with very long IDs", func() {
			longEventID := "event-" + string(make([]byte, 1000))
			longTalentID := "talent-" + string(make([]byte, 1000))
			longSkill := "skill-" + string(make([]byte, 1000))

			event := model.Event{
				EventID:   longEventID,
				TalentID:  longTalentID,
				RawMetric: 50.0,
				Skill:     longSkill,
				TS:        time.Now(),
			}

			convey.Convey("Then it should handle long strings", func() {
				convey.So(len(event.EventID), convey.ShouldBeGreaterThan, 1000)
				convey.So(len(event.TalentID), convey.ShouldBeGreaterThan, 1000)
				convey.So(len(event.Skill), convey.ShouldBeGreaterThan, 1000)
			})
		})

		convey.Convey("When creating an event with special characters", func() {
			event := model.Event{
				EventID:   "event-!@#$%^&*()",
				TalentID:  "talent-áéíóúñ",
				RawMetric: 75.5,
				Skill:     "skill-🚀🎯💻",
				TS:        time.Now(),
			}

			convey.Convey("Then it should handle special characters", func() {
				convey.So(event.EventID, convey.ShouldContainSubstring, "!@#$%^&*()")
				convey.So(event.TalentID, convey.ShouldContainSubstring, "áéíóúñ")
				convey.So(event.Skill, convey.ShouldContainSubstring, "🚀🎯💻")
			})
		})

		convey.Convey("When creating an event with extreme metric values", func() {
			event := model.Event{
				EventID:   "event-extreme",
				TalentID:  "talent-extreme",
				RawMetric: 1e308, // Very large number
				Skill:     "extreme-skill",
				TS:        time.Now(),
			}

			convey.Convey("Then it should handle extreme values", func() {
				convey.So(event.RawMetric, convey.ShouldEqual, 1e308)
			})
		})

		convey.Convey("When creating a talent score with extreme values", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-extreme",
				Score:    1e308,
			}

			convey.Convey("Then it should handle extreme values", func() {
				convey.So(talentScore.Score, convey.ShouldEqual, 1e308)
			})
		})
	})
}
