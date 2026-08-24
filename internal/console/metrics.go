// Package console exposes the operations dashboard: metrics aggregation,
// JSON APIs and the embedded web pages.
package console

import "edge-transcode/internal/store"

// SuccessRate returns the QC pass rate for a stream (1.0 when no samples).
func SuccessRate(samples []store.QualitySample, streamID string) float64 {
	var passed, total int
	for _, sample := range samples {
		if sample.StreamID != streamID {
			continue
		}
		total++
		if sample.Passed {
			passed++
		}
	}
	if total == 0 {
		return 1
	}
	return float64(passed) / float64(total)
}

// PassCount returns how many samples passed for a stream.
func PassCount(samples []store.QualitySample, streamID string) int {
	count := 0
	for _, sample := range samples {
		if (streamID == "" || sample.StreamID == streamID) && sample.Passed {
			count++
		}
	}
	return count
}

// RecentSamples returns the newest n samples.
func RecentSamples(samples []store.QualitySample, n int) []store.QualitySample {
	if n <= 0 || len(samples) <= n {
		return samples
	}
	return samples[len(samples)-n:]
}

// AverageScore returns the mean score of samples for a stream.
func AverageScore(samples []store.QualitySample, streamID string) float64 {
	var sum float64
	var count int
	for _, sample := range samples {
		if sample.StreamID != streamID {
			continue
		}
		sum += sample.Score
		count++
	}
	if count == 0 {
		return 0
	}
	return sum / float64(count)
}
