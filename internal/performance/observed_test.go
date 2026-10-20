package performance

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestLatencyRecorder_Percentiles pins the nearest-rank percentiles over a
// bounded ring: an empty recorder reports nothing (the UI must say
// "unmeasured", not "0ms"), a full sequence yields the textbook ranks, and
// once the ring wraps only the most recent window counts.
func TestLatencyRecorder_Percentiles(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		samples []time.Duration
		want    LatencyPercentiles
	}{
		{
			name: "empty recorder reports zero samples",
			size: 8,
			want: LatencyPercentiles{},
		},
		{
			name:    "single sample is every percentile",
			size:    8,
			samples: []time.Duration{7 * time.Millisecond},
			want:    LatencyPercentiles{P50: 7 * time.Millisecond, P95: 7 * time.Millisecond, P99: 7 * time.Millisecond, Samples: 1},
		},
		{
			name:    "1..100ms yields nearest-rank 50/95/99",
			size:    100,
			samples: ramp(100),
			want:    LatencyPercentiles{P50: 50 * time.Millisecond, P95: 95 * time.Millisecond, P99: 99 * time.Millisecond, Samples: 100},
		},
		{
			name:    "ring keeps only the newest window",
			size:    10,
			samples: ramp(100), // last ten are 91..100ms
			want:    LatencyPercentiles{P50: 95 * time.Millisecond, P95: 100 * time.Millisecond, P99: 100 * time.Millisecond, Samples: 10},
		},
		{
			name:    "unsorted input is sorted before ranking",
			size:    8,
			samples: []time.Duration{9 * time.Millisecond, 1 * time.Millisecond, 5 * time.Millisecond, 3 * time.Millisecond},
			want:    LatencyPercentiles{P50: 3 * time.Millisecond, P95: 9 * time.Millisecond, P99: 9 * time.Millisecond, Samples: 4},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewLatencyRecorder(tt.size)
			for _, d := range tt.samples {
				r.Record(d)
			}
			if got := r.Len(); got != tt.want.Samples {
				t.Errorf("Len() = %d, want %d", got, tt.want.Samples)
			}
			if got := r.Percentiles(); got != tt.want {
				t.Errorf("Percentiles() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func ramp(n int) []time.Duration {
	out := make([]time.Duration, 0, n)
	for i := 1; i <= n; i++ {
		out = append(out, time.Duration(i)*time.Millisecond)
	}
	return out
}

// TestObserver_Snapshot covers what the process can measure about itself:
// shipped-asset sizes from the static dir (gzipped, same lists CI gates),
// a startup duration recorded once, a memory high-water mark that never
// falls, and honest zeros when a measurement is unavailable.
func TestObserver_Snapshot(t *testing.T) {
	staticDir := t.TempDir()
	mustWrite(t, filepath.Join(staticDir, "js", "htmx.min.js"), 4000)
	mustWrite(t, filepath.Join(staticDir, "js", "alpine.min.js"), 3000)
	mustWrite(t, filepath.Join(staticDir, "js", "app.js"), 500)
	mustWrite(t, filepath.Join(staticDir, "css", "app.css"), 2500)

	o := NewObserver(staticDir)
	o.RecordStartup(120 * time.Millisecond)
	o.RecordStartup(900 * time.Millisecond) // second call is ignored: startup happens once
	o.RecordLatency(20 * time.Millisecond)
	o.RecordLatency(40 * time.Millisecond)

	snap := o.Snapshot()
	if snap.Startup != 120*time.Millisecond {
		t.Errorf("Startup = %v, want 120ms (first RecordStartup wins)", snap.Startup)
	}
	if snap.Samples != 2 || snap.P50 != 20*time.Millisecond || snap.P99 != 40*time.Millisecond {
		t.Errorf("latency snapshot = p50 %v p99 %v n=%d, want p50 20ms p99 40ms n=2", snap.P50, snap.P99, snap.Samples)
	}
	if snap.JSGzipped <= 0 || snap.JSGzipped >= 7500 {
		t.Errorf("JSGzipped = %d, want a gzipped total below the 7500 raw bytes", snap.JSGzipped)
	}
	if snap.CSSGzipped <= 0 || snap.CSSGzipped >= 2500 {
		t.Errorf("CSSGzipped = %d, want a gzipped total below the 2500 raw bytes", snap.CSSGzipped)
	}
	if snap.BinarySize <= 0 {
		t.Errorf("BinarySize = %d, want the running executable's size", snap.BinarySize)
	}
	if snap.Sys == 0 || snap.HeapInUse == 0 {
		t.Errorf("memory snapshot = sys %d heap %d, want non-zero", snap.Sys, snap.HeapInUse)
	}
	if snap.PeakSys < snap.Sys {
		t.Errorf("PeakSys = %d < Sys = %d; the high-water mark must never trail the current value", snap.PeakSys, snap.Sys)
	}
	if snap.Uptime <= 0 || snap.Goroutines <= 0 {
		t.Errorf("Uptime = %v, Goroutines = %d, want positive", snap.Uptime, snap.Goroutines)
	}

	// Missing assets: zeros, no panic, everything else still measured.
	empty := NewObserver(filepath.Join(staticDir, "nope"))
	es := empty.Snapshot()
	if es.JSGzipped != 0 || es.CSSGzipped != 0 {
		t.Errorf("missing static dir: JS %d CSS %d, want 0 (unmeasured)", es.JSGzipped, es.CSSGzipped)
	}
	if es.Startup != 0 || es.Samples != 0 {
		t.Errorf("fresh observer: Startup %v Samples %d, want zero (unmeasured)", es.Startup, es.Samples)
	}
}

func mustWrite(t *testing.T, path string, size int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, size)
	for i := range buf {
		buf[i] = byte('a' + i%26)
	}
	if err := os.WriteFile(path, buf, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestShippedAssetPaths pins that the runtime observer and the CI gate
// measure the same files: one list, joined onto whichever static root the
// caller has.
func TestShippedAssetPaths(t *testing.T) {
	got := ShippedAssetPaths("web/static", ShippedJS)
	want := []string{"web/static/js/htmx.min.js", "web/static/js/alpine.min.js", "web/static/js/app.js"}
	if len(got) != len(want) {
		t.Fatalf("ShippedAssetPaths = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ShippedAssetPaths[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// TestObserver_ObserveMemory pins the high-water mark over a sequence of
// readings: the peak follows Sys up, never falls when Sys does, and heap and
// sys are passed through as read. The reader is injected so the sequence is
// exact; TestReadMemory checks the real reader against the runtime.
func TestObserver_ObserveMemory(t *testing.T) {
	type reading struct{ heap, sys uint64 }
	tests := []struct {
		name     string
		readings []reading
		wantHeap uint64
		wantSys  uint64
		wantPeak uint64
	}{
		{
			name:     "single reading is its own peak",
			readings: []reading{{heap: 10, sys: 40}},
			wantHeap: 10, wantSys: 40, wantPeak: 40,
		},
		{
			name:     "peak follows sys up",
			readings: []reading{{10, 40}, {12, 50}, {15, 70}},
			wantHeap: 15, wantSys: 70, wantPeak: 70,
		},
		{
			name:     "peak holds when sys falls",
			readings: []reading{{10, 40}, {30, 90}, {8, 35}},
			wantHeap: 8, wantSys: 35, wantPeak: 90,
		},
		{
			name:     "unsupported metrics read as zero and leave the peak alone",
			readings: []reading{{10, 40}, {0, 0}},
			wantHeap: 0, wantSys: 0, wantPeak: 40,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewObserver(t.TempDir())
			i := 0
			o.readMemory = func() (uint64, uint64) {
				r := tt.readings[i]
				i++
				return r.heap, r.sys
			}
			var heap, sys, peak uint64
			for range tt.readings {
				heap, sys, peak = o.ObserveMemory()
			}
			if heap != tt.wantHeap || sys != tt.wantSys || peak != tt.wantPeak {
				t.Errorf("ObserveMemory() = heap %d sys %d peak %d, want %d %d %d",
					heap, sys, peak, tt.wantHeap, tt.wantSys, tt.wantPeak)
			}
		})
	}
}

// TestReadMemory pins that the runtime/metrics reading (no stop-the-world)
// means what the runtime.MemStats fields it replaced meant (#180): heap in
// use is HeapInuse and total mapped memory is Sys. Both are read back to back
// after a GC, so they agree to within a small allocation drift.
func TestReadMemory(t *testing.T) {
	runtime.GC()
	heap, sys := readMemory()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	const tolerance = 1 << 20 // 1 MiB of drift between the two reads
	tests := []struct {
		name      string
		got, want uint64
	}{
		{name: "heap in use matches MemStats.HeapInuse", got: heap, want: m.HeapInuse},
		{name: "total mapped matches MemStats.Sys", got: sys, want: m.Sys},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got == 0 {
				t.Fatal("reading is zero: a runtime/metrics name is unsupported by this Go version")
			}
			diff := max(tt.got, tt.want) - min(tt.got, tt.want)
			if diff > tolerance {
				t.Errorf("runtime/metrics = %d, MemStats = %d, differ by %d (> %d)", tt.got, tt.want, diff, tolerance)
			}
		})
	}
}

// BenchmarkObserveMemory compares the per-view cost of the memory reading
// before and after #180: runtime.ReadMemStats stops the world; runtime/metrics
// does not. Uses b.Loop (Go 1.24).
func BenchmarkObserveMemory(b *testing.B) {
	b.Run("ReadMemStats", func(b *testing.B) {
		var m runtime.MemStats
		for b.Loop() {
			runtime.ReadMemStats(&m)
		}
	})
	b.Run("runtime-metrics", func(b *testing.B) {
		o := NewObserver(b.TempDir())
		b.ReportAllocs()
		for b.Loop() {
			o.ObserveMemory()
		}
	})
}

// TestObserver_LatencyCache pins the percentile cache: sorting the 4096-sample
// window on every landing-page view dominated its cost once the memory read
// stopped stopping the world (#180), so Snapshot reuses percentiles for
// LatencyCacheTTL. A reading inside the TTL is the cached one; at or after
// it the window is re-ranked.
func TestObserver_LatencyCache(t *testing.T) {
	tests := []struct {
		name        string
		elapsed     time.Duration // between the first and second Snapshot
		wantSamples int
		wantP99     time.Duration
	}{
		{name: "same instant reuses the cached percentiles", elapsed: 0, wantSamples: 1, wantP99: 10 * time.Millisecond},
		{name: "inside the TTL reuses them", elapsed: LatencyCacheTTL - time.Nanosecond, wantSamples: 1, wantP99: 10 * time.Millisecond},
		{name: "at the TTL re-ranks", elapsed: LatencyCacheTTL, wantSamples: 11, wantP99: 50 * time.Millisecond},
		{name: "well past the TTL re-ranks", elapsed: time.Minute, wantSamples: 11, wantP99: 50 * time.Millisecond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := NewObserver(t.TempDir())
			now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
			o.now = func() time.Time { return now }

			o.RecordLatency(10 * time.Millisecond)
			if first := o.Snapshot(); first.Samples != 1 {
				t.Fatalf("first Snapshot Samples = %d, want 1", first.Samples)
			}
			for range 10 {
				o.RecordLatency(50 * time.Millisecond)
			}
			now = now.Add(tt.elapsed)
			got := o.Snapshot()
			if got.Samples != tt.wantSamples || got.P99 != tt.wantP99 {
				t.Errorf("second Snapshot = n=%d p99 %v, want n=%d p99 %v", got.Samples, got.P99, tt.wantSamples, tt.wantP99)
			}
		})
	}
}
