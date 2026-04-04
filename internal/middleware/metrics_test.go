package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResponseWriter_WriteHeader(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{name: "200 OK", statusCode: http.StatusOK},
		{name: "404 Not Found", statusCode: http.StatusNotFound},
		{name: "500 Internal Server Error", statusCode: http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			rw.WriteHeader(tt.statusCode)

			if rw.statusCode != tt.statusCode {
				t.Errorf("statusCode = %d, want %d", rw.statusCode, tt.statusCode)
			}
			if w.Code != tt.statusCode {
				t.Errorf("underlying ResponseWriter code = %d, want %d", w.Code, tt.statusCode)
			}
		})
	}
}

func TestResponseWriter_Write(t *testing.T) {
	w := httptest.NewRecorder()
	rw := &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}

	data := []byte("hello world")
	n, err := rw.Write(data)
	if err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if n != len(data) {
		t.Errorf("Write() returned %d bytes, want %d", n, len(data))
	}
	if rw.bytesWritten != len(data) {
		t.Errorf("bytesWritten = %d, want %d", rw.bytesWritten, len(data))
	}

	// Write again to test accumulation
	n2, err := rw.Write(data)
	if err != nil {
		t.Fatalf("second Write() returned error: %v", err)
	}
	if rw.bytesWritten != len(data)+n2 {
		t.Errorf("bytesWritten after two writes = %d, want %d", rw.bytesWritten, len(data)+n2)
	}
}

func TestGetRoutePattern_NoChiContext(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/test", nil)

	// Without chi route context, should fall back to URL path
	pattern := getRoutePattern(req)
	if pattern != "/api/test" {
		t.Errorf("getRoutePattern() = %q, want %q", pattern, "/api/test")
	}
}

func TestMetrics_HandlerChain(t *testing.T) {
	// Verify that the Metrics middleware passes through to the next handler
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	handler := Metrics(next)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if !called {
		t.Error("Metrics middleware did not call next handler")
	}
	if w.Code != http.StatusOK {
		t.Errorf("response status = %d, want %d", w.Code, http.StatusOK)
	}
}
