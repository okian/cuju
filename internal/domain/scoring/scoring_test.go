package scoring_test

import (
	"context"
	"math"
	"testing"
	"time"

	scoring "github.com/okian/cuju/internal/domain/scoring"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryScorer_Score(t *testing.T) {
	Convey("Given a new in-memory scorer", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding":    1.0,
				"design":    0.8,
				"writing":   0.9,
				"marketing": 0.7,
				"sales":     0.6,
				"dribble":   0.85,
			}, 100),
		)

		Convey("When scoring a coding event", func() {
			input := scoring.Input{
				TalentID:  "talent-123",
				RawMetric: 85.0,
				Skill:     "coding",
			}

			Convey("Then it should return a valid score", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.TalentID, ShouldEqual, "talent-123")
				So(result.Score, ShouldBeGreaterThan, 0)
				So(result.Score, ShouldBeLessThanOrEqualTo, 100)
			})

			Convey("And it should apply simple weight-based scoring", func() {
				// Test simple weight-based scoring
				highPerfInput := scoring.Input{
					TalentID:  "talent-456",
					RawMetric: 90.0,
					Skill:     "coding",
				}
				result, err := scorer.Score(context.Background(), highPerfInput)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 90.0) // Should be raw metric * weight (90 * 1.0 = 90)
			})
		})

		Convey("When scoring a design event", func() {
			input := scoring.Input{
				TalentID:  "talent-789",
				RawMetric: 75.0,
				Skill:     "design",
			}

			Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 60.0) // Should be raw metric * weight (75 * 0.8 = 60)
			})
		})

		Convey("When scoring a writing event", func() {
			Convey("And the raw metric is high", func() {
				input := scoring.Input{
					TalentID:  "talent-101",
					RawMetric: 80.0,
					Skill:     "writing",
				}

				Convey("Then it should apply simple weight-based scoring", func() {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.Score, ShouldEqual, 72.0) // Should be raw metric * weight (80 * 0.9 = 72)
				})
			})

			Convey("And the raw metric is low", func() {
				input := scoring.Input{
					TalentID:  "talent-102",
					RawMetric: 40.0,
					Skill:     "writing",
				}

				Convey("Then it should apply simple weight-based scoring", func() {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.Score, ShouldEqual, 36.0) // Should be raw metric * weight (40 * 0.9 = 36)
				})
			})
		})

		Convey("When scoring a marketing event", func() {
			input := scoring.Input{
				TalentID:  "talent-103",
				RawMetric: 80.0,
				Skill:     "marketing",
			}

			Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 56.0) // Should be raw metric * weight (80 * 0.7 = 56)
			})
		})

		Convey("When scoring a sales event", func() {
			input := scoring.Input{
				TalentID:  "talent-104",
				RawMetric: 70.0,
				Skill:     "sales",
			}

			Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 42.0) // Should be raw metric * weight (70 * 0.6 = 42)
			})
		})

		Convey("When scoring an unknown skill", func() {
			input := scoring.Input{
				TalentID:  "talent-105",
				RawMetric: 60.0,
				Skill:     "unknown_skill",
			}

			Convey("Then it should use default weight", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 100.0) // Should be raw metric * default weight (60 * 100 / 60 = 100)
			})
		})

		Convey("When scoring with extreme values", func() {
			Convey("And raw metric is 0", func() {
				input := scoring.Input{
					TalentID:  "talent-106",
					RawMetric: 0.0,
					Skill:     "coding",
				}

				Convey("Then score should be 0", func() {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.Score, ShouldEqual, 0.0)
				})
			})

			Convey("And raw metric is very high", func() {
				input := scoring.Input{
					TalentID:  "talent-107",
					RawMetric: 1000.0,
					Skill:     "coding",
				}

				Convey("Then score should be capped at 100", func() {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.Score, ShouldEqual, 100.0)
				})
			})

			Convey("And raw metric is negative", func() {
				input := scoring.Input{
					TalentID:  "talent-108",
					RawMetric: -50.0,
					Skill:     "coding",
				}

				Convey("Then score should be 0", func() {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.Score, ShouldEqual, 0.0)
				})
			})
		})

		Convey("When context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			input := scoring.Input{
				TalentID:  "talent-109",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			Convey("Then it should return context error", func() {
				result, err := scorer.Score(ctx, input)
				So(err, ShouldEqual, context.Canceled)
				So(result.TalentID, ShouldEqual, "")
				So(result.Score, ShouldEqual, 0.0)
			})
		})

		Convey("When scoring multiple events", func() {
			inputs := []scoring.Input{
				{TalentID: "talent-110", RawMetric: 80.0, Skill: "coding"},
				{TalentID: "talent-111", RawMetric: 70.0, Skill: "design"},
				{TalentID: "talent-112", RawMetric: 85.0, Skill: "writing"},
			}

			Convey("Then all should return valid scores", func() {
				for _, input := range inputs {
					result, err := scorer.Score(context.Background(), input)
					So(err, ShouldBeNil)
					So(result.TalentID, ShouldEqual, input.TalentID)
					So(result.Score, ShouldBeGreaterThan, 0)
					So(result.Score, ShouldBeLessThanOrEqualTo, 100)
				}
			})
		})
	})
}

