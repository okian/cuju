package scoring_test

import (
	"context"
	"math"
	"testing"
	"time"

	scoring "github.com/okian/cuju/internal/domain/scoring"
	"github.com/smartystreets/goconvey/convey"
)

func TestInMemoryScorer_Score(t *testing.T) {
	convey.Convey("Given a new in-memory scorer", t, func() {
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

		convey.Convey("When scoring a coding event", func() {
			input := scoring.Input{
				TalentID:  "talent-123",
				RawMetric: 85.0,
				Skill:     "coding",
			}

			convey.Convey("Then it should return a valid score", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.TalentID, convey.ShouldEqual, "talent-123")
				convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
				convey.So(result.Score, convey.ShouldBeLessThanOrEqualTo, 100)
			})

			convey.Convey("And it should apply simple weight-based scoring", func() {
				// Test simple weight-based scoring
				highPerfInput := scoring.Input{
					TalentID:  "talent-456",
					RawMetric: 90.0,
					Skill:     "coding",
				}
				result, err := scorer.Score(context.Background(), highPerfInput)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 90.0) // Should be raw metric * weight (90 * 1.0 = 90)
			})
		})

		convey.Convey("When scoring a design event", func() {
			input := scoring.Input{
				TalentID:  "talent-789",
				RawMetric: 75.0,
				Skill:     "design",
			}

			convey.Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 60.0) // Should be raw metric * weight (75 * 0.8 = 60)
			})
		})

		convey.Convey("When scoring a writing event", func() {
			convey.Convey("And the raw metric is high", func() {
				input := scoring.Input{
					TalentID:  "talent-101",
					RawMetric: 80.0,
					Skill:     "writing",
				}

				convey.Convey("Then it should apply simple weight-based scoring", func() {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.Score, convey.ShouldEqual, 72.0) // Should be raw metric * weight (80 * 0.9 = 72)
				})
			})

			convey.Convey("And the raw metric is low", func() {
				input := scoring.Input{
					TalentID:  "talent-102",
					RawMetric: 40.0,
					Skill:     "writing",
				}

				convey.Convey("Then it should apply simple weight-based scoring", func() {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.Score, convey.ShouldEqual, 36.0) // Should be raw metric * weight (40 * 0.9 = 36)
				})
			})
		})

		convey.Convey("When scoring a marketing event", func() {
			input := scoring.Input{
				TalentID:  "talent-103",
				RawMetric: 80.0,
				Skill:     "marketing",
			}

			convey.Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 56.0) // Should be raw metric * weight (80 * 0.7 = 56)
			})
		})

		convey.Convey("When scoring a sales event", func() {
			input := scoring.Input{
				TalentID:  "talent-104",
				RawMetric: 70.0,
				Skill:     "sales",
			}

			convey.Convey("Then it should apply simple weight-based scoring", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 42.0) // Should be raw metric * weight (70 * 0.6 = 42)
			})
		})

		convey.Convey("When scoring an unknown skill", func() {
			input := scoring.Input{
				TalentID:  "talent-105",
				RawMetric: 60.0,
				Skill:     "unknown_skill",
			}

			convey.Convey("Then it should use default weight", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 100.0) // Should be raw metric * default weight (60 * 100 / 60 = 100)
			})
		})

		convey.Convey("When scoring with extreme values", func() {
			convey.Convey("And raw metric is 0", func() {
				input := scoring.Input{
					TalentID:  "talent-106",
					RawMetric: 0.0,
					Skill:     "coding",
				}

				convey.Convey("Then score should be 0", func() {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.Score, convey.ShouldEqual, 0.0)
				})
			})

			convey.Convey("And raw metric is very high", func() {
				input := scoring.Input{
					TalentID:  "talent-107",
					RawMetric: 1000.0,
					Skill:     "coding",
				}

				convey.Convey("Then score should be capped at 100", func() {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.Score, convey.ShouldEqual, 100.0)
				})
			})

			convey.Convey("And raw metric is negative", func() {
				input := scoring.Input{
					TalentID:  "talent-108",
					RawMetric: -50.0,
					Skill:     "coding",
				}

				convey.Convey("Then score should be 0", func() {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.Score, convey.ShouldEqual, 0.0)
				})
			})
		})

		convey.Convey("When context is cancelled", func() {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // Cancel immediately

			input := scoring.Input{
				TalentID:  "talent-109",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			convey.Convey("Then it should return context error", func() {
				result, err := scorer.Score(ctx, input)
				convey.So(err, convey.ShouldEqual, context.Canceled)
				convey.So(result.TalentID, convey.ShouldEqual, "")
				convey.So(result.Score, convey.ShouldEqual, 0.0)
			})
		})

		convey.Convey("When scoring multiple events", func() {
			inputs := []scoring.Input{
				{TalentID: "talent-110", RawMetric: 80.0, Skill: "coding"},
				{TalentID: "talent-111", RawMetric: 70.0, Skill: "design"},
				{TalentID: "talent-112", RawMetric: 85.0, Skill: "writing"},
			}

			convey.Convey("Then all should return valid scores", func() {
				for _, input := range inputs {
					result, err := scorer.Score(context.Background(), input)
					convey.So(err, convey.ShouldBeNil)
					convey.So(result.TalentID, convey.ShouldEqual, input.TalentID)
					convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
					convey.So(result.Score, convey.ShouldBeLessThanOrEqualTo, 100)
				}
			})
		})
	})
}

