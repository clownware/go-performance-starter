package performance

import (
	"fmt"
	"os"
	"runtime"
	"time"
)

// Performance budgets as defined in ADR-000
const (
	// Response Time Budgets
	MaxP50ResponseTime = 50 * time.Millisecond
	MaxP95ResponseTime = 100 * time.Millisecond
	MaxP99ResponseTime = 200 * time.Millisecond

	// Resource Budgets
	MaxBinarySize  = 20 * 1024 * 1024       // 20MB
	MaxMemoryUsage = 128 * 1024 * 1024      // 128MB
	MaxPeakMemory  = 256 * 1024 * 1024      // 256MB
	MaxStartupTime = 500 * time.Millisecond // 500ms

	// Frontend Budgets
	MaxJavaScriptSize = 50 * 1024  // 50KB
	MaxCSSSize        = 30 * 1024  // 30KB
	MaxTotalPageSize  = 500 * 1024 // 500KB
)

// BudgetViolation represents a performance budget violation
type BudgetViolation struct {
	Budget   string
	Expected interface{}
	Actual   interface{}
	Message  string
}

// Error implements the error interface
func (v BudgetViolation) Error() string {
	return fmt.Sprintf("Budget violation: %s (expected: %v, actual: %v) - %s",
		v.Budget, v.Expected, v.Actual, v.Message)
}

// CheckBinarySize verifies the compiled binary size is within budget
func CheckBinarySize(binaryPath string) error {
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("failed to stat binary: %w", err)
	}

	size := info.Size()
	if size > MaxBinarySize {
		return BudgetViolation{
			Budget:   "Binary Size",
			Expected: formatBytes(MaxBinarySize),
			Actual:   formatBytes(size),
			Message:  "Binary size exceeds budget. Consider reducing dependencies or enabling more aggressive build flags.",
		}
	}

	return nil
}

// CheckMemoryUsage verifies current memory usage is within budget
func CheckMemoryUsage() error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check RSS (Alloc + Sys approximation)
	currentMem := m.Alloc

	if currentMem > MaxMemoryUsage {
		return BudgetViolation{
			Budget:   "Memory Usage",
			Expected: formatBytes(MaxMemoryUsage),
			Actual:   formatBytes(int64(currentMem)),
			Message:  "Memory usage exceeds budget. Check for memory leaks or excessive allocations.",
		}
	}

	return nil
}

// CheckResponseTime verifies response time is within budget
func CheckResponseTime(duration time.Duration, percentile string) error {
	var budget time.Duration
	var budgetName string

	switch percentile {
	case "p50":
		budget = MaxP50ResponseTime
		budgetName = "P50 Response Time"
	case "p95":
		budget = MaxP95ResponseTime
		budgetName = "P95 Response Time"
	case "p99":
		budget = MaxP99ResponseTime
		budgetName = "P99 Response Time"
	default:
		return fmt.Errorf("unknown percentile: %s", percentile)
	}

	if duration > budget {
		return BudgetViolation{
			Budget:   budgetName,
			Expected: budget,
			Actual:   duration,
			Message:  fmt.Sprintf("Response time exceeds %s budget", percentile),
		}
	}

	return nil
}

// CheckStartupTime verifies application startup time is within budget
func CheckStartupTime(duration time.Duration) error {
	if duration > MaxStartupTime {
		return BudgetViolation{
			Budget:   "Startup Time",
			Expected: MaxStartupTime,
			Actual:   duration,
			Message:  "Application startup time exceeds budget. Check for expensive initialization.",
		}
	}

	return nil
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
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

// GetMemoryStats returns current memory statistics
func GetMemoryStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"alloc_mb":        float64(m.Alloc) / 1024 / 1024,
		"total_alloc_mb":  float64(m.TotalAlloc) / 1024 / 1024,
		"sys_mb":          float64(m.Sys) / 1024 / 1024,
		"num_gc":          m.NumGC,
		"goroutines":      runtime.NumGoroutine(),
		"heap_objects":    m.HeapObjects,
		"stack_in_use_mb": float64(m.StackInuse) / 1024 / 1024,
	}
}