func TestInMemoryScorer_Options(t *testing.T) {
	Convey("Given a scorer with custom options", t, func() {
		Convey("When setting custom skill weights", func() {
			scorer := scoring.NewInMemoryScorer(
				scoring.WithSkillWeightsFromConfig(map[string]float64{
					"custom_skill":  2.0,
					"another_skill": 1.5,
				}, 100),
			)

			Convey("Then custom weights should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-113",
					RawMetric: 50.0,
					Skill:     "custom_skill",
				}

				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldBeGreaterThan, 50.0) // Should have higher weight
			})
		})

		Convey("When setting custom latency range", func() {
			minLatency := 5 * time.Millisecond
			maxLatency := 50 * time.Millisecond

			scorer := scoring.NewInMemoryScorer(
				scoring.WithLatencyRange(minLatency, maxLatency),
			)

			Convey("Then custom latency should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-114",
					RawMetric: 75.0,
					Skill:     "coding",
				}

				start := time.Now()
				result, err := scorer.Score(context.Background(), input)
				duration := time.Since(start)

				So(err, ShouldBeNil)
				So(result.Score, ShouldBeGreaterThan, 0)
				So(duration, ShouldBeGreaterThanOrEqualTo, minLatency)
				So(duration, ShouldBeLessThanOrEqualTo, maxLatency)
			})
		})

		// Removed WithMetrics test - option doesn't exist

		Convey("When setting custom default weight", func() {
			customDefault := 0.8
			scorer := scoring.NewInMemoryScorer(
				scoring.WithSkillWeightsFromConfig(map[string]float64{}, customDefault),
			)

			Convey("Then custom default weight should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-115",
					RawMetric: 60.0,
					Skill:     "unknown_skill",
				}

				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestInMemoryScorer_EdgeCases(t *testing.T) {
	Convey("Given a scorer", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		Convey("When scoring with empty talent ID", func() {
			input := scoring.Input{
				TalentID:  "",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			Convey("Then it should still work", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.TalentID, ShouldEqual, "")
				So(result.Score, ShouldBeGreaterThan, 0)
			})
		})

		Convey("When scoring with very small raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-116",
				RawMetric: 0.001,
				Skill:     "coding",
			}

			Convey("Then it should handle small values", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When scoring with NaN raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-117",
				RawMetric: math.NaN(),
				Skill:     "coding",
			}

			Convey("Then it should handle NaN gracefully", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		Convey("When scoring with infinity raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-118",
				RawMetric: math.Inf(1),
				Skill:     "coding",
			}

			Convey("Then it should cap at maximum score", func() {
				result, err := scorer.Score(context.Background(), input)
				So(err, ShouldBeNil)
				So(result.Score, ShouldEqual, 100.0)
			})
		})
	})
}

func TestInMemoryScorer_Deterministic(t *testing.T) {
	Convey("Given a scorer with fixed seed", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		Convey("When scoring the same input multiple times", func() {
			input := scoring.Input{
				TalentID:  "talent-119",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			Convey("Then results should be consistent", func() {
				result1, err1 := scorer.Score(context.Background(), input)
				result2, err2 := scorer.Score(context.Background(), input)

				So(err1, ShouldBeNil)
				So(err2, ShouldBeNil)
				So(result1.Score, ShouldEqual, result2.Score)
			})
		})
	})
}

func TestInMemoryScorer_Performance(t *testing.T) {
	Convey("Given a scorer", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		Convey("When scoring multiple events rapidly", func() {
			input := scoring.Input{
				TalentID:  "talent-120",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			Convey("Then it should handle concurrent requests", func() {
				results := make(chan scoring.Result, 10)
				errors := make(chan error, 10)

				// Start multiple goroutines
				for i := 0; i < 10; i++ {
					go func() {
						result, err := scorer.Score(context.Background(), input)
						results <- result
						errors <- err
					}()
				}

				// Collect results
				for i := 0; i < 10; i++ {
					result := <-results
					err := <-errors

					So(err, ShouldBeNil)
					So(result.TalentID, ShouldEqual, input.TalentID)
					So(result.Score, ShouldBeGreaterThan, 0)
				}
			})
		})
	})
}