func TestInMemoryScorer_Options(t *testing.T) {
	convey.Convey("Given a scorer with custom options", t, func() {
		convey.Convey("When setting custom skill weights", func() {
			scorer := scoring.NewInMemoryScorer(
				scoring.WithSkillWeightsFromConfig(map[string]float64{
					"custom_skill":  2.0,
					"another_skill": 1.5,
				}, 100),
			)

			convey.Convey("Then custom weights should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-113",
					RawMetric: 50.0,
					Skill:     "custom_skill",
				}

				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldBeGreaterThan, 50.0) // Should have higher weight
			})
		})

		convey.Convey("When setting custom latency range", func() {
			minLatency := 5 * time.Millisecond
			maxLatency := 50 * time.Millisecond

			scorer := scoring.NewInMemoryScorer(
				scoring.WithLatencyRange(minLatency, maxLatency),
			)

			convey.Convey("Then custom latency should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-114",
					RawMetric: 75.0,
					Skill:     "coding",
				}

				start := time.Now()
				result, err := scorer.Score(context.Background(), input)
				duration := time.Since(start)

				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
				convey.So(duration, convey.ShouldBeGreaterThanOrEqualTo, minLatency)
				convey.So(duration, convey.ShouldBeLessThanOrEqualTo, maxLatency)
			})
		})

		// Removed WithMetrics test - option doesn't exist

		convey.Convey("When setting custom default weight", func() {
			customDefault := 0.8
			scorer := scoring.NewInMemoryScorer(
				scoring.WithSkillWeightsFromConfig(map[string]float64{}, customDefault),
			)

			convey.Convey("Then custom default weight should be applied", func() {
				input := scoring.Input{
					TalentID:  "talent-115",
					RawMetric: 60.0,
					Skill:     "unknown_skill",
				}

				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
			})
		})
	})
}

func TestInMemoryScorer_EdgeCases(t *testing.T) {
	convey.Convey("Given a scorer", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		convey.Convey("When scoring with empty talent ID", func() {
			input := scoring.Input{
				TalentID:  "",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			convey.Convey("Then it should still work", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.TalentID, convey.ShouldEqual, "")
				convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
			})
		})

		convey.Convey("When scoring with very small raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-116",
				RawMetric: 0.001,
				Skill:     "coding",
			}

			convey.Convey("Then it should handle small values", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		convey.Convey("When scoring with NaN raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-117",
				RawMetric: math.NaN(),
				Skill:     "coding",
			}

			convey.Convey("Then it should handle NaN gracefully", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldBeGreaterThanOrEqualTo, 0)
			})
		})

		convey.Convey("When scoring with infinity raw metric", func() {
			input := scoring.Input{
				TalentID:  "talent-118",
				RawMetric: math.Inf(1),
				Skill:     "coding",
			}

			convey.Convey("Then it should cap at maximum score", func() {
				result, err := scorer.Score(context.Background(), input)
				convey.So(err, convey.ShouldBeNil)
				convey.So(result.Score, convey.ShouldEqual, 100.0)
			})
		})
	})
}

func TestInMemoryScorer_Deterministic(t *testing.T) {
	convey.Convey("Given a scorer with fixed seed", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		convey.Convey("When scoring the same input multiple times", func() {
			input := scoring.Input{
				TalentID:  "talent-119",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			convey.Convey("Then results should be consistent", func() {
				result1, err1 := scorer.Score(context.Background(), input)
				result2, err2 := scorer.Score(context.Background(), input)

				convey.So(err1, convey.ShouldBeNil)
				convey.So(err2, convey.ShouldBeNil)
				convey.So(result1.Score, convey.ShouldEqual, result2.Score)
			})
		})
	})
}

func TestInMemoryScorer_Performance(t *testing.T) {
	convey.Convey("Given a scorer", t, func() {
		scorer := scoring.NewInMemoryScorer(
			scoring.WithSkillWeightsFromConfig(map[string]float64{
				"coding": 1.0,
			}, 100),
		)

		convey.Convey("When scoring multiple events rapidly", func() {
			input := scoring.Input{
				TalentID:  "talent-120",
				RawMetric: 75.0,
				Skill:     "coding",
			}

			convey.Convey("Then it should handle concurrent requests", func() {
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

					convey.So(err, convey.ShouldBeNil)
					convey.So(result.TalentID, convey.ShouldEqual, input.TalentID)
					convey.So(result.Score, convey.ShouldBeGreaterThan, 0)
				}
			})
		})
	})
}
