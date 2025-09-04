package types_test

import (
	"testing"

	types "github.com/okian/cuju/internal/domain/types"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEntry(t *testing.T) {
	Convey("Given an Entry struct", t, func() {
		Convey("When creating a new entry", func() {
			rank := 1
			talentID := "talent-123"
			score := 95.5

			entry := types.Entry{
				Rank:     rank,
				TalentID: talentID,
				Score:    score,
			}

			Convey("Then it should have the correct values", func() {
				So(entry.Rank, ShouldEqual, rank)
				So(entry.TalentID, ShouldEqual, talentID)
				So(entry.Score, ShouldEqual, score)
			})
		})

		Convey("When creating an entry with zero values", func() {
			entry := types.Entry{}

			Convey("Then it should have default values", func() {
				So(entry.Rank, ShouldEqual, 0)
				So(entry.TalentID, ShouldEqual, "")
				So(entry.Score, ShouldEqual, 0.0)
			})
		})

		Convey("When creating an entry with negative rank", func() {
			entry := types.Entry{
				Rank:     -1,
				TalentID: "talent-neg",
				Score:    85.0,
			}

			Convey("Then it should accept negative rank", func() {
				So(entry.Rank, ShouldEqual, -1)
			})
		})

		Convey("When creating an entry with zero rank", func() {
			entry := types.Entry{
				Rank:     0,
				TalentID: "talent-zero",
				Score:    90.0,
			}

			Convey("Then it should accept zero rank", func() {
				So(entry.Rank, ShouldEqual, 0)
			})
		})

		Convey("When creating an entry with very high rank", func() {
			entry := types.Entry{
				Rank:     999999,
				TalentID: "talent-high-rank",
				Score:    75.0,
			}

			Convey("Then it should accept high rank", func() {
				So(entry.Rank, ShouldEqual, 999999)
			})
		})

		Convey("When creating an entry with negative score", func() {
			entry := types.Entry{
				Rank:     5,
				TalentID: "talent-neg-score",
				Score:    -15.5,
			}

			Convey("Then it should accept negative score", func() {
				So(entry.Score, ShouldEqual, -15.5)
			})
		})

		Convey("When creating an entry with zero score", func() {
			entry := types.Entry{
				Rank:     10,
				TalentID: "talent-zero-score",
				Score:    0.0,
			}

			Convey("Then it should accept zero score", func() {
				So(entry.Score, ShouldEqual, 0.0)
			})
		})

		Convey("When creating an entry with very high score", func() {
			entry := types.Entry{
				Rank:     2,
				TalentID: "talent-high-score",
				Score:    999999.999,
			}

			Convey("Then it should accept high score", func() {
				So(entry.Score, ShouldEqual, 999999.999)
			})
		})

		Convey("When creating an entry with decimal score", func() {
			entry := types.Entry{
				Rank:     3,
				TalentID: "talent-decimal",
				Score:    87.857,
			}

			Convey("Then it should maintain decimal precision", func() {
				So(entry.Score, ShouldEqual, 87.857)
			})
		})
	})
}

func TestEntryValidation(t *testing.T) {
	Convey("Given entry validation scenarios", t, func() {
		Convey("When creating an entry with valid data", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "valid-talent-123",
				Score:    92.5,
			}

			Convey("Then it should be a valid entry", func() {
				So(entry.Rank, ShouldNotBeNil)
				So(entry.TalentID, ShouldNotBeEmpty)
				So(entry.Score, ShouldNotBeNil)
			})
		})

		Convey("When creating an entry with minimal data", func() {
			entry := types.Entry{
				TalentID: "minimal",
			}

			Convey("Then it should have minimal required fields", func() {
				So(entry.Rank, ShouldEqual, 0)
				So(entry.TalentID, ShouldNotBeEmpty)
				So(entry.Score, ShouldEqual, 0.0)
			})
		})

		Convey("When creating multiple entries", func() {
			entries := []types.Entry{
				{Rank: 1, TalentID: "talent-1", Score: 95.0},
				{Rank: 2, TalentID: "talent-2", Score: 90.5},
				{Rank: 3, TalentID: "talent-3", Score: 88.0},
				{Rank: 4, TalentID: "talent-4", Score: 85.5},
				{Rank: 5, TalentID: "talent-5", Score: 82.0},
			}

			Convey("Then all entries should be valid", func() {
				for _, entry := range entries {
					So(entry.TalentID, ShouldNotBeEmpty)
					So(entry.Rank, ShouldBeGreaterThanOrEqualTo, 0)
				}
			})

			Convey("And ranks should be sequential", func() {
				for i, entry := range entries {
					So(entry.Rank, ShouldEqual, i+1)
				}
			})

			Convey("And scores should be in descending order", func() {
				for i := 0; i < len(entries)-1; i++ {
					So(entries[i].Score, ShouldBeGreaterThanOrEqualTo, entries[i+1].Score)
				}
			})
		})
	})
}

