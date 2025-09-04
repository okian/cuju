package model_test

import (
	"testing"
	"time"

	model "github.com/okian/cuju/internal/domain/model"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEvent(t *testing.T) {
	Convey("Given an Event struct", t, func() {
		Convey("When creating a new event", func() {
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

			Convey("Then it should have the correct values", func() {
				So(event.EventID, ShouldEqual, eventID)
				So(event.TalentID, ShouldEqual, talentID)
				So(event.RawMetric, ShouldEqual, rawMetric)
				So(event.Skill, ShouldEqual, skill)
				So(event.TS, ShouldEqual, ts)
			})
		})

		Convey("When creating an event with zero values", func() {
			event := model.Event{}

			Convey("Then it should have default values", func() {
				So(event.EventID, ShouldEqual, "")
				So(event.TalentID, ShouldEqual, "")
				So(event.RawMetric, ShouldEqual, 0.0)
				So(event.Skill, ShouldEqual, "")
				So(event.TS, ShouldEqual, time.Time{})
			})
		})

		Convey("When creating an event with negative metric", func() {
			event := model.Event{
				EventID:   "event-neg",
				TalentID:  "talent-789",
				RawMetric: -10.5,
				Skill:     "testing",
				TS:        time.Now(),
			}

			Convey("Then it should accept negative values", func() {
				So(event.RawMetric, ShouldEqual, -10.5)
			})
		})

		Convey("When creating an event with very large metric", func() {
			event := model.Event{
				EventID:   "event-large",
				TalentID:  "talent-999",
				RawMetric: 999999.999,
				Skill:     "performance",
				TS:        time.Now(),
			}

			Convey("Then it should accept large values", func() {
				So(event.RawMetric, ShouldEqual, 999999.999)
			})
		})

		Convey("When creating an event with empty skill", func() {
			event := model.Event{
				EventID:   "event-empty-skill",
				TalentID:  "talent-111",
				RawMetric: 50.0,
				Skill:     "",
				TS:        time.Now(),
			}

			Convey("Then it should accept empty skill", func() {
				So(event.Skill, ShouldEqual, "")
			})
		})

		Convey("When creating an event with past timestamp", func() {
			pastTime := time.Now().Add(-24 * time.Hour)
			event := model.Event{
				EventID:   "event-past",
				TalentID:  "talent-222",
				RawMetric: 75.0,
				Skill:     "history",
				TS:        pastTime,
			}

			Convey("Then it should accept past timestamps", func() {
				So(event.TS, ShouldEqual, pastTime)
			})
		})

		Convey("When creating an event with future timestamp", func() {
			futureTime := time.Now().Add(24 * time.Hour)
			event := model.Event{
				EventID:   "event-future",
				TalentID:  "talent-333",
				RawMetric: 85.0,
				Skill:     "prediction",
				TS:        futureTime,
			}

			Convey("Then it should accept future timestamps", func() {
				So(event.TS, ShouldEqual, futureTime)
			})
		})
	})
}

func TestTalentScore(t *testing.T) {
	Convey("Given a TalentScore struct", t, func() {
		Convey("When creating a new talent score", func() {
			talentID := "talent-123"
			score := 87.5

			talentScore := model.TalentScore{
				TalentID: talentID,
				Score:    score,
			}

			Convey("Then it should have the correct values", func() {
				So(talentScore.TalentID, ShouldEqual, talentID)
				So(talentScore.Score, ShouldEqual, score)
			})
		})

		Convey("When creating a talent score with zero values", func() {
			talentScore := model.TalentScore{}

			Convey("Then it should have default values", func() {
				So(talentScore.TalentID, ShouldEqual, "")
				So(talentScore.Score, ShouldEqual, 0.0)
			})
		})

		Convey("When creating a talent score with negative score", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-neg",
				Score:    -15.0,
			}

			Convey("Then it should accept negative scores", func() {
				So(talentScore.Score, ShouldEqual, -15.0)
			})
		})

		Convey("When creating a talent score with very high score", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-high",
				Score:    100.0,
			}

			Convey("Then it should accept high scores", func() {
				So(talentScore.Score, ShouldEqual, 100.0)
			})
		})

		Convey("When creating a talent score with decimal precision", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-precise",
				Score:    92.857,
			}

			Convey("Then it should maintain decimal precision", func() {
				So(talentScore.Score, ShouldEqual, 92.857)
			})
		})
	})
}

