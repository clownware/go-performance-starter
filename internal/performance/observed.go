package performance

import (
	"os"
	"runtime"
	"runtime/metrics"
	"slices"
	"sync"
	"time"
)

// This file is the observation side of ADR-000 (ADR-034): what the running
// process can measure about itself, so the landing page shows budget vs.
// observed instead of the budget alone. Everything here is per-process and
// since boot — a live reading, not an SLO — and every value has an honest
// zero meaning "not measured", which the UI must render as such.

// Class is the ADR-000 §1 classification of a budget.
type Class string

const (
	ClassEnforced     Class = "Enforced"     // task ci fails on violation
	ClassMonitored    Class = "Monitored"    // measured; informs, does not gate
	ClassAspirational Class = "Aspirational" // no measurement exists yet
)

// DefaultLatencyWindow is how many recent requests the percentiles cover.
const DefaultLatencyWindow = 4096

// LatencyCacheTTL is how long Snapshot reuses the latency percentiles.
// Ranking the window sorts up to DefaultLatencyWindow durations, which cost
// more than the rest of a landing-page view (#180); a reading at most a
// second old is still a reading of the recent window.
const LatencyCacheTTL = time.Second

// LatencyPercentiles is a nearest-rank summary of the recent window.
type LatencyPercentiles struct {
	P50, P95, P99 time.Duration
	Samples       int // 0 means nothing has been measured
}

// LatencyRecorder keeps the last N request durations in a ring so the
// percentiles reflect recent traffic on this process rather than its whole
// lifetime. Safe for concurrent use.
type LatencyRecorder struct {
	mu      sync.Mutex
	samples []time.Duration
	next    int
	filled  int
}

// NewLatencyRecorder returns a recorder covering the last size requests.
func NewLatencyRecorder(size int) *LatencyRecorder {
	if size < 1 {
		size = 1
	}
	return &LatencyRecorder{samples: make([]time.Duration, size)}
}

// Record adds one request duration, evicting the oldest once the ring is full.
func (r *LatencyRecorder) Record(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.samples[r.next] = d
	r.next = (r.next + 1) % len(r.samples)
	if r.filled < len(r.samples) {
		r.filled++
	}
}

// Len returns how many durations the window holds, without ranking them.
func (r *LatencyRecorder) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.filled
}

// Percentiles returns nearest-rank p50/p95/p99 over the recorded window.
func (r *LatencyRecorder) Percentiles() LatencyPercentiles {
	r.mu.Lock()
	sorted := make([]time.Duration, r.filled)
	copy(sorted, r.samples[:r.filled])
	r.mu.Unlock()

	n := len(sorted)
	if n == 0 {
		return LatencyPercentiles{}
	}
	slices.Sort(sorted)
	rank := func(p float64) time.Duration {
		idx := min(max(int(float64(n)*p+0.999999)-1, 0), n-1) // ceil(p*n) - 1, nearest-rank, clamped
		return sorted[idx]
	}
	return LatencyPercentiles{P50: rank(0.50), P95: rank(0.95), P99: rank(0.99), Samples: n}
}

// Snapshot is one reading of the process against the ADR-000 budgets. Zero
// values mean "not measured" (no requests yet, startup not recorded, assets
// or executable not found) — never "zero and therefore passing".
type Snapshot struct {
	Uptime     time.Duration
	Goroutines int

	// Request latency over the recent window (see DefaultLatencyWindow).
	Samples       int
	P50, P95, P99 time.Duration

	// Memory from runtime/metrics, with the meaning of the runtime.MemStats
	// fields it replaced (#180): heap in use (HeapInuse), total memory mapped
	// by the runtime (Sys), and the high-water mark of the latter since boot.
	HeapInUse, Sys, PeakSys uint64

	// Startup is process start → listening socket, recorded once by main.
	Startup time.Duration

	// BinarySize is the running executable's size on disk.
	BinarySize int64

	// Gzipped totals of the shipped assets (same lists the CI gate uses).
	JSGzipped, CSSGzipped int64
}

// Observer aggregates the per-process measurements. One lives in Default,
// fed by the metrics middleware and main; tests construct their own.
type Observer struct {
	latency    *LatencyRecorder
	readMemory func() (heapInUse, sys uint64) // injected by tests
	now        func() time.Time               // injected by tests
	startedAt  time.Time
	staticDir  string

	mu      sync.Mutex
	startup time.Duration
	peakSys uint64

	assetsOnce sync.Once
	jsGz       int64
	cssGz      int64

	binaryOnce sync.Once
	binarySize int64

	latencyMu     sync.Mutex
	latencyAt     time.Time
	latencyCached LatencyPercentiles
}

