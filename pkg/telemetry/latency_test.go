package telemetry

import (
	"testing"
	"time"
)

func TestLatencyPercentiles(t *testing.T) {
	r := NewLatencyRecorder("t")
	// 1..100 ms
	for i := 1; i <= 100; i++ {
		r.Record(time.Duration(i) * time.Millisecond)
	}
	d := r.Distribution()
	if d.Count != 100 {
		t.Fatalf("count=%d want 100", d.Count)
	}
	// nearest-rank: p50≈50, p95≈95, p99≈99, min 1, max 100
	check := func(name string, got, want float64) {
		if got < want-1.5 || got > want+1.5 {
			t.Errorf("%s=%.1f want ~%.1f", name, got, want)
		}
	}
	check("p50", d.P50MS, 50)
	check("p95", d.P95MS, 95)
	check("p99", d.P99MS, 99)
	check("min", d.MinMS, 1)
	check("max", d.MaxMS, 100)
	check("mean", d.MeanMS, 50.5)
}

func TestLatencyEmpty(t *testing.T) {
	if d := NewLatencyRecorder("e").Distribution(); d.Count != 0 || d.P50MS != 0 {
		t.Errorf("empty recorder should give zero dist, got %+v", d)
	}
}

func TestLatencySince(t *testing.T) {
	r := NewLatencyRecorder("s")
	start := time.Now()
	time.Sleep(2 * time.Millisecond)
	r.Since(start)
	if d := r.Distribution(); d.Count != 1 || d.P50MS < 1 {
		t.Errorf("Since should record a positive sample, got %+v", d)
	}
}