func TestEventValidation(t *testing.T) {
	Convey("Given event validation scenarios", t, func() {
		Convey("When creating an event with valid data", func() {
			event := model.Event{
				EventID:   "valid-event-123",
				TalentID:  "valid-talent-456",
				RawMetric: 88.5,
				Skill:     "valid-skill",
				TS:        time.Now(),
			}

			Convey("Then it should be a valid event", func() {
				So(event.EventID, ShouldNotBeEmpty)
				So(event.TalentID, ShouldNotBeEmpty)
				So(event.Skill, ShouldNotBeEmpty)
				So(event.TS, ShouldNotBeZeroValue)
			})
		})

		Convey("When creating an event with minimal data", func() {
			event := model.Event{
				EventID:  "minimal",
				TalentID: "minimal",
			}

			Convey("Then it should have minimal required fields", func() {
				So(event.EventID, ShouldNotBeEmpty)
				So(event.TalentID, ShouldNotBeEmpty)
				So(event.RawMetric, ShouldEqual, 0.0)
				So(event.Skill, ShouldEqual, "")
				So(event.TS, ShouldEqual, time.Time{})
			})
		})

		Convey("When creating multiple events", func() {
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

			Convey("Then all events should be valid", func() {
				for _, event := range events {
					So(event.EventID, ShouldNotBeEmpty)
					So(event.TalentID, ShouldNotBeEmpty)
					So(event.Skill, ShouldNotBeEmpty)
					So(event.TS, ShouldNotBeZeroValue)
				}
			})
		})
	})
}

func TestTalentScoreValidation(t *testing.T) {
	Convey("Given talent score validation scenarios", t, func() {
		Convey("When creating a talent score with valid data", func() {
			talentScore := model.TalentScore{
				TalentID: "valid-talent-123",
				Score:    89.5,
			}

			Convey("Then it should be a valid talent score", func() {
				So(talentScore.TalentID, ShouldNotBeEmpty)
				So(talentScore.Score, ShouldNotBeNil)
			})
		})

		Convey("When creating multiple talent scores", func() {
			talentScores := []model.TalentScore{
				{TalentID: "talent-1", Score: 90.0},
				{TalentID: "talent-2", Score: 85.5},
				{TalentID: "talent-3", Score: 92.0},
				{TalentID: "talent-4", Score: 78.5},
				{TalentID: "talent-5", Score: 95.0},
			}

			Convey("Then all talent scores should be valid", func() {
				for _, ts := range talentScores {
					So(ts.TalentID, ShouldNotBeEmpty)
					So(ts.Score, ShouldNotBeNil)
				}
			})

			Convey("And scores should be in expected range", func() {
				for _, ts := range talentScores {
					So(ts.Score, ShouldBeGreaterThanOrEqualTo, 0.0)
					So(ts.Score, ShouldBeLessThanOrEqualTo, 100.0)
				}
			})
		})
	})
}

func TestModelEdgeCases(t *testing.T) {
	Convey("Given model edge cases", t, func() {
		Convey("When creating an event with very long IDs", func() {
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

			Convey("Then it should handle long strings", func() {
				So(len(event.EventID), ShouldBeGreaterThan, 1000)
				So(len(event.TalentID), ShouldBeGreaterThan, 1000)
				So(len(event.Skill), ShouldBeGreaterThan, 1000)
			})
		})

		Convey("When creating an event with special characters", func() {
			event := model.Event{
				EventID:   "event-!@#$%^&*()",
				TalentID:  "talent-áéíóúñ",
				RawMetric: 75.5,
				Skill:     "skill-🚀🎯💻",
				TS:        time.Now(),
			}

			Convey("Then it should handle special characters", func() {
				So(event.EventID, ShouldContainSubstring, "!@#$%^&*()")
				So(event.TalentID, ShouldContainSubstring, "áéíóúñ")
				So(event.Skill, ShouldContainSubstring, "🚀🎯💻")
			})
		})

		Convey("When creating an event with extreme metric values", func() {
			event := model.Event{
				EventID:   "event-extreme",
				TalentID:  "talent-extreme",
				RawMetric: 1e308, // Very large number
				Skill:     "extreme-skill",
				TS:        time.Now(),
			}

			Convey("Then it should handle extreme values", func() {
				So(event.RawMetric, ShouldEqual, 1e308)
			})
		})

		Convey("When creating a talent score with extreme values", func() {
			talentScore := model.TalentScore{
				TalentID: "talent-extreme",
				Score:    1e308,
			}

			Convey("Then it should handle extreme values", func() {
				So(talentScore.Score, ShouldEqual, 1e308)
			})
		})
	})
}