func TestEntryEdgeCases(t *testing.T) {
	Convey("Given entry edge cases", t, func() {
		Convey("When creating an entry with very long talent ID", func() {
			longTalentID := "talent-" + string(make([]byte, 1000))

			entry := types.Entry{
				Rank:     1,
				TalentID: longTalentID,
				Score:    90.0,
			}

			Convey("Then it should handle long strings", func() {
				So(len(entry.TalentID), ShouldBeGreaterThan, 1000)
			})
		})

		Convey("When creating an entry with special characters in talent ID", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-!@#$%^&*()",
				Score:    85.0,
			}

			Convey("Then it should handle special characters", func() {
				So(entry.TalentID, ShouldContainSubstring, "!@#$%^&*()")
			})
		})

		Convey("When creating an entry with unicode characters in talent ID", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-áéíóúñ🚀🎯💻",
				Score:    88.0,
			}

			Convey("Then it should handle unicode characters", func() {
				So(entry.TalentID, ShouldContainSubstring, "áéíóúñ🚀🎯💻")
			})
		})

		Convey("When creating an entry with extreme rank values", func() {
			entry := types.Entry{
				Rank:     2147483647, // Max int32
				TalentID: "talent-extreme-rank",
				Score:    75.0,
			}

			Convey("Then it should handle extreme rank values", func() {
				So(entry.Rank, ShouldEqual, 2147483647)
			})
		})

		Convey("When creating an entry with extreme score values", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-extreme-score",
				Score:    1e308, // Very large number
			}

			Convey("Then it should handle extreme score values", func() {
				So(entry.Score, ShouldEqual, 1e308)
			})
		})

		Convey("When creating an entry with very small score values", func() {
			entry := types.Entry{
				Rank:     1,
				TalentID: "talent-small-score",
				Score:    1e-308, // Very small number
			}

			Convey("Then it should handle very small score values", func() {
				So(entry.Score, ShouldEqual, 1e-308)
			})
		})
	})
}

func TestEntryComparison(t *testing.T) {
	Convey("Given entry comparison scenarios", t, func() {
		Convey("When comparing entries by rank", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 95.0}
			entry2 := types.Entry{Rank: 2, TalentID: "talent-2", Score: 90.0}
			entry3 := types.Entry{Rank: 3, TalentID: "talent-3", Score: 85.0}

			Convey("Then ranks should be in ascending order", func() {
				So(entry1.Rank, ShouldBeLessThan, entry2.Rank)
				So(entry2.Rank, ShouldBeLessThan, entry3.Rank)
			})

			Convey("And scores should be in descending order", func() {
				So(entry1.Score, ShouldBeGreaterThan, entry2.Score)
				So(entry2.Score, ShouldBeGreaterThan, entry3.Score)
			})
		})

		Convey("When comparing entries with same rank", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 95.0}
			entry2 := types.Entry{Rank: 1, TalentID: "talent-2", Score: 95.0}

			Convey("Then ranks should be equal", func() {
				So(entry1.Rank, ShouldEqual, entry2.Rank)
			})

			Convey("And scores should be equal", func() {
				So(entry1.Score, ShouldEqual, entry2.Score)
			})

			Convey("But talent IDs should be different", func() {
				So(entry1.TalentID, ShouldNotEqual, entry2.TalentID)
			})
		})

		Convey("When comparing entries with same score", func() {
			entry1 := types.Entry{Rank: 1, TalentID: "talent-1", Score: 90.0}
			entry2 := types.Entry{Rank: 2, TalentID: "talent-2", Score: 90.0}

			Convey("Then scores should be equal", func() {
				So(entry1.Score, ShouldEqual, entry2.Score)
			})

			Convey("But ranks should be different", func() {
				So(entry1.Rank, ShouldNotEqual, entry2.Rank)
			})
		})
	})
}

func TestEntryDataIntegrity(t *testing.T) {
	Convey("Given entry data integrity scenarios", t, func() {
		Convey("When creating entries with various data types", func() {
			entries := []types.Entry{
				{Rank: 1, TalentID: "talent-string", Score: 90.0},
				{Rank: 2, TalentID: "123", Score: 85.0},
				{Rank: 3, TalentID: "talent-with-dash", Score: 80.0},
				{Rank: 4, TalentID: "talent_with_underscore", Score: 75.0},
				{Rank: 5, TalentID: "TALENT-UPPERCASE", Score: 70.0},
			}

			Convey("Then all entries should maintain data integrity", func() {
				for i, entry := range entries {
					So(entry.Rank, ShouldEqual, i+1)
					So(entry.TalentID, ShouldNotBeEmpty)
					So(entry.Score, ShouldBeGreaterThan, 0)
				}
			})
		})

		Convey("When creating entries with boundary values", func() {
			entries := []types.Entry{
				{Rank: 0, TalentID: "talent-zero-rank", Score: 0.0},
				{Rank: 1, TalentID: "talent-min-score", Score: 0.001},
				{Rank: 2, TalentID: "talent-max-score", Score: 999.999},
				{Rank: 999, TalentID: "talent-max-rank", Score: 100.0},
			}

			Convey("Then all entries should handle boundary values correctly", func() {
				for _, entry := range entries {
					So(entry.Rank, ShouldBeGreaterThanOrEqualTo, 0)
					So(entry.TalentID, ShouldNotBeEmpty)
					So(entry.Score, ShouldBeGreaterThanOrEqualTo, 0)
				}
			})
		})
	})
}
