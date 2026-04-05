package view

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
			for key, value := range tt.headers {
				r.Header.Set(key, value)
			}
			if result := IsHTMXRequest(r); result != tt.expected {
				t.Errorf("IsHTMXRequest() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestSetHXRedirect(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{name: "basic redirect", url: "/dashboard"},
		{name: "full url redirect", url: "https://example.com/path"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXRedirect(w, tt.url)
			if got := w.Header().Get("HX-Redirect"); got != tt.url {
				t.Errorf("SetHXRedirect() header = %v, expected %v", got, tt.url)
			}
		})
	}
}

func TestSetHXTrigger(t *testing.T) {
	tests := []struct {
		name  string
		event string
	}{
		{name: "simple event", event: "notification"},
		{name: "complex event", event: `{"showMessage": "Success"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			SetHXTrigger(w, tt.event)
			if got := w.Header().Get("HX-Trigger"); got != tt.event {
				t.Errorf("SetHXTrigger() header = %v, expected %v", got, tt.event)
			}
		})
	}
}
