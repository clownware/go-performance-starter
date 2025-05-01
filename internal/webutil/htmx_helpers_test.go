package webutil

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsHTMXRequest(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		expected bool
	}{
		{
			name:     "htmx request",
			headers:  map[string]string{"HX-Request": "true"},
			expected: true,
		},
		{
			name:     "non-htmx request",
			headers:  map[string]string{},
			expected: false,
		},
		{
			name:     "htmx request with wrong value",
			headers:  map[string]string{"HX-Request": "false"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _ := http.NewRequest("GET", "/", nil)

			// Set headers
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}

			result := IsHTMXRequest(r)
			if result != tt.expected {
				t.Errorf("IsHTMXRequest() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSetHXRedirect(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedHeader string
	}{
		{
			name:           "basic redirect",
			url:            "/dashboard",
			expectedHeader: "/dashboard",
		},
		{
			name:           "full url redirect",
			url:            "https://example.com/path",
			expectedHeader: "https://example.com/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXRedirect(w, tt.url)

			if w.Header().Get("HX-Redirect") != tt.expectedHeader {
				t.Errorf("SetHXRedirect() header = %v, expected %v",
					w.Header().Get("HX-Redirect"), tt.expectedHeader)
			}
		})
	}
}

func TestSetHXTrigger(t *testing.T) {
	tests := []struct {
		name           string
		event          string
		expectedHeader string
	}{
		{
			name:           "simple event",
			event:          "notification",
			expectedHeader: "notification",
		},
		{
			name:           "complex event",
			event:          "{\"showMessage\": \"Success\"}",
			expectedHeader: "{\"showMessage\": \"Success\"}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXTrigger(w, tt.event)

			if w.Header().Get("HX-Trigger") != tt.expectedHeader {
				t.Errorf("SetHXTrigger() header = %v, expected %v",
					w.Header().Get("HX-Trigger"), tt.expectedHeader)
			}
		})
	}
}

func TestSetHXTriggerAfterSwap(t *testing.T) {
	tests := []struct {
		name           string
		event          string
		expectedHeader string
	}{
		{
			name:           "simple event",
			event:          "notification",
			expectedHeader: "notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXTriggerAfterSwap(w, tt.event)

			if w.Header().Get("HX-Trigger-After-Swap") != tt.expectedHeader {
				t.Errorf("SetHXTriggerAfterSwap() header = %v, expected %v",
					w.Header().Get("HX-Trigger-After-Swap"), tt.expectedHeader)
			}
		})
	}
}

func TestSetHXTriggerAfterSettle(t *testing.T) {
	tests := []struct {
		name           string
		event          string
		expectedHeader string
	}{
		{
			name:           "simple event",
			event:          "notification",
			expectedHeader: "notification",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXTriggerAfterSettle(w, tt.event)

			if w.Header().Get("HX-Trigger-After-Settle") != tt.expectedHeader {
				t.Errorf("SetHXTriggerAfterSettle() header = %v, expected %v",
					w.Header().Get("HX-Trigger-After-Settle"), tt.expectedHeader)
			}
		})
	}
}
