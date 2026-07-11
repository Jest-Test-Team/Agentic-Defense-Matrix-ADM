package telemetry

import (
	"sort"
	"sync"
	"time"
)

// LatencyRecorder collects a stream of latency samples and reports the
// distribution (p50/p95/p99, min/max/mean) rather than a single mean. It is the
// instrumentation primitive behind the δ (detection delay) and κ (containment
// latency) measurements in docs/research/formalization-containment.md — a mean
// (MTTR) hides the tail that reviewers care about, so we keep the samples.
//
// It is safe for concurrent use, so live components (gateway detection path,
// green-team containment path, watchdog) can record into a shared recorder.
type LatencyRecorder struct {
	mu      sync.Mutex
	samples []time.Duration
	name    string
}

// NewLatencyRecorder creates a named recorder (e.g. "delta_detection",
// "kappa_containment").
func NewLatencyRecorder(name string) *LatencyRecorder {
	return &LatencyRecorder{name: name}
}

// Record adds one latency sample.
func (r *LatencyRecorder) Record(d time.Duration) {
	r.mu.Lock()
	r.samples = append(r.samples, d)
	r.mu.Unlock()
}

// Since records the elapsed time from start to now — the common call site:
//
//	start := time.Now(); doWork(); rec.Since(start)
func (r *LatencyRecorder) Since(start time.Time) { r.Record(time.Since(start)) }

// Time runs f, records how long it took, and returns f's result duration.
func (r *LatencyRecorder) Time(f func()) time.Duration {
	start := time.Now()
	f()
	d := time.Since(start)
	r.Record(d)
	return d
}

// Dist is a latency distribution in milliseconds (float for sub-ms resolution).
type Dist struct {
	Name   string  `json:"name"`
	Count  int     `json:"count"`
	MinMS  float64 `json:"min_ms"`
	P50MS  float64 `json:"p50_ms"`
	P95MS  float64 `json:"p95_ms"`
	P99MS  float64 `json:"p99_ms"`
	MaxMS  float64 `json:"max_ms"`
	MeanMS float64 `json:"mean_ms"`
}

// Distribution computes the percentile summary over the samples collected so far.
func (r *LatencyRecorder) Distribution() Dist {
	r.mu.Lock()
	s := append([]time.Duration(nil), r.samples...)
	r.mu.Unlock()

	d := Dist{Name: r.name, Count: len(s)}
	if len(s) == 0 {
		return d
	}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	ms := func(x time.Duration) float64 { return float64(x.Nanoseconds()) / 1e6 }

	var sum time.Duration
	for _, x := range s {
		sum += x
	}
	d.MinMS = ms(s[0])
	d.MaxMS = ms(s[len(s)-1])
	d.MeanMS = ms(sum) / float64(len(s))
	d.P50MS = ms(percentile(s, 0.50))
	d.P95MS = ms(percentile(s, 0.95))
	d.P99MS = ms(percentile(s, 0.99))
	return d
}

// Reset clears the collected samples.
func (r *LatencyRecorder) Reset() {
	r.mu.Lock()
	r.samples = r.samples[:0]
	r.mu.Unlock()
}

// percentile returns the q-quantile (0..1) of a sorted slice using the
// nearest-rank method.
func percentile(sorted []time.Duration, q float64) time.Duration {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}
	rank := int(q*float64(len(sorted)-1) + 0.5)
	return sorted[rank]
}
