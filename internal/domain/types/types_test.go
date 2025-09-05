package types_test

import (
	"testing"

	types "github.com/okian/cuju/internal/domain/types"
	"github.com/smartystreets/goconvey/convey"
)

func TestEntry(t *testing.T) {
	convey.Convey("Given an Entry struct", t, func() {
		convey.Convey("When creating a new entry", func() {
			rank := 1
			talentID := "talent-123"
			score := 95.5

			entry := types.Entry{
				Rank:     rank,
				TalentID: talentID,
				Score:    score,
			}

			convey.Convey("Then it should have the correct values", func() {
				convey.So(entry.Rank, convey.ShouldEqual, rank)
				convey.So(entry.TalentID, convey.ShouldEqual, talentID)
				convey.So(entry.Score, convey.ShouldEqual, score)
			})
		})

		convey.Convey("When creating an entry with zero values", func() {
			entry := types.Entry{}

			convey.Convey("Then it should have default values", func() {
				convey.So(entry.Rank, convey.ShouldEqual, 0)
				convey.So(entry.TalentID, convey.ShouldEqual, "")
				convey.So(entry.Score, convey.ShouldEqual, 0.0)
			})
		})

		convey.Convey("When creating an entry with negative rank", func() {
			entry := types.Entry{
				Rank:     -1,
				TalentID: "talent-neg",
				Score:    85.0,
			}

			convey.Convey("Then it should accept negative rank", func() {
				convey.So(entry.Rank, convey.ShouldEqual, -1)
			})
		})

		convey.Convey("When creating an entry with zero rank", func() {
			entry := types.Entry{
				Rank:     0,
				TalentID: "talent-zero",
				Score:    90.0,
			}

			convey.Convey("Then it should accept zero rank", func() {
				convey.So(entry.Rank, convey.ShouldEqual, 0)
			})
		})

		convey.Convey("When creating an entry with very high rank", func() {
			entry := types.Entry{
				Rank:     999999,
				TalentID: "talent-high-rank",
				Score:    75.0,
			}

			convey.Convey("Then it should accept high rank", func() {
				convey.So(entry.Rank, convey.ShouldEqual, 999999)
			})
		})

		convey.Convey("When creating an entry with negative score", func() {
			entry := types.Entry{
				Rank:     5,
				TalentID: "talent-neg-score",
				Score:    -15.5,
			}

			convey.Convey("Then it should accept negative score", func() {
				convey.So(entry.Score, convey.ShouldEqual, -15.5)
			})
		})

		convey.Convey("When creating an entry with zero score", func() {
			entry := types.Entry{
				Rank:     10,
				TalentID: "talent-zero-score",
				Score:    0.0,
			}

			convey.Convey("Then it should accept zero score", func() {
				convey.So(entry.Score, convey.ShouldEqual, 0.0)
			})
		})

		convey.Convey("When creating an entry with very high score", func() {
			entry := types.Entry{
				Rank:     2,
				TalentID: "talent-high-score",
				Score:    999999.999,
			}

			convey.Convey("Then it should accept high score", func() {
				convey.So(entry.Score, convey.ShouldEqual, 999999.999)
			})
		})

		convey.Convey("When creating an entry with decimal score", func() {
			entry := types.Entry{
				Rank:     3,
				TalentID: "talent-decimal",
				Score:    87.857,
			}

			convey.Convey("Then it should maintain decimal precision", func() {
				convey.So(entry.Score, convey.ShouldEqual, 87.857)
			})
		})
	})
}

func TestEntryValidation(t *testing.T) {
	convey.Convey("Given entry validation scenarios", t, func() {
		convey.Convey("When creating an entry with valid data", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "valid-talent-123",
				Score:    92.5,
			}

			convey.Convey("Then it should be a valid entry", func() {
				convey.So(entry.Rank, convey.ShouldNotBeNil)
				convey.So(entry.TalentID, convey.ShouldNotBeEmpty)
				convey.So(entry.Score, convey.ShouldNotBeNil)
			})
		})

		convey.Convey("When creating an entry with minimal data", func() {
			entry := types.Entry{
				TalentID: "minimal",
			}

			convey.Convey("Then it should have minimal required fields", func() {
				convey.So(entry.Rank, convey.ShouldEqual, 0)
				convey.So(entry.TalentID, convey.ShouldNotBeEmpty)
				convey.So(entry.Score, convey.ShouldEqual, 0.0)
			})
		})

		convey.Convey("When creating multiple entries", func() {
			entries := []types.Entry{
				{Rank: 1, TalentID: "talent-1", Score: 95.0},
				{Rank: 2, TalentID: "talent-2", Score: 90.5},
				{Rank: 3, TalentID: "talent-3", Score: 88.0},
				{Rank: 4, TalentID: "talent-4", Score: 85.5},
				{Rank: 5, TalentID: "talent-5", Score: 82.0},
			}

			convey.Convey("Then all entries should be valid", func() {
				for _, entry := range entries {
					convey.So(entry.TalentID, convey.ShouldNotBeEmpty)
					convey.So(entry.Rank, convey.ShouldBeGreaterThanOrEqualTo, 0)
				}
			})

			convey.Convey("And ranks should be sequential", func() {
				for i, entry := range entries {
					convey.So(entry.Rank, convey.ShouldEqual, i+1)
				}
			})

			convey.Convey("And scores should be in descending order", func() {
				for i := 0; i < len(entries)-1; i++ {
					convey.So(entries[i].Score, convey.ShouldBeGreaterThanOrEqualTo, entries[i+1].Score)
				}
			})
		})
	})
}

