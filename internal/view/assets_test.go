package view

import (
	"strings"
	"testing"
)

// TestAssetURL pins the cache-busting contract ADR-016 assumes but never
// implemented: static asset URLs carry a version query so the 1-year
// Cache-Control can't strand clients on stale CSS/JS across deploys.
func TestAssetURL(t *testing.T) {
	tests := []struct {
		name       string
		setVersion string // applied via SetAssetVersion when non-empty
		path       string
		wantSuffix string // exact suffix when a version is set
	}{
		{
			name:       "release version stamps the URL",
			setVersion: "v0.5.2",
			path:       "/static/css/app.css",
			wantSuffix: "/static/css/app.css?v=v0.5.2",
		},
		{
			name:       "dirty build versions are query-escaped",
			setVersion: "v0.5.2-3-gabc123 dirty",
			path:       "/static/js/app.js",
			wantSuffix: "/static/js/app.js?v=v0.5.2-3-gabc123+dirty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetAssetVersion(tt.setVersion)
			if got := AssetURL(tt.path); got != tt.wantSuffix {
				t.Errorf("AssetURL(%q) = %q, want %q", tt.path, got, tt.wantSuffix)
			}
		})
	}
}

// TestAssetURL_Defaults: whatever the current stamp is (process-start default
// in dev builds), blank and "dev" pseudo-versions must never erase it.
func TestAssetURL_Defaults(t *testing.T) {
	SetAssetVersion("") // no-op: keeps the current stamp
	got := AssetURL("/static/css/app.css")
	if !strings.HasPrefix(got, "/static/css/app.css?v=") {
		t.Fatalf("AssetURL without version = %q, want a ?v= stamp from process start", got)
	}
	if strings.TrimPrefix(got, "/static/css/app.css?v=") == "" {
		t.Error("default asset version is empty")
	}

	before := AssetURL("/static/js/app.js")
	SetAssetVersion("dev") // ldflags default: not a real version, keep the stamp
	if after := AssetURL("/static/js/app.js"); after != before {
		t.Errorf("SetAssetVersion(\"dev\") changed the URL: %q -> %q", before, after)
	}
}
