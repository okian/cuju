package repository

import (
	"context"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// BenchmarkResult holds the results of a benchmark run
type BenchmarkResult struct {
	Operation     string
	TotalOps      int64
	TotalTime     time.Duration
	AvgLatency    time.Duration
	P50Latency    time.Duration
	P90Latency    time.Duration
	P95Latency    time.Duration
	P99Latency    time.Duration
	Throughput    float64 // ops/sec
	MemoryUsage   uint64  // bytes
	SnapshotCount int64
	ErrorCount    int64
	SuccessRate   float64
}

// APIPerformance tracks performance metrics for each API
type APIPerformance struct {
	UpdateBest *BenchmarkResult
	Rank       *BenchmarkResult
	TopN       *BenchmarkResult
	Count      *BenchmarkResult
}

// StressTestConfig holds configuration for comprehensive stress testing
type StressTestConfig struct {
	TotalTalents      int
	ConcurrentWorkers int
	TestDuration      time.Duration
	SnapshotInterval  time.Duration
	TopCacheSize      int
	EnableMetrics     bool

	// API call distribution (percentages)
	UpdateBestRatio float64
	RankRatio       float64
	TopNRatio       float64
	CountRatio      float64

	// TopN query size distribution
	TopNSizes       []int
	TopNSizeWeights []float64
}

// DefaultStressTestConfig returns a configuration for comprehensive stress testing
func DefaultStressTestConfig() *StressTestConfig {
	return &StressTestConfig{
		TotalTalents:      30_000_000,
		ConcurrentWorkers: 2000,             // High concurrency to create real pressure
		TestDuration:      15 * time.Minute, // Longer duration for stable metrics
		SnapshotInterval:  1 * time.Second,
		TopCacheSize:      10000,
		EnableMetrics:     true,

		// Realistic API distribution based on typical usage patterns
		UpdateBestRatio: 0.40, // 40% updates (score changes)
		RankRatio:       0.35, // 35% rank queries (individual lookups)
		TopNRatio:       0.20, // 20% topN queries (leaderboard views)
		CountRatio:      0.05, // 5% count queries

		// TopN size distribution (weighted towards smaller queries)
		TopNSizes:       []int{10, 50, 100, 500, 1000, 5000, 10000},
		TopNSizeWeights: []float64{0.4, 0.25, 0.15, 0.1, 0.05, 0.03, 0.02},
	}
}

// ComprehensiveStressTest runs a realistic stress test with all APIs under pressure
func ComprehensiveStressTest(b *testing.B, config *StressTestConfig) {
	if config == nil {
		config = DefaultStressTestConfig()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)
	defer cancel()

	// Create store with configuration
	store := NewTreapStore(ctx,
		WithSnapshotInterval(config.SnapshotInterval),
		WithTopCacheSize(config.TopCacheSize),
	)
	defer func() {
		if err := store.Close(); err != nil {
			b.Errorf("failed to close store: %v", err)
		}
	}()

	// Pre-populate with 30M talents with realistic score distribution
	b.Logf("Pre-populating store with %d talents...", config.TotalTalents)
	start := time.Now()
	populateStoreRealistic(ctx, store, config.TotalTalents)
	b.Logf("Pre-population completed in %v", time.Since(start))

	// Run comprehensive stress test
	b.Log("Running comprehensive stress test with all APIs under pressure...")
	apiPerformance := runComprehensiveStressTest(ctx, store, config)

	// Generate detailed performance report
	generateComprehensiveReport(b, apiPerformance, config)
}

// populateStoreRealistic pre-populates the store with realistic talent distribution
func populateStoreRealistic(ctx context.Context, store *TreapStore, count int) {
	const batchSize = 10000
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, runtime.NumCPU()*2) // Higher concurrency for population

	// Create a realistic score distribution
	scoreRanges := []struct {
		min, max float64
		weight   float64
	}{
		{0, 100, 0.15},    // 15% beginners
		{100, 300, 0.25},  // 25% novices
		{300, 500, 0.20},  // 20% intermediates
		{500, 700, 0.20},  // 20% advanced
		{700, 850, 0.15},  // 15% experts
		{850, 1000, 0.05}, // 5% elite
	}

	for i := 0; i < count; i += batchSize {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(startIdx int) {
			defer wg.Done()
			defer func() { <-semaphore }()

			endIdx := startIdx + batchSize
			if endIdx > count {
				endIdx = count
			}

			r := rand.New(rand.NewSource(int64(startIdx)))

			for j := startIdx; j < endIdx; j++ {
				talentID := fmt.Sprintf("talent_%d", j)

				// Generate 3-8 scores per talent with realistic progression
				scoreCount := 3 + r.Intn(6)
				for k := 0; k < scoreCount; k++ {
					// Select score range based on weights
					randVal := r.Float64()
					var selectedRange struct {
						min, max float64
						weight   float64
					}

					cumulativeWeight := 0.0
					for _, rng := range scoreRanges {
						cumulativeWeight += rng.weight
						if randVal <= cumulativeWeight {
							selectedRange = rng
							break
						}
					}

					// Generate score within range with some variation
					baseScore := selectedRange.min + r.Float64()*(selectedRange.max-selectedRange.min)
					variation := (r.Float64() - 0.5) * 50.0 // ±25 variation
					score := baseScore + variation

					// Ensure score is within valid range
					if score < 0 {
						score = 0
					} else if score > 1000 {
						score = 1000
					}

					_, _ = store.UpdateBest(ctx, talentID, score)
				}
			}
		}(i)
	}

	wg.Wait()
}