func TestEntryEdgeCases(t *testing.T) {
	convey.Convey("Given entry edge cases", t, func() {
		convey.Convey("When creating an entry with very long talent ID", func() {
			longTalentID := "talent-" + string(make([]byte, 1000))

			entry := types.Entry{
				Rank:     1,
				TalentID: longTalentID,
				Score:    90.0,
			}

			convey.Convey("Then it should handle long strings", func() {
				convey.So(len(entry.TalentID), convey.ShouldBeGreaterThan, 1000)
			})
		})

		convey.Convey("When creating an entry with special characters in talent ID", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-!@#$%^&*()",
				Score:    85.0,
			}

			convey.Convey("Then it should handle special characters", func() {
				convey.So(entry.TalentID, convey.ShouldContainSubstring, "!@#$%^&*()")
			})
		})

		convey.Convey("When creating an entry with unicode characters in talent ID", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-áéíóúñ🚀🎯💻",
				Score:    88.0,
			}

			convey.Convey("Then it should handle unicode characters", func() {
				convey.So(entry.TalentID, convey.ShouldContainSubstring, "áéíóúñ🚀🎯💻")
			})
		})

		convey.Convey("When creating an entry with extreme rank values", func() {
			entry := types.Entry{
				Rank:     2147483647, // Max int32
				TalentID: "talent-extreme-rank",
				Score:    75.0,
			}

			convey.Convey("Then it should handle extreme rank values", func() {
				convey.So(entry.Rank, convey.ShouldEqual, 2147483647)
			})
		})

		convey.Convey("When creating an entry with extreme score values", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-extreme-score",
				Score:    1e308, // Very large number
			}

			convey.Convey("Then it should handle extreme score values", func() {
				convey.So(entry.Score, convey.ShouldEqual, 1e308)
			})
		})

		convey.Convey("When creating an entry with very small score values", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-small-score",
				Score:    1e-308, // Very small number
			}

			convey.Convey("Then it should handle very small score values", func() {
				convey.So(entry.Score, convey.ShouldEqual, 1e-308)
			})
		})
	})
}

func TestEntryComparison(t *testing.T) {
	convey.Convey("Given entry comparison scenarios", t, func() {
		convey.Convey("When comparing entries by rank", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 95.0}
			entry2 := types.Entry{Rank: 2, TalentID: "talent-2", Score: 90.0}
			entry3 := types.Entry{Rank: 3, TalentID: "talent-3", Score: 85.0}

			convey.Convey("Then ranks should be in ascending order", func() {
				convey.So(entry1.Rank, convey.ShouldBeLessThan, entry2.Rank)
				convey.So(entry2.Rank, convey.ShouldBeLessThan, entry3.Rank)
			})

			convey.Convey("And scores should be in descending order", func() {
				convey.So(entry1.Score, convey.ShouldBeGreaterThan, entry2.Score)
				convey.So(entry2.Score, convey.ShouldBeGreaterThan, entry3.Score)
			})
		})

		convey.Convey("When comparing entries with same rank", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 95.0}
			entry2 := types.Entry{Rank: 1, TalentID: "talent-2", Score: 95.0}

			convey.Convey("Then ranks should be equal", func() {
				convey.So(entry1.Rank, convey.ShouldEqual, entry2.Rank)
			})

			convey.Convey("And scores should be equal", func() {
				convey.So(entry1.Score, convey.ShouldEqual, entry2.Score)
			})

			convey.Convey("But talent IDs should be different", func() {
				convey.So(entry1.TalentID, convey.ShouldNotEqual, entry2.TalentID)
			})
		})

		convey.Convey("When comparing entries with same score", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 90.0}
			entry2 := types.Entry{Rank: 2, TalentID: "talent-2", Score: 90.0}

			convey.Convey("Then scores should be equal", func() {
				convey.So(entry1.Score, convey.ShouldEqual, entry2.Score)
			})

			convey.Convey("But ranks should be different", func() {
				convey.So(entry1.Rank, convey.ShouldNotEqual, entry2.Rank)
			})
		})
	})
}

func TestEntryDataIntegrity(t *testing.T) {
	convey.Convey("Given entry data integrity scenarios", t, func() {
		convey.Convey("When creating entries with various data types", func() {
			entries := []types.Entry{
				{Rank: 1, TalentID: "talent-string", Score: 90.0},
				{Rank: 2, TalentID: "123", Score: 85.0},
				{Rank: 3, TalentID: "talent-with-dash", Score: 80.0},
				{Rank: 4, TalentID: "talent_with_underscore", Score: 75.0},
				{Rank: 5, TalentID: "TALENT-UPPERCASE", Score: 70.0},
			}

			convey.Convey("Then all entries should maintain data integrity", func() {
				for i, entry := range entries {
					convey.So(entry.Rank, convey.ShouldEqual, i+1)
					convey.So(entry.TalentID, convey.ShouldNotBeEmpty)
					convey.So(entry.Score, convey.ShouldBeGreaterThan, 0)
				}
			})
		})

		convey.Convey("When creating entries with boundary values", func() {
			entries := []types.Entry{
				{Rank: 0, TalentID: "talent-zero-rank", Score: 0.0},
				{Rank: 1, TalentID: "talent-min-score", Score: 0.001},
				{Rank: 2, TalentID: "talent-max-score", Score: 999.999},
				{Rank: 999, TalentID: "talent-max-rank", Score: 100.0},
			}

			convey.Convey("Then all entries should handle boundary values correctly", func() {
				for _, entry := range entries {
					convey.So(entry.Rank, convey.ShouldBeGreaterThanOrEqualTo, 0)
					convey.So(entry.TalentID, convey.ShouldNotBeEmpty)
					convey.So(entry.Score, convey.ShouldBeGreaterThanOrEqualTo, 0)
				}
			})
		})
	})
}
