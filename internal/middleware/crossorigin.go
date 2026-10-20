package middleware

import (
	"log/slog"
	"net/http"
)

// CrossOrigin rejects cross-origin unsafe requests using the browser's
// Sec-Fetch-Site / Origin headers (http.CrossOriginProtection, Go 1.25).
// It runs in front of CSRF as defence-in-depth (ADR-035 Phase 1): an unsafe
// request must pass both. No trusted origins and no bypass patterns are
// configured; adrcheck's adr035-no-cop-bypass keeps it that way.
//
// Rejections answer 403 and are logged with the method, path and the two
// headers the decision was based on — never cookies.
func CrossOrigin() func(http.Handler) http.Handler {
	cop := http.NewCrossOriginProtection()
	cop.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.WarnContext(r.Context(), "cross-origin request rejected",
			"method", r.Method,
			"path", r.URL.Path,
			"sec_fetch_site", r.Header.Get("Sec-Fetch-Site"),
			"origin", r.Header.Get("Origin"),
		)
		http.Error(w, "Forbidden", http.StatusForbidden)
	}))
	return cop.Handler
}