// Default is the process-wide observer.
var Default = NewObserver("web/static")

// NewObserver returns an observer measuring the shipped assets under
// staticDir (the directory the file server serves).
func NewObserver(staticDir string) *Observer {
	return &Observer{
		latency:    NewLatencyRecorder(DefaultLatencyWindow),
		readMemory: readMemory,
		now:        time.Now,
		startedAt:  time.Now(),
		staticDir:  staticDir,
	}
}

// RecordLatency adds one completed request's duration.
func (o *Observer) RecordLatency(d time.Duration) {
	o.latency.Record(d)
}

// LatencySamples returns how many request durations the window holds now,
// bypassing the LatencyCacheTTL cache Snapshot reads through.
func (o *Observer) LatencySamples() int {
	return o.latency.Len()
}

// RecordStartup records process start → listening. Startup happens once, so
// the first call wins and later calls are ignored.
func (o *Observer) RecordStartup(d time.Duration) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.startup == 0 && d > 0 {
		o.startup = d
	}
}

// memoryMetrics are the runtime/metrics samples behind the memory row.
// Objects plus unused is MemStats.HeapInuse; total is MemStats.Sys (the sum
// of every /memory/classes metric).
var memoryMetrics = [...]string{
	"/memory/classes/heap/objects:bytes",
	"/memory/classes/heap/unused:bytes",
	"/memory/classes/total:bytes",
}

// readMemory reads heap in use and total mapped memory via runtime/metrics.
// Unlike runtime.ReadMemStats it does not stop the world, so the landing page
// can read it on every view without pausing other requests (#180). A metric
// this Go version does not support reads as zero ("not measured").
func readMemory() (heapInUse, sys uint64) {
	var samples [len(memoryMetrics)]metrics.Sample
	for i, name := range memoryMetrics {
		samples[i].Name = name
	}
	metrics.Read(samples[:])
	value := func(s metrics.Sample) uint64 {
		if s.Value.Kind() != metrics.KindUint64 {
			return 0
		}
		return s.Value.Uint64()
	}
	return value(samples[0]) + value(samples[1]), value(samples[2])
}

// ObserveMemory reads the runtime memory metrics and advances the high-water
// mark; the periodic metrics collector calls it so the peak is tracked even
// when nobody is looking at the page.
func (o *Observer) ObserveMemory() (heapInUse, sys, peakSys uint64) {
	heapInUse, sys = o.readMemory()
	o.mu.Lock()
	defer o.mu.Unlock()
	if sys > o.peakSys {
		o.peakSys = sys
	}
	return heapInUse, sys, o.peakSys
}

// percentiles returns the latency percentiles, re-ranking the window at most
// once per LatencyCacheTTL. Holding latencyMu while ranking means concurrent
// views that find the cache stale wait for one sort instead of each doing one.
func (o *Observer) percentiles() LatencyPercentiles {
	o.latencyMu.Lock()
	defer o.latencyMu.Unlock()
	now := o.now()
	if o.latencyAt.IsZero() || now.Sub(o.latencyAt) >= LatencyCacheTTL {
		o.latencyCached = o.latency.Percentiles()
		o.latencyAt = now
	}
	return o.latencyCached
}

// Snapshot takes one reading. Asset and executable sizes are measured on the
// first call and cached: they cannot change while the process runs. Latency
// percentiles are cached for LatencyCacheTTL.
func (o *Observer) Snapshot() Snapshot {
	heap, sys, peak := o.ObserveMemory()
	p := o.percentiles()

	o.assetsOnce.Do(func() {
		if js, err := GzippedTotal(ShippedAssetPaths(o.staticDir, ShippedJS)...); err == nil {
			o.jsGz = js
		}
		if css, err := GzippedTotal(ShippedAssetPaths(o.staticDir, ShippedCSS)...); err == nil {
			o.cssGz = css
		}
	})
	o.binaryOnce.Do(func() {
		if exe, err := os.Executable(); err == nil {
			if info, err := os.Stat(exe); err == nil {
				o.binarySize = info.Size()
			}
		}
	})

	o.mu.Lock()
	startup := o.startup
	o.mu.Unlock()

	return Snapshot{
		Uptime:     time.Since(o.startedAt),
		Goroutines: runtime.NumGoroutine(),
		Samples:    p.Samples,
		P50:        p.P50,
		P95:        p.P95,
		P99:        p.P99,
		HeapInUse:  heap,
		Sys:        sys,
		PeakSys:    peak,
		Startup:    startup,
		BinarySize: o.binarySize,
		JSGzipped:  o.jsGz,
		CSSGzipped: o.cssGz,
	}
}