// runComprehensiveStressTest runs all APIs simultaneously under pressure
func runComprehensiveStressTest(ctx context.Context, store *TreapStore, config *StressTestConfig) *APIPerformance {
	var wg sync.WaitGroup

	// Shared metrics collection
	updateBestMetrics := &MetricsCollector{}
	rankMetrics := &MetricsCollector{}
	topNMetrics := &MetricsCollector{}
	countMetrics := &MetricsCollector{}

	// Start time for the entire test
	testStart := time.Now()

	// Start concurrent workers that call all APIs randomly
	for i := 0; i < config.ConcurrentWorkers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			r := rand.New(rand.NewSource(int64(workerID) + time.Now().UnixNano()))

			for ctx.Err() == nil {
				// Determine which API to call based on distribution
				apiChoice := r.Float64()

				switch {
				case apiChoice < config.UpdateBestRatio:
					// UpdateBest operation
					talentID := fmt.Sprintf("stress_talent_%d", r.Intn(config.TotalTalents))
					score := r.Float64() * 1000.0

					start := time.Now()
					_, err := store.UpdateBest(ctx, talentID, score)
					latency := time.Since(start)

					updateBestMetrics.Record(latency, err == nil)

				case apiChoice < config.UpdateBestRatio+config.RankRatio:
					// Rank operation
					talentID := fmt.Sprintf("talent_%d", r.Intn(config.TotalTalents))

					start := time.Now()
					_, err := store.Rank(ctx, talentID)
					latency := time.Since(start)

					rankMetrics.Record(latency, err == nil)

				case apiChoice < config.UpdateBestRatio+config.RankRatio+config.TopNRatio:
					// TopN operation with weighted size selection
					randVal := r.Float64()
					cumulativeWeight := 0.0
					var selectedSize int

					for i, weight := range config.TopNSizeWeights {
						cumulativeWeight += weight
						if randVal <= cumulativeWeight {
							selectedSize = config.TopNSizes[i]
							break
						}
					}

					start := time.Now()
					_, err := store.TopN(ctx, selectedSize)
					latency := time.Since(start)

					topNMetrics.Record(latency, err == nil)

				default:
					// Count operation
					start := time.Now()
					_ = store.Count(ctx)
					latency := time.Since(start)

					// Count always succeeds, so no error to check
					countMetrics.Record(latency, true)
				}

				// Small random delay to prevent overwhelming
				time.Sleep(time.Duration(r.Intn(100)) * time.Microsecond)
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(config.TestDuration)
	wg.Wait()

	totalTime := time.Since(testStart)
	snapshotCount := int64(totalTime / config.SnapshotInterval)

	// Calculate results for each API
	return &APIPerformance{
		UpdateBest: updateBestMetrics.CalculateResult("UpdateBest", totalTime, snapshotCount),
		Rank:       rankMetrics.CalculateResult("Rank", totalTime, snapshotCount),
		TopN:       topNMetrics.CalculateResult("TopN", totalTime, snapshotCount),
		Count:      countMetrics.CalculateResult("Count", totalTime, snapshotCount),
	}
}

// MetricsCollector collects latency and success metrics for an API
type MetricsCollector struct {
	latencies  []time.Duration
	successOps int64
	totalOps   int64
	mu         sync.Mutex
}

// Record records a single operation result
func (mc *MetricsCollector) Record(latency time.Duration, success bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.latencies = append(mc.latencies, latency)
	mc.totalOps++
	if success {
		mc.successOps++
	}
}

// CalculateResult calculates benchmark results from collected metrics
func (mc *MetricsCollector) CalculateResult(operation string, totalTime time.Duration, snapshotCount int64) *BenchmarkResult {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if len(mc.latencies) == 0 {
		return &BenchmarkResult{
			Operation:     operation,
			TotalOps:      mc.totalOps,
			TotalTime:     totalTime,
			SnapshotCount: snapshotCount,
			ErrorCount:    mc.totalOps - mc.successOps,
			SuccessRate:   0.0,
		}
	}

	// Sort latencies for percentile calculation
	sorted := make([]time.Duration, len(mc.latencies))
	copy(sorted, mc.latencies)
	sortDurations(sorted)

	// Calculate percentiles
	p50Idx := int(float64(len(sorted)) * 0.50)
	p90Idx := int(float64(len(sorted)) * 0.90)
	p95Idx := int(float64(len(sorted)) * 0.95)
	p99Idx := int(float64(len(sorted)) * 0.99)

	// Calculate average
	var total time.Duration
	for _, lat := range mc.latencies {
		total += lat
	}
	avgLatency := total / time.Duration(len(mc.latencies))

	// Get memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	successRate := float64(mc.successOps) / float64(mc.totalOps) * 100.0

	return &BenchmarkResult{
		Operation:     operation,
		TotalOps:      mc.totalOps,
		TotalTime:     totalTime,
		AvgLatency:    avgLatency,
		P50Latency:    sorted[p50Idx],
		P90Latency:    sorted[p90Idx],
		P95Latency:    sorted[p95Idx],
		P99Latency:    sorted[p99Idx],
		Throughput:    float64(mc.totalOps) / totalTime.Seconds(),
		MemoryUsage:   m.Alloc,
		SnapshotCount: snapshotCount,
		ErrorCount:    mc.totalOps - mc.successOps,
		SuccessRate:   successRate,
	}
}

// sortDurations sorts durations in ascending order
func sortDurations(durations []time.Duration) {
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}
}

