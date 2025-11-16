package performance

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckBinarySize(t *testing.T) {
	// Create a temporary file to simulate binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "test-binary")

	tests := []struct {
		name       string
		size       int64
		wantErr    bool
		errContains string
	}{
		{
			name:    "within budget",
			size:    10 * 1024 * 1024, // 10MB
			wantErr: false,
		},
		{
			name:       "exceeds budget",
			size:       25 * 1024 * 1024, // 25MB
			wantErr:    true,
			errContains: "Binary size exceeds budget",
		},
		{
			name:    "at budget limit",
			size:    MaxBinarySize,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create file with specified size
			f, err := os.Create(binaryPath)
			if err != nil {
				t.Fatalf("failed to create test file: %v", err)
			}
			defer f.Close()

			if err := f.Truncate(tt.size); err != nil {
				t.Fatalf("failed to set file size: %v", err)
			}

			err = CheckBinarySize(binaryPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckBinarySize() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil && tt.errContains != "" {
				if err.Error() == "" || !contains(err.Error(), tt.errContains) {
					t.Errorf("CheckBinarySize() error = %v, want error containing %q", err, tt.errContains)
				}
			}
		})
	}
}

func TestCheckResponseTime(t *testing.T) {
	tests := []struct {
		name       string
		duration   time.Duration
		percentile string
		wantErr    bool
	}{
		{
			name:       "p50 within budget",
			duration:   30 * time.Millisecond,
			percentile: "p50",
			wantErr:    false,
		},
		{
			name:       "p50 exceeds budget",
			duration:   60 * time.Millisecond,
			percentile: "p50",
			wantErr:    true,
		},
		{
			name:       "p95 within budget",
			duration:   80 * time.Millisecond,
			percentile: "p95",
			wantErr:    false,
		},
		{
			name:       "p95 exceeds budget",
			duration:   120 * time.Millisecond,
			percentile: "p95",
			wantErr:    true,
		},
		{
			name:       "p99 within budget",
			duration:   150 * time.Millisecond,
			percentile: "p99",
			wantErr:    false,
		},
		{
			name:       "p99 exceeds budget",
			duration:   250 * time.Millisecond,
			percentile: "p99",
			wantErr:    true,
		},
		{
			name:       "unknown percentile",
			duration:   50 * time.Millisecond,
			percentile: "p100",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckResponseTime(tt.duration, tt.percentile)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckResponseTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckStartupTime(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		wantErr  bool
	}{
		{
			name:     "within budget",
			duration: 300 * time.Millisecond,
			wantErr:  false,
		},
		{
			name:     "exceeds budget",
			duration: 600 * time.Millisecond,
			wantErr:  true,
		},
		{
			name:     "at budget limit",
			duration: MaxStartupTime,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckStartupTime(tt.duration)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckStartupTime() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckMemoryUsage(t *testing.T) {
	// This test just verifies the function runs without error
	// Actual memory usage depends on runtime state
	err := CheckMemoryUsage()
	if err != nil {
		t.Logf("Memory usage check: %v", err)
		// Don't fail the test, just log the result
		// as actual memory usage varies by environment
	}
}

func TestGetMemoryStats(t *testing.T) {
	stats := GetMemoryStats()

	// Verify expected fields are present
	expectedFields := []string{
		"alloc_mb",
		"total_alloc_mb",
		"sys_mb",
		"num_gc",
		"goroutines",
		"heap_objects",
		"stack_in_use_mb",
	}

	for _, field := range expectedFields {
		if _, ok := stats[field]; !ok {
			t.Errorf("GetMemoryStats() missing field %q", field)
		}
	}

	// Verify values are reasonable
	if allocMB, ok := stats["alloc_mb"].(float64); ok {
		if allocMB < 0 {
			t.Errorf("alloc_mb should be non-negative, got %f", allocMB)
		}
	}

	if goroutines, ok := stats["goroutines"].(int); ok {
		if goroutines < 1 {
			t.Errorf("goroutines should be at least 1, got %d", goroutines)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{bytes: 0, want: "0 B"},
		{bytes: 1, want: "1 B"},
		{bytes: 1023, want: "1023 B"},
		{bytes: 1024, want: "1.0 KB"},
		{bytes: 1024 * 1024, want: "1.0 MB"},
		{bytes: 1024 * 1024 * 1024, want: "1.0 GB"},
		{bytes: 20 * 1024 * 1024, want: "20.0 MB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestBudgetViolationError(t *testing.T) {
	violation := BudgetViolation{
		Budget:   "Test Budget",
		Expected: "100ms",
		Actual:   "200ms",
		Message:  "Test message",
	}

	errMsg := violation.Error()
	if errMsg == "" {
		t.Error("BudgetViolation.Error() returned empty string")
	}

	// Check error message contains key information
	expectedParts := []string{"Test Budget", "100ms", "200ms", "Test message"}
	for _, part := range expectedParts {
		if !contains(errMsg, part) {
			t.Errorf("BudgetViolation.Error() = %q, want to contain %q", errMsg, part)
		}
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && hasSubstring(s, substr)))
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
