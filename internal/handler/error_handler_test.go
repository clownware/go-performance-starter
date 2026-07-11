package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONError(t *testing.T) {
	const secret = "pq: relation \"users\" violates row-level security policy"

	tests := []struct {
		name       string
		status     int
		err        error
		wantInBody string
		wantNoLeak bool // body must not contain the internal error text
	}{
		{
			name:       "client error is echoed",
			status:     http.StatusBadRequest,
			err:        errors.New("missing userID"),
			wantInBody: "missing userID",
		},
		{
			name:       "server error is genericized",
			status:     http.StatusInternalServerError,
			err:        errors.New(secret),
			wantInBody: http.StatusText(http.StatusInternalServerError),
			wantNoLeak: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			JSONError(rec, tt.status, tt.err)

			if rec.Code != tt.status {
				t.Errorf("status = %d, want %d", rec.Code, tt.status)
			}

			var body APIError
			if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if !strings.Contains(body.Error, tt.wantInBody) {
				t.Errorf("body = %q, want to contain %q", body.Error, tt.wantInBody)
			}
			if tt.wantNoLeak && strings.Contains(body.Error, secret) {
				t.Errorf("body leaked internal error: %q", body.Error)
			}
		})
	}
}