// generateComprehensiveReport generates a detailed performance report
func generateComprehensiveReport(b *testing.B, apiPerf *APIPerformance, config *StressTestConfig) {
	b.Log("\n" + strings.Repeat("=", 100))
	b.Log("COMPREHENSIVE STRESS TEST REPORT - ALL APIs UNDER PRESSURE")
	b.Log(strings.Repeat("=", 100))
	b.Logf("Configuration:")
	b.Logf("  Total Talents: %d", config.TotalTalents)
	b.Logf("  Concurrent Workers: %d", config.ConcurrentWorkers)
	b.Logf("  Snapshot Interval: %v", config.SnapshotInterval)
	b.Logf("  Top Cache Size: %d", config.TopCacheSize)
	b.Logf("  Test Duration: %v", config.TestDuration)
	b.Logf("  API Distribution: UpdateBest(%.1f%%) Rank(%.1f%%) TopN(%.1f%%) Count(%.1f%%)",
		config.UpdateBestRatio*100, config.RankRatio*100, config.TopNRatio*100, config.CountRatio*100)
	b.Logf("")

	// API Performance Summary Table
	b.Logf("API PERFORMANCE SUMMARY:")
	b.Logf("%-15s %12s %12s %12s %12s %12s %12s %10s %10s", "API", "Total Ops", "Throughput", "Avg Latency", "P90 Latency", "P95 Latency", "P99 Latency", "Success%", "Errors")
	b.Logf("%-15s %12s %12s %12s %12s %12s %12s %10s %10s", "", "", "(ops/sec)", "(μs)", "(μs)", "(μs)", "(μs)", "", "")
	b.Log(strings.Repeat("-", 100))

	apis := []struct {
		name   string
		result *BenchmarkResult
	}{
		{"UpdateBest", apiPerf.UpdateBest},
		{"Rank", apiPerf.Rank},
		{"TopN", apiPerf.TopN},
		{"Count", apiPerf.Count},
	}

	for _, api := range apis {
		if api.result.TotalOps > 0 {
			b.Logf("%-15s %12d %12.0f %12d %12d %12d %12d %10.1f %10d",
				api.name,
				api.result.TotalOps,
				api.result.Throughput,
				api.result.AvgLatency.Microseconds(),
				api.result.P90Latency.Microseconds(),
				api.result.P95Latency.Microseconds(),
				api.result.P99Latency.Microseconds(),
				api.result.SuccessRate,
				api.result.ErrorCount,
			)
		}
	}

	b.Logf("")
	b.Logf("DETAILED LATENCY ANALYSIS:")
	b.Logf("")

	// Latency distribution analysis
	for _, api := range apis {
		if api.result.TotalOps > 0 {
			b.Logf("%s Latency Distribution:", api.name)
			b.Logf("  P50: %8d μs (median)", api.result.P50Latency.Microseconds())
			b.Logf("  P90: %8d μs (90%% of requests faster)", api.result.P90Latency.Microseconds())
			b.Logf("  P95: %8d μs (95%% of requests faster)", api.result.P95Latency.Microseconds())
			b.Logf("  P99: %8d μs (99%% of requests faster)", api.result.P99Latency.Microseconds())
			b.Logf("  Tail Latency (P99-P50): %8d μs", (api.result.P99Latency - api.result.P50Latency).Microseconds())
			b.Logf("")
		}
	}

	// Performance insights
	b.Logf("PERFORMANCE INSIGHTS:")

	// Find best and worst performing APIs
	var bestAPI, worstAPI *BenchmarkResult
	var bestThroughput, worstThroughput float64

	for _, api := range apis {
		if api.result.Throughput > 0 {
			if bestAPI == nil || api.result.Throughput > bestThroughput {
				bestAPI = api.result
				bestThroughput = api.result.Throughput
			}
			if worstAPI == nil || api.result.Throughput < worstThroughput {
				worstAPI = api.result
				worstThroughput = api.result.Throughput
			}
		}
	}

	if bestAPI != nil && worstAPI != nil {
		b.Logf("  Best Performance: %s (%.0f ops/sec)", bestAPI.Operation, bestAPI.Throughput)
		b.Logf("  Worst Performance: %s (%.0f ops/sec)", worstAPI.Operation, worstAPI.Throughput)
		b.Logf("  Performance Ratio: %.2fx", bestAPI.Throughput/worstAPI.Throughput)
	}

	// Latency consistency analysis
	b.Logf("")
	b.Logf("LATENCY CONSISTENCY ANALYSIS:")
	for _, api := range apis {
		if api.result.TotalOps > 0 {
			latencySpread := float64(api.result.P99Latency) / float64(api.result.P50Latency)
			consistency := "Good"
			if latencySpread > 10 {
				consistency = "Poor"
			} else if latencySpread > 5 {
				consistency = "Fair"
			}
			b.Logf("  %s: P99/P50 ratio = %.2fx (%s consistency)",
				api.name, latencySpread, consistency)
		}
	}

	// Memory and resource analysis
	b.Logf("")
	b.Logf("RESOURCE ANALYSIS:")
	for _, api := range apis {
		if api.result.MemoryUsage > 0 {
			b.Logf("  %s Memory Usage: %s", api.name, formatBytes(api.result.MemoryUsage))
		}
	}

	b.Logf("")
	b.Logf("SNAPSHOT IMPACT:")
	for _, api := range apis {
		if api.result.SnapshotCount > 0 {
			b.Logf("  %s: %d snapshots during test", api.name, api.result.SnapshotCount)
		}
	}

	b.Logf("")
	b.Logf("STRESS TEST VALIDATION:")
	b.Logf("  ✓ Random talent selection across 30M talents")
	b.Logf("  ✓ Non-sequential updates (random scores and timing)")
	b.Logf("  ✓ All APIs called simultaneously under pressure")
	b.Logf("  ✓ Realistic API call distribution")
	b.Logf("  ✓ High concurrency (%d workers)", config.ConcurrentWorkers)
	b.Logf("  ✓ Extended duration (%v) for stable metrics", config.TestDuration)

	b.Log(strings.Repeat("=", 100))
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Benchmark functions for Go's testing framework
func BenchmarkTreapStore_30MTalents_ComprehensiveStressTest(b *testing.B) {
	config := DefaultStressTestConfig()
	ComprehensiveStressTest(b, config)
}

func BenchmarkTreapStore_30MTalents_ExtremePressure(b *testing.B) {
	config := DefaultStressTestConfig()
	config.ConcurrentWorkers = 5000 // Extreme pressure
	config.TestDuration = 20 * time.Minute
	ComprehensiveStressTest(b, config)
}

func BenchmarkTreapStore_30MTalents_WriteHeavyStress(b *testing.B) {
	config := DefaultStressTestConfig()
	config.UpdateBestRatio = 0.70 // 70% updates
	config.RankRatio = 0.20       // 20% rank queries
	config.TopNRatio = 0.08       // 8% topN queries
	config.CountRatio = 0.02      // 2% count queries
	ComprehensiveStressTest(b, config)
}

func BenchmarkTreapStore_30MTalents_ReadHeavyStress(b *testing.B) {
	config := DefaultStressTestConfig()
	config.UpdateBestRatio = 0.15 // 15% updates
	config.RankRatio = 0.50       // 50% rank queries
	config.TopNRatio = 0.30       // 30% topN queries
	config.CountRatio = 0.05      // 5% count queries
	ComprehensiveStressTest(b, config)
}

func BenchmarkTreapStore_30MTalents_TopNHeavyStress(b *testing.B) {
	config := DefaultStressTestConfig()
	config.UpdateBestRatio = 0.20 // 20% updates
	config.RankRatio = 0.25       // 25% rank queries
	config.TopNRatio = 0.50       // 50% topN queries
	config.CountRatio = 0.05      // 5% count queries
	ComprehensiveStressTest(b, config)
}
